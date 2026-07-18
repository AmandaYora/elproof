// Package application holds identity's use cases (login/refresh/logout). It
// depends only on repository/issuer interfaces defined here — never on a
// concrete database driver.
package application

import (
	"context"
	"time"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
)

type CredentialRepository interface {
	FindByUsername(ctx context.Context, username string) (*domain.Credential, error)
	FindByID(ctx context.Context, id int64) (*domain.Credential, error)
	FindByPrincipal(ctx context.Context, principalType domain.PrincipalType, principalID string) (*domain.Credential, error)
	Create(ctx context.Context, cred *domain.Credential) error
	UpdatePasswordHash(ctx context.Context, principalType domain.PrincipalType, principalID string, passwordHash string) error
	SetActive(ctx context.Context, principalType domain.PrincipalType, principalID string, isActive bool) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id int64) error
}

// TokenIssuer signs access tokens and mints/hashes opaque refresh tokens.
type TokenIssuer interface {
	IssueAccessToken(cred *domain.Credential, ttl time.Duration) (string, error)
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
	}
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
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*Session, error) {
	cred, err := s.credentials.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if cred == nil || !cred.IsActive {
		return nil, apperror.Unauthorized("Username atau password salah")
	}
	if err := s.hasher.Compare(cred.PasswordHash, password); err != nil {
		return nil, apperror.Unauthorized("Username atau password salah")
	}
	return s.issueSession(ctx, cred)
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

	if err := s.refreshTokens.Revoke(ctx, record.ID); err != nil {
		return nil, err
	}
	return s.issueSession(ctx, cred)
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
	return s.refreshTokens.Revoke(ctx, record.ID)
}

func (s *AuthService) issueSession(ctx context.Context, cred *domain.Credential) (*Session, error) {
	access, err := s.issuer.IssueAccessToken(cred, s.accessTTL)
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
		ExpiresAt:    time.Now().Add(s.refreshTTL),
	}
	if err := s.refreshTokens.Create(ctx, record); err != nil {
		return nil, err
	}

	return &Session{
		AccessToken:   access,
		RefreshToken:  plainRefresh,
		PrincipalType: string(cred.PrincipalType),
		PrincipalID:   cred.PrincipalID,
		TenantID:      cred.TenantID,
		Role:          cred.Role,
		DisplayName:   cred.DisplayName,
	}, nil
}
