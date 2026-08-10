// Package platform wires the platform module: tenant lifecycle and Platform
// Console's own admin accounts. It orchestrates staff, identity, and billing
// via their contracts — see ADR-0008.
package platform

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	billingcontracts "elproof/internal/modules/billing/contracts"
	identitycontracts "elproof/internal/modules/identity/contracts"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/infrastructure"
	"elproof/internal/modules/platform/presentation"
	projectscontracts "elproof/internal/modules/projects/contracts"
	staffcontracts "elproof/internal/modules/staff/contracts"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/storage"
)

type Module struct {
	tenantHandler      *presentation.TenantHandler
	adminHandler       *presentation.PlatformAdminHandler
	loginSlidesHandler *presentation.LoginSlidesHandler
	tenantService      *application.TenantService
}

func NewModule(
	db *sql.DB,
	staff staffcontracts.Contracts,
	identity identitycontracts.Contracts,
	billing billingcontracts.Contracts,
	payment paymentcontracts.Client,
	storageClient *storage.Client,
) *Module {
	tenantRepo := infrastructure.NewMySQLTenantRepository(db)
	pendingChargeRepo := infrastructure.NewMySQLPendingChargeRepository(db)
	adminRepo := infrastructure.NewMySQLPlatformAdminRepository(db)

	tenantService := application.NewTenantService(
		tenantRepo, pendingChargeRepo, staff, identity, billing, payment, storageClient, storage.BuildKey,
	)
	adminService := application.NewPlatformAdminService(adminRepo, identity)

	return &Module{
		tenantHandler:      presentation.NewTenantHandler(tenantService),
		adminHandler:       presentation.NewPlatformAdminHandler(adminService),
		loginSlidesHandler: presentation.NewLoginSlidesHandler(storageClient),
		tenantService:      tenantService,
	}
}

// SetVendors completes two-phase wiring with the vendors module — see
// TenantService.SetVendors. main.go calls this right after vendorsModule is
// built, the same slot as projectsModule.SetClientAccessResolver.
func (m *Module) SetVendors(vendors vendorscontracts.Contracts) {
	m.tenantService.SetVendors(vendors)
}

// SetProjects completes the same two-phase wiring as SetVendors above — see
// TenantService.SetProjects. main.go calls this right after projectsModule
// is built.
func (m *Module) SetProjects(projects projectscontracts.Contracts) {
	m.tenantService.SetProjects(projects)
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

// SiteMeta is the minimal, primitive-typed slice of a tenant's branding
// needed to render a request's <title>/Open Graph tags at HTML-serve time —
// deliberately not domain.Tenant itself, same primitive-typed-bridge
// reasoning as StaffNameResolver (projects/application): main.go's
// spaFileServer is composition-root code, not another module, but there's
// still no reason for it to see this module's domain internals for a need
// this narrow.
type SiteMeta struct {
	BusinessName string
	HasLogo      bool
}

// SiteMetaForHost resolves a tenant's own display name for injecting into
// index.html's <title>/Open Graph tags at serve time (spaFileServer in
// main.go) — same Host-based lookup as PublicBranding (ADR-0015), reused
// here so a shared link's preview card (WhatsApp/Telegram/etc., which never
// execute the SPA's JS) shows the tenant's real name instead of the
// platform's own. ok=false for the platform's own domain or any unmatched
// Host, exactly like PublicBranding's 404.
func (m *Module) SiteMetaForHost(ctx context.Context, host string) (meta SiteMeta, ok bool) {
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	tenant, err := m.tenantService.GetBrandingByDomain(ctx, strings.ToLower(host))
	if err != nil {
		return SiteMeta{}, false
	}
	return SiteMeta{BusinessName: tenant.BusinessName, HasLogo: tenant.LogoStoragePath != nil}, true
}

// RegisterPublicRoutes registers this module's pre-auth endpoints (ADR-0015)
// — Host-header-resolved tenant branding for a custom domain's login page,
// never wrapped in the `authed` middleware since no session exists yet.
func (m *Module) RegisterPublicRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/public/branding", httpx.Method(http.MethodGet, m.tenantHandler.PublicBranding))
	mux.Handle("/api/v1/public/logo", httpx.Method(http.MethodGet, m.tenantHandler.PublicLogo))
	mux.Handle("/api/v1/public/login-slides", httpx.Method(http.MethodGet, m.loginSlidesHandler.List))
	mux.Handle("/api/v1/public/login-slides/", httpx.Method(http.MethodGet, m.loginSlidesHandler.Slide))
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/tenants", authed(http.HandlerFunc(m.tenantHandler.Collection)))
	mux.Handle("/api/v1/tenants/", authed(http.HandlerFunc(m.tenantHandler.Item)))
	mux.Handle("/api/v1/subscriptions/pay", authed(httpx.Method(http.MethodPost, m.tenantHandler.Pay)))
	mux.Handle("/api/v1/subscriptions/pending-charge", authed(httpx.Method(http.MethodGet, m.tenantHandler.PendingCharge)))
	mux.Handle("/api/v1/subscriptions/pending-charge/cancel", authed(httpx.Method(http.MethodPost, m.tenantHandler.CancelPendingCharge)))
	mux.Handle("/api/v1/platform-admins", authed(http.HandlerFunc(m.adminHandler.Collection)))
	mux.Handle("/api/v1/platform-admins/", authed(http.HandlerFunc(m.adminHandler.Item)))
}
