package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type VenuePaymentRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.VenuePayment, error)
	// ListByProjects backs ComputeProgressBatch: every matching row across
	// the given projects in one query (WHERE project_id IN (...)).
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.VenuePayment, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.VenuePayment, error)
	Create(ctx context.Context, p *domain.VenuePayment) error
	// Patch writes the changed fields of a live payment (see PaymentRepository.Patch).
	Patch(ctx context.Context, p *domain.VenuePayment, changed []string, expected *time.Time) (bool, error)
	// SoftDelete hides a live payment (deleted_at = at); false when there was none.
	SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error)
	// FindDeletedByID is the restore lookup (nil unless the payment waits in the
	// "deleted" state).
	FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.VenuePayment, error)
	// Restore clears the stamp of a payment deleted after notBefore.
	Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error)
}

type VenuePaymentService struct {
	repo      VenuePaymentRepository
	evidence  *EvidenceService
	activity  *ActivityService
	contracts ContractReader
	proof     *ProofSupport
}

// SetProofSupport wires what CreateWithProof needs (transaction, upload checks).
func (s *VenuePaymentService) SetProofSupport(p *ProofSupport) { s.proof = p }

func NewVenuePaymentService(repo VenuePaymentRepository, evidence *EvidenceService, activity *ActivityService, contracts ContractReader) *VenuePaymentService {
	return &VenuePaymentService{repo: repo, evidence: evidence, activity: activity, contracts: contracts}
}

// checkChange refuses an add, edit or delete that would take the total paid
// to the venue past its rental price plus charge on the project.
func (s *VenuePaymentService) checkChange(ctx context.Context, tenantID, projectID, origID int64, next *domain.PaymentLine) error {
	cost, err := s.contracts.VenueCost(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	all, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	var before, after []domain.PaymentLine
	for _, p := range all {
		line := domain.PaymentLine{Type: p.Type, Amount: p.Amount}
		before = append(before, line)
		if p.ID != origID {
			after = append(after, line)
		}
	}
	if next != nil {
		after = append(after, *next)
	}
	return checkChange(before, after, cost, "biaya venue", next == nil)
}

func (s *VenuePaymentService) List(ctx context.Context, projectID int64) ([]domain.VenuePayment, error) {
	return s.repo.ListByProject(ctx, projectID)
}

type VenuePaymentInput struct {
	Type            domain.PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}

// Create records a venue payment and logs it under the same
// ActivityPaymentRecorded type PaymentService/ClientPaymentService already
// use — distinguished only by entityType/description.
func (s *VenuePaymentService) Create(ctx context.Context, tenantID, projectID int64, actorStaffID int64, input VenuePaymentInput) (*domain.VenuePayment, error) {
	if err := checkAmount(input.Amount); err != nil {
		return nil, err
	}
	if err := s.checkChange(ctx, tenantID, projectID, 0, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
		return nil, err
	}
	p := &domain.VenuePayment{
		ProjectID: projectID, Type: input.Type, Amount: input.Amount,
		PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "venue_payment", formatID(p.ID), string(p.Type),
		"Pembayaran venue dicatat")
	return p, nil
}

// Patch changes the fields of a venue payment that were sent, and only those; see
// PaymentService.Patch (overpayment rule on the merged type and amount, project
// lock, no-op writes nothing, compare-and-set with `expected`).
func (s *VenuePaymentService) Patch(ctx context.Context, tenantID, projectID, id int64, actorStaffID int64, patch PaymentPatch, expected *time.Time) (*domain.VenuePayment, error) {
	if err := checkPaymentPatch(patch); err != nil {
		return nil, err
	}
	var out *domain.VenuePayment
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
			if err := s.checkChange(ctx, tenantID, projectID, id, &domain.PaymentLine{Type: p.Type, Amount: p.Amount}); err != nil {
				return err
			}
		}
		wrote, err := s.repo.Patch(ctx, p, ch.keys, expected)
		if err != nil {
			return err
		}
		fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.VenuePayment, error) {
			return s.repo.FindByID(ctx, projectID, id)
		}, func(p *domain.VenuePayment) time.Time { return p.UpdatedAt }, apperror.NotFound("Pembayaran tidak ditemukan"))
		if err != nil {
			return err
		}
		out = fresh
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "venue_payment", formatID(fresh.ID), string(fresh.Type),
			"Pembayaran venue diperbarui: "+ch.summary())
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Delete soft-deletes the venue payment (ADR-0034); see PaymentService.Delete.
func (s *VenuePaymentService) Delete(ctx context.Context, tenantID, projectID, id int64, actorStaffID int64, now time.Time) (time.Time, error) {
	var until time.Time
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.repo.FindByID(ctx, projectID, id)
		if err != nil {
			return err
		}
		if p == nil {
			return apperror.NotFound("Pembayaran tidak ditemukan")
		}
		if err := s.checkChange(ctx, tenantID, projectID, id, nil); err != nil {
			return err
		}
		ok, err := s.repo.SoftDelete(ctx, projectID, id, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperror.NotFound("Pembayaran tidak ditemukan")
		}
		until = domain.RestorableUntil(now)
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentDeleted, actorStaffID, "venue_payment", formatID(p.ID), string(p.Type),
			"Pembayaran venue dihapus")
		return nil
	})
	return until, err
}

