package application

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
)

// Mirrors payment/domain.ChargeStatus's five values (verified PLAN.md T15,
// payment/domain package) — duplicated here as plain strings because
// `billing` may only import `payment/contracts`, never `payment/domain`
// (module-boundary rule). paymentcontracts.ChargeResult.Status is already a
// plain string for the same reason.
const (
	gatewayStatusUnpaid  = "unpaid"
	gatewayStatusPaid    = "paid"
	gatewayStatusExpired = "expired"
	gatewayStatusFailed  = "failed"
	gatewayStatusRefund  = "refund"
)

const (
	// subscriptionReconcileMinAge keeps a charge out of the sweep until
	// it's had a realistic chance to be paid — mirrors payment's own
	// reconcileMinAge.
	subscriptionReconcileMinAge = 2 * time.Minute
	// subscriptionReconcileBatchLimit caps how many charges one sweep tick
	// re-checks — mirrors payment's own reconcileBatchLimit.
	subscriptionReconcileBatchLimit = 50
	// subscriptionReconcileConcurrency bounds how many charges are checked
	// against the gateway at once within a single sweep tick — mirrors
	// payment's own reconcileConcurrency (A32: the two reconcilers call the
	// gateway independently, which is an accepted, documented consequence
	// of A16, not a rate-limit bug).
	subscriptionReconcileConcurrency = 10
)

// reconcileAction is what decideOutcome tells the reconciler to do with one
// charge.
type reconcileAction int

const (
	actionSkip reconcileAction = iota
	actionMarkPaid
	actionResolve
)

type reconcileDecision struct {
	action reconcileAction
	status domain.AppChargeStatus // meaningful only when action == actionResolve
}

// decideOutcome is a pure function (8.6b) mapping one CheckStatusForApp
// result to what the reconciler should do next — the only part of the
// reconciler testable without a database (TestMapGatewayStatus, §7.1).
//
// A call failure that ISN'T a confirmed 404 ghost row always yields
// actionSkip, no matter how old the charge is — the lesson from
// docs/plan/migrasi-data-jws/PLAN.md's D17: "we don't know" must never be
// treated as "not paid".
func decideOutcome(status string, statusErr error, createdAt, now time.Time, maxAge time.Duration) reconcileDecision {
	if statusErr != nil {
		if isNotFoundErr(statusErr) {
			// Ghost row (A20): the intent was written but payment.Client's
			// CreateChannelCharge was never (successfully) called, so
			// payment has no dispatch row for this order_ref at all —
			// there is no gateway state to ever converge on, so resolve
			// immediately rather than waiting for maxAge.
			return reconcileDecision{action: actionResolve, status: domain.AppChargeCancelled}
		}
		return reconcileDecision{action: actionSkip}
	}

	switch status {
	case gatewayStatusPaid:
		return reconcileDecision{action: actionMarkPaid}
	case gatewayStatusExpired:
		return reconcileDecision{action: actionResolve, status: domain.AppChargeExpired}
	case gatewayStatusFailed:
		return reconcileDecision{action: actionResolve, status: domain.AppChargeFailed}
	case gatewayStatusRefund:
		// subscription_app_charges.status has no "refund" value — the
		// nearest terminal-negative bucket is "cancelled" (the same one
		// the 404 ghost-row case above resolves to): a refunded charge
		// never ends in an active paid subscription either.
		return reconcileDecision{action: actionResolve, status: domain.AppChargeCancelled}
	case gatewayStatusUnpaid:
		if now.Sub(createdAt) > maxAge {
			return reconcileDecision{action: actionResolve, status: domain.AppChargeExpired}
		}
		return reconcileDecision{action: actionSkip}
	default:
		return reconcileDecision{action: actionSkip}
	}
}

func isNotFoundErr(err error) bool {
	appErr, ok := apperror.As(err)
	return ok && appErr.Kind == apperror.KindNotFound
}

// AppChargeReconcileRepo is the narrow slice of AppChargeRepository the
// reconciler needs — declared here so a stub can be injected for tests
// (A34).
type AppChargeReconcileRepo interface {
	FindUnresolved(ctx context.Context, olderThan time.Time, limit int) ([]domain.AppCharge, error)
	MarkPaidIfUnresolved(ctx context.Context, orderRef string, until time.Time) (bool, error)
	MarkResolvedIfUnresolved(ctx context.Context, orderRef string, status domain.AppChargeStatus) (bool, error)
	CurrentUntilForApp(ctx context.Context, appID string) (*time.Time, error)
}

// UntilComputer is the narrow slice of SubscriptionService the reconciler
// needs.
type UntilComputer interface {
	ComputeUntil(currentUntil *time.Time, durationMonths int) time.Time
}

// SubscriptionReconciler is the safety net for a subscription charge whose
// webhook was never received — polls payment.Client.CheckStatusForApp
// directly (A16). Deliberately isolated from payment's own switch-based
// dispatch (T15): payment's `payment` module is never touched by this file.
type SubscriptionReconciler struct {
	charges  AppChargeReconcileRepo
	payment  paymentcontracts.Client
	until    UntilComputer
	maxAge   time.Duration
	vouchers VoucherSweeper
}

func NewSubscriptionReconciler(charges AppChargeReconcileRepo, payment paymentcontracts.Client, until UntilComputer, maxAge time.Duration) *SubscriptionReconciler {
	return &SubscriptionReconciler{charges: charges, payment: payment, until: until, maxAge: maxAge}
}

