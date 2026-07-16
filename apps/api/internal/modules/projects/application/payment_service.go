package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

type PaymentRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.VendorPayment, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.VendorPayment, error)
	Create(ctx context.Context, p *domain.VendorPayment) error
}

type PaymentService struct {
	repo     PaymentRepository
	activity *ActivityService
}

func NewPaymentService(repo PaymentRepository, activity *ActivityService) *PaymentService {
	return &PaymentService{repo: repo, activity: activity}
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
