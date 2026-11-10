// Package billing wires the billing module: the subscription plan catalog
// (single source of truth for both consoles) and the subscription
// transaction ledger. See ADR-0008, docs/API_CONTRACT.md.
package billing

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/contracts"
	"elproof/internal/modules/billing/infrastructure"
	"elproof/internal/modules/billing/presentation"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/httpx"
)

type Module struct {
	transactionService          *application.TransactionService
	planHandler                 *presentation.PlanHandler
	transactionHandler          *presentation.TransactionHandler
	externalPlanHandler         *presentation.ExternalPlanHandler
	appPlanHandler              *presentation.AppPlanHandler
	externalSubscriptionHandler *presentation.ExternalSubscriptionHandler
	appSubscriptionHandler      *presentation.AppSubscriptionHandler
	publicPlanHandler           *presentation.PublicPlanHandler
	subscriptionReconciler      *application.SubscriptionReconciler
	contracts                   contracts.Contracts
}

// NewModule takes two dependencies from `payment` now: AppRegistry (Fase
// 5/7, checking App status for this module's own external/console
// endpoints) and Client (Fase 8, creating subscription charges) — the same
// direct-contracts-injection pattern `platform` already uses for
// payment.Client (main.go:210, A5).
func NewModule(db *sql.DB, appRegistry paymentcontracts.AppRegistry, paymentClient paymentcontracts.Client) *Module {
	planRepo := infrastructure.NewMySQLPlanRepository(db)
	transactionRepo := infrastructure.NewMySQLTransactionRepository(db)
	appChargeRepo := infrastructure.NewMySQLAppChargeRepository(db)

	planService := application.NewPlanService(planRepo)
	transactionService := application.NewTransactionService(transactionRepo)
	subscriptionService := application.NewSubscriptionService(planService, appChargeRepo, paymentClient)
	// 24h is a bootstrap default only — SetSubscriptionChargeMaxAge (called
	// from main.go right after NewModule, before StartSubscriptionReconciler)
	// applies cfg.SubscriptionChargeMaxAge before this reconciler ever ticks.
	subscriptionReconciler := application.NewSubscriptionReconciler(appChargeRepo, paymentClient, subscriptionService, 24*time.Hour)

	return &Module{
		transactionService:          transactionService,
		planHandler:                 presentation.NewPlanHandler(planService),
		transactionHandler:          presentation.NewTransactionHandler(transactionService),
		externalPlanHandler:         presentation.NewExternalPlanHandler(planService, appRegistry),
		appPlanHandler:              presentation.NewAppPlanHandler(planService, appRegistry),
		externalSubscriptionHandler: presentation.NewExternalSubscriptionHandler(subscriptionService, appRegistry),
		appSubscriptionHandler:      presentation.NewAppSubscriptionHandler(appChargeRepo),
		publicPlanHandler:           presentation.NewPublicPlanHandler(planService),
		subscriptionReconciler:      subscriptionReconciler,
		contracts:                   contracts.New(planService, transactionService),
	}
}

func (m *Module) Contracts() contracts.Contracts {
	return m.contracts
}

// SetTenantDirectory completes the two-phase wiring with `platform` (built after
// billing): tenant names and the tenant-name search of the transaction list.
func (m *Module) SetTenantDirectory(d application.TenantDirectory) {
	m.transactionService.SetTenants(d)
}

// StartSubscriptionReconciler starts the background safety net for
// subscription charges whose webhook was never received (A16) — mirrors
// payment.Module.StartReconciler exactly, including being called from the
// composition root (main.go) rather than from NewModule, so it starts only
// once the full server is otherwise wired up.
func (m *Module) StartSubscriptionReconciler(ctx context.Context, interval time.Duration) {
	m.subscriptionReconciler.Start(ctx, interval)
}

// SetSubscriptionChargeMaxAge applies cfg.SubscriptionChargeMaxAge (task
// 8.7) — call this before StartSubscriptionReconciler so the reconciler
// never ticks with the bootstrap default.
func (m *Module) SetSubscriptionChargeMaxAge(maxAge time.Duration) {
	m.subscriptionReconciler.SetMaxAge(maxAge)
}

// RegisterPublicRoutes registers this module's pre-auth endpoints, never
// wrapped in `authed` (same pattern as platform.RegisterPublicRoutes): the
// public homepage reads the active plan catalog before anyone signs in.
func (m *Module) RegisterPublicRoutes(mux *http.ServeMux) {
	registerPublicRoutes(mux, m.publicPlanHandler)
}

// registerPublicRoutes is split out so the route table (path, no auth, GET
// only) can be tested without a database.
func registerPublicRoutes(mux *http.ServeMux, plans *presentation.PublicPlanHandler) {
	mux.Handle("/api/v1/public/plans", httpx.Method(http.MethodGet, plans.List))
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/plans", authed(http.HandlerFunc(m.planHandler.Collection)))
	mux.Handle("/api/v1/plans/", authed(http.HandlerFunc(m.planHandler.Item)))
	mux.Handle("/api/v1/subscription-transactions", authed(httpx.Method(http.MethodGet, m.transactionHandler.List)))
	mux.Handle("/api/v1/subscription-transactions/", authed(httpx.Method(http.MethodGet, m.transactionHandler.Item)))

	// EXTERNAL: not wrapped in httpx.Method — each handler checks its own
	// method so a 405 still carries {code:"bad_request"}, matching the
	// vocabulary every other external route uses (T17/A24).
	mux.Handle("/api/v1/external/subscription-plans", authed(http.HandlerFunc(m.externalPlanHandler.List)))
	mux.Handle("/api/v1/external/subscriptions/charges", authed(http.HandlerFunc(m.externalSubscriptionHandler.CreateCharge)))

	// CONSOLE: /app-plans serves both GET and PUT on one handler, so
	// httpx.Method doesn't fit — same reasoning as /api/v1/plans above.
	// /app-subscriptions is GET-only, so httpx.Method applies.
	mux.Handle("/api/v1/app-plans", authed(http.HandlerFunc(m.appPlanHandler.Collection)))
	mux.Handle("/api/v1/app-subscriptions", authed(httpx.Method(http.MethodGet, m.appSubscriptionHandler.List)))
}
