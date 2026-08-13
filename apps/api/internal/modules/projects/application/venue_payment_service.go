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
	Update(ctx context.Context, projectID, id int64, p domain.VenuePayment) error
	Delete(ctx context.Context, projectID, id int64) error
}

type VenuePaymentService struct {
	repo     VenuePaymentRepository
	evidence *EvidenceService
	activity *ActivityService
}

func NewVenuePaymentService(repo VenuePaymentRepository, evidence *EvidenceService, activity *ActivityService) *VenuePaymentService {
	return &VenuePaymentService{repo: repo, evidence: evidence, activity: activity}
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
func (s *VenuePaymentService) Create(ctx context.Context, projectID int64, actorStaffID int64, input VenuePaymentInput) (*domain.VenuePayment, error) {
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

// Update is an unconditional full-field overwrite -- see PaymentService.Update.
func (s *VenuePaymentService) Update(ctx context.Context, projectID, id int64, actorStaffID int64, input VenuePaymentInput) (*domain.VenuePayment, error) {
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
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "venue_payment", formatID(p.ID), string(p.Type),
		"Pembayaran venue diperbarui")
	return p, nil
}

// Delete hard-deletes the payment and, first, its attached evidence -- see
// PaymentService.Delete for the full reasoning.
func (s *VenuePaymentService) Delete(ctx context.Context, projectID, id int64, actorStaffID int64) error {
	p, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return err
	}
	if p == nil {
		return apperror.NotFound("Pembayaran tidak ditemukan")
	}
	if err := s.evidence.DeleteForRelated(ctx, domain.RelatedVenuePayment, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, projectID, id); err != nil {
		return err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentDeleted, actorStaffID, "venue_payment", formatID(p.ID), string(p.Type),
		"Pembayaran venue dihapus")
	return nil
}
