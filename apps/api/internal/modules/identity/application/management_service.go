package application

import (
	"context"
	"errors"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
)

// ManagementService is what OTHER modules call (via identity/contracts) to
// provision/rotate/deactivate a login identity when they create, reset, or
// deactivate one of their own principals (a new tenant Owner, a new platform
// admin, a client contact, etc.) — identity never reaches into their tables,
// they never reach into identity's. See ADR-0005.
type ManagementService struct {
	credentials   CredentialRepository
	refreshTokens RefreshTokenRepository
	hasher        PasswordHasher
	sessions      *SessionChecker
	links         AccessLinkRepository
}

// SetAccessLinks lets deactivation and replacement revoke open links.
func (s *ManagementService) SetAccessLinks(repo AccessLinkRepository) {
	s.links = repo
}

// SetSessionChecker lets account changes invalidate cached session state
// immediately (identity.module.go).
func (s *ManagementService) SetSessionChecker(c *SessionChecker) {
	s.sessions = c
}

func NewManagementService(credentials CredentialRepository, refreshTokens RefreshTokenRepository, hasher PasswordHasher) *ManagementService {
	return &ManagementService{credentials: credentials, refreshTokens: refreshTokens, hasher: hasher}
}

// revokeSessions ends every open session of one principal. Called whenever
// that principal loses access (deactivated) or its password is replaced, so
// an already-issued refresh token can't keep the old session alive.
func (s *ManagementService) revokeSessions(ctx context.Context, cred *domain.Credential) error {
	if cred == nil {
		return nil
	}
	s.sessions.Invalidate(cred.PrincipalType, cred.PrincipalID)
	return s.refreshTokens.RevokeAllForCredential(ctx, cred.ID)
}

type CreateCredentialInput struct {
	TenantID      *int64
	PrincipalType domain.PrincipalType
	PrincipalID   string
	Username      string
	Email         string
	Password      string
	// GoogleSub creates a Google-only login (no password): Password must
	// then be empty.
	GoogleSub string
	// Pending creates a login with no way to sign in yet: the person
	// activates it through an access link (staff/client invitations, Fase
	// 3). Password and GoogleSub must be empty.
	Pending bool
	Role    string
	// RoleID: workspace role (tenant_roles.id) for staff and clients.
	RoleID      int64
	DisplayName string
}

func (s *ManagementService) CreateCredential(ctx context.Context, input CreateCredentialInput) error {
	existing, err := s.credentials.FindByUsername(ctx, input.Username)
	if err != nil {
		return err
	}
	if existing != nil {
		return apperror.Validation("Username sudah digunakan", map[string][]string{"username": {"Username ini sudah dipakai, pilih yang lain"}})
	}

	// Login can resolve either identifier now (see AuthService.Login), so an
	// email shared by two accounts would make login-by-email ambiguous by
	// construction -- reject it up front the same way a taken username
	// already is, rather than only catching it via the DB's unique
	// constraint (migration 000016).
	if input.Email != "" {
		existingByEmail, err := s.credentials.FindAllByEmail(ctx, input.Email)
		if err != nil {
			return err
		}
		if len(existingByEmail) > 0 {
			return apperror.Validation("Email sudah digunakan akun lain", map[string][]string{"email": {"Email ini sudah dipakai akun lain"}})
		}
	}

	var hash string
	if input.Pending {
		if input.Password != "" || input.GoogleSub != "" {
			return apperror.Internal("Akun undangan tidak memakai kata sandi atau Google")
		}
	} else if input.GoogleSub != "" {
		if input.Password != "" {
			return apperror.Internal("Akun Google tidak memakai kata sandi")
		}
		linked, err := s.credentials.FindByGoogleSub(ctx, input.GoogleSub)
		if err != nil {
			return err
		}
		if linked != nil {
			return apperror.Conflict("Akun Google ini sudah terhubung ke akun ElProof lain")
		}
	} else {
		if input.Password == "" {
			return apperror.Internal("Kata sandi wajib untuk akun tanpa Google")
		}
		hash, err = s.hasher.Hash(input.Password)
		if err != nil {
			return apperror.Internal("Gagal memproses password")
		}
	}

	return s.credentials.Create(ctx, &domain.Credential{
		TenantID:      input.TenantID,
		PrincipalType: input.PrincipalType,
		PrincipalID:   input.PrincipalID,
		Username:      input.Username,
		Email:         input.Email,
		GoogleSub:     input.GoogleSub,
		PasswordHash:  hash,
		Role:          input.Role,
		RoleID:        input.RoleID,
		DisplayName:   input.DisplayName,
		IsActive:      true,
	})
}

