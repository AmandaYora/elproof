// Package contracts is the ONLY package other modules may import from billing.
package contracts

import (
	"context"
	"strconv"
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
	TenantID int64
	Type     TransactionType
	Amount   int64
	// OriginalAmount, DiscountAmount and VoucherCode come from a VoucherQuote
	// when a voucher was used (Amount is then its FinalAmount); otherwise 0/"".
	OriginalAmount   int64
	DiscountAmount   int64
	VoucherCode      string
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
	// ReserveTenantVoucher takes one use of a voucher code on one of ElProof's
	// plans for the charge orderRef of a WO tenant (checked under a lock; the
	// preview before paying is billing's own POST /subscriptions/voucher-check).
	// A refused code is a 422 on voucherCode with errors.code (voucher_expired,
	// …). The use is settled by UpdateTransactionStatus on orderRef (paid ->
	// used; expired/cancelled -> released); if the charge is never recorded,
	// call ReleaseVoucher.
	ReserveTenantVoucher(ctx context.Context, tenantID, planID int64, code, orderRef string) (*VoucherReservation, error)
	// ReleaseVoucher deletes a reservation whose charge never came to exist.
	ReleaseVoucher(ctx context.Context, r *VoucherReservation) error
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

// VoucherQuote is what a voucher does to a plan price.
type VoucherQuote struct {
	Code           string
	OriginalAmount int64
	DiscountAmount int64
	FinalAmount    int64
}

// VoucherReservation is a reserved voucher use (opaque id for ReleaseVoucher).
type VoucherReservation struct {
	VoucherQuote
	id int64
}

type impl struct {
	plans        *application.PlanService
	transactions *application.TransactionService
	vouchers     *application.VoucherService
}

func New(plans *application.PlanService, transactions *application.TransactionService, vouchers *application.VoucherService) Contracts {
	return &impl{plans: plans, transactions: transactions, vouchers: vouchers}
}

func tenantCustomer(tenantID int64) domain.Customer {
	return domain.Customer{Type: domain.CustomerTenant, Ref: strconv.FormatInt(tenantID, 10)}
}

func quoteOf(q application.VoucherQuote) VoucherQuote {
	return VoucherQuote{Code: q.Code, OriginalAmount: q.OriginalAmount, DiscountAmount: q.DiscountAmount, FinalAmount: q.FinalAmount}
}

func (c *impl) ReserveTenantVoucher(ctx context.Context, tenantID, planID int64, code, orderRef string) (*VoucherReservation, error) {
	plan, err := c.plans.GetTenantPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	r, err := c.vouchers.Reserve(ctx, code, plan, tenantCustomer(tenantID), orderRef)
	if err != nil {
		return nil, err
	}
	return &VoucherReservation{VoucherQuote: quoteOf(r.VoucherQuote), id: r.RedemptionID}, nil
}

func (c *impl) ReleaseVoucher(ctx context.Context, r *VoucherReservation) error {
	if r == nil {
		return nil
	}
	return c.vouchers.Cancel(ctx, &application.Reservation{RedemptionID: r.id})
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
		OriginalAmount:   input.OriginalAmount,
		DiscountAmount:   input.DiscountAmount,
		VoucherCode:      input.VoucherCode,
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
