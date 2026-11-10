package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type PaymentRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.VendorPayment, error)
	// ListByProjects backs ComputeProgressBatch: every matching row across
	// the given projects in one query (WHERE project_id IN (...)).
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.VendorPayment, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.VendorPayment, error)
	Create(ctx context.Context, p *domain.VendorPayment) error
	// Patch writes the changed fields of a live payment (compare-and-set on
	// updated_at when expected is given; see infrastructure.patchRow).
	Patch(ctx context.Context, p *domain.VendorPayment, changed []string, expected *time.Time) (bool, error)
	// SoftDelete hides a live payment (deleted_at = at); false when there was none.
	SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error)
	// FindDeletedByID is the restore lookup (nil unless the payment waits in the
	// "deleted" state).
	FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.VendorPayment, error)
	// Restore clears the stamp of a payment deleted after notBefore.
	Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error)
}

type PaymentService struct {
	repo      PaymentRepository
	evidence  *EvidenceService
	activity  *ActivityService
	contracts ContractReader
	proof     *ProofSupport
}

// SetProofSupport wires what CreateWithProof needs (transaction, upload checks).
func (s *PaymentService) SetProofSupport(p *ProofSupport) { s.proof = p }

func NewPaymentService(repo PaymentRepository, evidence *EvidenceService, activity *ActivityService, contracts ContractReader) *PaymentService {
	return &PaymentService{repo: repo, evidence: evidence, activity: activity, contracts: contracts}
}

// checkChange refuses an add, edit or delete that would take the total paid
// to one vendor engagement past that engagement's agreed price.
func (s *PaymentService) checkChange(ctx context.Context, projectID, projectVendorID, origID int64, next *domain.PaymentLine) error {
	contract, err := s.contracts.EngagementContractValue(ctx, projectID, projectVendorID)
	if err != nil {
		return err
	}
	all, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	var before, after []domain.PaymentLine
	for _, p := range all {
		if p.ProjectVendorID != projectVendorID {
			continue
		}
		line := domain.PaymentLine{Type: p.Type, Amount: p.Amount}
		before = append(before, line)
		if p.ID != origID {
			after = append(after, line)
		}
	}
	if next != nil {
		after = append(after, *next)
	}
	return checkChange(before, after, contract, "harga vendor", next == nil)
}

func (s *PaymentService) List(ctx context.Context, projectID int64) ([]domain.VendorPayment, error) {
	return s.repo.ListByProject(ctx, projectID)
}

type PaymentInput struct {
	ProjectVendorID int64
	Type            domain.PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}

func (s *PaymentService) Create(ctx context.Context, projectID int64, actorStaffID int64, input PaymentInput) (*domain.VendorPayment, error) {
	if err := checkAmount(input.Amount); err != nil {
		return nil, err
	}
	if err := s.checkChange(ctx, projectID, input.ProjectVendorID, 0, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
		return nil, err
	}
	p := &domain.VendorPayment{
		ProjectID: projectID, ProjectVendorID: input.ProjectVendorID, Type: input.Type, Amount: input.Amount,
		PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "vendor_payment", formatID(p.ID), string(p.Type),
		"Pembayaran vendor dicatat")
	return p, nil
}

// Exists is nil when the vendor payment is on this project and not deleted, a not
// found otherwise. The PATCH handler asks it before it answers a missing
// If-Match (428), so a payment that is not there is a 404 with or without the header.
func (s *PaymentService) Exists(ctx context.Context, projectID, id int64) error {
	p, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return err
	}
	if p == nil {
		return apperror.NotFound("Pembayaran tidak ditemukan")
	}
	return nil
}

