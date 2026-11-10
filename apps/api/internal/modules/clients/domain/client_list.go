package domain

import (
	"sort"
	"strings"
	"time"
)

// LoginStatuses are the states of a client's sign-in as the list derives them
// (identity owns the login): "active" (can sign in), "invited" (no password yet),
// "inactive" (switched off) and "none" (no login at all).
var LoginStatuses = []string{"active", "invited", "inactive", "none"}

// IsLoginStatus reports whether s is a value the loginStatus filter accepts.
func IsLoginStatus(s string) bool {
	for _, v := range LoginStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// IsClientRole reports whether s is one of the three client relations.
func IsClientRole(s string) bool {
	switch ClientRole(s) {
	case RoleBride, RoleGroom, RoleFamilyRepresentative:
		return true
	}
	return false
}

// ListFilter narrows GET /clients (the paginated list).
type ListFilter struct {
	// Search matches the name or the email.
	Search string
	// Role is one of the three client relations ("" = any).
	Role string
	// LoginStatus is one of LoginStatuses ("" = any). The login lives in
	// `identity`, so this filter is applied after the login states are read.
	LoginStatus string
	// ProjectID narrows to one project's clients.
	ProjectID *int64
}

// ClientRow is a client with what other modules own and the list shows next to
// it: the project's name and event date and the sign-in state.
type ClientRow struct {
	Client           Client
	ProjectName      string
	ProjectEventDate *time.Time
	LoginStatus      string
}

// Sort fields whose value the clients table does not hold: they are resolved
// from `projects` and `identity`, so ordering by them happens in memory.
const (
	SortProjectName      = "projectName"
	SortProjectEventDate = "projectEventDate"
	SortLoginStatus      = "loginStatus"
)

// NeedsMemoryPaging reports whether ordering by field (or filtering by
// loginStatus) needs data of another module, which a SQL ORDER BY in this
// module cannot reach.
func NeedsMemoryPaging(sortField, loginStatusFilter string) bool {
	return loginStatusFilter != "" || sortField == SortProjectName || sortField == SortProjectEventDate || sortField == SortLoginStatus
}

// FilterByLoginStatus keeps the rows in that sign-in state ("" keeps all).
func FilterByLoginStatus(rows []ClientRow, status string) []ClientRow {
	if status == "" {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if r.LoginStatus == status {
			out = append(out, r)
		}
	}
	return out
}

func roleRank(r ClientRole) int {
	switch r {
	case RoleBride:
		return 0
	case RoleGroom:
		return 1
	}
	return 2
}

// SortRows orders the rows by any list field (the cross-module ones, or a column
// of the table when a loginStatus filter forced the paging into memory), ties broken by client id in
// the same direction (the paging must be stable). Texts compare without regard
// to case, like the database's collation does for the SQL-ordered columns; a
// client whose project does not resolve sorts after the others whatever the
// direction.
func SortRows(rows []ClientRow, field string, desc bool) {
	less := func(a, b ClientRow) (lt, eq bool) {
		switch field {
		case "name":
			x, y := strings.ToLower(a.Client.Name), strings.ToLower(b.Client.Name)
			return x < y, x == y
		case "role":
			// The column is an ENUM: the database orders it by declaration order.
			x, y := roleRank(a.Client.Role), roleRank(b.Client.Role)
			return x < y, x == y
		case "createdAt":
			return a.Client.CreatedAt.Before(b.Client.CreatedAt), a.Client.CreatedAt.Equal(b.Client.CreatedAt)
		case SortProjectName:
			x, y := strings.ToLower(a.ProjectName), strings.ToLower(b.ProjectName)
			return x < y, x == y
		case SortLoginStatus:
			return a.LoginStatus < b.LoginStatus, a.LoginStatus == b.LoginStatus
		case SortProjectEventDate:
			switch {
			case a.ProjectEventDate == nil && b.ProjectEventDate == nil:
				return false, true
			case a.ProjectEventDate == nil, b.ProjectEventDate == nil:
				// handled below: absent always last
				return a.ProjectEventDate != nil, false
			}
			return a.ProjectEventDate.Before(*b.ProjectEventDate), a.ProjectEventDate.Equal(*b.ProjectEventDate)
		}
		return false, true
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if field == SortProjectEventDate && (a.ProjectEventDate == nil) != (b.ProjectEventDate == nil) {
			return a.ProjectEventDate != nil // absent is last, in both directions
		}
		lt, eq := less(a, b)
		if eq {
			if desc {
				return a.Client.ID > b.Client.ID
			}
			return a.Client.ID < b.Client.ID
		}
		if desc {
			return !lt
		}
		return lt
	})
}
