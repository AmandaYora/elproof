package main

import (
	"net/http"
	"strconv"
	"strings"
)

// clientPortalAllowed is the complete list of authenticated endpoints a
// `client` principal (bride/groom/family on the Client Portal) may call —
// deny by default (docs/plan/registrasi-wo-self-service/PLAN.md Fase 1, G1).
// Each owning module still enforces its own rules (project scoping, per-file
// visibility, field redaction); this gate guarantees a client can never reach
// a staff-only endpoint just because that module forgot to check the
// principal type. Add a route here only when the portal really calls it.
//
// The client portal is read-only, so only GET is ever allowed.
func clientPortalAllowed(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	s := strings.Split(strings.Trim(path, "/"), "/")
	// Every route below lives under /api/v1/.
	if len(s) < 3 || s[0] != "api" || s[1] != "v1" {
		return false
	}
	s = s[2:]

	switch {
	case match(s, "auth", "me"), match(s, "access", "me"):
		return true
	case match(s, "tenants", "me", "branding"), match(s, "tenants", "me", "logo"), match(s, "tenants", "me", "contact"):
		return true
	case match(s, "projects", "me"), match(s, "projects", "me", "home"):
		return true
	case len(s) >= 2 && s[0] == "projects" && isID(s[1]):
		// /projects/{id}/... — the projects module narrows this further
		// (own project only, allowlisted sub-resources, redaction).
		return clientProjectSubpathAllowed(s[2:])
	// NOT allowed: vendors/summary and vendor-categories list every vendor and
	// category of the workspace. The portal gets vendor and category names
	// inside GET /projects/{id}/vendors, for the project's own vendors only
	// (E6, ADR-0032).
	case len(s) == 3 && s[0] == "venues" && isID(s[1]) && s[2] == "attachment":
		return true
	default:
		return false
	}
}

func clientProjectSubpathAllowed(rest []string) bool {
	switch len(rest) {
	case 0:
		return true
	case 1:
		switch rest[0] {
		case "milestones", "vendors", "client-payments", "payment-terms", "issues", "evidence", "documents", "venue":
			return true
		}
		return false
	case 3:
		return rest[0] == "evidence" && isID(rest[1]) && (rest[2] == "file" || rest[2] == "link")
	default:
		return false
	}
}

func match(segments []string, want ...string) bool {
	if len(segments) != len(want) {
		return false
	}
	for i := range want {
		if segments[i] != want[i] {
			return false
		}
	}
	return true
}

func isID(s string) bool {
	n, err := strconv.ParseInt(s, 10, 64)
	return err == nil && n > 0
}