// Patch changes the fields of a vendor payment that were sent, and only those. No
// status/role guard (only Delete escalates to Owner-only, PLAN.md §2.3).
// ProjectVendorID is deliberately not patchable (PLAN.md §2.5): re-parenting a
// payment to another engagement is not supported; delete and recreate instead.
// The overpayment rule is judged on the merged type and amount (and only when one
// of them changes, so an old overpaid row can still get a new note). Runs under the
// project lock like every money write, so two edits cannot overpay together. A
// patch that changes nothing writes nothing; with `expected` the write is a
// compare-and-set on updated_at (a stale one is a 409 carrying the current payment).
func (s *PaymentService) Patch(ctx context.Context, projectID, id int64, actorStaffID int64, patch PaymentPatch, expected *time.Time) (*domain.VendorPayment, error) {
	if err := checkPaymentPatch(patch); err != nil {
		return nil, err
	}
	var out *domain.VendorPayment
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.repo.FindByID(ctx, projectID, id)
		if err != nil {
			return err
		}
		if p == nil {
			return apperror.NotFound("Pembayaran tidak ditemukan")
		}
		loaded := *p
		var ch changeSet
		applyPayment(&ch, patch, &p.Type, &p.Amount, &p.PaymentDate, &p.Method, &p.ReferenceNumber, &p.Notes)
		if !ch.any() {
			out = p
			return nil
		}
		if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
			return err
		}
		if ch.moneyChanged() {
			if err := s.checkChange(ctx, projectID, p.ProjectVendorID, id, &domain.PaymentLine{Type: p.Type, Amount: p.Amount}); err != nil {
				return err
			}
		}
		wrote, err := s.repo.Patch(ctx, p, ch.keys, expected)
		if err != nil {
			return err
		}
		fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.VendorPayment, error) {
			return s.repo.FindByID(ctx, projectID, id)
		}, func(p *domain.VendorPayment) time.Time { return p.UpdatedAt }, apperror.NotFound("Pembayaran tidak ditemukan"))
		if err != nil {
			return err
		}
		out = fresh
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "vendor_payment", formatID(fresh.ID), string(fresh.Type),
			"Pembayaran vendor diperbarui: "+ch.summary())
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Delete soft-deletes the payment (ADR-0034): it leaves the list, the vendor's
// paid total and progress, and its proof files are hidden with it, until it is
// restored (24 hours) or purged. The permission check lives in the HTTP handler.
// It returns the deleted payment (the handler needs its engagement for the
// ledger) and the instant restoring stops being possible.
func (s *PaymentService) Delete(ctx context.Context, projectID, id int64, actorStaffID int64, now time.Time) (*domain.VendorPayment, time.Time, error) {
	var deleted *domain.VendorPayment
	var until time.Time
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.repo.FindByID(ctx, projectID, id)
		if err != nil {
			return err
		}
		if p == nil {
			return apperror.NotFound("Pembayaran tidak ditemukan")
		}
		if err := s.checkChange(ctx, projectID, p.ProjectVendorID, id, nil); err != nil {
			return err
		}
		ok, err := s.repo.SoftDelete(ctx, projectID, id, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperror.NotFound("Pembayaran tidak ditemukan")
		}
		deleted, until = p, domain.RestorableUntil(now)
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentDeleted, actorStaffID, "vendor_payment", formatID(p.ID), string(p.Type),
			"Pembayaran vendor dihapus")
		return nil
	})
	return deleted, until, err
}

// Restore brings a deleted vendor payment back, after re-checking what can now
// fail: its vendor engagement must still exist (422 on `projectVendorId`:
// restore the vendor first) and the payment must not take the engagement past
// its agreed price (422 on `amount`). Not deleted, or past the retention: 404.
func (s *PaymentService) Restore(ctx context.Context, projectID, id int64, actorStaffID int64, now time.Time) (*domain.VendorPayment, error) {
	var restored *domain.VendorPayment
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.repo.FindDeletedByID(ctx, projectID, id)
		if err != nil {
			return err
		}
		if p == nil || p.DeletedAt == nil || !domain.CanRestore(*p.DeletedAt, now) {
			return errNotRestorable()
		}
		if _, err := s.contracts.EngagementContractValue(ctx, projectID, p.ProjectVendorID); err != nil {
			if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
				msg := "Kerja sama vendor untuk pembayaran ini sudah dihapus. Pulihkan vendornya lebih dulu."
				return apperror.Validation(msg, map[string][]string{"projectVendorId": {msg}})
			}
			return err
		}
		if err := s.checkChange(ctx, projectID, p.ProjectVendorID, 0, &domain.PaymentLine{Type: p.Type, Amount: p.Amount}); err != nil {
			return err
		}
		ok, err := s.repo.Restore(ctx, projectID, id, domain.PurgeCutoff(now))
		if err != nil {
			return err
		}
		if !ok {
			return errNotRestorable()
		}
		p.DeletedAt = nil
		restored = p
		s.activity.Record(ctx, &projectID, domain.ActivityRestored, actorStaffID, "vendor_payment", formatID(p.ID), string(p.Type),
			"Pembayaran vendor dipulihkan")
		return nil
	})
	return restored, err
}

