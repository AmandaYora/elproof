// Package platform wires the platform module: tenant lifecycle and Platform
// Console's own admin accounts. It orchestrates staff, identity, and billing
// via their contracts — see ADR-0008.
package platform

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	accesscontracts "elproof/internal/modules/access/contracts"
	billingcontracts "elproof/internal/modules/billing/contracts"
	clientscontracts "elproof/internal/modules/clients/contracts"
	identitycontracts "elproof/internal/modules/identity/contracts"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/infrastructure"
	"elproof/internal/modules/platform/presentation"
	projectscontracts "elproof/internal/modules/projects/contracts"
	staffcontracts "elproof/internal/modules/staff/contracts"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/storage"
)

type Module struct {
	tenantHandler      *presentation.TenantHandler
	adminHandler       *presentation.PlatformAdminHandler
	loginSlidesHandler *presentation.LoginSlidesHandler
	signupHandler      *presentation.SignupHandler
	consoleHandler     *presentation.ConsoleHandler
	consoleService     *application.ConsoleService
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
	tenantService.SetManualActivations(infrastructure.NewMySQLManualActivationRepository(db))
	tenantService.SetSignups(infrastructure.NewMySQLSignupRepository(db))
	tenantService.SetTrialLimits(infrastructure.NewMySQLTrialLimitsRepository(db))
	adminService := application.NewPlatformAdminService(adminRepo, identity)
	consoleService := application.NewConsoleService(infrastructure.NewMySQLConsoleRepository(db), tenantService, identity)
	consoleService.SetAdmins(adminRepo)
	consoleService.OnPolicyChanged(func(ctx context.Context) {
		if _, err := tenantService.SweepSubscriptionStatuses(ctx); err != nil {
			logger.Error("platform: refreshing subscription statuses after a policy change: %v", err)
		}
	})

	return &Module{
		tenantHandler:      presentation.NewTenantHandler(tenantService),
		adminHandler:       presentation.NewPlatformAdminHandler(adminService),
		loginSlidesHandler: presentation.NewLoginSlidesHandler(storageClient),
		signupHandler:      presentation.NewSignupHandler(tenantService),
		consoleHandler:     presentation.NewConsoleHandler(consoleService, tenantService),
		consoleService:     consoleService,
		tenantService:      tenantService,
	}
}

// SetAuditRecorder lets the tenant CSV export write its own audit record.
func (m *Module) SetAuditRecorder(rec middleware.AuditRecorder) {
	m.tenantHandler.SetAuditRecorder(rec)
}

// SupportSessionValidator is asked on every request made with a support
// token (wired into the authed chain in cmd/server).
func (m *Module) SupportSessionValidator() middleware.SupportSessionValidator {
	return m.consoleService
}

// LoadSubscriptionPolicy applies the lifecycle policy stored by the platform
// team (main.go calls this at startup; until then the defaults are in force).
func (m *Module) LoadSubscriptionPolicy(ctx context.Context) error {
	return m.consoleService.LoadPolicy(ctx)
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

// SetAccess completes two-phase wiring with the access module.
func (m *Module) SetAccess(a accesscontracts.Contracts) {
	m.tenantService.SetAccess(a)
}

// SetClients completes two-phase wiring with the clients module (client
// usage on the onboarding checklist).
func (m *Module) SetClients(clients clientscontracts.Contracts) {
	m.tenantService.SetClients(clients)
}

// TrialLimit implements the TrialQuota port that staff, projects and
// clients declare (wired in cmd/server): the cap for one resource while the
// tenant is on its free trial.
func (m *Module) TrialLimit(ctx context.Context, tenantID int64, resource string) (int, bool, error) {
	return m.tenantService.TrialLimit(ctx, tenantID, resource)
}

// TenantName implements identity's TenantNameResolver (invitation pages
// name the WO).
func (m *Module) TenantName(ctx context.Context, tenantID int64) (string, error) {
	t, err := m.tenantService.Get(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return t.BusinessName, nil
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

// TenantAccessLevel implements middleware.TenantAccessResolver for the
// per-request subscription gate (wired in cmd/server).
func (m *Module) TenantAccessLevel(ctx context.Context, tenantID int64) (string, error) {
	return m.tenantService.TenantAccessLevel(ctx, tenantID)
}

// CheckTenantLogin implements identity's TenantGuard (frozen tenants cannot
// log in or refresh).
func (m *Module) CheckTenantLogin(ctx context.Context, tenantID int64) error {
	return m.tenantService.CheckTenantLogin(ctx, tenantID)
}

// StartSubscriptionStatusSweeper keeps the cached subscription_status column
// current (see TenantService.SweepSubscriptionStatuses).
func (m *Module) StartSubscriptionStatusSweeper(ctx context.Context, interval time.Duration) {
	m.tenantService.StartSubscriptionStatusSweeper(ctx, interval)
}

// RegisterPublicRoutes registers this module's pre-auth endpoints (login
// slides, self-service signup), never wrapped in `authed` since no session
// exists yet.
func (m *Module) RegisterPublicRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/public/login-slides", httpx.Method(http.MethodGet, m.loginSlidesHandler.List))
	mux.Handle("/api/v1/public/login-slides/", httpx.Method(http.MethodGet, m.loginSlidesHandler.Slide))
	// Self-service WO registration (PLAN.md §3): Google credential + the
	// one-screen workspace form.
	mux.Handle("/api/v1/public/signup/workspace", httpx.Method(http.MethodPost, m.signupHandler.CreateWorkspace))
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/tenants", authed(http.HandlerFunc(m.tenantHandler.Collection)))
	mux.Handle("/api/v1/tenants/", authed(http.HandlerFunc(m.tenantHandler.Item)))
	mux.Handle("/api/v1/tenants/export", authed(httpx.Method(http.MethodGet, m.tenantHandler.Export)))
	mux.Handle("/api/v1/subscriptions/pay", authed(httpx.Method(http.MethodPost, m.tenantHandler.Pay)))
	mux.Handle("/api/v1/subscriptions/pending-charge", authed(httpx.Method(http.MethodGet, m.tenantHandler.PendingCharge)))
	mux.Handle("/api/v1/subscriptions/pending-charge/cancel", authed(httpx.Method(http.MethodPost, m.tenantHandler.CancelPendingCharge)))
	mux.Handle("/api/v1/trial-limits", authed(http.HandlerFunc(m.tenantHandler.TrialLimits)))
	mux.Handle("/api/v1/platform/metrics", authed(httpx.Method(http.MethodGet, m.tenantHandler.Metrics)))
	// Platform Console back office (notes, data requests, announcements,
	// policy, support sessions) and the Owner-side counterparts. The exact
	// "/tenants/me/support-access" pattern outranks the "/tenants/" prefix.
	mux.Handle("/api/v1/console/", authed(http.HandlerFunc(m.consoleHandler.Console)))
	// Super Admin replaces one of the login page's photos.
	mux.Handle("/api/v1/console/login-slides/", authed(httpx.Method(http.MethodPut, m.loginSlidesHandler.Replace)))
	mux.Handle("/api/v1/tenants/me/support-access", authed(http.HandlerFunc(m.consoleHandler.MySupportAccess)))
	mux.Handle("/api/v1/announcements/active", authed(httpx.Method(http.MethodGet, m.consoleHandler.ActiveAnnouncements)))
	mux.Handle("/api/v1/platform-admins", authed(http.HandlerFunc(m.adminHandler.Collection)))
	mux.Handle("/api/v1/platform-admins/", authed(http.HandlerFunc(m.adminHandler.Item)))
}
