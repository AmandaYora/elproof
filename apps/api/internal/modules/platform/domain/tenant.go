package domain

import (
	"errors"
	"time"
)

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
	CustomDomain          *string
	// SignupSource: self-service signup (Fase 2) or created by ElProof's team.
	SignupSource SignupSource
	// OnboardingDismissedAt: the Owner closed the "Mulai di sini" checklist.
	OnboardingDismissedAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type SignupSource string

const (
	SignupSourcePlatform    SignupSource = "platform"
	SignupSourceSelfService SignupSource = "self_service"
)

// ErrDuplicateCustomDomain signals a unique-key collision on custom_domain —
// the repository returns this instead of a raw driver error so the
// application layer can translate it into a field-level validation error
// (see ADR-0015) without knowing MySQL error codes itself.
var ErrDuplicateCustomDomain = errors.New("domain kustom sudah digunakan tenant lain")