// LedgerSummary is where one engagement's agreed price stands against its live
// payments (the `ledgerSummary` of a mutation response).
func (s *PaymentService) LedgerSummary(ctx context.Context, projectID, projectVendorID int64) (domain.Ledger, error) {
	contract, err := s.contracts.EngagementContractValue(ctx, projectID, projectVendorID)
	if err != nil {
		return domain.Ledger{}, err
	}
	all, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return domain.Ledger{}, err
	}
	var lines []domain.PaymentLine
	for _, q := range all {
		if q.ProjectVendorID == projectVendorID {
			lines = append(lines, lineOf(q.Type, q.Amount))
		}
	}
	return domain.ComputeLedger(lines, contract), nil
}

// VendorPaymentCreated is the outcome of recording a vendor payment: the
// payment, its proof files and where that engagement's price stands.
type VendorPaymentCreated struct {
	Payment  *domain.VendorPayment
	Evidence []domain.Evidence
	Ledger   domain.Ledger
}

// CreateWithProof is Create plus the proof files, as one atomic action: see
// ClientPaymentService.CreateWithProof for the contract (uploads verified first,
// then one transaction for payment + evidence + activity + ledger, all or nothing).
func (s *PaymentService) CreateWithProof(ctx context.Context, tenantID, projectID, actorStaffID int64, input PaymentInput, items []ProofItem) (*VendorPaymentCreated, error) {
	proof := s.proof
	if proof == nil {
		proof = NewProofSupport(nil, nil, s.evidence)
	}
	fe := newFieldErrors()
	validatePaymentFields(fe, input.Type, input.Amount)
	resolved, err := proof.resolve(ctx, tenantID, projectID, items, outgoingProofTypes)
	if err != nil && !fe.mergeErr(err) {
		return nil, err
	}
	if !fe.empty() {
		return nil, fe.err()
	}

	var out VendorPaymentCreated
	err = proof.tx.InTx(ctx, func(ctx context.Context) error {
		if err := proof.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		if err := s.checkChange(ctx, projectID, input.ProjectVendorID, 0, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
			return err
		}
		p := &domain.VendorPayment{
			ProjectID: projectID, ProjectVendorID: input.ProjectVendorID, Type: input.Type, Amount: input.Amount,
			PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		evidence, err := proof.attach(ctx, proofTarget{
			ProjectID: projectID, ActorStaffID: actorStaffID, Kind: domain.RelatedPayment, RelatedID: p.ID,
			Date: input.PaymentDate, Reference: input.ReferenceNumber,
		}, resolved)
		if err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "vendor_payment", formatID(p.ID), string(p.Type),
			"Pembayaran vendor dicatat")
		contract, err := s.contracts.EngagementContractValue(ctx, projectID, input.ProjectVendorID)
		if err != nil {
			return err
		}
		all, err := s.repo.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		var lines []domain.PaymentLine
		for _, q := range all {
			if q.ProjectVendorID == input.ProjectVendorID {
				lines = append(lines, lineOf(q.Type, q.Amount))
			}
		}
		out = VendorPaymentCreated{Payment: p, Evidence: evidence, Ledger: domain.ComputeLedger(lines, contract)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
