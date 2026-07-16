package application

import (
	"context"
	"time"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/pagination"
)

type TransactionRepository interface {
	List(ctx context.Context, tenantID *int64) ([]domain.Transaction, error)
	ListPaginated(ctx context.Context, tenantID *int64, params pagination.Params, status string) ([]domain.Transaction, int64, error)
	Create(ctx context.Context, tx *domain.Transaction) error
	UpdateStatusByReference(ctx context.Context, paymentReference string, status domain.TransactionStatus, paidAt *time.Time) error
}

type TransactionService struct {
	repo TransactionRepository
}

func NewTransactionService(repo TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) List(ctx context.Context, tenantID *int64) ([]domain.Transaction, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *TransactionService) ListPaginated(ctx context.Context, tenantID *int64, params pagination.Params, status string) ([]domain.Transaction, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, params, status)
}

type RecordTransactionInput struct {
	TenantID         int64
	Type             domain.TransactionType
	Amount           int64
	PaymentMethod    string
	PaymentReference string
	Status           domain.TransactionStatus
}

// RecordTransaction is called, via contracts, by `platform`'s orchestration
// for tenant registration (status "unpaid"), self-service payment ("paid"),
// and manual activation ("granted") — only "paid"/"granted" get a PaidAt.
func (s *TransactionService) RecordTransaction(ctx context.Context, input RecordTransactionInput) (*domain.Transaction, error) {
	tx := &domain.Transaction{
		TenantID:         input.TenantID,
		Type:             input.Type,
		Amount:           input.Amount,
		PaymentMethod:    input.PaymentMethod,
		PaymentReference: input.PaymentReference,
		Status:           input.Status,
	}
	if input.Status == domain.StatusPaid || input.Status == domain.StatusGranted {
		now := time.Now()
		tx.PaidAt = &now
	}
	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// UpdateStatus is called, via contracts, by `platform`'s webhook consumer
// (Fase 9) once a gateway charge is confirmed paid/expired — stamps PaidAt
// itself when the new status is Paid, same convention as RecordTransaction.
func (s *TransactionService) UpdateStatus(ctx context.Context, paymentReference string, status domain.TransactionStatus) error {
	var paidAt *time.Time
	if status == domain.StatusPaid {
		now := time.Now()
		paidAt = &now
	}
	return s.repo.UpdateStatusByReference(ctx, paymentReference, status, paidAt)
}
