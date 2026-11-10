package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The web app sends If-Match (ADR-0036) and Idempotency-Key on writes; a dev
// preflight that does not allow them blocks every such save across origins.
func TestCORSAllowsWriteHeaders(t *testing.T) {
	h := CORS(true)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/vouchers/1", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	allowed := w.Header().Get("Access-Control-Allow-Headers")
	for _, header := range []string{"If-Match", "Idempotency-Key", "Authorization", "Content-Type"} {
		if !strings.Contains(allowed, header) {
			t.Errorf("preflight does not allow %s: %q", header, allowed)
		}
	}
}