func (s *ManagementService) ResetPassword(ctx context.Context, principalType domain.PrincipalType, principalID string, newPassword string) error {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil {
		return err
	}
	if cred == nil {
		return apperror.NotFound("Kredensial tidak ditemukan")
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return apperror.Internal("Gagal memproses password")
	}
	if err := s.credentials.UpdatePasswordHash(ctx, principalType, principalID, hash); err != nil {
		return err
	}
	return s.revokeSessions(ctx, cred)
}

func (s *ManagementService) SetActive(ctx context.Context, principalType domain.PrincipalType, principalID string, isActive bool) error {
	if err := s.credentials.SetActive(ctx, principalType, principalID, isActive); err != nil {
		return err
	}
	s.sessions.Invalidate(principalType, principalID)
	if isActive {
		return nil
	}
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil {
		return err
	}
	if cred != nil && s.links != nil {
		if err := s.links.RevokeAll(ctx, cred.ID); err != nil {
			return err
		}
	}
	return s.revokeSessions(ctx, cred)
}

// UpdateRole syncs this table's own denormalized `role` column — see
// CredentialRepository.UpdateRole's doc comment for why this exists at all.
func (s *ManagementService) UpdateRole(ctx context.Context, principalType domain.PrincipalType, principalID string, role string, roleID int64) error {
	if err := s.credentials.UpdateRole(ctx, principalType, principalID, role, roleID); err != nil {
		return err
	}
	// Tokens carrying the old role stop working at once; the next refresh
	// issues one with the new role.
	s.sessions.Invalidate(principalType, principalID)
	return nil
}

// ResetPasswordByUsername lets a caller reset a password without knowing the
// principal's (type, id) pair — used by `platform`'s tenant-credential-reset
// flow, since `platform` only ever tracks the Owner's username, never the
// staff_members.id `identity` uses internally (no cross-module FK).
func (s *ManagementService) ResetPasswordByUsername(ctx context.Context, username string, newPassword string) error {
	cred, err := s.credentials.FindByUsername(ctx, username)
	if err != nil {
		return err
	}
	if cred == nil {
		return apperror.NotFound("Kredensial tidak ditemukan")
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return apperror.Internal("Gagal memproses password")
	}
	if err := s.credentials.UpdatePasswordHash(ctx, cred.PrincipalType, cred.PrincipalID, hash); err != nil {
		return err
	}
	return s.revokeSessions(ctx, cred)
}

// ResetMFA clears two-factor for one principal and revokes their sessions.
func (s *ManagementService) ResetMFA(ctx context.Context, principalType domain.PrincipalType, principalID string) error {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil {
		return err
	}
	if cred == nil {
		return apperror.NotFound("Kredensial tidak ditemukan")
	}
	if err := s.credentials.ClearTOTP(ctx, principalType, principalID); err != nil {
		return err
	}
	return s.revokeSessions(ctx, cred)
}

func (s *ManagementService) MFAEnabled(ctx context.Context, principalType domain.PrincipalType, principalID string) (bool, error) {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil || cred == nil {
		return false, err
	}
	return cred.TOTPEnabled, nil
}

// UsernameTaken reports whether a login already uses username.
func (s *ManagementService) UsernameTaken(ctx context.Context, username string) (bool, error) {
	return s.credentials.UsernameTaken(ctx, username)
}

// CredentialExists reports whether a principal already has a login.
func (s *ManagementService) CredentialExists(ctx context.Context, principalType domain.PrincipalType, principalID string) (bool, error) {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	return cred != nil, err
}

// CheckGoogleSignup refuses a new workspace for a Google account that is
// already linked to a login, or whose email belongs to a login linked to a
// Google account (that person should sign in). An email used by a login
// without Google does NOT block signup — a WO Owner could otherwise squat
// anyone's address by adding it as staff; the new Owner login is then
// created without the email (see EmailTaken).
func (s *ManagementService) CheckGoogleSignup(ctx context.Context, sub, email string) error {
	linked, err := s.credentials.FindByGoogleSub(ctx, sub)
	if err != nil {
		return err
	}
	if linked != nil {
		return apperror.Conflict("Akun Google ini sudah terdaftar. Silakan masuk.").WithCode("google_account_exists")
	}
	byEmail, err := s.credentials.FindAllByEmail(ctx, email)
	if err != nil {
		return err
	}
	for _, c := range byEmail {
		if c.GoogleSub != "" {
			return apperror.Conflict("Email " + email + " sudah terhubung ke akun ElProof lain. Silakan masuk.").WithCode("email_in_use")
		}
	}
	return nil
}

