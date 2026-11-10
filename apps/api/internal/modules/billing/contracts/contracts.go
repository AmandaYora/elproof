// Package contracts is the ONLY package other modules may import from billing.
package contracts

import (
	"context"
	"time"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
)

type TransactionType string
type TransactionStatus string

const (
	TransactionNew     TransactionType = "new"
	TransactionRenewal TransactionType = "renewal"

	StatusUnpaid    TransactionStatus = "unpaid"
	StatusPending   TransactionStatus = "pending"
	StatusPaid      TransactionStatus = "paid"
	StatusExpired   TransactionStatus = "expired"
	StatusGranted   TransactionStatus = "granted"
	StatusCancelled TransactionStatus = "cancelled"
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
	// GetPlan reads any plan (e.g. to name the plan a tenant already has).
	GetPlan(ctx context.Context, id int64) (*Plan, error)
	// GetTenantPlan reads a plan a WO tenant may be subscribed to: only a plan
	// of ElProof's own catalog. Any other plan — one sold by another App —
	// answers 422 on planId, exactly like a plan that does not exist.
	GetTenantPlan(ctx context.Context, id int64) (*Plan, error)
	// ListPlans lists the plan catalog (the Platform Console orders its tenant
	// list by plan name, which this module owns).
	ListPlans(ctx context.Context) ([]Plan, error)
	RecordTransaction(ctx context.Context, input RecordTransactionInput) error
	UpdateTransactionStatus(ctx context.Context, paymentReference string, status TransactionStatus) error
	// PaidSince lists paid (never granted) subscription transactions since a
	// moment, for the platform revenue metrics.
	PaidSince(ctx context.Context, since time.Time) ([]PaidTransaction, error)
}

// PaidTransaction is one paid subscription transaction.
type PaidTransaction struct {
	TenantID int64
	Type     TransactionType
	Amount   int64
	PaidAt   time.Time
}

type impl struct {
	plans        *application.PlanService
	transactions *application.TransactionService
}

func New(plans *application.PlanService, transactions *application.TransactionService) Contracts {
	return &impl{plans: plans, transactions: transactions}
}

func (c *impl) ListPlans(ctx context.Context) ([]Plan, error) {
	plans, err := c.plans.List(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]Plan, 0, len(plans))
	for _, p := range plans {
		out = append(out, Plan{ID: p.ID, Name: p.Name, DurationMonths: p.DurationMonths, Price: p.Price})
	}
	return out, nil
}

func (c *impl) GetPlan(ctx context.Context, id int64) (*Plan, error) {
	plan, err := c.plans.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Plan{ID: plan.ID, Name: plan.Name, DurationMonths: plan.DurationMonths, Price: plan.Price}, nil
}

func (c *impl) GetTenantPlan(ctx context.Context, id int64) (*Plan, error) {
	plan, err := c.plans.GetTenantPlan(ctx, id)
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

func (c *impl) PaidSince(ctx context.Context, since time.Time) ([]PaidTransaction, error) {
	list, err := c.transactions.PaidSince(ctx, since)
	if err != nil {
		return nil, err
	}
	out := make([]PaidTransaction, 0, len(list))
	for _, t := range list {
		if t.PaidAt == nil {
			continue
		}
		out = append(out, PaidTransaction{TenantID: t.TenantID, Type: TransactionType(t.Type), Amount: t.Amount, PaidAt: *t.PaidAt})
	}
	return out, nil
}
