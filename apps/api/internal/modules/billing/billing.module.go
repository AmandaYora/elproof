// Package billing wires the billing module: the subscription plan catalog
// (single source of truth for both consoles) and the subscription
// transaction ledger. See ADR-0008, docs/API_CONTRACT.md.
package billing

import (
	"database/sql"
	"net/http"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/contracts"
	"elproof/internal/modules/billing/infrastructure"
	"elproof/internal/modules/billing/presentation"
	"elproof/internal/shared/httpx"
)

type Module struct {
	planHandler        *presentation.PlanHandler
	transactionHandler *presentation.TransactionHandler
	contracts          contracts.Contracts
}

func NewModule(db *sql.DB) *Module {
	planRepo := infrastructure.NewMySQLPlanRepository(db)
	transactionRepo := infrastructure.NewMySQLTransactionRepository(db)

	planService := application.NewPlanService(planRepo)
	transactionService := application.NewTransactionService(transactionRepo)

	return &Module{
		planHandler:        presentation.NewPlanHandler(planService),
		transactionHandler: presentation.NewTransactionHandler(transactionService),
		contracts:          contracts.New(planService, transactionService),
	}
}

func (m *Module) Contracts() contracts.Contracts {
	return m.contracts
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/plans", authed(http.HandlerFunc(m.planHandler.Collection)))
	mux.Handle("/api/v1/plans/", authed(http.HandlerFunc(m.planHandler.Item)))
	mux.Handle("/api/v1/subscription-transactions", authed(httpx.Method(http.MethodGet, m.transactionHandler.List)))
}
