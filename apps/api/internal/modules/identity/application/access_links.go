package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
)

// AccessLinkRepository stores one-time access links (auth_tokens).
type AccessLinkRepository interface {
	Issue(ctx context.Context, credentialID int64, purpose domain.LinkPurpose, tokenHash string, expiresAt time.Time) error
	RevokeAll(ctx context.Context, credentialID int64) error
	FindByHash(ctx context.Context, tokenHash string) (*domain.AccessLink, error)
	ConsumeAndLinkGoogle(ctx context.Context, linkID, credentialID int64, sub string) (bool, error)
	ConsumeAndSetPassword(ctx context.Context, linkID, credentialID int64, passwordHash string) (bool, error)
}

// TenantNameResolver names the WO on an invitation page. Declared here,
// implemented by `platform`, wired in cmd/server.
type TenantNameResolver interface {
	TenantName(ctx context.Context, tenantID int64) (string, error)
}

// SetAccessLinks plugs in the link store (identity.module.go).
func (s *AuthService) SetAccessLinks(repo AccessLinkRepository) {
	s.links = repo
}

// SetTenantNames completes two-phase wiring with `platform`.
func (s *AuthService) SetTenantNames(r TenantNameResolver) {
	s.tenantNames = r
}

// Link lifetimes (PLAN.md §3 "Undangan staf & klien").
const (
	staffInviteTTL  = 7 * 24 * time.Hour
	clientInviteTTL = 7 * 24 * time.Hour
	clientResetTTL  = 24 * time.Hour

	minClientPasswordLen = 8
	maxClientPasswordLen = 72 // bcrypt's limit
)

// AccessLinkResult is a freshly issued link; Token goes into the URL
// /undangan/{token} and is never stored in plain text.
type AccessLinkResult struct {
	Token     string
	Purpose   domain.LinkPurpose
	ExpiresAt time.Time
}

// IssueAccessLink creates a one-time link for a login, revoking any earlier
// open link of that login:
//   - staff: always an invitation (also to re-link a lost Google account);
//   - client: an invitation while no password was ever set, otherwise a
//     password reset.
func (s *AuthService) IssueAccessLink(ctx context.Context, principalType domain.PrincipalType, principalID string) (*AccessLinkResult, error) {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil {
		return nil, err
	}
	if cred == nil {
		return nil, apperror.NotFound("Akun login tidak ditemukan")
	}
	if !cred.IsActive {
		return nil, apperror.Conflict("Akun ini nonaktif. Aktifkan dulu sebelum membuat link.")
	}
	var purpose domain.LinkPurpose
	var ttl time.Duration
	switch cred.PrincipalType {
	case domain.PrincipalStaff:
		if strings.TrimSpace(cred.Email) == "" {
			return nil, apperror.Validation("Email belum diisi", map[string][]string{
				"email": {"Isi email akun Google orang ini dulu; undangan hanya bisa diterima dengan email tersebut"},
			})
		}
		purpose, ttl = domain.PurposeStaffInvite, staffInviteTTL
	case domain.PrincipalClient:
		if cred.PasswordHash == "" {
			purpose, ttl = domain.PurposeClientInvite, clientInviteTTL
		} else {
			purpose, ttl = domain.PurposeClientReset, clientResetTTL
		}
	default:
		return nil, apperror.Forbidden("Link akses hanya untuk staf dan client WO")
	}
	plain, hash, err := s.issuer.NewRefreshTokenValue()
	if err != nil {
		return nil, apperror.Internal("Gagal membuat link")
	}
	expiresAt := time.Now().Add(ttl).Truncate(time.Second)
	if err := s.links.Issue(ctx, cred.ID, purpose, hash, expiresAt); err != nil {
		return nil, err
	}
	return &AccessLinkResult{Token: plain, Purpose: purpose, ExpiresAt: expiresAt}, nil
}

// LinkInfo is what the invitation page shows before the person acts.
type LinkInfo struct {
	Purpose     domain.LinkPurpose
	TenantName  string
	DisplayName string
	Username    string
	Email       string
	ExpiresAt   time.Time
}

func linkError(code, msg string) error {
	return apperror.NotFound(msg).WithCode(code)
}

// openLink resolves a token to its link and login, refusing used, revoked,
// expired or deactivated ones with a reason the page can show.
func (s *AuthService) openLink(ctx context.Context, token string) (*domain.AccessLink, *domain.Credential, error) {
	if s.links == nil || token == "" {
		return nil, nil, linkError("link_invalid", "Link tidak dikenali. Minta link baru ke WO Anda.")
	}
	link, err := s.links.FindByHash(ctx, s.issuer.HashToken(token))
	if err != nil {
		return nil, nil, err
	}
	switch {
	case link == nil:
		return nil, nil, linkError("link_invalid", "Link tidak dikenali. Minta link baru ke WO Anda.")
	case link.UsedAt != nil:
		return nil, nil, linkError("link_used", "Link ini sudah dipakai. Silakan masuk, atau minta link baru ke WO Anda.")
	case link.RevokedAt != nil:
		return nil, nil, linkError("link_revoked", "Link ini sudah diganti dengan link yang lebih baru. Gunakan link terakhir dari WO Anda.")
	case !time.Now().Before(link.ExpiresAt):
		return nil, nil, linkError("link_expired", "Link ini sudah kedaluwarsa. Minta link baru ke WO Anda.")
	}
	cred, err := s.credentials.FindByID(ctx, link.CredentialID)
	if err != nil {
		return nil, nil, err
	}
	if cred == nil || !cred.IsActive {
		return nil, nil, inactiveAccount()
	}
	return link, cred, nil
}

