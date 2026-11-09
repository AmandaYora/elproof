package domain

import "time"

type ClientRole string

const (
	RoleBride                ClientRole = "Bride"
	RoleGroom                ClientRole = "Groom"
	RoleFamilyRepresentative ClientRole = "Family Representative"
)

type Client struct {
	ID        int64
	TenantID  int64
	ProjectID int64
	Role      ClientRole
	// AccessRoleID is the portal role (access module) deciding what this
	// client sees — Pengantin / Keluarga by default (ADR-0026).
	AccessRoleID          int64
	Username              string
	RelationNote          string
	Name                  string
	Phone                 string
	Email                 string
	IsActive              bool
	LastCredentialResetAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	// DeletedAt is set while the client waits in the "deleted" state (soft
	// delete, ADR-0034); only the restore and purge paths ever see such a row.
	DeletedAt *time.Time
}

// SoftDeleteRetention is how long a deleted client can still be restored; the
// purge removes the row (and its login) after it. Same window as the payments
// and vendor engagements of `projects`.
const SoftDeleteRetention = 24 * time.Hour

// RestorableUntil is the instant after which a client deleted at deletedAt can
// no longer be restored.
func RestorableUntil(deletedAt time.Time) time.Time { return deletedAt.Add(SoftDeleteRetention) }

// CanRestore reports whether a client deleted at deletedAt may still be restored
// at now. A row past its retention that the purge has not reached yet counts as
// already purged.
func CanRestore(deletedAt, now time.Time) bool { return now.Before(RestorableUntil(deletedAt)) }

// PurgeCutoff is the deleted_at value at or before which a client is due for the
// purge at now (the exact complement of CanRestore).
func PurgeCutoff(now time.Time) time.Time { return now.Add(-SoftDeleteRetention) }
