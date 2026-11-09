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
	Update(ctx context.Context, projectID, id int64, p domain.VendorPayment) error
	Delete(ctx context.Context, projectID, id int64) error
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

// Update is an unconditional full-field overwrite -- same convention as
// VendorEngagementService.Update/IssueService.Update, no status/role guard
// (PLAN.md §2.3's correction: only Delete escalates to Owner-only, Edit
// never does). ProjectVendorID is deliberately not accepted here (PLAN.md
// §2.5) -- re-parenting a payment to a different engagement isn't supported;
// delete and recreate instead.
func (s *PaymentService) Update(ctx context.Context, projectID, id int64, actorStaffID int64, input PaymentInput) (*domain.VendorPayment, error) {
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
	if err := s.checkChange(ctx, projectID, p.ProjectVendorID, id, &domain.PaymentLine{Type: input.Type, Amount: input.Amount}); err != nil {
		return nil, err
	}
	p.Type, p.Amount, p.PaymentDate, p.Method, p.ReferenceNumber, p.Notes =
		input.Type, input.Amount, input.PaymentDate, input.Method, input.ReferenceNumber, input.Notes
	if err := s.repo.Update(ctx, projectID, id, *p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "vendor_payment", formatID(p.ID), string(p.Type),
		"Pembayaran vendor diperbarui")
	return p, nil
}

// Delete hard-deletes the payment and, first, every evidence row attached to
// it -- see EvidenceService.DeleteForRelated and PLAN.md §2.4 for why
// evidence is cleaned up BEFORE the payment row itself (so a failure here
// never leaves an un-deletable "ghost" evidence row behind). The Owner-only
// role check lives in the HTTP handler (deletePayment), matching
// deleteProject's own convention -- this service method has no opinion on
// who's calling.
func (s *PaymentService) Delete(ctx context.Context, projectID, id int64, actorStaffID int64) error {
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
	if err := s.evidence.DeleteForRelated(ctx, domain.RelatedPayment, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, projectID, id); err != nil {
		return err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentDeleted, actorStaffID, "vendor_payment", formatID(p.ID), string(p.Type),
		"Pembayaran vendor dihapus")
	return nil
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
