package main

import (
	"net/http"
	"testing"

	"elproof/internal/shared/middleware"
)

func TestPlatformAdminAllowed(t *testing.T) {
	type req struct{ method, path string }
	var (
		register   = req{http.MethodPost, "/api/v1/tenants"}
		editTenant = req{http.MethodPatch, "/api/v1/tenants/4"}
		resetOwner = req{http.MethodPost, "/api/v1/tenants/4/owner-invite"}
		freeze     = req{http.MethodPost, "/api/v1/tenants/4/toggle-suspension"}
		activate   = req{http.MethodPost, "/api/v1/tenants/4/activate-subscription"}
		createPlan = req{http.MethodPost, "/api/v1/plans"}
		editPlan   = req{http.MethodPatch, "/api/v1/plans/3"}
		trialCaps  = req{http.MethodPut, "/api/v1/trial-limits"}
		newVoucher = req{http.MethodPost, "/api/v1/vouchers"}
		editVouch  = req{http.MethodPatch, "/api/v1/vouchers/3"}
		newAdmin   = req{http.MethodPost, "/api/v1/platform-admins"}
		gateway    = req{http.MethodPut, "/api/v1/payment/gateway-config"}
		appSecret  = req{http.MethodPost, "/api/v1/payment/apps/app_1/reset-secret"}
		readAny    = req{http.MethodGet, "/api/v1/payment/gateway-config"}

		overview    = req{http.MethodGet, "/api/v1/console/tenants/4/overview"}
		notesRead   = req{http.MethodGet, "/api/v1/console/tenants/4/notes"}
		openReqs    = req{http.MethodGet, "/api/v1/console/data-requests"}
		policyRead  = req{http.MethodGet, "/api/v1/console/subscription-policy"}
		export      = req{http.MethodGet, "/api/v1/tenants/export"}
		noteWrite   = req{http.MethodPost, "/api/v1/console/tenants/4/notes"}
		reqWrite    = req{http.MethodPost, "/api/v1/console/tenants/4/data-requests"}
		reqComplete = req{http.MethodPost, "/api/v1/console/data-requests/9/complete"}
		impersonate = req{http.MethodPost, "/api/v1/console/tenants/4/impersonate"}
		endSession  = req{http.MethodPost, "/api/v1/console/support-sessions/3/end"}
		announce    = req{http.MethodPost, "/api/v1/console/announcements"}
		policyWrite = req{http.MethodPut, "/api/v1/console/subscription-policy"}
		slide       = req{http.MethodPut, "/api/v1/console/login-slides/1"}
		trends      = req{http.MethodGet, "/api/v1/platform/trends"}
		names       = req{http.MethodGet, "/api/v1/tenants/names"}
	)
	matrix := map[string]map[req]bool{
		roleSuperAdmin: {newVoucher: true, editVouch: true, slide: true, trends: true, names: true, export: true, overview: true, notesRead: true, openReqs: true, policyRead: true, noteWrite: true, reqWrite: true, reqComplete: true, impersonate: true, endSession: true, announce: true, policyWrite: true, register: true, editTenant: true, resetOwner: true, freeze: true, activate: true, createPlan: true, editPlan: true, trialCaps: true, newAdmin: true, gateway: true, appSecret: true, readAny: true},
		roleBilling:    {newVoucher: true, editVouch: true, slide: false, trends: true, names: true, export: true, overview: true, notesRead: true, openReqs: true, policyRead: true, noteWrite: false, reqWrite: false, reqComplete: false, impersonate: false, endSession: false, announce: false, policyWrite: false, register: false, editTenant: false, resetOwner: false, freeze: false, activate: true, createPlan: true, editPlan: true, trialCaps: true, newAdmin: false, gateway: false, appSecret: false, readAny: true},
		roleSupport:    {newVoucher: false, editVouch: false, slide: false, trends: true, names: true, export: true, overview: true, notesRead: true, openReqs: true, policyRead: true, noteWrite: true, reqWrite: true, reqComplete: true, impersonate: true, endSession: true, announce: false, policyWrite: false, register: true, editTenant: true, resetOwner: true, freeze: false, activate: false, createPlan: false, editPlan: false, trialCaps: false, newAdmin: false, gateway: false, appSecret: false, readAny: true},
		roleReadOnly:   {newVoucher: false, editVouch: false, slide: false, trends: true, names: true, export: false, overview: false, notesRead: false, openReqs: false, policyRead: true, noteWrite: false, reqWrite: false, reqComplete: false, impersonate: false, endSession: false, announce: false, policyWrite: false, register: false, editTenant: false, resetOwner: false, freeze: false, activate: false, createPlan: false, editPlan: false, trialCaps: false, newAdmin: false, gateway: false, appSecret: false, readAny: true},
		"":             {register: false, freeze: false, readAny: true},
	}
	for role, cases := range matrix {
		for r, want := range cases {
			got, _ := platformAdminAllowed(&middleware.Claims{PrincipalType: "platform_admin", Role: role}, r.method, r.path)
			if got != want {
				t.Errorf("role %q %s %s: got %v, want %v", role, r.method, r.path, got, want)
			}
		}
	}
}

func TestPlatformAdminPolicyRequiresMFA(t *testing.T) {
	noMFA := &middleware.Claims{PrincipalType: "platform_admin", Role: roleSuperAdmin}
	for _, c := range []struct {
		method, path string
		want         bool
	}{
		{http.MethodGet, "/api/v1/auth/me", true},
		{http.MethodPost, "/api/v1/auth/mfa/setup", true},
		{http.MethodPost, "/api/v1/auth/mfa/enable", true},
		{http.MethodGet, "/api/v1/tenants", false},
		{http.MethodPost, "/api/v1/tenants", false},
	} {
		got, code, _ := platformAdminPolicy(noMFA, c.method, c.path)
		if got != c.want || (!got && code != "mfa_required") {
			t.Errorf("no-MFA %s %s: got (%v,%s), want %v", c.method, c.path, got, code, c.want)
		}
	}
	for _, role := range []string{roleReadOnly, roleSupport, roleBilling} {
		c := &middleware.Claims{PrincipalType: "platform_admin", Role: role}
		for _, p := range []string{"/api/v1/auth/mfa/setup", "/api/v1/auth/mfa/enable"} {
			if ok, _, _ := platformAdminPolicy(c, http.MethodPost, p); !ok {
				t.Errorf("%s must be able to call %s to enrol", role, p)
			}
		}
	}
	withMFA := &middleware.Claims{PrincipalType: "platform_admin", Role: roleReadOnly, MFA: true}
	if ok, _, _ := platformAdminPolicy(withMFA, http.MethodGet, "/api/v1/tenants"); !ok {
		t.Error("MFA session may read")
	}
	if ok, code, _ := platformAdminPolicy(withMFA, http.MethodPost, "/api/v1/tenants"); ok || code != "forbidden_role" {
		t.Errorf("read-only write must be forbidden_role, got %v %s", ok, code)
	}
}
