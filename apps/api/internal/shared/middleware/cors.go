package middleware

import "net/http"

// CORS is only needed in local development, where the Vite dev server
// (npm run dev:web) and the Go API (npm run dev:api) run on different ports.
// In production, one Docker app container serves both under the same origin
// (see .claude/rules/monorepo.md), so this middleware has nothing to do there.
// devMode reflects whatever Origin sent the request rather than pinning one
// port, since the frontend dev port isn't guaranteed to stay fixed.
//
// Credentials (the refresh-token cookie, ADR-0044) are only allowed for
// loopback origins: a page on another machine may read the dev API's public
// endpoints, but never use a developer's session cookie.
func CORS(devMode bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if devMode {
				if origin := r.Header.Get("Origin"); origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					if IsLoopbackOrigin(origin) {
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				// If-Match (ADR-0036) and Idempotency-Key are sent by the web app on writes;
				// X-Requested-With on the cookie-authenticated auth endpoints (ADR-0044).
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match, Idempotency-Key, X-Requested-With")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
