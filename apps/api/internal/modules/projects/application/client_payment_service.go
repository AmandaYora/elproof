package application

import (
	"context"
	"errors"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type ClientPaymentRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ClientPayment, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.ClientPayment, error)
	Create(ctx context.Context, p *domain.ClientPayment) error
	Update(ctx context.Context, projectID, id int64, p domain.ClientPayment) error
	Delete(ctx context.Context, projectID, id int64) error
	// SetVerification records a verification only when there is none yet and
	// reports whether this call set it (idempotent under concurrent verifies).
	SetVerification(ctx context.Context, projectID, id int64, at time.Time, byStaffID int64) (bool, error)
	ClearVerification(ctx context.Context, projectID, id int64) error
}

type ClientPaymentService struct {
	repo      ClientPaymentRepository
	evidence  *EvidenceService
	activity  *ActivityService
	contracts ContractReader
	// proof/terms are wired by SetAtomicSupport (CreateWithProof needs them).
	proof *ProofSupport
	terms PaymentTermRepository
}

// SetAtomicSupport wires what CreateWithProof needs: the transaction, the
// upload checks and the payment schedule (read for the ledger summary).
func (s *ClientPaymentService) SetAtomicSupport(proof *ProofSupport, terms PaymentTermRepository) {
	s.proof, s.terms = proof, terms
}

func NewClientPaymentService(repo ClientPaymentRepository, evidence *EvidenceService, activity *ActivityService, contracts ContractReader) *ClientPaymentService {
	s := &ClientPaymentService{repo: repo, evidence: evidence, activity: activity, contracts: contracts}
	// A verified payment must keep at least one proof: react when one is removed.
	evidence.OnRemoved(s)
	return s
}

// checkChange refuses an add, edit or delete that would take the client's
// paid total past the project's contract value (FLOW-IDEAL 2.4, ADR-0027).
// origID is the payment being edited or deleted (0 when adding); next is its
// new version (nil when deleting).
func (s *ClientPaymentService) checkChange(ctx context.Context, tenantID, projectID, origID int64, next *domain.PaymentLine) error {
	contract, err := s.contracts.ProjectContractValue(ctx, tenantID, projectID)
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
	return checkChange(before, after, contract, "nilai kontrak", next == nil)
}

func (s *ClientPaymentService) List(ctx context.Context, projectID int64) ([]domain.ClientPayment, error) {
	return s.repo.ListByProject(ctx, projectID)
}

type ClientPaymentInput struct {
	Type            domain.PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}

// Create records a client payment and logs it under the same
// ActivityPaymentRecorded type PaymentService.Create already uses for
// vendor payments (mirrors ActivityMilestoneUpdated's own reuse across
// Project Milestones and Vendor Milestones) — the two are distinguished
// only by entityType/description, not a new ActivityType value.
func (s *ClientPaymentService) Create(ctx context.Context, tenantID, projectID int64, actorStaffID int64, input ClientPaymentInput) (*domain.ClientPayment, error) {
	if err := checkAmount(input.Amount); err != nil {
		return nil, err
	}
	if err := s.checkChange(ctx, tenantID, projectID, 0, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
		return nil, err
	}
	p := &domain.ClientPayment{
		ProjectID: projectID, Type: input.Type, Amount: input.Amount,
		PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
		RecordedByStaffID: actorStaffID,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
		"Pembayaran client dicatat")
	return p, nil
}

// Update is an unconditional full-field overwrite -- see PaymentService.Update.
func (s *ClientPaymentService) Update(ctx context.Context, tenantID, projectID, id int64, actorStaffID int64, input ClientPaymentInput) (*domain.ClientPayment, error) {
	p, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperror.NotFound("Pembayaran tidak ditemukan")
	}
	if err := checkAmount(input.Amount); err != nil {
		return nil, err
	}
	if err := s.checkChange(ctx, tenantID, projectID, id, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
		return nil, err
	}
	// Changing what was paid invalidates a verification (ADR-0032).
	cleared := p.ApplyEdit(input.Type, input.Amount, input.PaymentDate, input.Method, input.ReferenceNumber, input.Notes)
	if err := s.repo.Update(ctx, projectID, id, *p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
		"Pembayaran client diperbarui")
	if cleared {
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
			"Verifikasi pembayaran client dibatalkan karena data pembayaran diubah")
	}
	return p, nil
}

