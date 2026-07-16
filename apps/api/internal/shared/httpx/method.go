// Package httpx holds tiny net/http helpers shared across modules.
package httpx

import (
	"net/http"

	"elproof/internal/shared/response"
)

// Method wraps a handler so it only responds to the given HTTP method — kept as
// a plain wrapper (rather than Go 1.22's "METHOD /path" mux pattern syntax) so
// routing works the same regardless of the exact Go patch version building it.
func Method(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
			return
		}
		h(w, r)
	}
}
