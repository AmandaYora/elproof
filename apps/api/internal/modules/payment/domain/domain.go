// Package domain holds generic, provider-agnostic value objects for the
// `payment` module. None of these types represent a business ledger — see
// MODULE_PAYMENT.md §4: this module only ever knows about gateway config,
// the App registry, a thin dispatch index, and webhook idempotency.
package domain

import "time"

type AppKind string

const (
	AppKindInternal AppKind = "internal"
	AppKindExternal AppKind = "external"
)

// App is one row in the registry — one consumer allowed to create charges
// through the single active gateway credential.
type App struct {
	ID              int64
	AppID           string
	Name            string
	Kind            AppKind
	SecretHash      string
	SecretEncrypted string
	CallbackURL     string
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// GatewayConfig is the single-row config for the one active provider
// credential — see MODULE_PAYMENT.md §4 "Gateway Config (baris tunggal)".
type GatewayConfig struct {
	ID                        int64
	ActiveProvider            string
	IsSandbox                 bool
	TripayMerchantCode        string
	TripayAPIKeyEncrypted     string
	TripayPrivateKeyEncrypted string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// Enabled reports whether a provider is actually configured — false means
// the module is in "simulation mode" (see contracts.Client.Enabled).
func (c GatewayConfig) Enabled() bool {
	return c.ActiveProvider != ""
}

// ChargeDispatch is the thin order_ref -> app_id index — never a ledger.
type ChargeDispatch struct {
	OrderRef    string
	AppID       string
	ProviderRef string
	CreatedAt   time.Time
}

// Channel is a payment channel (QRIS, a specific VA bank, etc.) as reported
// live by the active gateway — never a hardcoded list. Fees are flat + a
// percentage of the charge amount (Tripay's actual fee shape for most
// channels, QRIS included).
type Channel struct {
	Code               string
	Name               string
	Type               string
	FeeCustomerFlat    int64
	FeeCustomerPercent float64
	FeeMerchantFlat    int64
	FeeMerchantPercent float64
	IconURL            string
	Active             bool
}

// QuoteFee computes the customer-facing fee for a charge of the given amount
// on this channel — flat component plus a percentage of the amount.
func (c Channel) QuoteFee(amount int64) int64 {
	return c.FeeCustomerFlat + int64(float64(amount)*c.FeeCustomerPercent/100)
}

// Charge is the result of creating (or checking) a charge at the gateway.
type Charge struct {
	OrderRef    string
	ProviderRef string
	Channel     string
	QRImageURL  string
	PayCode     string
	CheckoutURL string
	Amount      int64
	FeeAmount   int64
	ExpiresAt   time.Time
	Status      ChargeStatus
}

type ChargeStatus string

const (
	ChargeUnpaid  ChargeStatus = "unpaid"
	ChargePaid    ChargeStatus = "paid"
	ChargeExpired ChargeStatus = "expired"
	ChargeFailed  ChargeStatus = "failed"
	ChargeRefund  ChargeStatus = "refund"
)

// WebhookEvent is the generic, provider-agnostic shape a gateway callback is
// parsed into before being dispatched to whichever App owns the order ref.
type WebhookEvent struct {
	ProviderRef string
	OrderRef    string
	Status      ChargeStatus
	Amount      int64
	PaidAt      time.Time
}

func (e WebhookEvent) Paid() bool {
	return e.Status == ChargePaid
}
