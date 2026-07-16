// Package domain holds identity's core entities. No framework or DB imports here.
package domain

import "time"

type PrincipalType string

const (
	PrincipalStaff         PrincipalType = "staff"
	PrincipalClient        PrincipalType = "client"
	PrincipalPlatformAdmin PrincipalType = "platform_admin"
)

// Credential is a login identity for one of the three principal types. It does
// not own profile data (name, contact info, etc.) beyond DisplayName, which is
// kept only for immediate post-login UI display — see ADR-0005.
type Credential struct {
	ID            int64
	TenantID      *int64
	PrincipalType PrincipalType
	PrincipalID   string
	Username      string
	PasswordHash  string
	Role          string
	DisplayName   string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RefreshToken is a rotatable, revocable session token tied to one Credential.
type RefreshToken struct {
	ID           int64
	CredentialID int64
	TokenHash    string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}
