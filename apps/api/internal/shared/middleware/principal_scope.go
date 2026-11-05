package middleware

import (
	"context"
	"net/http"
	"sync"

	"elproof/internal/shared/response"
)

// RouteAllowFunc reports whether a request (method + URL path) may proceed.
type RouteAllowFunc func(method, path string) bool

// RestrictPrincipal is a deny-by-default gate for one principal type: once
// RequireAuth has put Claims in the context, a request from principalType is
// only passed on when allow returns true; every other principal type is
// passed through untouched. It is a technical utility — the actual allowlist
// is composed in cmd/server, which is the only place that knows every
// module's routes. Requests without claims are passed on unchanged (the
// wrapped handler, or RequireAuth before it, decides what to do).
func RestrictPrincipal(principalType string, allow RouteAllowFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if ok && claims.PrincipalType == principalType && !allow(r.Method, r.URL.Path) {
				response.Error(w, http.StatusForbidden, "Akun ini tidak memiliki akses ke endpoint ini", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsAllowFunc is RouteAllowFunc with access to the caller's claims (for
// role- or flag-based rules). It returns a user-facing reason when denying.
type ClaimsAllowFunc func(claims *Claims, method, path string) (allowed bool, code, reason string)

// RestrictPrincipalBy is RestrictPrincipal for rules that depend on the
// claims (e.g. Platform Console roles). Other principal types pass through.
func RestrictPrincipalBy(principalType string, allow ClaimsAllowFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if ok && claims.PrincipalType == principalType {
				if allowed, code, reason := allow(claims, r.Method, r.URL.Path); !allowed {
					response.Error(w, http.StatusForbidden, reason, map[string]string{"code": code})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SessionValidator checks that an access token still matches its account
// (active, same role, same two-factor state) — implemented by identity.
type SessionValidator interface {
	ValidateSession(ctx context.Context, claims *Claims) error
}

// RequireLiveSession rejects (401) a valid-looking token whose account was
// deactivated, demoted or lost two-factor since it was issued. The web
// client then refreshes (which re-reads the account) or logs out. Bound late
// because identity is built after the authed chain is assembled.
type RequireLiveSession struct {
	mu        sync.RWMutex
	validator SessionValidator
}

func (l *RequireLiveSession) SetValidator(v SessionValidator) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.validator = v
}

func (l *RequireLiveSession) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		l.mu.RLock()
		v := l.validator
		l.mu.RUnlock()
		if ok && v != nil {
			if err := v.ValidateSession(r.Context(), claims); err != nil {
				response.Error(w, http.StatusUnauthorized, "Sesi tidak berlaku lagi, silakan masuk kembali", map[string]string{"code": "session_revoked"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
