package application

import (
	"context"
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
}

type ClientPaymentService struct {
	repo     ClientPaymentRepository
	evidence *EvidenceService
	activity *ActivityService
}

func NewClientPaymentService(repo ClientPaymentRepository, evidence *EvidenceService, activity *ActivityService) *ClientPaymentService {
	return &ClientPaymentService{repo: repo, evidence: evidence, activity: activity}
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
func (s *ClientPaymentService) Create(ctx context.Context, projectID int64, actorStaffID int64, input ClientPaymentInput) (*domain.ClientPayment, error) {
	p := &domain.ClientPayment{
		ProjectID: projectID, Type: input.Type, Amount: input.Amount,
		PaymentDate: input.PaymentDate, Method: input.Method, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
		"Pembayaran client dicatat")
	return p, nil
}

// Update is an unconditional full-field overwrite -- see PaymentService.Update.
func (s *ClientPaymentService) Update(ctx context.Context, projectID, id int64, actorStaffID int64, input ClientPaymentInput) (*domain.ClientPayment, error) {
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
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "client_payment", formatID(p.ID), string(p.Type),
		"Pembayaran client diperbarui")
	return p, nil
}

// Delete hard-deletes the payment and, first, its attached evidence -- see
// PaymentService.Delete for the full reasoning (evidence-first ordering,
// Owner-only role check living in the HTTP handler, not here).
func (s *ClientPaymentService) Delete(ctx context.Context, projectID, id int64, actorStaffID int64) error {
	p, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return err
	}
	if p == nil {
		return apperror.NotFound("Pembayaran tidak ditemukan")
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
