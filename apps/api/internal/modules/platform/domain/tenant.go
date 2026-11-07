package domain

import "time"

type SubscriptionStatus string

// Subscription lifecycle (docs/plan/registrasi-wo-self-service/FLOW-IDEAL.md
// §1.4). The stored column is a cache refreshed by the status sweeper;
// ComputeLifecycle (subscription_lifecycle.go) is the source of truth.
const (
	StatusTrialing       SubscriptionStatus = "trialing"
	StatusActive         SubscriptionStatus = "active"
	StatusExpiringSoon   SubscriptionStatus = "expiring_soon"
	StatusPastDue        SubscriptionStatus = "past_due"
	StatusReadOnly       SubscriptionStatus = "read_only"
	StatusSuspended      SubscriptionStatus = "suspended"
	StatusPendingPayment SubscriptionStatus = "pending_payment"
	// StatusExpired is legacy (pre-Fase 1 rows); never produced anymore.
	StatusExpired SubscriptionStatus = "expired"
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
	// TrialEndsAt is set when a tenant starts on a free trial (every new
	// tenant, Fase 1); nil only for legacy rows.
	TrialEndsAt *time.Time
	// IsSuspended is a freeze by ElProof's platform team (abuse, fraud) —
	// separate from the billing lifecycle and blocks every principal.
	IsSuspended           bool
	LastCredentialResetAt *time.Time
	BrandColorPreset      string
	LogoStoragePath       *string
	// SignupSource: self-service signup (Fase 2) or created by ElProof's team.
	SignupSource SignupSource
	// OnboardingDismissedAt: the Owner closed the "Mulai di sini" checklist.
	OnboardingDismissedAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// BusinessDetails is what the Owner shares with clients: where the business is
// and which bank account clients pay into. All optional.
type BusinessDetails struct {
	Address           string
	BankName          string
	BankAccountNumber string
	BankAccountHolder string
}

// HasBank reports whether a complete account is set.
func (d BusinessDetails) HasBank() bool {
	return d.BankName != "" && d.BankAccountNumber != "" && d.BankAccountHolder != ""
}

// TenantFilter narrows the Platform Console's tenant list.
type TenantFilter struct {
	Search string
	// Status is the stored subscription status (refreshed hourly by the sweeper).
	Status string
	PlanID *int64
	// Source is "self_service" or "platform" (how the workspace came to exist).
	Source string
}

type SignupSource string

const (
	SignupSourcePlatform    SignupSource = "platform"
	SignupSourceSelfService SignupSource = "self_service"
)
