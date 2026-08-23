package application

import (
	"context"
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
	GetAppPlanIDs(ctx context.Context, appID string) ([]int64, error)
	Get(ctx context.Context, id int64) (*domain.Plan, error)
}

type SubscriptionService struct {
	plans   PlanCatalog
	charges AppChargeWriter
	payment paymentcontracts.Client
}

func NewSubscriptionService(plans PlanCatalog, charges AppChargeWriter, payment paymentcontracts.Client) *SubscriptionService {
	return &SubscriptionService{plans: plans, charges: charges, payment: payment}
}

// StartSubscriptionCharge is the business logic behind
// POST /external/subscriptions/charges (D9) — kept out of the handler so it
// can be tested with stubs (A34/§7.1).
//
// Validates planID is actually mapped to appID (the real guard against one
// App buying another App's plan — R1/R2; planId can be guessed, so the
// catalog endpoint alone isn't a sufficient gate), reads price/duration
// from the catalog rather than trusting the caller (A18), writes the
// intent row BEFORE calling payment (A20) so a process death right after
// leaves a ghost row the reconciler can resolve via a 404 from
// CheckStatusForApp, and deletes that intent again if the charge call
// itself fails so orderRef stays retryable.
func (s *SubscriptionService) StartSubscriptionCharge(ctx context.Context, appID, orderRef string, planID int64, opts paymentcontracts.ChargeOptions) (*paymentcontracts.ChargeResult, error) {
	if appID == "" || orderRef == "" || planID == 0 {
		return nil, apperror.Validation("orderRef dan planId wajib diisi", nil)
	}

	mappedIDs, err := s.plans.GetAppPlanIDs(ctx, appID)
	if err != nil {
		return nil, err
	}
	if !containsInt64(mappedIDs, planID) {
		return nil, apperror.Validation("Paket ini tidak tersedia untuk aplikasi Anda", nil)
	}

	plan, err := s.plans.Get(ctx, planID)
	if err != nil {
		return nil, err
	}

	if err := s.charges.Create(ctx, &domain.AppCharge{
		OrderRef:           orderRef,
		AppID:              appID,
		PlanID:             plan.ID,
		PlanName:           plan.Name,
		PlanPrice:          plan.Price,
		PlanDurationMonths: plan.DurationMonths,
	}); err != nil {
		return nil, err
	}

	result, err := s.payment.CreateChannelCharge(ctx, appID, "", orderRef, plan.Price, defaultSubscriptionChannel, opts)
	if err != nil {
		// Best-effort cleanup: if this Delete itself fails, the intent row
		// becomes a ghost the reconciler will still resolve (CheckStatusForApp
		// answers 404 for an order_ref the gateway never created, A20) — not
		// fatal to the caller, who already has the real error to report.
		_ = s.charges.Delete(ctx, orderRef)
		return nil, err
	}
	return result, nil
}

func containsInt64(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
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
