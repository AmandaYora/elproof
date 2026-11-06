// Package access wires the access module: per-workspace roles and
// permissions (PLAN.md Fase 3a, ADR-0026).
package access

import (
	"context"
	"database/sql"
	"net/http"

	"elproof/internal/modules/access/application"
	"elproof/internal/modules/access/contracts"
	"elproof/internal/modules/access/domain"
	"elproof/internal/modules/access/infrastructure"
	"elproof/internal/modules/access/presentation"
	"elproof/internal/shared/middleware"
)

type Module struct {
	roles     *application.RoleService
	handler   *presentation.Handler
	contracts contracts.Contracts
}

func NewModule(db *sql.DB) *Module {
	roles := application.NewRoleService(infrastructure.NewMySQLRoleRepository(db))
	return &Module{roles: roles, handler: presentation.NewHandler(roles), contracts: contracts.New(roles)}
}

func (m *Module) Contracts() contracts.Contracts { return m.contracts }

// SetMemberCounter completes two-phase wiring with identity.
func (m *Module) SetMemberCounter(c application.MemberCounter) { m.roles.SetMemberCounter(c) }

// ResolveGrants implements middleware.GrantsResolver.
func (m *Module) ResolveGrants(ctx context.Context, tenantID, roleID int64) (*middleware.Grants, error) {
	role, err := m.roles.Grants(ctx, tenantID, roleID)
	if err != nil || role == nil {
		return nil, err
	}
	perms := make(map[string]bool, len(role.Permissions))
	for _, p := range role.Permissions {
		perms[p] = true
	}
	return &middleware.Grants{
		RoleID: role.ID, RoleName: role.Name, Owner: role.IsSystem,
		AllProjects: role.Scope == domain.ScopeAll, Permissions: perms,
	}, nil
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/access/me", authed(http.HandlerFunc(m.handler.Me)))
	mux.Handle("/api/v1/access/catalog", authed(http.HandlerFunc(m.handler.Catalog)))
	mux.Handle("/api/v1/access/roles", authed(http.HandlerFunc(m.handler.Roles)))
	mux.Handle("/api/v1/access/roles/", authed(http.HandlerFunc(m.handler.Role)))
}
