package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

type VenuePaymentRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.VenuePayment, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.VenuePayment, error)
	Create(ctx context.Context, p *domain.VenuePayment) error
}

type VenuePaymentService struct {
	repo     VenuePaymentRepository
	activity *ActivityService
}

func NewVenuePaymentService(repo VenuePaymentRepository, activity *ActivityService) *VenuePaymentService {
	return &VenuePaymentService{repo: repo, activity: activity}
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
