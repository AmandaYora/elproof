package domain

import (
	"sort"
	"strings"
)

// StaffListFilter narrows GET /staff (the team list).
type StaffListFilter struct {
	Search string
	// RoleID narrows to one workspace role (0 = any).
	RoleID int64
	// IsActive keeps only active (true) or inactive (false) members; nil = both.
	IsActive *bool
}

// SortRoleName is the sort field whose display value (a role's current name) lives
// in the `access` module while staff_members stores the role's id: the list orders
// by the role ids ranked by name (see SortRoleIDsByName).
const SortRoleName = "roleName"

// SortRoleIDsByName lists the role ids ordered by role name, ignoring case, ties
// by id: the order the roleName sort ranks members by.
func SortRoleIDsByName(names map[int64]string) []int64 {
	ids := make([]int64, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := strings.ToLower(names[ids[i]]), strings.ToLower(names[ids[j]])
		if a == b {
			return ids[i] < ids[j]
		}
		return a < b
	})
	return ids
}
