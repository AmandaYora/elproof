// Package clients wires the clients module — Fase 4, bundled with `projects`
// since a client row can't exist without a real project_id to reference
// (see PLAN.md Fase 3 scope note).
package clients

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"elproof/internal/modules/clients/application"
	"elproof/internal/modules/clients/contracts"
	"elproof/internal/modules/clients/infrastructure"
	"elproof/internal/modules/clients/presentation"
	identitycontracts "elproof/internal/modules/identity/contracts"
	projectscontracts "elproof/internal/modules/projects/contracts"
)

type Module struct {
	handler   *presentation.Handler
	contracts contracts.Contracts
	service   *application.ClientService
}

func NewModule(db *sql.DB, projects projectscontracts.Contracts, identity identitycontracts.Contracts) *Module {
	repo := infrastructure.NewMySQLClientRepository(db)
	service := application.NewClientService(repo, projects, identity)
	return &Module{handler: presentation.NewHandler(service), contracts: contracts.New(service), service: service}
}

// SetRoles plugs in the access module's portal roles (main.go).
func (m *Module) SetRoles(r application.RoleResolver) {
	m.service.SetRoles(r)
}

// SetTrialQuota plugs in the platform module's trial caps (main.go).
func (m *Module) SetTrialQuota(q application.TrialQuota) {
	m.service.SetTrialQuota(q)
}

// PurgeSummary is what one purge run removed (the composition root logs it).
type PurgeSummary struct {
	// Ran is false when another instance held the purge lock.
	Ran     bool
	Removed int
}

// PurgeExpired permanently removes the clients that have been deleted for longer
// than the retention, with their logins. Idempotent and safe with several
// instances; it is what `api purge` and the server's ticker run.
func (m *Module) PurgeExpired(ctx context.Context, now time.Time) (PurgeSummary, error) {
	removed, ran, err := m.service.PurgeExpired(ctx, now)
	return PurgeSummary{Ran: ran, Removed: removed}, err
}

func (m *Module) Contracts() contracts.Contracts {
	return m.contracts
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/clients", authed(http.HandlerFunc(m.handler.Collection)))
	mux.Handle("/api/v1/clients/", authed(http.HandlerFunc(m.handler.Item)))
}
