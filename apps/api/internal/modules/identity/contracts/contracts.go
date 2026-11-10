// Package contracts is the ONLY package other modules may import from
// identity. It re-declares the small vocabulary other modules need
// (PrincipalType, CreateCredentialInput) instead of exposing identity/domain,
// so a caller like `platform` never needs to import identity's internals.
package contracts

import (
	"context"
	"errors"
	"time"

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
	Email         string
	Password      string
	// GoogleSub creates a Google-only login (Password must be empty).
	GoogleSub string
	// Pending creates a login activated later through an access link
	// (invitation); Password and GoogleSub must be empty.
	Pending bool
	Role    string
	// RoleID: workspace role (tenant_roles.id) for staff and clients.
	RoleID      int64
	DisplayName string
}

// GoogleIdentity is a Google account proven by a verified ID token.
type GoogleIdentity struct {
	Sub       string
	Email     string
	Name      string
	ExpiresAt time.Time
}

// Contracts is what other modules depend on to provision/rotate/deactivate a
// login identity for one of their own principals — see ADR-0005.
type Contracts interface {
	CreateCredential(ctx context.Context, input CreateCredentialInput) error
	ResetPassword(ctx context.Context, principalType PrincipalType, principalID string, newPassword string) error
	ResetPasswordByUsername(ctx context.Context, username string, newPassword string) error
	SetActive(ctx context.Context, principalType PrincipalType, principalID string, isActive bool) error
	// UpdateRole syncs the denormalized `role` this module's own `credentials`
	// table keeps (baked into every login/refresh JWT) with the owning
	// module's current role value. Call this from every code path that edits
	// a principal's role after account creation — CreateCredential only ever
	// sets it once, at creation time.
	// roleID is the workspace role (tenant_roles.id); 0 for platform admins.
	UpdateRole(ctx context.Context, principalType PrincipalType, principalID string, role string, roleID int64) error
	// CountLoginsWithRole counts a workspace's logins holding a role.
	CountLoginsWithRole(ctx context.Context, tenantID, roleID int64) (int, error)
	// IssueServiceToken mints a bearer token for a principal not backed by a
	// Credential row — the caller has already authenticated it against its
	// own store (e.g. `payment`'s external Apps, see
	// knowledge/MODULE_PAYMENT.md §7.1). No refresh token is issued.
	IssueServiceToken(ctx context.Context, principalType string, principalID string, ttl time.Duration) (string, error)
	// IssueSupportToken mints a read-only support-session token for a
	// workspace Owner's staff login on behalf of a platform admin.
	IssueSupportToken(ctx context.Context, staffPrincipalID, adminID string, sessionID int64, ttl time.Duration) (string, error)
	// ResetMFA removes a principal's two-factor setup (lost device) and ends
	// their sessions; they must enrol again at next login.
	ResetMFA(ctx context.Context, principalType PrincipalType, principalID string) error
	// MFAEnabled reports whether a principal has confirmed two-factor.
	MFAEnabled(ctx context.Context, principalType PrincipalType, principalID string) (bool, error)

	// Sign in with Google / self-service signup (PLAN.md Fase 2).
	// VerifyGoogleIDToken validates a Google ID token without signing in.
	VerifyGoogleIDToken(ctx context.Context, idToken string) (GoogleIdentity, error)
	// CheckGoogleSignup errors when this Google account (or a Google-linked
	// login with its email) already exists (codes google_account_exists,
	// email_in_use).
	CheckGoogleSignup(ctx context.Context, sub, email string) error
	// EmailTaken reports whether any login uses this email.
	EmailTaken(ctx context.Context, email string) (bool, error)
	UsernameTaken(ctx context.Context, username string) (bool, error)
	CredentialExists(ctx context.Context, principalType PrincipalType, principalID string) (bool, error)

	// Access links (PLAN.md Fase 3). IssueAccessLink returns a one-time
	// link token for a staff invitation, client invitation or client
	// password reset (chosen from the login's state); earlier open links of
	// that login stop working.
	IssueAccessLink(ctx context.Context, principalType PrincipalType, principalID string) (AccessLink, error)
	// UpdateEmail syncs a login's email with the owning module.
	UpdateEmail(ctx context.Context, principalType PrincipalType, principalID string, email string) error
	// ResetSignIn removes every way to sign in and ends sessions and links
	// (the person behind the login changed); a non-empty newUsername renames
	// the login. Issue a new link after it.
	ResetSignIn(ctx context.Context, principalType PrincipalType, principalID string, newUsername string) error
	// LoginStatus: "invited", "active", "inactive" or "none".
	LoginStatus(ctx context.Context, principalType PrincipalType, principalID string) (string, error)
	// LoginStatuses is LoginStatus for many principals of one workspace in a
	// few queries (the clients list). Every id asked for has an entry; a
	// principal with no login in that workspace is "none".
	LoginStatuses(ctx context.Context, principalType PrincipalType, tenantID int64, principalIDs []string) (map[string]string, error)
	// DeleteCredential removes a login with its sessions and links (person
	// deleted, invitation rolled back); its username and email become free.
	DeleteCredential(ctx context.Context, principalType PrincipalType, principalID string) error
	// LoginEmail is the email on a principal's login ("" when none).
	LoginEmail(ctx context.Context, principalType PrincipalType, principalID string) (string, error)
}

// AccessLink is a freshly issued one-time link: the web app's URL is
// /undangan/{Token}. Purpose is staff_invite, client_invite or client_reset.
type AccessLink struct {
	Token     string
	Purpose   string
	ExpiresAt time.Time
}

