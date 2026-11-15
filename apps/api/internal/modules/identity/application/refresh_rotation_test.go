package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
)

// memRefreshTokens is an in-memory RefreshTokenRepository.
type memRefreshTokens struct {
	mu     sync.Mutex
	rows   []*domain.RefreshToken
	now    func() time.Time
	nextID int64
}

func (m *memRefreshTokens) Create(_ context.Context, t *domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	t.ID = m.nextID
	cp := *t
	m.rows = append(m.rows, &cp)
	return nil
}

func (m *memRefreshTokens) FindByTokenHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.TokenHash == hash {
			cp := *r
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memRefreshTokens) Revoke(_ context.Context, id int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == id && r.RevokedAt == nil {
			t := m.now()
			r.RevokedAt = &t
			return true, nil
		}
	}
	return false, nil
}

func (m *memRefreshTokens) RevokeFamily(_ context.Context, family int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.Family() == family && r.RevokedAt == nil {
			t := m.now()
			r.RevokedAt = &t
		}
	}
	return nil
}

func (m *memRefreshTokens) FamilyActive(_ context.Context, family int64, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.Family() == family && r.RevokedAt == nil && r.ExpiresAt.After(now) {
			return true, nil
		}
	}
	return false, nil
}

func (m *memRefreshTokens) RevokeAllForCredential(_ context.Context, credentialID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.CredentialID == credentialID && r.RevokedAt == nil {
			t := m.now()
			r.RevokedAt = &t
		}
	}
	return nil
}

// oneCredential serves FindByID for a single account; any other repository
// method is not used by these tests (and would panic on the nil interface).
type oneCredential struct {
	CredentialRepository
	cred *domain.Credential
}

func (o oneCredential) FindByID(_ context.Context, id int64) (*domain.Credential, error) {
	if o.cred != nil && o.cred.ID == id {
		cp := *o.cred
		return &cp, nil
	}
	return nil, nil
}

// countingIssuer mints predictable tokens; the "hash" is the token itself.
type countingIssuer struct {
	mu sync.Mutex
	n  int
}

func (c *countingIssuer) IssueAccessToken(cred *domain.Credential, mfa bool, _ time.Duration) (string, error) {
	return fmt.Sprintf("access-%d-%v", cred.ID, mfa), nil
}
func (c *countingIssuer) IssueServiceToken(string, string, time.Duration) (string, error) {
	return "", nil
}
func (c *countingIssuer) IssueImpersonationToken(*domain.Credential, string, int64, time.Duration) (string, error) {
	return "", nil
}
func (c *countingIssuer) NewRefreshTokenValue() (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	v := fmt.Sprintf("rt-%d", c.n)
	return v, v, nil
}
func (c *countingIssuer) HashToken(plain string) string { return plain }

type rotationRig struct {
	svc    *AuthService
	tokens *memRefreshTokens
	clock  time.Time
}

func newRotationRig(t *testing.T) *rotationRig {
	t.Helper()
	rig := &rotationRig{clock: time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)}
	now := func() time.Time { return rig.clock }
	rig.tokens = &memRefreshTokens{now: now}
	cred := &domain.Credential{ID: 7, PrincipalType: domain.PrincipalClient, PrincipalID: "c-7", Role: "Bride", DisplayName: "Citra", IsActive: true}
	rig.svc = NewAuthService(oneCredential{cred: cred}, rig.tokens, &countingIssuer{}, nil, 30*time.Minute, 168*time.Hour)
	rig.svc.now = now
	return rig
}

