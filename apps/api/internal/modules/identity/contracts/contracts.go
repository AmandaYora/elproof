// Package contracts is the ONLY package other modules may import from
// identity. It re-declares the small vocabulary other modules need
// (PrincipalType, CreateCredentialInput) instead of exposing identity/domain,
// so a caller like `platform` never needs to import identity's internals.
package contracts

import (
	"context"

	"elproof/internal/modules/identity/application"
	"elproof/internal/modules/identity/domain"
)

type PrincipalType string

const (
	PrincipalStaff         PrincipalType = "staff"
	PrincipalClient        PrincipalType = "client"
	PrincipalPlatformAdmin PrincipalType = "platform_admin"
)

type CreateCredentialInput struct {
	TenantID      *int64
	PrincipalType PrincipalType
	PrincipalID   string
	Username      string
	Password      string
	Role          string
	DisplayName   string
}

// Contracts is what other modules depend on to provision/rotate/deactivate a
// login identity for one of their own principals — see ADR-0005.
type Contracts interface {
	CreateCredential(ctx context.Context, input CreateCredentialInput) error
	ResetPassword(ctx context.Context, principalType PrincipalType, principalID string, newPassword string) error
	ResetPasswordByUsername(ctx context.Context, username string, newPassword string) error
	SetActive(ctx context.Context, principalType PrincipalType, principalID string, isActive bool) error
}

type impl struct {
	management *application.ManagementService
}

func New(management *application.ManagementService) Contracts {
	return &impl{management: management}
}

func (c *impl) CreateCredential(ctx context.Context, input CreateCredentialInput) error {
	return c.management.CreateCredential(ctx, application.CreateCredentialInput{
		TenantID:      input.TenantID,
		PrincipalType: domain.PrincipalType(input.PrincipalType),
		PrincipalID:   input.PrincipalID,
		Username:      input.Username,
		Password:      input.Password,
		Role:          input.Role,
		DisplayName:   input.DisplayName,
	})
}

func (c *impl) ResetPassword(ctx context.Context, principalType PrincipalType, principalID string, newPassword string) error {
	return c.management.ResetPassword(ctx, domain.PrincipalType(principalType), principalID, newPassword)
}

func (c *impl) ResetPasswordByUsername(ctx context.Context, username string, newPassword string) error {
	return c.management.ResetPasswordByUsername(ctx, username, newPassword)
}

func (c *impl) SetActive(ctx context.Context, principalType PrincipalType, principalID string, isActive bool) error {
	return c.management.SetActive(ctx, domain.PrincipalType(principalType), principalID, isActive)
}
