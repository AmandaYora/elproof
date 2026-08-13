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
	repo     PaymentRepository
	evidence *EvidenceService
	activity *ActivityService
}

func NewPaymentService(repo PaymentRepository, evidence *EvidenceService, activity *ActivityService) *PaymentService {
	return &PaymentService{repo: repo, evidence: evidence, activity: activity}
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
