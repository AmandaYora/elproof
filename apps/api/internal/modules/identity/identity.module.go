// Package identity wires the identity module together: authentication
// (login/refresh/logout) for all three principal types. See ADR-0005.
package identity

import (
	"database/sql"
	"net/http"
	"time"

	"elproof/internal/modules/identity/application"
	"elproof/internal/modules/identity/contracts"
	"elproof/internal/modules/identity/infrastructure"
	"elproof/internal/modules/identity/presentation"
	"elproof/internal/shared/httpx"
)

type Module struct {
	handler   *presentation.Handler
	contracts contracts.Contracts
}

func NewModule(db *sql.DB, jwtSecret string, accessTTL, refreshTTL time.Duration) *Module {
	credentialRepo := infrastructure.NewMySQLCredentialRepository(db)
	refreshTokenRepo := infrastructure.NewMySQLRefreshTokenRepository(db)
	issuer := infrastructure.NewJWTIssuer(jwtSecret)
	hasher := infrastructure.NewBcryptHasher()

	authService := application.NewAuthService(credentialRepo, refreshTokenRepo, issuer, hasher, accessTTL, refreshTTL)
	managementService := application.NewManagementService(credentialRepo, hasher)

	return &Module{
		handler:   presentation.NewHandler(authService),
		contracts: contracts.New(managementService, authService),
	}
}

// Contracts exposes the only surface other modules may depend on — see
// internal/modules/identity/contracts.
func (m *Module) Contracts() contracts.Contracts {
	return m.contracts
}

// RegisterRoutes registers this module's public (unauthenticated) endpoints.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/login", httpx.Method(http.MethodPost, m.handler.Login))
	mux.HandleFunc("/api/v1/auth/refresh", httpx.Method(http.MethodPost, m.handler.Refresh))
	mux.HandleFunc("/api/v1/auth/logout", httpx.Method(http.MethodPost, m.handler.Logout))
}
