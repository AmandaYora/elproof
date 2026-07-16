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

	"elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/payment/infrastructure"
	"elproof/internal/modules/payment/presentation"
	"elproof/internal/shared/httpx"
)

type Module struct {
	service *infrastructure.PaymentService
	handler *presentation.Handler
}

func NewModule(db *sql.DB, encryptionKey string) (*Module, error) {
	service, err := infrastructure.NewPaymentService(db, encryptionKey)
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

// RegisterRoutes registers the module's own HTTP surface: the permanent
// webhook route (unauthenticated — trust comes from signature verification)
// and the gateway-config admin CRUD (platform_admin only, via `authed`).
func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.HandleFunc("/webhooks/payment", httpx.Method(http.MethodPost, m.handler.Webhook))
	mux.Handle("/api/v1/payment/gateway-config", authed(http.HandlerFunc(m.handler.Config)))
}
