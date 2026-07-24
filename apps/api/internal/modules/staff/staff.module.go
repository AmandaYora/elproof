// Package staff wires the staff module together: WO Console's own tenant-
// scoped user management (Fase 3), plus CreateOwner (via contracts, called by
// `platform`'s tenant-registration orchestration, Fase 2).
package staff

import (
	"database/sql"
	"net/http"

	identitycontracts "elproof/internal/modules/identity/contracts"

	"elproof/internal/modules/staff/application"
	"elproof/internal/modules/staff/contracts"
	"elproof/internal/modules/staff/infrastructure"
	"elproof/internal/modules/staff/presentation"
)

type Module struct {
	contracts contracts.Contracts
	handler   *presentation.Handler
}

func NewModule(db *sql.DB, identity identitycontracts.Contracts) *Module {
	repo := infrastructure.NewMySQLStaffRepository(db)
	service := application.NewStaffService(repo, identity)
	return &Module{
		contracts: contracts.New(service),
		handler:   presentation.NewHandler(service),
	}
}

func (m *Module) Contracts() contracts.Contracts {
	return m.contracts
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/staff", authed(http.HandlerFunc(m.handler.Collection)))
	mux.Handle("/api/v1/staff/", authed(http.HandlerFunc(m.handler.Item)))
}
