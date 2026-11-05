package domain

import "time"

type PlatformAdminRole string

// Platform Console roles (FLOW-IDEAL.md §1.3, least privilege). What each
// role may do is enforced centrally in cmd/server/platform_scope.go.
const (
	// RoleSuperAdmin: everything, incl. admins, gateway and tenant freeze.
	RoleSuperAdmin PlatformAdminRole = "Super Admin"
	// RoleBilling: plans, plan mapping and manual subscription activation.
	RoleBilling PlatformAdminRole = "Billing"
	// RoleSupport: register/edit tenants and reset their Owner credential.
	RoleSupport PlatformAdminRole = "Support"
	// RoleReadOnly: view only.
	RoleReadOnly PlatformAdminRole = "Read-only"
)

// IsValidPlatformAdminRole reports whether r is one of the roles above.
func IsValidPlatformAdminRole(r PlatformAdminRole) bool {
	switch r {
	case RoleSuperAdmin, RoleBilling, RoleSupport, RoleReadOnly:
		return true
	}
	return false
}

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
