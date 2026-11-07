package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/platform/domain"
	staffcontracts "elproof/internal/modules/staff/contracts"
	"elproof/internal/shared/apperror"
)

type consoleFakeRepo struct {
	ConsoleRepository
	grant    *domain.SupportAccessGrant
	sessions []domain.SupportSession
	ended    map[int64]bool
	policy   domain.SubscriptionPolicy
	revoked  int
	deleted  []int64
}

func (r *consoleFakeRepo) LatestGrant(context.Context, int64) (*domain.SupportAccessGrant, error) {
	return r.grant, nil
}
func (r *consoleFakeRepo) CreateGrant(_ context.Context, g *domain.SupportAccessGrant) error {
	g.ID = 1
	r.grant = g
	return nil
}
func (r *consoleFakeRepo) RevokeGrants(_ context.Context, _ int64, at time.Time) error {
	r.revoked++
	if r.grant != nil {
		r.grant.RevokedAt = &at
	}
	return nil
}
func (r *consoleFakeRepo) CreateSession(_ context.Context, s *domain.SupportSession) error {
	s.ID = int64(len(r.sessions) + 1)
	r.sessions = append(r.sessions, *s)
	return nil
}
func (r *consoleFakeRepo) EndSession(_ context.Context, id int64, adminID string, _ time.Time) (bool, error) {
	if r.ended == nil {
		r.ended = map[int64]bool{}
	}
	for _, s := range r.sessions {
		if s.ID == id && s.AdminID == adminID && !r.ended[id] {
			r.ended[id] = true
			return true, nil
		}
	}
	return false, nil
}
func (r *consoleFakeRepo) FindSession(_ context.Context, id int64) (*domain.SupportSession, error) {
	for i := range r.sessions {
		if r.sessions[i].ID == id {
			s := r.sessions[i]
			if r.ended[id] {
				at := time.Now()
				s.EndedAt = &at
			}
			return &s, nil
		}
	}
	return nil, nil
}
func (r *consoleFakeRepo) DeleteSession(_ context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	return nil
}
func (r *consoleFakeRepo) ListSessions(context.Context, int64, int) ([]domain.SupportSession, error) {
	return r.sessions, nil
}
func (r *consoleFakeRepo) SavePolicy(_ context.Context, p domain.SubscriptionPolicy, _ string) error {
	r.policy = p
	return nil
}

type consoleFakeTenants struct {
	TenantRepository
}

func (consoleFakeTenants) FindByID(_ context.Context, id int64) (*domain.Tenant, error) {
	return &domain.Tenant{ID: id, BusinessName: "WO Uji"}, nil
}

type consoleFakeStaff struct {
	staffcontracts.Contracts
	owner string
}

func (s consoleFakeStaff) FindOwner(context.Context, int64) (string, error) { return s.owner, nil }

type fakeIssuer struct {
	err    error
	called int
}

func (f *fakeIssuer) IssueSupportToken(_ context.Context, staffID, adminID string, sessionID int64, ttl time.Duration) (string, error) {
	f.called++
	if f.err != nil {
		return "", f.err
	}
	return "token-for-" + staffID + "-by-" + adminID, nil
}

func newConsole(repo *consoleFakeRepo, owner string, issuer *fakeIssuer) *ConsoleService {
	tenants := &TenantService{repo: consoleFakeTenants{}, staff: consoleFakeStaff{owner: owner}}
	return NewConsoleService(repo, tenants, issuer)
}

func codeOf(err error) string {
	if ae, ok := apperror.As(err); ok {
		return ae.Code
	}
	return ""
}

