package domain

import "time"

type PlatformAdminRole string

const (
	RoleSuperAdmin PlatformAdminRole = "Super Admin"
	RoleSupport    PlatformAdminRole = "Support"
)

type PlatformAdmin struct {
	ID        int64
	Name      string
	Title     string
	Role      PlatformAdminRole
	Username  string
	Email     string
	Phone     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