// AccessLinkInfo backs GET /auth/links/{token}.
func (s *AuthService) AccessLinkInfo(ctx context.Context, token string) (*LinkInfo, error) {
	link, cred, err := s.openLink(ctx, token)
	if err != nil {
		return nil, err
	}
	info := &LinkInfo{Purpose: link.Purpose, DisplayName: cred.DisplayName, ExpiresAt: link.ExpiresAt}
	switch link.Purpose {
	case domain.PurposeStaffInvite:
		info.Email = cred.Email // the Google account to use
	default:
		info.Username = cred.Username // what the client will type to log in
	}
	if cred.TenantID != nil && s.tenantNames != nil {
		if name, err := s.tenantNames.TenantName(ctx, *cred.TenantID); err == nil {
			info.TenantName = name
		}
	}
	return info, nil
}

// AcceptStaffInvite redeems a staff invitation with a Google sign-in. The
// Google account's verified email must equal the invited email, so a link
// forwarded to someone else is useless to them.
func (s *AuthService) AcceptStaffInvite(ctx context.Context, token, idToken string) (*Session, error) {
	link, cred, err := s.openLink(ctx, token)
	if err != nil {
		return nil, err
	}
	if link.Purpose != domain.PurposeStaffInvite {
		return nil, linkError("link_invalid", "Link ini bukan undangan staf.")
	}
	google, err := s.VerifyGoogle(ctx, idToken)
	if err != nil {
		return nil, err
	}
	if cred.Email == "" || !strings.EqualFold(strings.TrimSpace(google.Email), strings.TrimSpace(cred.Email)) {
		return nil, apperror.Forbidden("Undangan ini untuk " + cred.Email + ". Masuk dengan akun Google yang memakai email tersebut.").WithCode("email_mismatch")
	}
	if err := s.checkTenant(ctx, cred); err != nil {
		return nil, err
	}
	if other, err := s.credentials.FindByGoogleSub(ctx, google.Sub); err != nil {
		return nil, err
	} else if other != nil && other.ID != cred.ID {
		return nil, apperror.Conflict("Akun Google ini sudah dipakai akun ElProof lain.").WithCode("google_account_exists")
	}
	ok, err := s.links.ConsumeAndLinkGoogle(ctx, link.ID, cred.ID, google.Sub)
	if errors.Is(err, domain.ErrGoogleAccountTaken) {
		return nil, apperror.Conflict("Akun Google ini sudah dipakai akun ElProof lain.").WithCode("google_account_exists")
	}
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, s.linkGone(ctx, token)
	}
	// A re-invitation (lost Google account) must end the old sessions.
	if err := s.endSessions(ctx, cred); err != nil {
		return nil, err
	}
	cred.GoogleSub, cred.PasswordHash = google.Sub, ""
	return s.issueSession(ctx, cred, false)
}

// SetPasswordWithLink redeems a client invitation or password reset: the
// client chooses their own password, every other session ends, and they
// are signed in.
func (s *AuthService) SetPasswordWithLink(ctx context.Context, token, password string) (*Session, error) {
	link, cred, err := s.openLink(ctx, token)
	if err != nil {
		return nil, err
	}
	if link.Purpose != domain.PurposeClientInvite && link.Purpose != domain.PurposeClientReset {
		return nil, linkError("link_invalid", "Link ini bukan untuk mengatur kata sandi.")
	}
	if n := utf8.RuneCountInString(password); n < minClientPasswordLen || len(password) > maxClientPasswordLen {
		return nil, apperror.Validation("Kata sandi belum memenuhi syarat", map[string][]string{
			"password": {"Kata sandi minimal 8 karakter"},
		})
	}
	if strings.EqualFold(password, cred.Username) {
		return nil, apperror.Validation("Kata sandi belum memenuhi syarat", map[string][]string{
			"password": {"Kata sandi tidak boleh sama dengan nama pengguna"},
		})
	}
	if err := s.checkTenant(ctx, cred); err != nil {
		return nil, err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, apperror.Internal("Gagal memproses kata sandi")
	}
	ok, err := s.links.ConsumeAndSetPassword(ctx, link.ID, cred.ID, hash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, s.linkGone(ctx, token)
	}
	if err := s.endSessions(ctx, cred); err != nil {
		return nil, err
	}
	s.passwordFailures.Reset(credentialKey(cred.ID))
	cred.PasswordHash = hash
	return s.issueSession(ctx, cred, false)
}

func (s *AuthService) endSessions(ctx context.Context, cred *domain.Credential) error {
	if s.sessions != nil {
		s.sessions.Invalidate(cred.PrincipalType, cred.PrincipalID)
	}
	return s.refreshTokens.RevokeAllForCredential(ctx, cred.ID)
}

// checkTenant refuses before a link is spent when the login's workspace is
// frozen (the session could not be issued anyway).
func (s *AuthService) checkTenant(ctx context.Context, cred *domain.Credential) error {
	if cred.TenantID != nil && s.tenantGuard != nil {
		return s.tenantGuard.CheckTenantLogin(ctx, *cred.TenantID)
	}
	return nil
}

// linkGone explains why a link could not be redeemed after all (used,
// revoked or expired in the meantime).
func (s *AuthService) linkGone(ctx context.Context, token string) error {
	if _, _, err := s.openLink(ctx, token); err != nil {
		return err
	}
	return linkError("link_used", "Link ini sudah dipakai. Silakan masuk.")
}
