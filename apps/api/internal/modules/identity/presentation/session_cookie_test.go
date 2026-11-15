package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"elproof/internal/modules/identity/application"
	"elproof/internal/modules/identity/domain"
)

// ---- minimal fakes (the application package's own are not exported) ----

type tokens struct {
	mu   sync.Mutex
	rows []*domain.RefreshToken
}

func (m *tokens) Create(_ context.Context, t *domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t.ID = int64(len(m.rows) + 1)
	cp := *t
	m.rows = append(m.rows, &cp)
	return nil
}
func (m *tokens) FindByTokenHash(_ context.Context, h string) (*domain.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.TokenHash == h {
			cp := *r
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *tokens) Revoke(_ context.Context, id int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == id && r.RevokedAt == nil {
			now := time.Now()
			r.RevokedAt = &now
			return true, nil
		}
	}
	return false, nil
}
func (m *tokens) RevokeFamily(_ context.Context, family int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.Family() == family && r.RevokedAt == nil {
			now := time.Now()
			r.RevokedAt = &now
		}
	}
	return nil
}
func (m *tokens) FamilyActive(_ context.Context, family int64, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.Family() == family && r.RevokedAt == nil && r.ExpiresAt.After(now) {
			return true, nil
		}
	}
	return false, nil
}
func (m *tokens) RevokeAllForCredential(context.Context, int64) error { return nil }

type clientLogin struct {
	application.CredentialRepository
	cred *domain.Credential
}

func (c clientLogin) FindByUsername(_ context.Context, u string) (*domain.Credential, error) {
	if u == c.cred.Username {
		cp := *c.cred
		return &cp, nil
	}
	return nil, nil
}
func (c clientLogin) FindAllByEmail(context.Context, string) ([]*domain.Credential, error) {
	return nil, nil
}
func (c clientLogin) FindByID(_ context.Context, id int64) (*domain.Credential, error) {
	if id == c.cred.ID {
		cp := *c.cred
		return &cp, nil
	}
	return nil, nil
}

type issuer struct {
	mu sync.Mutex
	n  int
}

func (i *issuer) IssueAccessToken(*domain.Credential, bool, time.Duration) (string, error) {
	return "access", nil
}
func (i *issuer) IssueServiceToken(string, string, time.Duration) (string, error) { return "", nil }
func (i *issuer) IssueImpersonationToken(*domain.Credential, string, int64, time.Duration) (string, error) {
	return "", nil
}
func (i *issuer) NewRefreshTokenValue() (string, string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.n++
	v := fmt.Sprintf("refresh-%d", i.n)
	return v, v, nil
}
func (i *issuer) HashToken(p string) string { return p }

type plainHasher struct{}

func (plainHasher) Compare(hash, pw string) error {
	if hash != pw {
		return fmt.Errorf("mismatch")
	}
	return nil
}
func (plainHasher) Hash(pw string) (string, error) { return pw, nil }

func newHandler(t *testing.T, secure bool) (*Handler, *tokens) {
	t.Helper()
	store := &tokens{}
	cred := &domain.Credential{ID: 3, PrincipalType: domain.PrincipalClient, PrincipalID: "c-3", Username: "citra", PasswordHash: "rahasia", IsActive: true}
	auth := application.NewAuthService(clientLogin{cred: cred}, store, &issuer{}, plainHasher{}, 30*time.Minute, 168*time.Hour)
	h := NewHandler(auth)
	h.SetCookieSecure(secure)
	return h, store
}

func sessionCookieOf(t *testing.T, w *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %s cookie in %v", name, w.Header().Values("Set-Cookie"))
	return nil
}

func login(t *testing.T, h *Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"citra","password":"rahasia"}`))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	return w
}

// ---- tests ----

func TestLoginSetsAHardenedRefreshCookieAndNoTokenInTheBody(t *testing.T) {
	h, _ := newHandler(t, true)
	w := login(t, h)

	c := sessionCookieOf(t, w, "__Secure-elproof_refresh")
	if c.Value != "refresh-1" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/v1/auth" || c.Domain != "" {
		t.Errorf("cookie attributes: %+v", c)
	}
	if c.MaxAge != int((168 * time.Hour).Seconds()) {
		t.Errorf("Max-Age must match the refresh TTL, got %d", c.MaxAge)
	}
	if strings.Contains(w.Body.String(), "refresh-1") || strings.Contains(w.Body.String(), "refreshToken") {
		t.Errorf("the refresh token must not appear in the JSON body: %s", w.Body.String())
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data["accessToken"] != "access" || body.Data["principalType"] != "client" {
		t.Errorf("session body: %v", body.Data)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("a response that sets the session cookie must not be cached")
	}
}

func TestDevelopmentCookieIsNotSecure(t *testing.T) {
	h, _ := newHandler(t, false)
	c := sessionCookieOf(t, login(t, h), "elproof_refresh")
	if c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Errorf("dev cookie: %+v", c)
	}
}

func TestRefreshReadsTheCookieAndRotatesIt(t *testing.T) {
	h, store := newHandler(t, true)
	first := sessionCookieOf(t, login(t, h), "__Secure-elproof_refresh")

	// A token in the body is ignored: browsers never send one, and the
	// cookie is the only accepted carrier.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refreshToken":"refresh-1"}`))
	w := httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh without the cookie: want 401, got %d", w.Code)
	}
	if c := sessionCookieOf(t, w, "__Secure-elproof_refresh"); c.MaxAge >= 0 {
		t.Errorf("a refused refresh must clear the cookie, got %+v", c)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: first.Name, Value: first.Value})
	w = httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh with the cookie: %d %s", w.Code, w.Body.String())
	}
	next := sessionCookieOf(t, w, "__Secure-elproof_refresh")
	if next.Value == first.Value || next.Value == "" || !next.HttpOnly {
		t.Errorf("refresh must set a new cookie: %+v", next)
	}
	if old, _ := store.FindByTokenHash(context.Background(), first.Value); old.RevokedAt == nil {
		t.Error("the old refresh token must be revoked")
	}
	if strings.Contains(w.Body.String(), next.Value) {
		t.Error("the rotated token must not appear in the body")
	}
}

func TestLogoutRevokesAndClearsTheCookie(t *testing.T) {
	h, store := newHandler(t, true)
	c := sessionCookieOf(t, login(t, h), "__Secure-elproof_refresh")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})
	w := httptest.NewRecorder()
	h.Logout(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: %d", w.Code)
	}
	cleared := sessionCookieOf(t, w, "__Secure-elproof_refresh")
	if cleared.Value != "" || cleared.MaxAge >= 0 || cleared.Path != "/api/v1/auth" {
		t.Errorf("logout must expire the cookie on the same path: %+v", cleared)
	}
	if rec, _ := store.FindByTokenHash(context.Background(), c.Value); rec.RevokedAt == nil {
		t.Error("logout must revoke the refresh token")
	}

	// Without a cookie logout still answers 200 and clears.
	w = httptest.NewRecorder()
	h.Logout(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
	if w.Code != http.StatusOK {
		t.Errorf("logout without cookie: %d", w.Code)
	}
}
