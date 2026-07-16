package infrastructure

import (
	"context"

	"elproof/internal/modules/payment/domain"
)

// ChargeRequest is the provider-agnostic input for creating a charge.
type ChargeRequest struct {
	OrderRef      string
	Amount        int64
	Channel       string // gateway channel code, e.g. "QRIS"
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	CallbackURL   string
	ExpiresInSecs int
}

// gateway is the provider-agnostic interface every provider adapter (Tripay
// today, Midtrans/Xendit later) implements — see MODULE_PAYMENT.md §3. Only
// `client.go` (same package) depends on this; it is never part of the public
// `contracts` package.
type gateway interface {
	Name() string
	ListChannels(ctx context.Context) ([]domain.Channel, error)
	CreateCharge(ctx context.Context, req ChargeRequest) (*domain.Charge, error)
	GetChargeStatus(ctx context.Context, providerRef string) (*domain.Charge, error)
	// VerifyAndParseWebhook checks the raw request signature and, if valid,
	// returns the parsed event plus a provider-scoped event id (used for the
	// webhook idempotency log) and the raw event id/name for logging.
	VerifyAndParseWebhook(signature string, rawBody []byte) (event *domain.WebhookEvent, eventID string, err error)
}
