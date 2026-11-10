package application

import (
	"context"
	"strings"
	"time"

	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
)

// defaultSubscriptionChannel is the payment channel every subscription
// charge uses — task 8.3's request body has no `channel` field, mirroring
// payment's own defaultExternalChannel fallback for
// /external/payments/charges.
const defaultSubscriptionChannel = "QRIS"

// AppChargeWriter is the narrow slice of AppChargeRepository
// SubscriptionService needs — declared here (not *infrastructure.
// MySQLAppChargeRepository) so a stub can be injected for testing (A34).
type AppChargeWriter interface {
	Create(ctx context.Context, charge *domain.AppCharge) error
	Delete(ctx context.Context, orderRef string) error
}

// PlanCatalog is the narrow slice of PlanService SubscriptionService needs.
type PlanCatalog interface {
	// GetForApp returns the plan only when it belongs to appID (422 otherwise).
	GetForApp(ctx context.Context, appID string, id int64) (*domain.Plan, error)
}

type SubscriptionService struct {
	plans    PlanCatalog
	charges  AppChargeWriter
	payment  paymentcontracts.Client
	vouchers VoucherReserver
}

// VoucherReserver is the slice of VoucherService a subscription charge needs.
type VoucherReserver interface {
	Quote(ctx context.Context, code string, plan *domain.Plan, c domain.Customer) (*VoucherQuote, error)
	Reserve(ctx context.Context, code string, plan *domain.Plan, c domain.Customer, orderRef string) (*Reservation, error)
	Cancel(ctx context.Context, r *Reservation) error
}

// SetVouchers wires the voucher service (same module).
func (s *SubscriptionService) SetVouchers(v VoucherReserver) { s.vouchers = v }

// appCustomer: the subscriber of an App charge is the App itself — the
// ledger (subscription_app_charges) and the subscription period
// (CurrentUntilForApp) are keyed on it, and the charge contract carries no
// other required identity (customerEmail is optional).
func appCustomer(appID string) domain.Customer {
	return domain.Customer{Type: domain.CustomerApp, Ref: appID}
}

// QuoteVoucher previews voucherCode on the App's plan without reserving it
// (POST /external/subscriptions/voucher-check).
func (s *SubscriptionService) QuoteVoucher(ctx context.Context, appID string, planID int64, voucherCode string) (*VoucherQuote, error) {
	if appID == "" || planID == 0 {
		return nil, apperror.Validation("planId wajib diisi", nil)
	}
	plan, err := s.plans.GetForApp(ctx, appID, planID)
	if err != nil {
		return nil, err
	}
	return s.vouchers.Quote(ctx, voucherCode, plan, appCustomer(appID))
}

func NewSubscriptionService(plans PlanCatalog, charges AppChargeWriter, payment paymentcontracts.Client) *SubscriptionService {
	return &SubscriptionService{plans: plans, charges: charges, payment: payment}
}

// SubscriptionCharge is a started subscription charge and what the voucher
// (if any) took off the plan price. Charge.Amount is the amount charged.
type SubscriptionCharge struct {
	Charge         *paymentcontracts.ChargeResult
	OriginalAmount int64
	DiscountAmount int64
	VoucherCode    string
}

// StartSubscriptionCharge is the business logic behind
// POST /external/subscriptions/charges (D9) — kept out of the handler so it
// can be tested with stubs (A34/§7.1).
//
// Validates planID belongs to appID (the real guard against one App buying
// another App's plan — planId can be guessed, so the catalog endpoint alone
// isn't a sufficient gate; ADR-0040), reads price/duration
// from the catalog rather than trusting the caller (A18), writes the
// intent row BEFORE calling payment (A20) so a process death right after
// leaves a ghost row the reconciler can resolve via a 404 from
// CheckStatusForApp, and deletes that intent again if the charge call
// itself fails so orderRef stays retryable.
//
// With a voucherCode, one use of the voucher is reserved first (under a row
// lock, so quota and once-per-customer hold under concurrent charges) and the
// discounted amount is what is written and charged. The reservation is
// deleted again whenever the charge does not come to exist; the reconciler
// settles it (used / released) with the charge.
func (s *SubscriptionService) StartSubscriptionCharge(ctx context.Context, appID, orderRef string, planID int64, voucherCode string, opts paymentcontracts.ChargeOptions) (*SubscriptionCharge, error) {
	if appID == "" || orderRef == "" || planID == 0 {
		return nil, apperror.Validation("orderRef dan planId wajib diisi", nil)
	}

	plan, err := s.plans.GetForApp(ctx, appID, planID)
	if err != nil {
		return nil, err
	}

	out := &SubscriptionCharge{OriginalAmount: plan.Price}
	amount := plan.Price
	var reservation *Reservation
	if strings.TrimSpace(voucherCode) != "" {
		if s.vouchers == nil {
			return nil, apperror.Validation("Voucher tidak tersedia", map[string][]string{"voucherCode": {"Voucher tidak tersedia"}})
		}
		if reservation, err = s.vouchers.Reserve(ctx, voucherCode, plan, appCustomer(appID), orderRef); err != nil {
			return nil, err
		}
		amount = reservation.FinalAmount
		out.DiscountAmount = reservation.DiscountAmount
		out.VoucherCode = reservation.Code
	}
	cancelReservation := func() {
		if reservation != nil {
			_ = s.vouchers.Cancel(ctx, reservation) // best effort: ReleaseStale is the safety net
		}
	}

	if err := s.charges.Create(ctx, &domain.AppCharge{
		OrderRef:           orderRef,
		AppID:              appID,
		PlanID:             plan.ID,
		PlanName:           plan.Name,
		PlanPrice:          plan.Price,
		DiscountAmount:     out.DiscountAmount,
		Amount:             amount,
		VoucherCode:        out.VoucherCode,
		PlanDurationMonths: plan.DurationMonths,
	}); err != nil {
		cancelReservation()
		return nil, err
	}

	result, err := s.payment.CreateChannelCharge(ctx, appID, "", orderRef, amount, defaultSubscriptionChannel, opts)
	if err != nil {
		// Best-effort cleanup: if this Delete itself fails, the intent row
		// becomes a ghost the reconciler will still resolve (CheckStatusForApp
		// answers 404 for an order_ref the gateway never created, A20) — not
		// fatal to the caller, who already has the real error to report.
		_ = s.charges.Delete(ctx, orderRef)
		cancelReservation()
		return nil, err
	}
	out.Charge = result
	return out, nil
}

// ComputeUntil stacks a renewal on top of remaining time rather than
// discarding it: if currentUntil is still in the future, the new period
// starts from there so paying early never costs the customer time already
// owned. If currentUntil is nil or already in the past, the new period
// starts from now — paying after a lapse doesn't produce a subscription
// that's already expired. Deliberately stateless (doesn't read any
// SubscriptionService field) so it's testable without a database (§7.1).
func (s *SubscriptionService) ComputeUntil(currentUntil *time.Time, durationMonths int) time.Time {
	now := time.Now()
	base := now
	if currentUntil != nil && currentUntil.After(now) {
		base = *currentUntil
	}
	return base.AddDate(0, durationMonths, 0)
}