// errVerifyNeedsEvidence is the 422 returned when a payment with no proof file
// is asked to be verified (field key "evidence").
func errVerifyNeedsEvidence() error {
	msg := "Lampirkan bukti transfer sebelum memverifikasi pembayaran."
	return apperror.Validation(msg, map[string][]string{"evidence": {msg}})
}

// Verify confirms that a client payment arrived and its proof is attached. It
// is idempotent (an already verified payment is returned untouched: same
// person, same time) and refuses a payment with no evidence file.
func (s *ClientPaymentService) Verify(ctx context.Context, projectID, id, actorStaffID int64, now time.Time) (*domain.ClientPayment, error) {
	p, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperror.NotFound("Pembayaran tidak ditemukan")
	}
	if p.IsVerified() {
		return p, nil
	}
	files, err := s.evidence.ListForRelated(ctx, domain.RelatedClientPayment, id)
	if err != nil {
		return nil, err
	}
	if _, err := p.Verify(now, actorStaffID, len(files)); err != nil {
		if errors.Is(err, domain.ErrVerifyNeedsEvidence) {
			return nil, errVerifyNeedsEvidence()
		}
		return nil, err
	}
	won, err := s.repo.SetVerification(ctx, projectID, id, *p.VerifiedAt, actorStaffID)
	if err != nil {
		return nil, err
	}
	if !won {
		// Someone verified between our read and write: report theirs, unchanged.
		return s.repo.FindByID(ctx, projectID, id)
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
		"Pembayaran client diverifikasi")
	return p, nil
}

// AfterEvidenceRemoved keeps the invariant "a verified payment always has at
// least one proof": when the last file of a verified client payment goes, the
// verification goes with it.
func (s *ClientPaymentService) AfterEvidenceRemoved(ctx context.Context, e domain.Evidence, actorStaffID int64) {
	if e.RelatedKind != domain.RelatedClientPayment {
		return
	}
	p, err := s.repo.FindByID(ctx, e.ProjectID, e.RelatedID)
	if err != nil || p == nil || !p.IsVerified() {
		return
	}
	left, err := s.evidence.ListForRelated(ctx, domain.RelatedClientPayment, e.RelatedID)
	if err != nil || len(left) > 0 {
		return
	}
	if err := s.repo.ClearVerification(ctx, e.ProjectID, e.RelatedID); err != nil {
		return
	}
	pid := e.ProjectID
	s.activity.Record(ctx, &pid, domain.ActivityPaymentUpdated, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
		"Verifikasi pembayaran client dibatalkan karena bukti terakhir dihapus")
}

// Delete hard-deletes the payment and, first, its attached evidence -- see
// PaymentService.Delete for the full reasoning (evidence-first ordering,
// Owner-only role check living in the HTTP handler, not here).
func (s *ClientPaymentService) Delete(ctx context.Context, tenantID, projectID, id int64, actorStaffID int64) error {
	p, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return err
	}
	if p == nil {
		return apperror.NotFound("Pembayaran tidak ditemukan")
	}
	// Removing a Refund raises what counts as paid, so a delete is checked too.
	if err := s.checkChange(ctx, tenantID, projectID, id, nil); err != nil {
		return err
	}
	if err := s.evidence.DeleteForRelated(ctx, domain.RelatedClientPayment, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, projectID, id); err != nil {
		return err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentDeleted, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
		"Pembayaran client dihapus")
	return nil
}

