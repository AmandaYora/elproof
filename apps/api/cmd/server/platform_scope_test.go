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
		resetOwner = req{http.MethodPost, "/api/v1/tenants/4/reset-credential"}
		freeze     = req{http.MethodPost, "/api/v1/tenants/4/toggle-suspension"}
		activate   = req{http.MethodPost, "/api/v1/tenants/4/activate-subscription"}
		createPlan = req{http.MethodPost, "/api/v1/plans"}
		mapPlans   = req{http.MethodPut, "/api/v1/app-plans"}
		trialCaps  = req{http.MethodPut, "/api/v1/trial-limits"}
		newAdmin   = req{http.MethodPost, "/api/v1/platform-admins"}
		gateway    = req{http.MethodPut, "/api/v1/payment/gateway-config"}
		appSecret  = req{http.MethodPost, "/api/v1/payment/apps/app_1/reset-secret"}
		readAny    = req{http.MethodGet, "/api/v1/payment/gateway-config"}
	)
	matrix := map[string]map[req]bool{
		roleSuperAdmin: {register: true, editTenant: true, resetOwner: true, freeze: true, activate: true, createPlan: true, mapPlans: true, trialCaps: true, newAdmin: true, gateway: true, appSecret: true, readAny: true},
		roleBilling:    {register: false, editTenant: false, resetOwner: false, freeze: false, activate: true, createPlan: true, mapPlans: true, trialCaps: true, newAdmin: false, gateway: false, appSecret: false, readAny: true},
		roleSupport:    {register: true, editTenant: true, resetOwner: true, freeze: false, activate: false, createPlan: false, mapPlans: false, trialCaps: false, newAdmin: false, gateway: false, appSecret: false, readAny: true},
		roleReadOnly:   {register: false, editTenant: false, resetOwner: false, freeze: false, activate: false, createPlan: false, mapPlans: false, trialCaps: false, newAdmin: false, gateway: false, appSecret: false, readAny: true},
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
