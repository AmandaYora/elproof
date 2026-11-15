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
	for _, header := range []string{"If-Match", "Idempotency-Key", "Authorization", "Content-Type", "X-Requested-With"} {
		if !strings.Contains(allowed, header) {
			t.Errorf("preflight does not allow %s: %q", header, allowed)
		}
	}
}

// The refresh cookie may only be used by the developer's own pages.
func TestCORSCredentialsOnlyForLoopbackOrigins(t *testing.T) {
	h := CORS(true)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for origin, want := range map[string]string{
		"http://localhost:5173":      "true",
		"http://127.0.0.1:14173":     "true",
		"http://elproof.localhost:9": "true",
		"http://[::1]:5173":          "true",
		"http://192.168.1.20:5173":   "",
		"https://evil.example":       "",
		"http://localhost.evil.com":  "",
	} {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/refresh", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if got := w.Header().Get("Access-Control-Allow-Credentials"); got != want {
			t.Errorf("%s: Allow-Credentials %q, want %q", origin, got, want)
		}
	}

	// Production: no CORS headers at all.
	prod := CORS(false)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/refresh", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	prod.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Errorf("production must not send CORS headers: %v", w.Header())
	}
}
