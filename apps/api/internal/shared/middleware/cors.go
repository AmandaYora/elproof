package middleware

import "net/http"

// CORS is only needed in local development, where the Vite dev server
// (npm run dev:web) and the Go API (npm run dev:api) run on different ports.
// In production, one Docker app container serves both under the same origin
// (see .claude/rules/monorepo.md), so this middleware has nothing to do there.
// devMode reflects whatever Origin sent the request rather than pinning one
// port, since the frontend dev port isn't guaranteed to stay fixed.
func CORS(devMode bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if devMode {
				if origin := r.Header.Get("Origin"); origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
