// Package application holds identity's use cases (login/refresh/logout). It
// depends only on repository/issuer interfaces defined here — never on a
// concrete database driver.
package application

import (
	"context"
	"strconv"
	"time"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/middleware"
)

type CredentialRepository interface {
	FindByUsername(ctx context.Context, username string) (*domain.Credential, error)
	// FindAllByEmail backs Login's email fallback — see AuthService.Login for
	// why this returns every match instead of one.
	FindAllByEmail(ctx context.Context, email string) ([]*domain.Credential, error)
	FindByID(ctx context.Context, id int64) (*domain.Credential, error)
	// Sign in with Google (google.go).
	FindByGoogleSub(ctx context.Context, sub string) (*domain.Credential, error)
	LinkGoogle(ctx context.Context, credentialID int64, sub string) (bool, error)
	UsernameTaken(ctx context.Context, username string) (bool, error)
	// Access links (access_links.go) and owning-module sync.
	UpdateEmail(ctx context.Context, principalType domain.PrincipalType, principalID string, email string) error
	ClearSignIn(ctx context.Context, credentialID int64) error
	UpdateUsername(ctx context.Context, credentialID int64, username string) error
	Delete(ctx context.Context, credentialID int64) error
	FindByPrincipal(ctx context.Context, principalType domain.PrincipalType, principalID string) (*domain.Credential, error)
	Create(ctx context.Context, cred *domain.Credential) error
	UpdatePasswordHash(ctx context.Context, principalType domain.PrincipalType, principalID string, passwordHash string) error
	SetActive(ctx context.Context, principalType domain.PrincipalType, principalID string, isActive bool) error
	// UpdateRole keeps this table's own denormalized `role` column (baked
	// into the JWT at login/refresh — see JWTIssuer.IssueAccessToken) in
	// sync with the owning module's own role field whenever that changes
	// after account creation. Without this, a role edit (e.g. `staff`'s
	// Wedding Planner -> Admin) would silently never take effect for
	// authorization purposes, since Login/Refresh never re-derive it from
	// the owning module — they only ever read this column.
	UpdateRole(ctx context.Context, principalType domain.PrincipalType, principalID string, role string, roleID int64) error
	CountWithRole(ctx context.Context, tenantID, roleID int64) (int, error)
	// Two-factor (TOTP) state — see mfa.go.
	SetTOTPSecret(ctx context.Context, credentialID int64, enc []byte) (bool, error)
	EnableTOTP(ctx context.Context, credentialID int64, step int64) (bool, error)
	SetTOTPLastStep(ctx context.Context, credentialID int64, step int64) (bool, error)
	ClearTOTP(ctx context.Context, principalType domain.PrincipalType, principalID string) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	// Revoke reports whether this call revoked the token (false: it was
	// already revoked, e.g. by a concurrent refresh).
	Revoke(ctx context.Context, id int64) (bool, error)
	// RevokeAllForCredential ends every open session of one login identity
	// (used when the owning module deactivates that principal).
	RevokeAllForCredential(ctx context.Context, credentialID int64) error
}

// TenantGuard lets the tenant's owning module veto a login/refresh for a
// tenant-scoped principal (staff or client) — e.g. a tenant frozen by
// ElProof's platform team. Declared here, implemented by `platform`, wired in
// cmd/server, so identity never imports platform. A nil guard allows all.
type TenantGuard interface {
	CheckTenantLogin(ctx context.Context, tenantID int64) error
}

// TokenIssuer signs access tokens and mints/hashes opaque refresh tokens.
type TokenIssuer interface {
	// IssueAccessToken signs an access token; mfa records whether the
	// session was established with a second factor.
	IssueAccessToken(cred *domain.Credential, mfa bool, ttl time.Duration) (string, error)
	// IssueServiceToken signs an access-token-only JWT for a principal that
	// isn't backed by a Credential row (e.g. payment's external Apps, see
	// knowledge/MODULE_PAYMENT.md §7.1) — no refresh token, no tenant/role.
	IssueServiceToken(principalType, principalID string, ttl time.Duration) (string, error)
	NewRefreshTokenValue() (plain string, hash string, err error)
	HashToken(plain string) string
}

// PasswordHasher verifies a plaintext password against a stored hash, and
// hashes new passwords for credential creation/reset.
type PasswordHasher interface {
	Compare(hash, password string) error
	Hash(password string) (string, error)
}

