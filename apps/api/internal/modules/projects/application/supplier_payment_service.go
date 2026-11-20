package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// SupplierPaymentRepository stores money paid to the vendors and venues of
// projects (supplier_payments). Lists skip deleted rows.
type SupplierPaymentRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.SupplierPayment, error)
	// ListByProjects backs ComputeProgressBatch: every live row across the given
	// projects in one query.
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.SupplierPayment, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.SupplierPayment, error)
	Create(ctx context.Context, p *domain.SupplierPayment) error
	// Patch writes the changed fields of a live payment (compare-and-set on
	// updated_at when expected is given; see infrastructure.patchRow).
	Patch(ctx context.Context, p *domain.SupplierPayment, changed []string, expected *time.Time) (bool, error)
	SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error)
	// FindDeletedByID is the restore lookup (nil unless the payment waits in the
	// "deleted" state).
	FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.SupplierPayment, error)
	Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error)
}

// SupplierPaymentService records what is paid to a project's vendors and
// venues. A payment's payer is fixed when it is recorded: the supplier's
// "dibayar oleh" at that moment (and that cannot change while a payment hangs
// off it). What the WO pays is checked against the cost it agreed with the
// supplier; what the client pays a supplier directly is outside the contract
// and limited by nothing.
type SupplierPaymentService struct {
	repo      SupplierPaymentRepository
	evidence  *EvidenceService
	activity  *ActivityService
	contracts ContractReader
	proof     *ProofSupport
}

// SetProofSupport wires what CreateWithProof needs (transaction, upload checks).
func (s *SupplierPaymentService) SetProofSupport(p *ProofSupport) { s.proof = p }

func NewSupplierPaymentService(repo SupplierPaymentRepository, evidence *EvidenceService, activity *ActivityService, contracts ContractReader) *SupplierPaymentService {
	return &SupplierPaymentService{repo: repo, evidence: evidence, activity: activity, contracts: contracts}
}

var errSupplierPaymentNotFound = apperror.NotFound("Pembayaran tidak ditemukan")

// Supplier is the live vendor or venue a payment is (to be) recorded for: the
// handler reads its payer to pick the permission.
func (s *SupplierPaymentService) Supplier(ctx context.Context, projectID, projectSupplierID int64) (*domain.ProjectSupplier, error) {
	return s.contracts.Supplier(ctx, projectID, projectSupplierID)
}

func (s *SupplierPaymentService) List(ctx context.Context, projectID int64) ([]domain.SupplierPayment, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// Get is one live payment of the project.
func (s *SupplierPaymentService) Get(ctx context.Context, projectID, id int64) (*domain.SupplierPayment, error) {
	p, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errSupplierPaymentNotFound
	}
	return p, nil
}

// FindDeleted is one payment waiting in the "deleted" state (nil when none).
func (s *SupplierPaymentService) FindDeleted(ctx context.Context, projectID, id int64) (*domain.SupplierPayment, error) {
	return s.repo.FindDeletedByID(ctx, projectID, id)
}

// supplierLines is the ledger view of one supplier's live payments by one payer
// that count as paid (a client's proof the WO has not marked right does not:
// domain.SupplierPayment.CountsAsPaid; the WO's own payments always count).
func supplierLines(all []domain.SupplierPayment, supplierID int64, payer domain.Payer, skipID int64) []domain.PaymentLine {
	var lines []domain.PaymentLine
	for _, p := range all {
		if p.ProjectSupplierID == supplierID && p.Payer == payer && p.ID != skipID && p.CountsAsPaid() {
			lines = append(lines, lineOf(p.Type, p.Amount))
		}
	}
	return lines
}

// owedTo is what a payer's payments to a supplier are checked against: the
// agreed cost for the WO (0, not set yet, limits nothing), nothing for the
// client.
func owedTo(sup *domain.ProjectSupplier, payer domain.Payer) int64 {
	if payer != domain.PayerWO || sup.AgreedCost == nil {
		return 0
	}
	return *sup.AgreedCost
}

