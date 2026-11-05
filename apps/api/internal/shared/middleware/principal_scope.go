package middleware

import (
	"net/http"

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