// SetMaxAge overrides the max-age threshold after construction — used by
// the composition root to apply cfg.SubscriptionChargeMaxAge once config is
// loaded, without changing NewModule's fixed 3-argument shape (task 9.1).
func (s *SubscriptionReconciler) SetMaxAge(maxAge time.Duration) {
	s.maxAge = maxAge
}

// Start runs ReconcilePending on a fixed interval until ctx is cancelled —
// same shape as payment.PaymentService.StartReconciler. Errors are logged,
// never fatal.
func (s *SubscriptionReconciler) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.releaseStaleVouchers(ctx)
				checked, resolved, err := s.ReconcilePending(ctx)
				if err != nil {
					log.Printf("billing: subscription reconciliation sweep gagal: %v", err)
					continue
				}
				if checked > 0 {
					log.Printf("billing: subscription reconciliation sweep — %d diperiksa, %d diselesaikan", checked, resolved)
				}
			}
		}
	}()
}

// ReconcilePending re-checks every still-open subscription intent directly
// against the gateway. Bounded fan-out (subscriptionReconcileConcurrency),
// same reasoning as payment's own sweep: each iteration's dominant cost is
// a live network round trip.
func (s *SubscriptionReconciler) ReconcilePending(ctx context.Context) (checked int, resolved int, err error) {
	pending, err := s.charges.FindUnresolved(ctx, time.Now().Add(-subscriptionReconcileMinAge), subscriptionReconcileBatchLimit)
	if err != nil {
		return 0, 0, err
	}

	sem := make(chan struct{}, subscriptionReconcileConcurrency)
	var wg sync.WaitGroup
	var resolvedCount int64
	for _, charge := range pending {
		wg.Add(1)
		sem <- struct{}{}
		go func(c domain.AppCharge) {
			defer wg.Done()
			defer func() { <-sem }()
			if s.reconcileOne(ctx, c) {
				atomic.AddInt64(&resolvedCount, 1)
			}
		}(charge)
	}
	wg.Wait()

	return len(pending), int(resolvedCount), nil
}

// reconcileOne handles a single charge, returning whether it ended up
// resolved (a terminal outcome was written) this round.
func (s *SubscriptionReconciler) reconcileOne(ctx context.Context, charge domain.AppCharge) bool {
	result, statusErr := s.payment.CheckStatusForApp(ctx, charge.AppID, charge.OrderRef)
	status := ""
	if result != nil {
		status = result.Status
	}
	decision := decideOutcome(status, statusErr, charge.CreatedAt, time.Now(), s.maxAge)

	switch decision.action {
	case actionMarkPaid:
		currentUntil, err := s.charges.CurrentUntilForApp(ctx, charge.AppID)
		if err != nil {
			log.Printf("billing: subscription reconciliation — gagal membaca subscription_until App %q: %v", charge.AppID, err)
			return false
		}
		// until is computed BEFORE the write — it's an input to
		// MarkPaidIfUnresolved, not something that write derives itself.
		until := s.until.ComputeUntil(currentUntil, charge.PlanDurationMonths)
		won, err := s.charges.MarkPaidIfUnresolved(ctx, charge.OrderRef, until)
		if err != nil {
			log.Printf("billing: subscription reconciliation — gagal menandai lunas order_ref %q: %v", charge.OrderRef, err)
			return false
		}
		if won {
			s.settleVoucher(ctx, charge.OrderRef, true)
		}
		return won
	case actionResolve:
		won, err := s.charges.MarkResolvedIfUnresolved(ctx, charge.OrderRef, decision.status)
		if err != nil {
			log.Printf("billing: subscription reconciliation — gagal menyelesaikan order_ref %q: %v", charge.OrderRef, err)
			return false
		}
		if won {
			s.settleVoucher(ctx, charge.OrderRef, false)
		}
		return won
	default: // actionSkip
		return false
	}
}

// SetVouchers wires the voucher ledger: a charge's voucher use is settled
// with the charge, and reservations left behind by a dead process are
// released on every tick.
func (s *SubscriptionReconciler) SetVouchers(v VoucherSweeper) { s.vouchers = v }

// VoucherSweeper is the slice of VoucherService the reconciler needs.
type VoucherSweeper interface {
	Settle(ctx context.Context, orderRef string, paid bool) error
	ReleaseStale(ctx context.Context) (int64, error)
}

// settleVoucher is best effort: a use left reserved after a paid charge
// still counts (as it should); one left after a failed charge is released
// by the stale sweep once the charge is no longer pending.
func (s *SubscriptionReconciler) settleVoucher(ctx context.Context, orderRef string, paid bool) {
	if s.vouchers == nil {
		return
	}
	if err := s.vouchers.Settle(ctx, orderRef, paid); err != nil {
		log.Printf("billing: gagal mencatat pemakaian voucher order_ref %q: %v", orderRef, err)
	}
}

func (s *SubscriptionReconciler) releaseStaleVouchers(ctx context.Context) {
	if s.vouchers == nil {
		return
	}
	n, err := s.vouchers.ReleaseStale(ctx)
	if err != nil {
		log.Printf("billing: gagal melepas voucher yang tertahan: %v", err)
		return
	}
	if n > 0 {
		log.Printf("billing: %d pemakaian voucher tanpa tagihan dilepas", n)
	}
}
