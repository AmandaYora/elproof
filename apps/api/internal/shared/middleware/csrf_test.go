package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func csrfRequest(host string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Host = host
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func TestSameOriginPolicy(t *testing.T) {
	xhr := map[string]string{CSRFHeader: "XMLHttpRequest"}
	with := func(extra map[string]string) map[string]string {
		out := map[string]string{CSRFHeader: "XMLHttpRequest"}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	cases := []struct {
		name          string
		dev           bool
		requireHeader bool
		host          string
		headers       map[string]string
		want          bool
	}{
		{"same-origin browser", false, true, "elproof.elcodelabs.com", with(map[string]string{"Origin": "https://elproof.elcodelabs.com", "Sec-Fetch-Site": "same-origin"}), true},
		{"api client without browser headers", false, true, "elproof.elcodelabs.com", xhr, true},
		{"missing custom header", false, true, "elproof.elcodelabs.com", map[string]string{"Origin": "https://elproof.elcodelabs.com"}, false},
		{"blank custom header", false, true, "elproof.elcodelabs.com", map[string]string{CSRFHeader: "  "}, false},
		{"header not required (login)", false, false, "elproof.elcodelabs.com", map[string]string{"Origin": "https://elproof.elcodelabs.com"}, true},
		{"foreign origin", false, false, "elproof.elcodelabs.com", map[string]string{"Origin": "https://evil.example"}, false},
		{"foreign origin with header", false, true, "elproof.elcodelabs.com", with(map[string]string{"Origin": "https://evil.example"}), false},
		{"opaque origin", false, false, "elproof.elcodelabs.com", map[string]string{"Origin": "null"}, false},
		{"cross-site fetch metadata", false, false, "elproof.elcodelabs.com", map[string]string{"Sec-Fetch-Site": "cross-site"}, false},
		{"origin matches proxied host", false, true, "127.0.0.1:8082", with(map[string]string{"Origin": "https://elproof.elcodelabs.com", "X-Forwarded-Host": "elproof.elcodelabs.com"}), true},
		{"origin matches neither host", false, true, "127.0.0.1:8082", with(map[string]string{"Origin": "https://elproof.elcodelabs.com"}), false},
		{"lookalike host", false, false, "elproof.elcodelabs.com", map[string]string{"Origin": "https://elproof.elcodelabs.com.evil.example"}, false},
		{"dev: vite on another port", true, true, "localhost:8080", with(map[string]string{"Origin": "http://localhost:5173", "Sec-Fetch-Site": "same-site"}), true},
		{"dev: loopback across host names", true, true, "127.0.0.1:8080", with(map[string]string{"Origin": "http://localhost:5173", "Sec-Fetch-Site": "cross-site"}), true},
		{"dev: LAN origin is not loopback", true, true, "localhost:8080", with(map[string]string{"Origin": "http://192.168.1.4:5173"}), false},
		{"prod: loopback origin is foreign", false, true, "elproof.elcodelabs.com", with(map[string]string{"Origin": "http://localhost:5173"}), false},
	}
	for _, c := range cases {
		got := SameOrigin{DevMode: c.dev}.Allowed(csrfRequest(c.host, c.headers), c.requireHeader)
		if got != c.want {
			t.Errorf("%s: Allowed = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSameOriginGuardRefusesWith403(t *testing.T) {
	called := false
	h := SameOrigin{}.Guard(true, func(http.ResponseWriter, *http.Request) { called = true })
	w := httptest.NewRecorder()
	h(w, csrfRequest("elproof.elcodelabs.com", map[string]string{"Origin": "https://evil.example"}))
	if called || w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without calling the handler, got %d (called=%v)", w.Code, called)
	}
}
