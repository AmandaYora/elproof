package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRestrictImpersonation(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := RestrictImpersonation(next)
	call := func(method string, claims *Claims) int {
		r := httptest.NewRequest(method, "/api/v1/projects", nil)
		if claims != nil {
			r = r.WithContext(context.WithValue(r.Context(), claimsContextKey, claims))
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	imp := &Claims{PrincipalType: "staff", Impersonator: "7"}
	for _, m := range []string{http.MethodGet, http.MethodHead} {
		if got := call(m, imp); got != http.StatusNoContent {
			t.Errorf("%s while impersonating: %d", m, got)
		}
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if got := call(m, imp); got != http.StatusForbidden {
			t.Errorf("%s while impersonating must be refused, got %d", m, got)
		}
	}
	// A normal session and an unauthenticated request are untouched here.
	if got := call(http.MethodPost, &Claims{PrincipalType: "staff"}); got != http.StatusNoContent {
		t.Errorf("normal staff POST: %d", got)
	}
	if got := call(http.MethodPost, nil); got != http.StatusNoContent {
		t.Errorf("no claims POST: %d", got)
	}
}

type stubSupportValidator struct {
	err    error
	called int
}

func (s *stubSupportValidator) CheckSupportSession(context.Context, int64, int64, string, time.Time) error {
	s.called++
	return s.err
}

func TestRequireLiveSupportSession(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	call := func(l *RequireLiveSupportSession, claims *Claims) int {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
		if claims != nil {
			r = r.WithContext(context.WithValue(r.Context(), claimsContextKey, claims))
		}
		w := httptest.NewRecorder()
		l.Middleware(next).ServeHTTP(w, r)
		return w.Code
	}
	support := &Claims{PrincipalType: "staff", TenantID: "3", Impersonator: "7", ImpersonationID: 11}

	ok := &RequireLiveSupportSession{}
	v := &stubSupportValidator{}
	ok.SetValidator(v)
	if got := call(ok, support); got != http.StatusNoContent || v.called != 1 {
		t.Errorf("live session: status %d, validator calls %d", got, v.called)
	}
	// Normal logins are not checked at all (no extra query per request).
	if got := call(ok, &Claims{PrincipalType: "staff", TenantID: "3"}); got != http.StatusNoContent || v.called != 1 {
		t.Errorf("normal staff: status %d, validator calls %d", got, v.called)
	}

	ended := &RequireLiveSupportSession{}
	ended.SetValidator(&stubSupportValidator{err: errors.New("ended")})
	if got := call(ended, support); got != http.StatusUnauthorized {
		t.Errorf("ended session must be refused, got %d", got)
	}

	// A support token that cannot be checked is never trusted.
	if got := call(&RequireLiveSupportSession{}, support); got != http.StatusUnauthorized {
		t.Errorf("no validator: %d", got)
	}
	noSession := &Claims{PrincipalType: "staff", TenantID: "3", Impersonator: "7"}
	if got := call(ok, noSession); got != http.StatusUnauthorized {
		t.Errorf("token without session id: %d", got)
	}
}
