package domain

import "time"

// AppChargeStatus mirrors the subscription_app_charges.status ENUM
// (migration 000044).
type AppChargeStatus string

const (
	AppChargePending   AppChargeStatus = "pending"
	AppChargePaid      AppChargeStatus = "paid"
	AppChargeExpired   AppChargeStatus = "expired"
	AppChargeFailed    AppChargeStatus = "failed"
	AppChargeCancelled AppChargeStatus = "cancelled"
)

// AppCharge is a subscription-purchase intent for an external App (D7/D9),
// snapshotting the plan's name/price/duration at the moment the intent is
// written — before payment.Client is ever called (A20) — so a later catalog
// price change never rewrites what a customer already agreed to pay.
type AppCharge struct {
	OrderRef  string
	AppID     string
	PlanID    int64
	PlanName  string
	PlanPrice int64
	// DiscountAmount is the voucher discount (0 without one) and Amount what
	// the gateway was asked to charge: PlanPrice - DiscountAmount.
	DiscountAmount     int64
	Amount             int64
	VoucherCode        string
	PlanDurationMonths int
	Status             AppChargeStatus
	SubscriptionUntil  *time.Time
	CreatedAt          time.Time
	PaidAt             *time.Time
	ResolvedAt         *time.Time
}

// AppSubscriptionSummary is one row of "who's paying for what, until when" —
// backs GET /api/v1/app-subscriptions (T13/D7).
type AppSubscriptionSummary struct {
	AppID             string
	PlanName          string
	SubscriptionUntil time.Time
	Status            AppChargeStatus
}