// Restore brings a deleted venue payment back after re-checking the overpayment
// rule against the venue's cost (422 on `amount`); see PaymentService.Restore.
func (s *VenuePaymentService) Restore(ctx context.Context, tenantID, projectID, id int64, actorStaffID int64, now time.Time) (*domain.VenuePayment, error) {
	var restored *domain.VenuePayment
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.repo.FindDeletedByID(ctx, projectID, id)
		if err != nil {
			return err
		}
		if p == nil || p.DeletedAt == nil || !domain.CanRestore(*p.DeletedAt, now) {
			return errNotRestorable()
		}
		if err := s.checkChange(ctx, tenantID, projectID, 0, &domain.PaymentLine{Type: p.Type, Amount: p.Amount}); err != nil {
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
		s.activity.Record(ctx, &projectID, domain.ActivityRestored, actorStaffID, "venue_payment", formatID(p.ID), string(p.Type),
			"Pembayaran venue dipulihkan")
		return nil
	})
	return restored, err
}

// LedgerSummary is where the venue's cost (rental price + charge) stands against
// its live payments (the `ledgerSummary` of a mutation response).
func (s *VenuePaymentService) LedgerSummary(ctx context.Context, tenantID, projectID int64) (domain.Ledger, error) {
	cost, err := s.contracts.VenueCost(ctx, tenantID, projectID)
	if err != nil {
		return domain.Ledger{}, err
	}
	all, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return domain.Ledger{}, err
	}
	lines := make([]domain.PaymentLine, 0, len(all))
	for _, q := range all {
		lines = append(lines, lineOf(q.Type, q.Amount))
	}
	return domain.ComputeLedger(lines, cost), nil
}

// VenuePaymentCreated is the outcome of recording a venue payment: the payment,
// its proof files and where the venue's cost stands.
type VenuePaymentCreated struct {
	Payment  *domain.VenuePayment
	Evidence []domain.Evidence
	Ledger   domain.Ledger
}

// CreateWithProof is Create plus the proof files, as one atomic action: see
// ClientPaymentService.CreateWithProof.
func (s *VenuePaymentService) CreateWithProof(ctx context.Context, tenantID, projectID, actorStaffID int64, input VenuePaymentInput, items []ProofItem) (*VenuePaymentCreated, error) {
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

	var out VenuePaymentCreated
	err = proof.tx.InTx(ctx, func(ctx context.Context) error {
		if err := proof.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		if err := s.checkChange(ctx, tenantID, projectID, 0, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
			return err
		}
		p := &domain.VenuePayment{
			ProjectID: projectID, Type: input.Type, Amount: input.Amount,
			PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		evidence, err := proof.attach(ctx, proofTarget{
			ProjectID: projectID, ActorStaffID: actorStaffID, Kind: domain.RelatedVenuePayment, RelatedID: p.ID,
			Date: input.PaymentDate, Reference: input.ReferenceNumber,
		}, resolved)
		if err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "venue_payment", formatID(p.ID), string(p.Type),
			"Pembayaran venue dicatat")
		cost, err := s.contracts.VenueCost(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		all, err := s.repo.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		lines := make([]domain.PaymentLine, 0, len(all))
		for _, q := range all {
			lines = append(lines, lineOf(q.Type, q.Amount))
		}
		out = VenuePaymentCreated{Payment: p, Evidence: evidence, Ledger: domain.ComputeLedger(lines, cost)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
