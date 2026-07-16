// Package platform wires the platform module: tenant lifecycle and Platform
// Console's own admin accounts. It orchestrates staff, identity, and billing
// via their contracts — see ADR-0008.
package platform

import (
	"context"
	"database/sql"
	"net/http"

	billingcontracts "elproof/internal/modules/billing/contracts"
	identitycontracts "elproof/internal/modules/identity/contracts"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/infrastructure"
	"elproof/internal/modules/platform/presentation"
	staffcontracts "elproof/internal/modules/staff/contracts"
	"elproof/internal/shared/httpx"
)

type Module struct {
	tenantHandler *presentation.TenantHandler
	adminHandler  *presentation.PlatformAdminHandler
	tenantService *application.TenantService
}

func NewModule(
	db *sql.DB,
	staff staffcontracts.Contracts,
	identity identitycontracts.Contracts,
	billing billingcontracts.Contracts,
	payment paymentcontracts.Client,
) *Module {
	tenantRepo := infrastructure.NewMySQLTenantRepository(db)
	pendingChargeRepo := infrastructure.NewMySQLPendingChargeRepository(db)
	adminRepo := infrastructure.NewMySQLPlatformAdminRepository(db)

	tenantService := application.NewTenantService(tenantRepo, pendingChargeRepo, staff, identity, billing, payment)
	adminService := application.NewPlatformAdminService(adminRepo, identity)

	return &Module{
		tenantHandler: presentation.NewTenantHandler(tenantService),
		adminHandler:  presentation.NewPlatformAdminHandler(adminService),
		tenantService: tenantService,
	}
}

// ApplyWebhookEvent makes *Module itself satisfy
// `paymentcontracts.WebhookConsumer` — main.go registers this module
// directly with the payment module's Dispatcher
// (`paymentModule.Dispatcher().RegisterConsumer(paymentcontracts.InternalAppBilling, platformModule)`)
// after both modules are constructed, the same bridging pattern used for
// Fase 6's projects<->clients wiring.
func (m *Module) ApplyWebhookEvent(ctx context.Context, orderRef string, event paymentcontracts.WebhookEvent) error {
	return m.tenantService.ApplyWebhookEvent(ctx, orderRef, event)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/tenants", authed(http.HandlerFunc(m.tenantHandler.Collection)))
	mux.Handle("/api/v1/tenants/", authed(http.HandlerFunc(m.tenantHandler.Item)))
	mux.Handle("/api/v1/subscriptions/pay", authed(httpx.Method(http.MethodPost, m.tenantHandler.Pay)))
	mux.Handle("/api/v1/platform-admins", authed(http.HandlerFunc(m.adminHandler.Collection)))
	mux.Handle("/api/v1/platform-admins/", authed(http.HandlerFunc(m.adminHandler.Item)))
}
