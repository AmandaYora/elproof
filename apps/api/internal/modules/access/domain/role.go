package domain

import (
	"errors"
	"time"
)

// Role is a named set of permissions inside one workspace.
type Role struct {
	ID          int64
	TenantID    int64
	Kind        Kind
	Name        string
	Description string
	// TemplateKey is set for built-in roles ("owner", "admin", …).
	TemplateKey string
	// IsSystem: the Owner role — every permission, cannot be edited or
	// deleted, so a WO can never lock itself out.
	IsSystem    bool
	Scope       Scope
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Has reports whether the role holds a permission.
func (r Role) Has(key string) bool {
	for _, p := range r.Permissions {
		if p == key {
			return true
		}
	}
	return false
}

// ErrDuplicateRoleName: the workspace already has a role with this name.
var ErrDuplicateRoleName = errors.New("role name already used")