type AuthService struct {
	credentials   CredentialRepository
	refreshTokens RefreshTokenRepository
	issuer        TokenIssuer
	hasher        PasswordHasher
	accessTTL     time.Duration
	refreshTTL    time.Duration
	tenantGuard   TenantGuard
	mfa           MFAEngine
	google        GoogleVerifier
	links         AccessLinkRepository
	tenantNames   TenantNameResolver
	challenges    *mfaChallenges
	sessions      *SessionChecker
	// Failed attempts are counted per credential (not per typed string,
	// which accent/case variants and username-vs-email could multiply), so
	// an account's budget is the same from any IP and any spelling.
	passwordFailures *middleware.RateLimiter
	totpFailures     *middleware.RateLimiter
}

const (
	failureWindow       = 15 * time.Minute
	maxPasswordFailures = 10
	maxTOTPFailures     = 5
)

func credentialKey(id int64) string { return strconv.FormatInt(id, 10) }

func tooManyAttempts() error {
	return apperror.RateLimited("Terlalu banyak percobaan masuk untuk akun ini. Silakan coba lagi dalam 15 menit.")
}

// checkPassword tries one candidate, honouring and updating its failure
// budget. ok=false with err=nil means "wrong password".
func (s *AuthService) checkPassword(cred *domain.Credential, password string) (bool, error) {
	if cred.PasswordHash == "" {
		// Google-only account: it has no password, so none can match.
		return false, nil
	}
	key := credentialKey(cred.ID)
	if s.passwordFailures.Exceeded(key) {
		return false, tooManyAttempts()
	}
	if s.hasher.Compare(cred.PasswordHash, password) != nil {
		s.passwordFailures.Record(key)
		return false, nil
	}
	s.passwordFailures.Reset(key)
	return true, nil
}

// SetSessionChecker lets two-factor changes invalidate cached session state.
func (s *AuthService) SetSessionChecker(c *SessionChecker) {
	s.sessions = c
}

// SetTenantGuard completes two-phase wiring: identity is built before the
// platform module that implements the guard (see cmd/server/main.go).
func (s *AuthService) SetTenantGuard(guard TenantGuard) {
	s.tenantGuard = guard
}

func NewAuthService(
	credentials CredentialRepository,
	refreshTokens RefreshTokenRepository,
	issuer TokenIssuer,
	hasher PasswordHasher,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		credentials:   credentials,
		refreshTokens: refreshTokens,
		issuer:        issuer,
		hasher:        hasher,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		challenges:    newMFAChallenges(),

		passwordFailures: middleware.NewRateLimiter(maxPasswordFailures, failureWindow),
		totpFailures:     middleware.NewRateLimiter(maxTOTPFailures, failureWindow),
	}
}

// SetMFAEngine plugs in the TOTP implementation (identity.module.go).
func (s *AuthService) SetMFAEngine(engine MFAEngine) {
	s.mfa = engine
}

// Session is what the presentation layer returns to the client after a
// successful login or refresh.
type Session struct {
	AccessToken   string
	RefreshToken  string
	PrincipalType string
	PrincipalID   string
	TenantID      *int64
	Role          string
	DisplayName   string
	// MFA is true when this session was established with a second factor;
	// MFASetupRequired tells a Platform Console admin to enrol first.
	MFA              bool
	MFASetupRequired bool
}

// LoginResult is either a session, or — for an admin with two-factor on —
// a short-lived challenge to answer with a TOTP code (VerifyMFA).
type LoginResult struct {
	Session     *Session
	MFARequired bool
	MFAToken    string
}