// ClientPaymentCreated is the outcome of recording a client payment: the
// payment, the proof files recorded with it, where the contract stands and how
// each installment of the schedule now stands.
type ClientPaymentCreated struct {
	Payment  *domain.ClientPayment
	Evidence []domain.Evidence
	Ledger   domain.Ledger
	Terms    []domain.TermProgress
}

// validatePaymentFields collects what is wrong with the type and amount.
func validatePaymentFields(fe *fieldErrors, t domain.PaymentType, amount int64) {
	if !domain.IsValidPaymentType(t) {
		fe.add("Pilih jenis pembayaran: DP, Termin, Pelunasan, Tambahan, atau Refund.", "type")
	}
	if err := checkAmount(amount); err != nil {
		fe.mergeErr(err)
	}
}

// CreateWithProof records a client payment and its proof files as ONE business
// action. The uploads are verified first (nothing written); then, in a single
// database transaction, the project row is locked, the overpayment rule is
// checked, the payment and every evidence row are inserted, activity is logged
// and the ledger summary is computed from the very rows just written. Any
// failure rolls the whole transaction back: there is never a payment without
// the proof it was recorded with. Every field problem (type, amount, each
// evidence item) is reported together.
func (s *ClientPaymentService) CreateWithProof(ctx context.Context, tenantID, projectID, actorStaffID int64, input ClientPaymentInput, items []ProofItem, today time.Time) (*ClientPaymentCreated, error) {
	proof := s.proof
	if proof == nil {
		proof = NewProofSupport(nil, nil, s.evidence)
	}
	fe := newFieldErrors()
	validatePaymentFields(fe, input.Type, input.Amount)
	resolved, err := proof.resolve(ctx, tenantID, projectID, items, clientProofTypes)
	if err != nil && !fe.mergeErr(err) {
		return nil, err
	}
	if !fe.empty() {
		return nil, fe.err()
	}

	var out ClientPaymentCreated
	err = proof.tx.InTx(ctx, func(ctx context.Context) error {
		if err := proof.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		if err := s.checkChange(ctx, tenantID, projectID, 0, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
			return err
		}
		p := &domain.ClientPayment{
			ProjectID: projectID, Type: input.Type, Amount: input.Amount,
			PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
			RecordedByStaffID: actorStaffID,
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		evidence, err := proof.attach(ctx, proofTarget{
			ProjectID: projectID, ActorStaffID: actorStaffID, Kind: domain.RelatedClientPayment, RelatedID: p.ID,
			Date: input.PaymentDate, Reference: input.ReferenceNumber,
		}, resolved)
		if err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
			"Pembayaran client dicatat")
		ledger, terms, err := s.ledgerAndTerms(ctx, tenantID, projectID, today)
		if err != nil {
			return err
		}
		out = ClientPaymentCreated{Payment: p, Evidence: evidence, Ledger: ledger, Terms: terms}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ledgerAndTerms is where the client's contract stands, and each installment of
// the schedule, as of `today`, read on whatever transaction ctx carries.
func (s *ClientPaymentService) ledgerAndTerms(ctx context.Context, tenantID, projectID int64, today time.Time) (domain.Ledger, []domain.TermProgress, error) {
	contract, err := s.contracts.ProjectContractValue(ctx, tenantID, projectID)
	if err != nil {
		return domain.Ledger{}, nil, err
	}
	all, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return domain.Ledger{}, nil, err
	}
	lines := make([]domain.PaymentLine, 0, len(all))
	for _, p := range all {
		lines = append(lines, lineOf(p.Type, p.Amount))
	}
	ledger := domain.ComputeLedger(lines, contract)
	terms := []domain.TermProgress{}
	if s.terms != nil {
		schedule, err := s.terms.ListByProject(ctx, projectID)
		if err != nil {
			return domain.Ledger{}, nil, err
		}
		terms = domain.AllocateTerms(schedule, domain.AppliedToContract(lines), today)
	}
	return ledger, terms, nil
}
