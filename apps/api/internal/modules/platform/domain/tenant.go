package domain

import "time"

type SubscriptionStatus string

const (
	StatusActive        SubscriptionStatus = "active"
	StatusExpiringSoon   SubscriptionStatus = "expiring_soon"
	StatusExpired        SubscriptionStatus = "expired"
	StatusPendingPayment SubscriptionStatus = "pending_payment"
)

// PendingCharge remembers, for a subscription charge still awaiting gateway
// confirmation, which tenant+plan it was for — `payment` itself is never
// told about tenants/plans (see MODULE_PAYMENT.md's non-goals), so this
// mapping has to live in `platform`, not inside the payment module.
type PendingCharge struct {
	OrderRef string
	TenantID int64
	PlanID   int64
}

type Tenant struct {
	ID                    int64
	BusinessName          string
	OwnerName             string
	Username              string
	Email                 string
	Phone                 string
	City                  string
	JoinedAt              time.Time
	PlanID                *int64
	SubscriptionStatus    SubscriptionStatus
	SubscriptionExpiresAt *time.Time
	IsSuspended           bool
	LastCredentialResetAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
