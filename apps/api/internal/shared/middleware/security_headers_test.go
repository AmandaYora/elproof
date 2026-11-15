package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serveWith(cfg SecurityHeadersConfig, path string) http.Header {
	h := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Header()
}

// directive returns one CSP directive's sources ("" when absent).
func directive(policy, name string) string {
	for _, d := range strings.Split(policy, ";") {
		d = strings.TrimSpace(d)
		if d == name || strings.HasPrefix(d, name+" ") {
			return d
		}
	}
	return ""
}

var prodStorage = SecurityHeadersConfig{StorageEndpoint: "is3.cloudhost.id", StorageUseSSL: true, StoragePresign: true}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	for _, path := range []string{"/", "/masuk", "/assets/index-abc.js", "/api/v1/health"} {
		h := serveWith(prodStorage, path)
		want := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"Referrer-Policy":           "strict-origin-when-cross-origin",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		}
		for k, v := range want {
			if got := h.Get(k); got != v {
				t.Errorf("%s %s = %q, want %q", path, k, got, v)
			}
		}
		pp := h.Get("Permissions-Policy")
		for _, f := range []string{"camera=()", "microphone=()", "geolocation=()"} {
			if !strings.Contains(pp, f) {
				t.Errorf("%s Permissions-Policy lacks %s: %q", path, f, pp)
			}
		}
		if !strings.Contains(h.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Errorf("%s: CSP must forbid framing: %q", path, h.Get("Content-Security-Policy"))
		}
	}
}

func TestNoHSTSInDevelopment(t *testing.T) {
	if got := serveWith(SecurityHeadersConfig{Development: true}, "/").Get("Strict-Transport-Security"); got != "" {
		t.Errorf("development is plain http: no HSTS, got %q", got)
	}
}

func TestAPIGetsTheStrictPolicy(t *testing.T) {
	if got := serveWith(prodStorage, "/api/v1/projects").Get("Content-Security-Policy"); got != APIContentSecurityPolicy {
		t.Errorf("API CSP = %q", got)
	}
}

func TestAppPolicyFitsTheApp(t *testing.T) {
	csp := serveWith(prodStorage, "/").Get("Content-Security-Policy")
	checks := map[string][]string{
		"default-src":     {"'self'"},
		"script-src":      {"'self'", "https://accounts.google.com/gsi/client"},
		"style-src":       {"'self'", "'unsafe-inline'", "https://accounts.google.com/gsi/style"},
		"img-src":         {"'self'", "data:", "blob:", "https://is3.cloudhost.id", "https://*.is3.cloudhost.id", "https://tripay.co.id"},
		"connect-src":     {"'self'", "https://accounts.google.com/gsi/", "https://is3.cloudhost.id"},
		"frame-src":       {"blob:", "https://accounts.google.com/gsi/", "https://is3.cloudhost.id"},
		"worker-src":      {"'self'"},
		"object-src":      {"'none'"},
		"base-uri":        {"'self'"},
		"form-action":     {"'self'"},
		"frame-ancestors": {"'none'"},
	}
	for name, sources := range checks {
		d := directive(csp, name)
		if d == "" {
			t.Errorf("CSP lacks %s: %q", name, csp)
			continue
		}
		for _, s := range sources {
			if !strings.Contains(" "+d+" ", " "+s+" ") {
				t.Errorf("%s lacks %s: %q", name, s, d)
			}
		}
	}
	script := directive(csp, "script-src")
	for _, forbidden := range []string{"'unsafe-inline'", "'unsafe-eval'", "*", "data:", "https:"} {
		if strings.Contains(" "+script+" ", " "+forbidden+" ") {
			t.Errorf("script-src must not allow %s: %q", forbidden, script)
		}
	}
}

func TestStorageSourcesFollowConfiguration(t *testing.T) {
	cases := []struct {
		name string
		cfg  SecurityHeadersConfig
		want []string
		none []string
	}{
		{"https DNS endpoint", prodStorage, []string{"https://is3.cloudhost.id", "https://*.is3.cloudhost.id"}, nil},
		{"local minio with port", SecurityHeadersConfig{StorageEndpoint: "127.0.0.1:19000", StoragePresign: true}, []string{"http://127.0.0.1:19000"}, []string{"*.127"}},
		{"presign off", SecurityHeadersConfig{StorageEndpoint: "is3.cloudhost.id", StorageUseSSL: true}, nil, []string{"is3.cloudhost.id"}},
		{"no endpoint", SecurityHeadersConfig{StoragePresign: true}, nil, []string{"https://*"}},
		{"endpoint cannot inject a directive", SecurityHeadersConfig{StorageEndpoint: "x.test; script-src *", StoragePresign: true}, nil, []string{"script-src *"}},
	}
	for _, c := range cases {
		csp := AppContentSecurityPolicy(c.cfg)
		img := directive(csp, "img-src")
		for _, s := range c.want {
			if !strings.Contains(" "+img+" ", " "+s+" ") {
				t.Errorf("%s: img-src lacks %s: %q", c.name, s, img)
			}
		}
		for _, s := range c.none {
			if strings.Contains(csp, s) {
				t.Errorf("%s: CSP must not contain %q: %q", c.name, s, csp)
			}
		}
	}
}