// checkChange refuses an add, edit or delete that would take what the WO paid
// a supplier past the cost it agreed with it.
func (s *SupplierPaymentService) checkChange(ctx context.Context, projectID int64, sup *domain.ProjectSupplier, payer domain.Payer, origID int64, next *domain.PaymentLine) error {
	limit := owedTo(sup, payer)
	if limit <= 0 {
		return nil
	}
	all, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	before := supplierLines(all, sup.ID, payer, 0)
	after := supplierLines(all, sup.ID, payer, origID)
	if next != nil {
		after = append(after, *next)
	}
	return checkChange(before, after, limit, "biaya rekanan", next == nil)
}

// SupplierPaymentInput is one payment to a supplier.
type SupplierPaymentInput struct {
	ProjectSupplierID int64
	Type              domain.PaymentType
	Amount            int64
	PaymentDate       time.Time
	Method            string
	ReferenceNumber   string
	Notes             string
}

// LedgerSummary is where one supplier stands against its live payments by its
// payer (the `ledgerSummary` of a mutation response). For a supplier the client
// pays the owed amount is 0: the WO does not track what it costs.
func (s *SupplierPaymentService) LedgerSummary(ctx context.Context, projectID, projectSupplierID int64) (domain.Ledger, error) {
	sup, err := s.contracts.Supplier(ctx, projectID, projectSupplierID)
	if err != nil {
		return domain.Ledger{}, err
	}
	all, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return domain.Ledger{}, err
	}
	return domain.ComputeLedger(supplierLines(all, sup.ID, sup.PaidBy, 0), owedTo(sup, sup.PaidBy)), nil
}

// SupplierPaymentCreated is the outcome of recording a supplier payment: the
// payment, its proof files and where that supplier stands.
type SupplierPaymentCreated struct {
	Payment  *domain.SupplierPayment
	Evidence []domain.Evidence
	Ledger   domain.Ledger
}

