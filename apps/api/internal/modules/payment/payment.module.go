// Package payment wraps a single payment-gateway merchant account (Tripay)
// behind a "one wallet, many consumers" pattern — see MODULE_PAYMENT.md.
// Fase 9 wires up internal-mode only (App internal, in-process); Fase 10
// adds external-mode (App eksternal, HTTP + client-credentials auth) on top
// without touching how App internal works.
package payment

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	identitycontracts "elproof/internal/modules/identity/contracts"
	"elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/payment/infrastructure"
	"elproof/internal/modules/payment/presentation"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
)

type Module struct {
	service *infrastructure.PaymentService
	handler *presentation.Handler

	// appTokenLimiter guards POST /auth/app/token — separate limiter
	// instance per module (not shared global state) so tests/restarts start
	// with a clean budget.
	appTokenLimiter *middleware.RateLimiter
}

func NewModule(db *sql.DB, encryptionKey string, identity identitycontracts.Contracts, appTokenTTL time.Duration) (*Module, error) {
	service, err := infrastructure.NewPaymentService(db, encryptionKey, identity, appTokenTTL)
	if err != nil {
		return nil, err
	}

	// Bootstrap the fixed internal App row every startup — idempotent, see
	// MySQLAppRepository.EnsureInternalApp. A fresh `docker compose up` (or
	// any clean clone) needs this to exist without a manual seed step.
	if err := service.EnsureInternalApp(context.Background(), contracts.InternalAppBilling, "ElProof Billing (internal)"); err != nil {
		log.Printf("payment: gagal bootstrap App internal %q: %v", contracts.InternalAppBilling, err)
	}

	return &Module{
		service: service,
		handler: presentation.NewHandler(service),
		// 10 attempts/minute/IP — strict enough to blunt secret-guessing
		// against /auth/app/token (see knowledge/MODULE_PAYMENT.md §7.1),
		// generous enough that a legitimate App re-authenticating after
		// token expiry never trips it under normal use.
		appTokenLimiter: middleware.NewRateLimiter(10, time.Minute),
	}, nil
}

// Client exposes the contract App internal (e.g. `platform`) receives at
// construction time.
func (m *Module) Client() contracts.Client {
	return m.service
}

// Dispatcher exposes the contract main.go uses, after every module is built,
// to register each App internal's webhook consumer — mirrors the
// dependency-inversion bridge pattern from Fase 6's projects<->clients wiring
// (SetClientAccessResolver), for the same underlying reason: `payment` must
// not import `platform` (or any other App internal's package) to know about
// its consumer, so the composition root bridges them instead.
func (m *Module) Dispatcher() contracts.Dispatcher {
	return m.service
}

// StartReconciler starts the background safety net for charges whose
// webhook was never received — see knowledge/MODULE_PAYMENT.md. Must be
// called after every App internal has already registered its consumer
// (`Dispatcher().RegisterConsumer(...)`), since a sweep tick may need to
// dispatch to one immediately.
func (m *Module) StartReconciler(ctx context.Context, interval time.Duration) {
	m.service.StartReconciler(ctx, interval)
}

// RegisterRoutes registers the module's own HTTP surface:
//   - the permanent webhook route (unauthenticated — trust comes from
//     signature verification)
//   - gateway-config + App-registry admin CRUD (platform_admin only, via
//     `authed`)
//   - `/auth/app/token` (Fase 10) — unauthenticated (it IS the login step
//     for Apps), but rate-limited per IP
//   - `/external/payments/*` (Fase 10) — `authed` (valid JWT) then
//     `requireActiveApp` (principal must be an `app`, checked live against
//     the registry) — see knowledge/MODULE_PAYMENT.md §7.1/§7.2
func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.HandleFunc("/webhooks/payment", httpx.Method(http.MethodPost, m.handler.Webhook))
	mux.Handle("/api/v1/payment/gateway-config", authed(http.HandlerFunc(m.handler.Config)))
	mux.Handle("/api/v1/payment/apps", authed(http.HandlerFunc(m.handler.Apps)))
	mux.Handle("/api/v1/payment/apps/", authed(http.HandlerFunc(m.handler.AppItem)))

	rateLimited := middleware.RateLimit(m.appTokenLimiter)
	mux.Handle("/api/v1/auth/app/token", rateLimited(http.HandlerFunc(httpx.Method(http.MethodPost, m.handler.AppToken))))

	mux.Handle("/api/v1/external/payments/charges", authed(http.HandlerFunc(m.handler.RequireActiveApp(m.handler.ExternalCreateCharge))))
	mux.Handle("/api/v1/external/payments/charges/", authed(http.HandlerFunc(m.handler.RequireActiveApp(m.handler.ExternalChargeStatus))))
	mux.Handle("/api/v1/external/payments/channels", authed(http.HandlerFunc(m.handler.RequireActiveApp(m.handler.ExternalListChannels))))
}
