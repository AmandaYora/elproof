package main

import (
	"net/http"
	"strings"

	"elproof/internal/shared/middleware"
)

// Platform Console roles (platform/domain.PlatformAdminRole values).
const (
	roleSuperAdmin = "Super Admin"
	roleBilling    = "Billing"
	roleSupport    = "Support"
	roleReadOnly   = "Read-only"
)

// platformAdminAllowed is the least-privilege policy for ElProof's own team
// (FLOW-IDEAL.md §1.3), enforced once here for every module's routes:
//
//   - every role may read (GET/HEAD);
//   - Billing may manage plans, plan↔App mapping, trial caps and manual
//     activation;
//   - Support may register/edit tenants and reset an Owner's credential;
//   - only Super Admin may freeze tenants, manage admins, change the payment
//     gateway or manage external Apps;
//   - Read-only (and any unknown role) may not change anything.
func platformAdminAllowed(claims *middleware.Claims, method, path string) (bool, string) {
	if method == http.MethodGet || method == http.MethodHead {
		// Read-only sees the summaries and reports, not a workspace's internal
		// back office (notes, members, consent, data requests, support sessions).
		if claims.Role != roleSuperAdmin && claims.Role != roleBilling && claims.Role != roleSupport && consoleInternalRead(path) {
			return false, "Peran Anda (" + roleLabel(claims.Role) + ") tidak dapat melihat data internal tenant"
		}
		return true, ""
	}
	if claims.Role == roleSuperAdmin {
		return true, ""
	}
	return platformRoleAllows(claims, method, path)
}

// platformAdminPolicy is what cmd/server enforces for every platform_admin
// request: two-factor first (mandatory for the Platform Console, decision
// K14) — until an admin has a TOTP-verified session, only enrolment is
// reachable — then the role policy above.
func platformAdminPolicy(claims *middleware.Claims, method, path string) (bool, string, string) {
	// Every admin manages their own two-factor, whatever their role.
	if mfaEnrolmentRoute(method, path) {
		return true, "", ""
	}
	if !claims.MFA {
		return false, "mfa_required", "Aktifkan verifikasi dua langkah terlebih dahulu untuk menggunakan Platform Console"
	}
	if ok, reason := platformAdminAllowed(claims, method, path); !ok {
		return false, "forbidden_role", reason
	}
	return true, "", ""
}

func mfaEnrolmentRoute(method, path string) bool {
	p := strings.TrimSuffix(path, "/")
	return (method == http.MethodGet && p == "/api/v1/auth/me") ||
		(method == http.MethodPost && (p == "/api/v1/auth/mfa/setup" || p == "/api/v1/auth/mfa/enable"))
}

// consoleInternalRead: the Console endpoints that expose one workspace's
// internal back office. Announcements and the subscription policy are
// platform-wide settings anyone on the team may read.
func consoleInternalRead(path string) bool {
	s := strings.Split(strings.Trim(path, "/"), "/")
	if len(s) == 4 && s[0] == "api" && s[1] == "v1" && s[2] == "tenants" && s[3] == "export" {
		return true // the full customer list as a file
	}
	if len(s) < 4 || s[0] != "api" || s[1] != "v1" || s[2] != "console" {
		return false
	}
	return s[3] == "tenants" || s[3] == "data-requests" || s[3] == "support-sessions"
}

func platformRoleAllows(claims *middleware.Claims, method, path string) (bool, string) {
	action := platformWriteAction(method, path)
	switch claims.Role {
	case roleBilling:
		if action == "plans" || action == "activate-subscription" {
			return true, ""
		}
	case roleSupport:
		if action == "tenant-manage" || action == "support-access" {
			return true, ""
		}
	}
	return false, "Peran Anda (" + roleLabel(claims.Role) + ") tidak memiliki izin untuk aksi ini"
}

// platformWriteAction classifies a write request into the permission it
// needs. Anything unrecognised is "super-admin" (deny by default).
func platformWriteAction(method, path string) string {
	s := strings.Split(strings.Trim(path, "/"), "/")
	if len(s) < 3 || s[0] != "api" || s[1] != "v1" {
		return "super-admin"
	}
	s = s[2:]
	switch {
	case s[0] == "plans", s[0] == "trial-limits":
		return "plans" // pricing & trial caps are Billing's job
	case s[0] == "tenants" && len(s) == 1 && method == http.MethodPost:
		return "tenant-manage" // register a tenant
	case s[0] == "tenants" && len(s) == 2 && isID(s[1]) && method == http.MethodPatch:
		return "tenant-manage" // edit a tenant
	case s[0] == "console" && len(s) == 4 && s[1] == "tenants" && isID(s[2]):
		switch s[3] {
		case "notes", "data-requests":
			return "tenant-manage" // Support keeps notes and logs data requests
		case "impersonate":
			return "support-access"
		}
	case s[0] == "console" && len(s) == 4 && s[1] == "data-requests" && isID(s[2]) && s[3] == "complete":
		return "tenant-manage"
	case s[0] == "console" && len(s) == 4 && s[1] == "support-sessions" && isID(s[2]) && s[3] == "end":
		return "support-access"
	case s[0] == "tenants" && len(s) == 3 && isID(s[1]):
		switch s[2] {
		case "logo", "owner-invite":
			return "tenant-manage"
		case "activate-subscription":
			return "activate-subscription"
		}
	}
	return "super-admin"
}

func roleLabel(role string) string {
	if role == "" {
		return "tanpa peran"
	}
	return role
}