// CreateWithProof records a payment and its proof files as one atomic action:
// see ClientPaymentService.CreateWithProof for the contract (uploads verified
// first, then one transaction for payment + evidence + activity + ledger, all
// or nothing).
func (s *SupplierPaymentService) CreateWithProof(ctx context.Context, tenantID, projectID, actorStaffID int64, input SupplierPaymentInput, items []ProofItem) (*SupplierPaymentCreated, error) {
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

	var out SupplierPaymentCreated
	err = proof.tx.InTx(ctx, func(ctx context.Context) error {
		if err := proof.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		sup, err := s.contracts.Supplier(ctx, projectID, input.ProjectSupplierID)
		if err != nil {
			return err
		}
		if err := s.checkChange(ctx, projectID, sup, sup.PaidBy, 0, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
			return err
		}
		p := &domain.SupplierPayment{
			ProjectID: projectID, ProjectSupplierID: sup.ID, Payer: sup.PaidBy, Type: input.Type, Amount: input.Amount,
			PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
			RecordedByStaffID: actorStaffID,
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		evidence, err := proof.attach(ctx, proofTarget{
			ProjectID: projectID, ActorStaffID: actorStaffID, Kind: domain.RelatedSupplierPayment, RelatedID: p.ID,
			Date: input.PaymentDate, Reference: input.ReferenceNumber,
		}, resolved)
		if err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "supplier_payment", formatID(p.ID), string(p.Type),
			"Pembayaran ke "+sup.Name+" dicatat")
		all, err := s.repo.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		out = SupplierPaymentCreated{Payment: p, Evidence: evidence,
			Ledger: domain.ComputeLedger(supplierLines(all, sup.ID, sup.PaidBy, 0), owedTo(sup, sup.PaidBy))}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Exists is nil when the payment is on this project and not deleted, a not
// found otherwise (asked before a missing If-Match is answered with 428).
func (s *SupplierPaymentService) Exists(ctx context.Context, projectID, id int64) error {
	_, err := s.Get(ctx, projectID, id)
	return err
}

// Patch changes the fields of a payment that were sent, and only those. The
// supplier and the payer are not patchable: delete and record again. The
// overpayment rule is judged on the merged type and amount, under the project
// lock like every money write.
func (s *SupplierPaymentService) Patch(ctx context.Context, projectID, id, actorStaffID int64, patch PaymentPatch, expected *time.Time) (*domain.SupplierPayment, error) {
	if err := checkPaymentPatch(patch); err != nil {
		return nil, err
	}
	var out *domain.SupplierPayment
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.Get(ctx, projectID, id)
		if err != nil {
			return err
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
		sup, err := s.contracts.Supplier(ctx, projectID, p.ProjectSupplierID)
		if err != nil {
			return err
		}
		if ch.moneyChanged() {
			if err := s.checkChange(ctx, projectID, sup, p.Payer, id, &domain.PaymentLine{Type: p.Type, Amount: p.Amount}); err != nil {
				return err
			}
		}
		// A client's proof the WO marked right is unchecked again once what was
		// paid changes (ADR-0032); a change of the note keeps it.
		cleared := false
		if ch.has("type") || ch.has("amount") || ch.has("paymentDate") || ch.has("method") || ch.has("referenceNumber") {
			if cleared = p.ClearVerification(); cleared {
				ch.note("verification", "")
			}
		}
		wrote, err := s.repo.Patch(ctx, p, ch.keys, expected)
		if err != nil {
			return err
		}
		fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.SupplierPayment, error) {
			return s.repo.FindByID(ctx, projectID, id)
		}, func(p *domain.SupplierPayment) time.Time { return p.UpdatedAt }, errSupplierPaymentNotFound)
		if err != nil {
			return err
		}
		out = fresh
		desc := "Pembayaran ke " + sup.Name + " diperbarui: " + ch.summary()
		if cleared {
			desc += ". Pemeriksaan bukti dibatalkan karena data pembayaran diubah"
		}
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "supplier_payment", formatID(fresh.ID), string(fresh.Type), desc)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Delete soft-deletes the payment (ADR-0034): it leaves the list, the supplier's
// paid total and progress, and its proof files are hidden with it, until it is
// restored (24 hours) or purged. Removing a Refund is checked like adding a
// payment.
func (s *SupplierPaymentService) Delete(ctx context.Context, projectID, id, actorStaffID int64, now time.Time) (*domain.SupplierPayment, time.Time, error) {
	var deleted *domain.SupplierPayment
	var until time.Time
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.Get(ctx, projectID, id)
		if err != nil {
			return err
		}
		sup, err := s.contracts.Supplier(ctx, projectID, p.ProjectSupplierID)
		if err != nil {
			return err
		}
		if err := s.checkChange(ctx, projectID, sup, p.Payer, id, nil); err != nil {
			return err
		}
		ok, err := s.repo.SoftDelete(ctx, projectID, id, now)
		if err != nil {
			return err
		}
		if !ok {
			return errSupplierPaymentNotFound
		}
		deleted, until = p, domain.RestorableUntil(now)
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentDeleted, actorStaffID, "supplier_payment", formatID(p.ID), string(p.Type),
			"Pembayaran ke "+sup.Name+" dihapus")
		return nil
	})
	return deleted, until, err
}

// Restore brings a deleted payment back, after re-checking what can now fail:
// its supplier must still be live (422 on `projectSupplierId`: restore the
// supplier first) and it must not take the WO past the agreed cost (422 on
// `amount`). Not deleted, or past the retention: 404.
func (s *SupplierPaymentService) Restore(ctx context.Context, projectID, id, actorStaffID int64, now time.Time) (*domain.SupplierPayment, error) {
	var restored *domain.SupplierPayment
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		p, err := s.repo.FindDeletedByID(ctx, projectID, id)
		if err != nil {
			return err
		}
		if p == nil || p.DeletedAt == nil || !domain.CanRestore(*p.DeletedAt, now) {
			return errNotRestorable()
		}
		sup, err := s.contracts.Supplier(ctx, projectID, p.ProjectSupplierID)
		if err != nil {
			if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
				msg := "Vendor atau venue untuk pembayaran ini sudah dihapus. Pulihkan lebih dulu."
				return apperror.Validation(msg, map[string][]string{"projectSupplierId": {msg}})
			}
			return err
		}
		if err := s.checkChange(ctx, projectID, sup, p.Payer, 0, &domain.PaymentLine{Type: p.Type, Amount: p.Amount}); err != nil {
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
		s.activity.Record(ctx, &projectID, domain.ActivityRestored, actorStaffID, "supplier_payment", formatID(p.ID), string(p.Type),
			"Pembayaran ke "+sup.Name+" dipulihkan")
		return nil
	})
	return restored, err
}
