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
	"elproof/internal/shared/middleware"
)

type Module struct {
	handler   *presentation.Handler
	contracts contracts.Contracts
	auth      *application.AuthService
	sessions  *application.SessionChecker
	// sameOrigin refuses cross-site calls to the endpoints that set or read
	// the refresh-token cookie (ADR-0042).
	sameOrigin middleware.SameOrigin
}

// SessionOptions configures the refresh-token cookie and its rotation.
type SessionOptions struct {
	// Development: local http, so the cookie is not Secure and loopback
	// origins on other ports count as ElProof's own pages.
	Development bool
	// ReuseGrace overrides application.DefaultRefreshReuseGrace when > 0.
	ReuseGrace time.Duration
}

// ConfigureSessions applies opts; call it before RegisterRoutes.
func (m *Module) ConfigureSessions(opts SessionOptions) {
	m.handler.SetCookieSecure(!opts.Development)
	m.sameOrigin = middleware.SameOrigin{DevMode: opts.Development}
	if opts.ReuseGrace > 0 {
		m.auth.SetRefreshReuseGrace(opts.ReuseGrace)
	}
}

// SessionValidator is plugged into middleware.RequireLiveSession so a token
// stops working as soon as its account is deactivated or changed.
func (m *Module) SessionValidator() *application.SessionChecker {
	return m.sessions
}

// SetTenantNames lets invitation pages name the WO (implemented by
// `platform`, wired in cmd/server).
func (m *Module) SetTenantNames(r application.TenantNameResolver) {
	m.auth.SetTenantNames(r)
}

// SetTenantGuard lets cmd/server plug in the platform module's tenant check
// (frozen tenants cannot log in or refresh) after both modules exist.
func (m *Module) SetTenantGuard(guard application.TenantGuard) {
	m.auth.SetTenantGuard(guard)
}

// GoogleConfig configures Sign in with Google. ClientID is the OAuth web
// client ID (public); JWKSURL is Google's key set (overridable for tests).
type GoogleConfig struct {
	ClientID string
	JWKSURL  string
}

func NewModule(db *sql.DB, jwtSecret string, accessTTL, refreshTTL time.Duration, google GoogleConfig) *Module {
	credentialRepo := infrastructure.NewMySQLCredentialRepository(db)
	refreshTokenRepo := infrastructure.NewMySQLRefreshTokenRepository(db)
	issuer := infrastructure.NewJWTIssuer(jwtSecret)
	hasher := infrastructure.NewBcryptHasher()

	authService := application.NewAuthService(credentialRepo, refreshTokenRepo, issuer, hasher, accessTTL, refreshTTL)
	totp, err := infrastructure.NewTOTP("ElProof", jwtSecret)
	if err != nil {
		panic("identity: cannot initialise TOTP: " + err.Error())
	}
	authService.SetMFAEngine(totp)
	jwksURL := google.JWKSURL
	if jwksURL == "" {
		jwksURL = infrastructure.GoogleJWKSURL
	}
	authService.SetGoogleVerifier(infrastructure.NewGoogleVerifier(google.ClientID, jwksURL))
	managementService := application.NewManagementService(credentialRepo, refreshTokenRepo, hasher)
	linkRepo := infrastructure.NewMySQLAccessLinkRepository(db)
	authService.SetAccessLinks(linkRepo)
	managementService.SetAccessLinks(linkRepo)
	sessions := application.NewSessionChecker(credentialRepo)
	managementService.SetSessionChecker(sessions)
	authService.SetSessionChecker(sessions)

	return &Module{
		handler:   presentation.NewHandler(authService),
		contracts: contracts.New(managementService, authService),
		auth:      authService,
		sessions:  sessions,
	}
}

// Contracts exposes the only surface other modules may depend on — see
// internal/modules/identity/contracts.
func (m *Module) Contracts() contracts.Contracts {
	return m.contracts
}

// RegisterRoutes registers this module's public (unauthenticated) endpoints.
//
// Endpoints that read the refresh-token cookie (refresh, logout) demand
// X-Requested-With and a same-origin request; endpoints that set it (every
// sign-in) refuse cross-site browser requests, so another site cannot sign
// a visitor into an account of its choosing.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	readsCookie := func(h http.HandlerFunc) http.HandlerFunc { return m.sameOrigin.Guard(true, h) }
	setsCookie := func(h http.HandlerFunc) http.HandlerFunc { return m.sameOrigin.Guard(false, h) }
	mux.HandleFunc("/api/v1/auth/login", httpx.Method(http.MethodPost, setsCookie(m.handler.Login)))
	mux.HandleFunc("/api/v1/auth/refresh", httpx.Method(http.MethodPost, readsCookie(m.handler.Refresh)))
	mux.HandleFunc("/api/v1/auth/logout", httpx.Method(http.MethodPost, readsCookie(m.handler.Logout)))
	mux.HandleFunc("/api/v1/auth/mfa/verify", httpx.Method(http.MethodPost, setsCookie(m.handler.VerifyMFA)))
	mux.HandleFunc("/api/v1/auth/google", httpx.Method(http.MethodPost, setsCookie(m.handler.LoginGoogle)))
	mux.HandleFunc("/api/v1/auth/google/link", httpx.Method(http.MethodPost, setsCookie(m.handler.LinkGoogle)))
	// One-time invitation / password-reset links (Fase 3); accepting one signs in.
	mux.HandleFunc("/api/v1/auth/links/", m.linksGuard(m.handler.Links))
}

// linksGuard applies the sign-in check to the POSTs under /auth/links/ (the
// GET only describes the link).
func (m *Module) linksGuard(next http.HandlerFunc) http.HandlerFunc {
	guarded := m.sameOrigin.Guard(false, next)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}
		guarded(w, r)
	}
}

// RegisterAuthedRoutes registers endpoints that need a signed-in principal.
func (m *Module) RegisterAuthedRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/auth/mfa/setup", authed(httpx.Method(http.MethodPost, m.handler.SetupMFA)))
	mux.Handle("/api/v1/auth/mfa/enable", authed(httpx.Method(http.MethodPost, m.handler.EnableMFA)))
}