type impl struct {
	management *application.ManagementService
	auth       *application.AuthService
}

// New builds a Contracts implementation. auth may be nil for callers that
// only ever need credential CRUD (e.g. internal/adminseed) — IssueServiceToken
// errors clearly if called on such an instance instead of panicking.
func New(management *application.ManagementService, auth *application.AuthService) Contracts {
	return &impl{management: management, auth: auth}
}

func (c *impl) CreateCredential(ctx context.Context, input CreateCredentialInput) error {
	return c.management.CreateCredential(ctx, application.CreateCredentialInput{
		TenantID:      input.TenantID,
		PrincipalType: domain.PrincipalType(input.PrincipalType),
		PrincipalID:   input.PrincipalID,
		Username:      input.Username,
		Email:         input.Email,
		Password:      input.Password,
		GoogleSub:     input.GoogleSub,
		Pending:       input.Pending,
		RoleID:        input.RoleID,
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

func (c *impl) UpdateRole(ctx context.Context, principalType PrincipalType, principalID string, role string, roleID int64) error {
	return c.management.UpdateRole(ctx, domain.PrincipalType(principalType), principalID, role, roleID)
}

func (c *impl) CountLoginsWithRole(ctx context.Context, tenantID, roleID int64) (int, error) {
	return c.management.CountLoginsWithRole(ctx, tenantID, roleID)
}

func (c *impl) ResetMFA(ctx context.Context, principalType PrincipalType, principalID string) error {
	return c.management.ResetMFA(ctx, domain.PrincipalType(principalType), principalID)
}

func (c *impl) MFAEnabled(ctx context.Context, principalType PrincipalType, principalID string) (bool, error) {
	return c.management.MFAEnabled(ctx, domain.PrincipalType(principalType), principalID)
}

func (c *impl) IssueServiceToken(ctx context.Context, principalType string, principalID string, ttl time.Duration) (string, error) {
	if c.auth == nil {
		return "", errors.New("identity: instance Contracts ini dibuat tanpa AuthService, tidak bisa menerbitkan token")
	}
	return c.auth.IssueServiceToken(ctx, principalType, principalID, ttl)
}

func (c *impl) VerifyGoogleIDToken(ctx context.Context, idToken string) (GoogleIdentity, error) {
	if c.auth == nil {
		return GoogleIdentity{}, errors.New("identity: instance Contracts ini dibuat tanpa AuthService")
	}
	id, err := c.auth.VerifyGoogle(ctx, idToken)
	if err != nil {
		return GoogleIdentity{}, err
	}
	return GoogleIdentity{Sub: id.Sub, Email: id.Email, Name: id.Name, ExpiresAt: id.ExpiresAt}, nil
}

func (c *impl) CheckGoogleSignup(ctx context.Context, sub, email string) error {
	return c.management.CheckGoogleSignup(ctx, sub, email)
}

func (c *impl) UsernameTaken(ctx context.Context, username string) (bool, error) {
	return c.management.UsernameTaken(ctx, username)
}

func (c *impl) CredentialExists(ctx context.Context, principalType PrincipalType, principalID string) (bool, error) {
	return c.management.CredentialExists(ctx, domain.PrincipalType(principalType), principalID)
}

func (c *impl) EmailTaken(ctx context.Context, email string) (bool, error) {
	return c.management.EmailTaken(ctx, email)
}

func (c *impl) IssueAccessLink(ctx context.Context, principalType PrincipalType, principalID string) (AccessLink, error) {
	if c.auth == nil {
		return AccessLink{}, errors.New("identity: instance Contracts ini dibuat tanpa AuthService")
	}
	l, err := c.auth.IssueAccessLink(ctx, domain.PrincipalType(principalType), principalID)
	if err != nil {
		return AccessLink{}, err
	}
	return AccessLink{Token: l.Token, Purpose: string(l.Purpose), ExpiresAt: l.ExpiresAt}, nil
}

func (c *impl) UpdateEmail(ctx context.Context, principalType PrincipalType, principalID string, email string) error {
	return c.management.UpdateEmail(ctx, domain.PrincipalType(principalType), principalID, email)
}

func (c *impl) ResetSignIn(ctx context.Context, principalType PrincipalType, principalID string, newUsername string) error {
	return c.management.ResetSignIn(ctx, domain.PrincipalType(principalType), principalID, newUsername)
}

func (c *impl) LoginStatus(ctx context.Context, principalType PrincipalType, principalID string) (string, error) {
	return c.management.LoginStatus(ctx, domain.PrincipalType(principalType), principalID)
}

func (c *impl) LoginStatuses(ctx context.Context, principalType PrincipalType, tenantID int64, principalIDs []string) (map[string]string, error) {
	return c.management.LoginStatuses(ctx, domain.PrincipalType(principalType), tenantID, principalIDs)
}

func (c *impl) DeleteCredential(ctx context.Context, principalType PrincipalType, principalID string) error {
	return c.management.DeleteCredential(ctx, domain.PrincipalType(principalType), principalID)
}

func (c *impl) LoginEmail(ctx context.Context, principalType PrincipalType, principalID string) (string, error) {
	return c.management.LoginEmail(ctx, domain.PrincipalType(principalType), principalID)
}

func (c *impl) IssueSupportToken(ctx context.Context, staffPrincipalID, adminID string, sessionID int64, ttl time.Duration) (string, error) {
	if c.auth == nil {
		return "", errors.New("identity: instance Contracts ini dibuat tanpa AuthService, tidak bisa menerbitkan token")
	}
	return c.auth.IssueSupportToken(ctx, staffPrincipalID, adminID, sessionID, ttl)
}