func (r *rotationRig) login(t *testing.T) *Session {
	t.Helper()
	s, err := r.svc.issueSession(context.Background(), &domain.Credential{ID: 7, PrincipalType: domain.PrincipalClient, PrincipalID: "c-7", IsActive: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (r *rotationRig) advance(d time.Duration) { r.clock = r.clock.Add(d) }

func isUnauthorized(err error) bool {
	ae, ok := apperror.As(err)
	return ok && ae.Kind == apperror.KindUnauthorized
}

func TestRefreshRotatesAndKeepsTheFamily(t *testing.T) {
	rig := newRotationRig(t)
	first := rig.login(t)

	second, err := rig.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == "" || second.RefreshToken == first.RefreshToken || second.AccessToken == "" {
		t.Fatalf("refresh must issue a new refresh and access token: %+v", second)
	}
	old, _ := rig.tokens.FindByTokenHash(context.Background(), first.RefreshToken)
	next, _ := rig.tokens.FindByTokenHash(context.Background(), second.RefreshToken)
	if old.RevokedAt == nil {
		t.Error("the presented token must be revoked")
	}
	if next.Family() != old.Family() || next.FamilyID != old.ID {
		t.Errorf("rotated token must join the first token's family: old=%+v next=%+v", old, next)
	}
	if !next.ExpiresAt.Equal(rig.clock.Add(168 * time.Hour)) {
		t.Errorf("new token must live the refresh TTL, got %v", next.ExpiresAt)
	}
}

func TestReusingARotatedTokenAfterTheGraceRevokesTheFamily(t *testing.T) {
	rig := newRotationRig(t)
	first := rig.login(t)
	second, err := rig.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	other := rig.login(t) // another device: its own family

	rig.advance(DefaultRefreshReuseGrace + time.Second)
	if _, err := rig.svc.Refresh(context.Background(), first.RefreshToken); !isUnauthorized(err) {
		t.Fatalf("reuse after the grace must be refused with 401, got %v", err)
	}
	if _, err := rig.svc.Refresh(context.Background(), second.RefreshToken); !isUnauthorized(err) {
		t.Fatalf("the newest token of a reused family must be revoked too, got %v", err)
	}
	if _, err := rig.svc.Refresh(context.Background(), other.RefreshToken); err != nil {
		t.Fatalf("another sign-in of the same account must survive, got %v", err)
	}
}

func TestReusingARotatedTokenWithinTheGraceIsAccepted(t *testing.T) {
	rig := newRotationRig(t)
	first := rig.login(t)
	tabA, err := rig.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	rig.advance(5 * time.Second)
	// A second tab sent the same cookie before tab A's response updated it.
	tabB, err := rig.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("a rotated token within the grace must be accepted, got %v", err)
	}
	if tabB.RefreshToken == tabA.RefreshToken {
		t.Error("each redemption gets its own successor")
	}
	for _, s := range []*Session{tabA, tabB} {
		if _, err := rig.svc.Refresh(context.Background(), s.RefreshToken); err != nil {
			t.Errorf("successor %s must still work: %v", s.RefreshToken, err)
		}
	}
}

func TestGraceDoesNotReviveASignedOutSession(t *testing.T) {
	rig := newRotationRig(t)
	first := rig.login(t)
	second, err := rig.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := rig.svc.Logout(context.Background(), second.RefreshToken); err != nil {
		t.Fatal(err)
	}
	rig.advance(time.Second)
	if _, err := rig.svc.Refresh(context.Background(), first.RefreshToken); !isUnauthorized(err) {
		t.Fatalf("a token of a signed-out family must be refused even within the grace, got %v", err)
	}
	if _, err := rig.svc.Refresh(context.Background(), second.RefreshToken); !isUnauthorized(err) {
		t.Fatalf("logout must revoke the presented token, got %v", err)
	}
}

func TestLogoutRevokesTheWholeFamily(t *testing.T) {
	rig := newRotationRig(t)
	first := rig.login(t)
	second, _ := rig.svc.Refresh(context.Background(), first.RefreshToken)
	// Signing out with the older (rotated) cookie still ends the sign-in.
	if err := rig.svc.Logout(context.Background(), first.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := rig.svc.Refresh(context.Background(), second.RefreshToken); !isUnauthorized(err) {
		t.Fatalf("logout must revoke every token of the family, got %v", err)
	}
	if err := rig.svc.Logout(context.Background(), ""); err != nil {
		t.Errorf("logout without a cookie is a no-op, got %v", err)
	}
	if err := rig.svc.Logout(context.Background(), "unknown"); err != nil {
		t.Errorf("logout with an unknown token is a no-op, got %v", err)
	}
}

func TestRefreshRefusesMissingExpiredAndUnknownTokens(t *testing.T) {
	rig := newRotationRig(t)
	first := rig.login(t)
	for _, token := range []string{"", "never-issued"} {
		if _, err := rig.svc.Refresh(context.Background(), token); !isUnauthorized(err) {
			t.Errorf("token %q: want 401, got %v", token, err)
		}
	}
	rig.advance(169 * time.Hour)
	if _, err := rig.svc.Refresh(context.Background(), first.RefreshToken); !isUnauthorized(err) {
		t.Errorf("expired token: want 401, got %v", err)
	}
}

func TestZeroGraceRefusesEveryReuse(t *testing.T) {
	rig := newRotationRig(t)
	rig.svc.SetRefreshReuseGrace(0)
	first := rig.login(t)
	second, err := rig.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rig.svc.Refresh(context.Background(), first.RefreshToken); !isUnauthorized(err) {
		t.Fatalf("without a grace any reuse is refused, got %v", err)
	}
	if _, err := rig.svc.Refresh(context.Background(), second.RefreshToken); !isUnauthorized(err) {
		t.Fatalf("and the family is revoked, got %v", err)
	}
}

func TestConcurrentRefreshesWithOneToken(t *testing.T) {
	rig := newRotationRig(t)
	first := rig.login(t)
	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = rig.svc.Refresh(context.Background(), first.RefreshToken)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d: concurrent refreshes within the grace must all succeed, got %v", i, err)
		}
	}
}
