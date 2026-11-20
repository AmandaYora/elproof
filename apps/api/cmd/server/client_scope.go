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
// The client portal is read-only except for exactly five writes on the
// client's own project (ADR-0047), each behind its own portal permission in
// the projects module: the two halves of a signed upload (POST
// /projects/{id}/uploads, PUT /projects/{id}/uploads/{token}), sending a
// transfer proof to the WO (POST /projects/{id}/client-payments/submissions)
// or to a vendor the client pays directly (POST
// /projects/{id}/supplier-bills/submissions, Paket WO Tahap 4) and marking a
// task done (POST /projects/{id}/milestones/{mid}/complete). Every other
// method is refused here.
func clientPortalAllowed(method, path string) bool {
	s := strings.Split(strings.Trim(path, "/"), "/")
	// Every route below lives under /api/v1/.
	if len(s) < 3 || s[0] != "api" || s[1] != "v1" {
		return false
	}
	s = s[2:]

	if method != http.MethodGet {
		return clientPortalWriteAllowed(method, s)
	}

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
	// NOT allowed: vendors/summary (a route removed in Tahap 9; the path stays
	// refused here, see client_scope_test.go) and vendor-categories list every
	// vendor and category of the workspace. The portal gets vendor and category names
	// inside GET /projects/{id}/suppliers, for the project's own vendors and
	// venues only, a hidden vendor by its category (E6, ADR-0032).
	case len(s) == 3 && s[0] == "venues" && isID(s[1]) && s[2] == "attachment":
		return true
	default:
		return false
	}
}

// clientPortalWriteAllowed is the five non-GET routes of the portal (segments
// after /api/v1/). Anything else — another method, another path, a malformed
// id — is refused.
func clientPortalWriteAllowed(method string, s []string) bool {
	if len(s) < 3 || s[0] != "projects" || !isID(s[1]) {
		return false
	}
	rest := s[2:]
	switch {
	case method == http.MethodPost && match(rest, "uploads"):
		return true
	case method == http.MethodPut && len(rest) == 2 && rest[0] == "uploads" && rest[1] != "":
		return true
	case method == http.MethodPost && match(rest, "client-payments", "submissions"):
		return true
	case method == http.MethodPost && match(rest, "supplier-bills", "submissions"):
		return true
	case method == http.MethodPost && len(rest) == 3 && rest[0] == "milestones" && isID(rest[1]) && rest[2] == "complete":
		return true
	}
	return false
}

func clientProjectSubpathAllowed(rest []string) bool {
	switch len(rest) {
	case 0:
		return true
	case 1:
		switch rest[0] {
		case "milestones", "events", "suppliers", "lines", "supplier-payments", "supplier-bills", "client-payments", "payment-terms", "issues", "evidence", "documents", "cover":
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
