package application

import (
	"context"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
)

// ManagementService is what OTHER modules call (via identity/contracts) to
// provision/rotate/deactivate a login identity when they create, reset, or
// deactivate one of their own principals (a new tenant Owner, a new platform
// admin, a client contact, etc.) — identity never reaches into their tables,
// they never reach into identity's. See ADR-0005.
type ManagementService struct {
	credentials CredentialRepository
	hasher      PasswordHasher
}

func NewManagementService(credentials CredentialRepository, hasher PasswordHasher) *ManagementService {
	return &ManagementService{credentials: credentials, hasher: hasher}
}

type CreateCredentialInput struct {
	TenantID      *int64
	PrincipalType domain.PrincipalType
	PrincipalID   string
	Username      string
	Password      string
	Role          string
	DisplayName   string
}

func (s *ManagementService) CreateCredential(ctx context.Context, input CreateCredentialInput) error {
	existing, err := s.credentials.FindByUsername(ctx, input.Username)
	if err != nil {
		return err
	}
	if existing != nil {
		return apperror.Conflict("Username sudah digunakan")
	}

	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return apperror.Internal("Gagal memproses password")
	}

	return s.credentials.Create(ctx, &domain.Credential{
		TenantID:      input.TenantID,
		PrincipalType: input.PrincipalType,
		PrincipalID:   input.PrincipalID,
		Username:      input.Username,
		PasswordHash:  hash,
		Role:          input.Role,
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
	return s.credentials.UpdatePasswordHash(ctx, principalType, principalID, hash)
}

func (s *ManagementService) SetActive(ctx context.Context, principalType domain.PrincipalType, principalID string, isActive bool) error {
	return s.credentials.SetActive(ctx, principalType, principalID, isActive)
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
	return s.credentials.UpdatePasswordHash(ctx, cred.PrincipalType, cred.PrincipalID, hash)
}
