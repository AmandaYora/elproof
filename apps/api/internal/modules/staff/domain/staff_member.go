// Package domain holds staff's core entity. Full CRUD lands in Fase 3 — Fase 2
// only needs enough of this module for `platform` to create the Owner row
// when a tenant registers. See PLAN.md Fase 2/3, ADR-0008.
package domain

import "time"

// StaffRole is the name of the member's workspace role (a snapshot shown
// in lists); authority comes from RoleID and the access module (ADR-0026).
type StaffRole string

// RoleOwner names the workspace's system role. There is exactly one Owner,
// created at registration; it is never assigned or removed here.
const RoleOwner StaffRole = "Owner"

type StaffMember struct {
	ID       int64
	TenantID int64
	Name     string
	Title    string
	Initials string
	Role     StaffRole
	// RoleID is the workspace role (access module's tenant_roles.id).
	RoleID    int64
	Username  string
	Email     string
	Phone     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
