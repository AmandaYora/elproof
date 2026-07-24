// Package domain holds staff's core entity. Full CRUD lands in Fase 3 — Fase 2
// only needs enough of this module for `platform` to create the Owner row
// when a tenant registers. See PLAN.md Fase 2/3, ADR-0008.
package domain

import "time"

type StaffRole string

const (
	RoleOwner StaffRole = "Owner"
	RoleAdmin StaffRole = "Admin"
	RoleStaff StaffRole = "Staff"
)

type StaffMember struct {
	ID        int64
	TenantID  int64
	Name      string
	Title     string
	Initials  string
	Role      StaffRole
	Username  string
	Email     string
	Phone     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
