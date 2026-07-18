// Package contracts is the ONLY package other modules may import from
// `payment` — see MODULE_PAYMENT.md §3/§5.
package contracts

import (
	"context"
	"time"
)

// InternalAppBilling is the fixed App ID `platform` registers itself under —
// shared as a constant here (rather than a string literal duplicated in both
// modules) so the bootstrap-seed in payment.module.go and platform's own
// calls to CreateCharge/RegisterConsumer can never drift out of sync.
const InternalAppBilling = "platform-billing"

// Channel is serialized directly as the /external/payments/channels response
// body (Fase 10) — json tags keep it camelCase, consistent with every other
// endpoint in this API, since this is the only place this type is ever
// marshaled (no frontend consumer exists yet).
type Channel struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	FeeCustomer int64  `json:"feeCustomer"`
	FeeMerchant int64  `json:"feeMerchant"`
	IconURL     string `json:"iconUrl"`
	Active      bool   `json:"active"`
}

// ChargeResult is what CreateCharge/CreateChannelCharge/CheckStatus return —
// enough for a caller to render a QRIS image or VA pay code and to poll
// status, without ever depending on which gateway produced it.
type ChargeResult struct {
	OrderRef    string
	ProviderRef string
	Channel     string
	QRImageURL  string
	PayCode     string
	CheckoutURL string
	Amount      int64
	FeeAmount   int64
	ExpiresAt   time.Time
	Status      string
}

// WebhookEvent is the generic, provider-agnostic shape a gateway callback is
// parsed into before being dispatched to the App that owns the order ref.
type WebhookEvent struct {
	ProviderRef string
	OrderRef    string
	Paid        bool
	Amount      int64
	PaidAt      time.Time
}

type ChargeOptions struct {
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
}

// Client is the contract an App internal calls directly, in-process — see
// MODULE_PAYMENT.md §5. `contextID` is opaque to the gateway itself (tenant/
// workspace/whatever the caller needs) and may be left empty.
type Client interface {
	Enabled(ctx context.Context) (bool, error)
	QuoteFee(ctx context.Context, channel string, amount int64) (int64, error)
	CreateCharge(ctx context.Context, appID, contextID, orderRef string, amount int64) (*ChargeResult, error)
	CreateChannelCharge(ctx context.Context, appID, contextID, orderRef string, amount int64, channel string, opts ChargeOptions) (*ChargeResult, error)
	ListChannels(ctx context.Context) ([]Channel, error)
	CheckStatus(ctx context.Context, providerRef string) (*ChargeResult, error)
}

// WebhookConsumer is implemented by every App internal that registers with
// the Dispatcher — invoked in-process, same request, when a webhook event
// for that App's order ref arrives (see MODULE_PAYMENT.md §6).
type WebhookConsumer interface {
	ApplyWebhookEvent(ctx context.Context, orderRef string, event WebhookEvent) error
}

// Dispatcher is used only by this module's own composition-root wiring and
// presentation layer — never by an App (see MODULE_PAYMENT.md §5).
type Dispatcher interface {
	RegisterConsumer(appID string, consumer WebhookConsumer)
}