func TestSupportSessionNeedsConsentAndReason(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	repo := &consoleFakeRepo{}
	issuer := &fakeIssuer{}
	s := newConsole(repo, "5", issuer)

	// No consent from the Owner: refused, no token, nothing recorded.
	_, err := s.StartSupportSession(ctx, 1, "9", "support1", "Membantu atur timeline default", now)
	if codeOf(err) != "support_access_not_granted" || issuer.called != 0 || len(repo.sessions) != 0 {
		t.Fatalf("without consent: %v issued=%d sessions=%d", err, issuer.called, len(repo.sessions))
	}

	if _, err := s.GrantSupportAccess(ctx, 1, 5, now); err != nil {
		t.Fatal(err)
	}
	// Consent given but no (real) reason.
	if _, err := s.StartSupportSession(ctx, 1, "9", "support1", "ya", now); err == nil {
		t.Fatal("a reason is required")
	}
	res, err := s.StartSupportSession(ctx, 1, "9", "support1", "Membantu atur timeline default", now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Token != "token-for-5-by-9" {
		t.Errorf("token %q: it must be the Owner's login, issued for the admin", res.Token)
	}
	if got := res.Session.ExpiresAt.Sub(now); got != 60*time.Minute {
		t.Errorf("session lasts %v, want 60m", got)
	}

	// The consent lapses after 24 hours.
	_, err = s.StartSupportSession(ctx, 1, "9", "support1", "Membantu atur timeline default", now.Add(25*time.Hour))
	if codeOf(err) != "support_access_not_granted" {
		t.Errorf("after the grant expired: %v", err)
	}

	// Only the admin who started a session can end it.
	if err := s.EndSupportSession(ctx, res.Session.ID, "someone-else", now); err == nil {
		t.Error("another admin must not end the session")
	}
	if err := s.EndSupportSession(ctx, res.Session.ID, "9", now); err != nil {
		t.Errorf("own session: %v", err)
	}
	if err := s.EndSupportSession(ctx, res.Session.ID, "9", now); err == nil {
		t.Error("ending twice must fail")
	}
}

func TestRevokedSupportAccessStopsNewSessions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	repo := &consoleFakeRepo{}
	s := newConsole(repo, "5", &fakeIssuer{})
	if _, err := s.GrantSupportAccess(ctx, 1, 5, now); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeSupportAccess(ctx, 1, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartSupportSession(ctx, 1, "9", "s", "Membantu atur timeline default", now.Add(2*time.Minute)); err == nil {
		t.Fatal("a revoked consent must stop new sessions")
	}
	st, _ := s.SupportStatus(ctx, 1, now.Add(2*time.Minute))
	if st.Grant != nil {
		t.Error("a revoked grant must not be reported as active")
	}
}

func TestSupportSessionWithoutOwnerOrToken(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	repo := &consoleFakeRepo{}
	s := newConsole(repo, "", &fakeIssuer{})
	_, _ = s.GrantSupportAccess(ctx, 1, 5, now)
	if _, err := s.StartSupportSession(ctx, 1, "9", "s", "Membantu atur timeline default", now); err == nil {
		t.Fatal("a workspace without an Owner cannot be entered")
	}

	// Token failure: the session record is closed again, not left as a view that never happened.
	repo2 := &consoleFakeRepo{}
	s2 := newConsole(repo2, "5", &fakeIssuer{err: apperror.Forbidden("dibekukan")})
	_, _ = s2.GrantSupportAccess(ctx, 1, 5, now)
	if _, err := s2.StartSupportSession(ctx, 1, "9", "s", "Membantu atur timeline default", now); err == nil {
		t.Fatal("token failure must fail the start")
	}
	if len(repo2.deleted) != 1 {
		t.Errorf("failed start must remove its session record, deleted=%v", repo2.deleted)
	}
}

func TestSavePolicyValidatesAndApplies(t *testing.T) {
	defer domain.SetSubscriptionPolicy(domain.DefaultSubscriptionPolicy)
	repo := &consoleFakeRepo{}
	s := newConsole(repo, "5", &fakeIssuer{})
	if err := s.SavePolicy(context.Background(), domain.SubscriptionPolicy{TrialDays: 0}, "1"); err == nil {
		t.Fatal("invalid policy must be refused")
	}
	if domain.CurrentSubscriptionPolicy().TrialDays != 14 {
		t.Fatal("a refused policy must not change anything")
	}
	good := domain.SubscriptionPolicy{TrialDays: 30, ExpiringSoonDays: 7, GraceDays: 7, ReadOnlyDays: 23}
	if err := s.SavePolicy(context.Background(), good, "1"); err != nil {
		t.Fatal(err)
	}
	if repo.policy != good || domain.CurrentSubscriptionPolicy() != good {
		t.Fatal("policy must be stored and applied")
	}
}

func TestDataRequestValidation(t *testing.T) {
	ctx := context.Background()
	s := newConsole(&consoleFakeRepo{}, "5", &fakeIssuer{})
	if _, err := s.AddDataRequest(ctx, 1, "bogus", "owner@wo.id", "", "1", time.Now()); err == nil {
		t.Error("unknown kind")
	}
	if _, err := s.AddDataRequest(ctx, 1, domain.DataRequestExport, " ", "", "1", time.Now()); err == nil {
		t.Error("who asked is required")
	}
	if err := s.CompleteDataRequest(ctx, 1, domain.DataRequestDone, "1", "ok", time.Now()); err == nil {
		t.Error("proof of completion is required (5+ chars)")
	}
	if err := s.CompleteDataRequest(ctx, 1, "open", "1", "bukti lengkap", time.Now()); err == nil {
		t.Error("a request can only be closed as done or rejected")
	}
}

type fakeAdmins struct{ admin *domain.PlatformAdmin }

func (f fakeAdmins) FindByID(context.Context, int64) (*domain.PlatformAdmin, error) {
	return f.admin, nil
}

func TestSupportSessionStopsWhenConditionsChange(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	admin := &domain.PlatformAdmin{ID: 9, IsActive: true, Role: domain.RoleSupport}

	start := func() (*ConsoleService, *consoleFakeRepo, int64) {
		repo := &consoleFakeRepo{}
		s := newConsole(repo, "5", &fakeIssuer{})
		s.SetAdmins(fakeAdmins{admin: admin})
		if _, err := s.GrantSupportAccess(ctx, 1, 5, now); err != nil {
			t.Fatal(err)
		}
		res, err := s.StartSupportSession(ctx, 1, "9", "support1", "Membantu atur timeline default", now)
		if err != nil {
			t.Fatal(err)
		}
		return s, repo, res.Session.ID
	}

	s, repo, id := start()
	if err := s.CheckSupportSession(ctx, 1, id, "9", now.Add(time.Minute)); err != nil {
		t.Fatalf("a live session must pass: %v", err)
	}
	if err := s.CheckSupportSession(ctx, 2, id, "9", now.Add(time.Minute)); err == nil {
		t.Error("another workspace must not accept the session")
	}
	if err := s.CheckSupportSession(ctx, 1, id, "8", now.Add(time.Minute)); err == nil {
		t.Error("another admin's claim must not match the session")
	}
	if err := s.CheckSupportSession(ctx, 1, id, "9", now.Add(61*time.Minute)); err == nil {
		t.Error("the session must lapse after 60 minutes")
	}
	if err := s.CheckSupportSession(ctx, 1, 999, "9", now); err == nil {
		t.Error("an unknown session must be refused")
	}

	// The Owner withdraws consent: the running session ends too.
	if err := s.RevokeSupportAccess(ctx, 1, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckSupportSession(ctx, 1, id, "9", now.Add(3*time.Minute)); err == nil {
		t.Error("withdrawing consent must end a running session")
	}

	// The admin ends it: a copied token stops working.
	s, repo, id = start()
	_ = repo
	if err := s.EndSupportSession(ctx, id, "9", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckSupportSession(ctx, 1, id, "9", now.Add(2*time.Minute)); err == nil {
		t.Error("an ended session must not accept its token")
	}

	// The admin is deactivated or demoted.
	s, _, id = start()
	admin.IsActive = false
	if err := s.CheckSupportSession(ctx, 1, id, "9", now.Add(time.Minute)); err == nil {
		t.Error("a deactivated admin must lose the session")
	}
	admin.IsActive, admin.Role = true, domain.RoleReadOnly
	if err := s.CheckSupportSession(ctx, 1, id, "9", now.Add(time.Minute)); err == nil {
		t.Error("an admin demoted to Read-only must lose the session")
	}
}