// Login accepts either a username or an email as identifier. Username is
// tried first (it's the unique, unambiguous case); if that doesn't resolve to
// a matching, active credential whose password checks out, identifier is
// tried again as an email. Unlike username, email isn't guaranteed unique
// across principal types/tenants (no owning module enforces it — see
// domain.Credential's doc comment), so FindAllByEmail can return more than
// one candidate; each is tried in turn rather than assuming the first match
// is the intended account — the correct one is whichever candidate's
// password actually matches.
func (s *AuthService) Login(ctx context.Context, identifier, password string) (*LoginResult, error) {
	cred, err := s.credentials.FindByUsername(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if cred != nil && cred.IsActive {
		ok, err := s.checkPassword(cred, password)
		if err != nil {
			return nil, err
		}
		if ok {
			return s.completeLogin(ctx, cred)
		}
	}

	candidates, err := s.credentials.FindAllByEmail(ctx, identifier)
	if err != nil {
		return nil, err
	}
	for _, c := range candidates {
		if !c.IsActive || (cred != nil && c.ID == cred.ID) {
			continue
		}
		ok, err := s.checkPassword(c, password)
		if err != nil {
			return nil, err
		}
		if ok {
			return s.completeLogin(ctx, c)
		}
	}

	return nil, apperror.Unauthorized("Username/email atau password salah")
}

// completeLogin runs after the password matched: an admin with two-factor
// on gets a challenge instead of a session.
func (s *AuthService) completeLogin(ctx context.Context, cred *domain.Credential) (*LoginResult, error) {
	// WO staff sign in with Google only (decision K5). The password was
	// right, so saying how to sign in leaks nothing new.
	if cred.PrincipalType == domain.PrincipalStaff {
		return nil, apperror.Forbidden("Tim WO masuk dengan tombol \"Masuk dengan Google\". Akun lama yang belum terhubung: klik tombol tersebut, lalu konfirmasi kata sandi ini sekali.").WithCode("use_google")
	}
	if requiresMFA(cred) && cred.TOTPEnabled {
		// A frozen tenant is checked again in issueSession; admins have no
		// tenant, so nothing else to check before the second factor.
		return &LoginResult{MFARequired: true, MFAToken: s.challenges.issue(cred.ID)}, nil
	}
	session, err := s.issueSession(ctx, cred, false)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Session: session}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshTokenPlain string) (*Session, error) {
	hash := s.issuer.HashToken(refreshTokenPlain)
	record, err := s.refreshTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if record == nil || record.RevokedAt != nil || record.ExpiresAt.Before(time.Now()) {
		return nil, apperror.Unauthorized("Sesi sudah berakhir, silakan login kembali")
	}

	cred, err := s.credentials.FindByID(ctx, record.CredentialID)
	if err != nil {
		return nil, err
	}
	if cred == nil || !cred.IsActive {
		return nil, apperror.Unauthorized("Sesi sudah berakhir, silakan login kembali")
	}

	revoked, err := s.refreshTokens.Revoke(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	if !revoked { // redeemed concurrently by another request
		return nil, apperror.Unauthorized("Sesi sudah berakhir, silakan login kembali")
	}
	// The refreshed session keeps exactly the second-factor status of the
	// session that created this token, and loses it if two-factor was
	// reset since.
	return s.issueSession(ctx, cred, record.MFA && cred.TOTPEnabled)
}

// IssueServiceToken mints a bearer access token for a principal the caller
// has already authenticated against its own store (e.g. `payment` verifying
// an external App's appId+secret) — identity only ever signs the token here,
// never checks a password or touches the credentials table. No refresh token
// is issued; the caller re-authenticates from scratch once this expires.
func (s *AuthService) IssueServiceToken(ctx context.Context, principalType, principalID string, ttl time.Duration) (string, error) {
	token, err := s.issuer.IssueServiceToken(principalType, principalID, ttl)
	if err != nil {
		return "", apperror.Internal("Gagal menerbitkan token")
	}
	return token, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshTokenPlain string) error {
	hash := s.issuer.HashToken(refreshTokenPlain)
	record, err := s.refreshTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		return err
	}
	if record == nil {
		return nil // idempotent — already gone
	}
	_, err = s.refreshTokens.Revoke(ctx, record.ID)
	return err
}

func (s *AuthService) issueSession(ctx context.Context, cred *domain.Credential, mfa bool) (*Session, error) {
	// Both Login (after the password matched) and Refresh funnel through
	// here, so a frozen tenant can neither start nor extend a session.
	if cred.TenantID != nil && s.tenantGuard != nil {
		if err := s.tenantGuard.CheckTenantLogin(ctx, *cred.TenantID); err != nil {
			return nil, err
		}
	}
	access, err := s.issuer.IssueAccessToken(cred, mfa, s.accessTTL)
	if err != nil {
		return nil, apperror.Internal("Gagal membuat sesi login")
	}

	plainRefresh, hash, err := s.issuer.NewRefreshTokenValue()
	if err != nil {
		return nil, apperror.Internal("Gagal membuat sesi login")
	}

	record := &domain.RefreshToken{
		CredentialID: cred.ID,
		TokenHash:    hash,
		MFA:          mfa,
		ExpiresAt:    time.Now().Add(s.refreshTTL),
	}
	if err := s.refreshTokens.Create(ctx, record); err != nil {
		return nil, err
	}

	return &Session{
		AccessToken:      access,
		RefreshToken:     plainRefresh,
		PrincipalType:    string(cred.PrincipalType),
		PrincipalID:      cred.PrincipalID,
		TenantID:         cred.TenantID,
		Role:             cred.Role,
		DisplayName:      cred.DisplayName,
		MFA:              mfa,
		MFASetupRequired: requiresMFA(cred) && !cred.TOTPEnabled,
	}, nil
}