// EmailTaken reports whether any login already uses email.
func (s *ManagementService) EmailTaken(ctx context.Context, email string) (bool, error) {
	byEmail, err := s.credentials.FindAllByEmail(ctx, email)
	return len(byEmail) > 0, err
}

// UpdateEmail keeps the login's email in step with the owning module (for a
// pending staff invitation it is the Google account that must accept).
func (s *ManagementService) UpdateEmail(ctx context.Context, principalType domain.PrincipalType, principalID string, email string) error {
	err := s.credentials.UpdateEmail(ctx, principalType, principalID, email)
	if errors.Is(err, domain.ErrEmailTaken) {
		return apperror.Validation("Email sudah digunakan akun lain", map[string][]string{"email": {"Email ini sudah dipakai akun lain"}})
	}
	return err
}

// ResetSignIn removes every way to sign in to a login and ends its sessions
// and open links — used when the person behind it changes (a client's
// family representative is replaced); a new invitation follows. A non-empty
// newUsername renames the login for the new person.
func (s *ManagementService) ResetSignIn(ctx context.Context, principalType domain.PrincipalType, principalID string, newUsername string) error {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil {
		return err
	}
	if cred == nil {
		return apperror.NotFound("Akun login tidak ditemukan")
	}
	if newUsername != "" && newUsername != cred.Username {
		if err := s.credentials.UpdateUsername(ctx, cred.ID, newUsername); err != nil {
			if errors.Is(err, domain.ErrUsernameTaken) {
				return apperror.Validation("Nama pengguna sudah dipakai", map[string][]string{"username": {"Nama pengguna ini sudah dipakai, pilih yang lain"}})
			}
			return err
		}
	}
	if err := s.credentials.ClearSignIn(ctx, cred.ID); err != nil {
		return err
	}
	if s.links != nil {
		if err := s.links.RevokeAll(ctx, cred.ID); err != nil {
			return err
		}
	}
	return s.revokeSessions(ctx, cred)
}

// LoginStatus is "invited" (no way to sign in yet), "active" or "inactive";
// "none" when the principal has no login at all.
func (s *ManagementService) LoginStatus(ctx context.Context, principalType domain.PrincipalType, principalID string) (string, error) {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil || cred == nil {
		return "none", err
	}
	return loginStatusOf(cred), nil
}

// loginStatusOf: "inactive" (switched off), "invited" (no way to sign in yet) or "active".
func loginStatusOf(cred *domain.Credential) string {
	switch {
	case !cred.IsActive:
		return "inactive"
	case cred.PasswordHash == "" && cred.GoogleSub == "":
		return "invited"
	}
	return "active"
}

// loginStatusBatch is how many principals one query looks up.
const loginStatusBatch = 500

// LoginStatuses is LoginStatus for many principals of one workspace, in one
// query per 500 ids instead of one per principal. Every requested id has an
// entry: a principal with no login (or one of another workspace) is "none".
func (s *ManagementService) LoginStatuses(ctx context.Context, principalType domain.PrincipalType, tenantID int64, principalIDs []string) (map[string]string, error) {
	out := make(map[string]string, len(principalIDs))
	for _, id := range principalIDs {
		out[id] = "none"
	}
	for start := 0; start < len(principalIDs); start += loginStatusBatch {
		end := start + loginStatusBatch
		if end > len(principalIDs) {
			end = len(principalIDs)
		}
		creds, err := s.credentials.ListByPrincipals(ctx, principalType, tenantID, principalIDs[start:end])
		if err != nil {
			return nil, err
		}
		for _, c := range creds {
			out[c.PrincipalID] = loginStatusOf(c)
		}
	}
	return out, nil
}

// DeleteCredential removes a principal's login entirely (sessions and links
// included), freeing its username and email — when the owning module deletes
// the person or rolls back a failed invitation. Missing logins are fine.
func (s *ManagementService) DeleteCredential(ctx context.Context, principalType domain.PrincipalType, principalID string) error {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil || cred == nil {
		return err
	}
	s.sessions.Invalidate(principalType, principalID)
	return s.credentials.Delete(ctx, cred.ID)
}

// LoginEmail is the email stored on a principal's login ("" when none).
func (s *ManagementService) LoginEmail(ctx context.Context, principalType domain.PrincipalType, principalID string) (string, error) {
	cred, err := s.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil || cred == nil {
		return "", err
	}
	return cred.Email, nil
}

// CountLoginsWithRole backs access's "role still in use" check.
func (s *ManagementService) CountLoginsWithRole(ctx context.Context, tenantID, roleID int64) (int, error) {
	return s.credentials.CountWithRole(ctx, tenantID, roleID)
}
