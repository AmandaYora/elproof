package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func payOnly(method, path string) bool {
	return method == http.MethodPost && path == "/api/v1/subscriptions/pay"
}

func TestDecideTenantAccess(t *testing.T) {
	cases := []struct {
		level, principal, method, path string
		want                           bool
		code                           string
	}{
		{TenantAccessFull, "staff", http.MethodPost, "/api/v1/projects", true, ""},
		{TenantAccessReadOnly, "staff", http.MethodGet, "/api/v1/projects", true, ""},
		{TenantAccessReadOnly, "staff", http.MethodPost, "/api/v1/projects", false, "subscription_read_only"},
		{TenantAccessReadOnly, "staff", http.MethodPost, "/api/v1/subscriptions/pay", true, ""},
		{TenantAccessBillingOnly, "staff", http.MethodGet, "/api/v1/projects", false, "subscription_suspended"},
		{TenantAccessBillingOnly, "staff", http.MethodPost, "/api/v1/subscriptions/pay", true, ""},
		// The couple keeps reading the portal whatever the WO's billing state.
		{TenantAccessBillingOnly, "client", http.MethodGet, "/api/v1/projects/me", true, ""},
		{TenantAccessReadOnly, "client", http.MethodGet, "/api/v1/projects/me", true, ""},
		// A platform freeze stops everyone.
		{TenantAccessBlocked, "client", http.MethodGet, "/api/v1/projects/me", false, "tenant_frozen"},
		{TenantAccessBlocked, "staff", http.MethodPost, "/api/v1/subscriptions/pay", false, "tenant_frozen"},
	}
	for _, c := range cases {
		got, code, _ := decideTenantAccess(c.level, c.principal, c.method, c.path, payOnly)
		if got != c.want || code != c.code {
			t.Errorf("%s %s %s %s: got (%v,%q), want (%v,%q)", c.level, c.principal, c.method, c.path, got, code, c.want, c.code)
		}
	}
}

type fakeResolver struct {
	level string
	err   error
}

func (f fakeResolver) TenantAccessLevel(context.Context, int64) (string, error) {
	return f.level, f.err
}

func TestTenantAccessGateMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	serve := func(g *TenantAccessGate, claims *Claims, method string) int {
		req := httptest.NewRequest(method, "/api/v1/projects", nil)
		if claims != nil {
			req = req.WithContext(NewContextWithClaims(req.Context(), claims))
		}
		rec := httptest.NewRecorder()
		g.Middleware(next).ServeHTTP(rec, req)
		return rec.Code
	}
	staff := &Claims{PrincipalType: "staff", TenantID: "7"}

	g := NewTenantAccessGate(payOnly)
	if code := serve(g, staff, http.MethodPost); code != http.StatusNoContent {
		t.Errorf("no resolver bound yet should pass, got %d", code)
	}

	g.SetResolver(fakeResolver{level: TenantAccessReadOnly})
	if code := serve(g, staff, http.MethodPost); code != http.StatusForbidden {
		t.Errorf("read-only write should be 403, got %d", code)
	}
	if code := serve(g, &Claims{PrincipalType: "platform_admin"}, http.MethodPost); code != http.StatusNoContent {
		t.Errorf("platform admin must not be gated, got %d", code)
	}

	g.SetResolver(fakeResolver{err: errors.New("db down")})
	if code := serve(g, staff, http.MethodGet); code != http.StatusServiceUnavailable {
		t.Errorf("resolver error should fail closed with 503, got %d", code)
	}
}
