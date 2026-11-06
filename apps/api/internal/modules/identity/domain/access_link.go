package domain

import (
	"errors"
	"time"
)

// LinkPurpose is what a one-time access link does (PLAN.md Fase 3).
type LinkPurpose string

const (
	// PurposeStaffInvite: the invited person signs in with the Google
	// account whose email matches the invitation; staff never get an
	// ElProof password (decision K5).
	PurposeStaffInvite LinkPurpose = "staff_invite"
	// PurposeClientInvite: a new client sets their own password.
	PurposeClientInvite LinkPurpose = "client_invite"
	// PurposeClientReset: a client who forgot their password sets a new one.
	PurposeClientReset LinkPurpose = "client_reset"
)

// AccessLink is a stored one-time link. Only the token's SHA-256 hash is
// kept; the plain token exists only in the URL handed to the person.
type AccessLink struct {
	ID           int64
	CredentialID int64
	Purpose      LinkPurpose
	ExpiresAt    time.Time
	UsedAt       *time.Time
	RevokedAt    *time.Time
}

var (
	// ErrGoogleAccountTaken: the Google account is already linked to another
	// login (credentials.google_sub is unique).
	ErrGoogleAccountTaken = errors.New("google account already linked")
	// ErrEmailTaken: another login already uses this email.
	ErrEmailTaken = errors.New("email already used by another login")
	// ErrUsernameTaken: another login already uses this username.
	ErrUsernameTaken = errors.New("username already used by another login")
)
