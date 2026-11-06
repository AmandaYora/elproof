package application

import (
	"context"
	"sync"
	"time"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/middleware"
)

// SessionChecker re-validates an access token against the credential it was
// issued for, on every request (middleware.RequireLiveSession). Without it a
// token stays fully valid for its whole lifetime (30 min) after the account
// is deactivated, demoted or loses two-factor — long enough for a demoted
// Super Admin to create a new admin (security review, Fase 1).
//
// Lookups are cached briefly; every change made through this module
// (deactivate, role change, password reset, two-factor change) invalidates
// the entry immediately, so the cache never extends a revoked session.
type SessionChecker struct {
	credentials CredentialRepository
	ttl         time.Duration

	mu      sync.Mutex
	entries map[string]sessionEntry
}

type sessionEntry struct {
	cred *domain.Credential
	at   time.Time
}

const sessionCheckTTL = 30 * time.Second

func NewSessionChecker(credentials CredentialRepository) *SessionChecker {
	return &SessionChecker{credentials: credentials, ttl: sessionCheckTTL, entries: make(map[string]sessionEntry)}
}

func sessionKey(principalType domain.PrincipalType, principalID string) string {
	return string(principalType) + "/" + principalID
}

// Invalidate drops the cached state of one principal.
func (c *SessionChecker) Invalidate(principalType domain.PrincipalType, principalID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, sessionKey(principalType, principalID))
}

func (c *SessionChecker) lookup(ctx context.Context, principalType domain.PrincipalType, principalID string) (*domain.Credential, error) {
	key := sessionKey(principalType, principalID)
	now := time.Now()
	c.mu.Lock()
	e, ok := c.entries[key]
	c.mu.Unlock()
	if ok && now.Sub(e.at) < c.ttl {
		return e.cred, nil
	}
	cred, err := c.credentials.FindByPrincipal(ctx, principalType, principalID)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if len(c.entries) > 10_000 { // bounded: drop everything rather than grow
		c.entries = make(map[string]sessionEntry)
	}
	c.entries[key] = sessionEntry{cred: cred, at: now}
	c.mu.Unlock()
	return cred, nil
}

// ValidateSession implements middleware.SessionValidator. Only principals
// backed by a credential row are checked; service tokens (external Apps)
// are validated by their own module.
func (c *SessionChecker) ValidateSession(ctx context.Context, claims *middleware.Claims) error {
	pt := domain.PrincipalType(claims.PrincipalType)
	switch pt {
	case domain.PrincipalStaff, domain.PrincipalClient, domain.PrincipalPlatformAdmin:
	default:
		return nil
	}
	cred, err := c.lookup(ctx, pt, claims.PrincipalID)
	if err != nil {
		return err
	}
	expired := apperror.Unauthorized("Sesi tidak berlaku lagi, silakan masuk kembali")
	switch {
	case cred == nil, !cred.IsActive:
		return expired
	case cred.Role != claims.Role, cred.RoleID != claims.RoleID: // role changed: the refreshed token carries the new one
		return expired
	case claims.MFA && !cred.TOTPEnabled: // two-factor was reset
		return expired
	}
	return nil
}
