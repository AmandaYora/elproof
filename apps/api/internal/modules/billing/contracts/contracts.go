// Package contracts is the ONLY package other modules may import from billing.
package contracts

import (
	"context"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
)

type TransactionType string
type TransactionStatus string

const (
	TransactionNew     TransactionType = "new"
	TransactionRenewal TransactionType = "renewal"

	StatusUnpaid  TransactionStatus = "unpaid"
	StatusPending TransactionStatus = "pending"
	StatusPaid    TransactionStatus = "paid"
	StatusExpired TransactionStatus = "expired"
	StatusGranted TransactionStatus = "granted"
)

type Plan struct {
	ID             int64
	Name           string
	DurationMonths int
	Price          int64
}

type RecordTransactionInput struct {
	TenantID         int64
	Type             TransactionType
	Amount           int64
	PaymentMethod    string
	PaymentReference string
	Status           TransactionStatus
}

// Contracts is what `platform` calls to read the plan catalog and append to
// the transaction ledger when it orchestrates tenant registration, manual
// activation, and self-service payment — `platform` owns the resulting
// tenant-status update, `billing` only owns its own tables. See ADR-0008.
type Contracts interface {
	GetPlan(ctx context.Context, id int64) (*Plan, error)
	RecordTransaction(ctx context.Context, input RecordTransactionInput) error
	UpdateTransactionStatus(ctx context.Context, paymentReference string, status TransactionStatus) error
}

type impl struct {
	plans        *application.PlanService
	transactions *application.TransactionService
}

func New(plans *application.PlanService, transactions *application.TransactionService) Contracts {
	return &impl{plans: plans, transactions: transactions}
}

func (c *impl) GetPlan(ctx context.Context, id int64) (*Plan, error) {
	plan, err := c.plans.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Plan{ID: plan.ID, Name: plan.Name, DurationMonths: plan.DurationMonths, Price: plan.Price}, nil
}

func (c *impl) RecordTransaction(ctx context.Context, input RecordTransactionInput) error {
	_, err := c.transactions.RecordTransaction(ctx, application.RecordTransactionInput{
		TenantID:         input.TenantID,
		Type:             domain.TransactionType(input.Type),
		Amount:           input.Amount,
		PaymentMethod:    input.PaymentMethod,
		PaymentReference: input.PaymentReference,
		Status:           domain.TransactionStatus(input.Status),
	})
	return err
}

func (c *impl) UpdateTransactionStatus(ctx context.Context, paymentReference string, status TransactionStatus) error {
	return c.transactions.UpdateStatus(ctx, paymentReference, domain.TransactionStatus(status))
}
