package application

import (
	"testing"

	"elproof/internal/shared/middleware"
)

func TestActionLabel(t *testing.T) {
	cases := map[string]string{
		"/api/v1/tenants/12/toggle-suspension":               "POST /tenants/{id}/toggle-suspension",
		"/api/v1/payment/apps/app_02ef90c52704/reset-secret": "POST /payment/apps/{id}/reset-secret",
		"/api/v1/plans": "POST /plans",
	}
	for path, want := range cases {
		if got := ActionLabel("POST", path); got != want {
			t.Errorf("%s: got %q, want %q", path, got, want)
		}
	}
}

func TestTenantOfAdminActions(t *testing.T) {
	cases := []struct {
		name string
		rec  middleware.AuditRecord
		want int64 // 0: none
	}{
		{"admin on tenant", middleware.AuditRecord{ActorType: "platform_admin", Path: "/api/v1/tenants/12/toggle-suspension"}, 12},
		{"admin on console tenant", middleware.AuditRecord{ActorType: "platform_admin", Path: "/api/v1/console/tenants/7/impersonate"}, 7},
		{"admin on a list", middleware.AuditRecord{ActorType: "platform_admin", Path: "/api/v1/tenants"}, 0},
		{"admin elsewhere", middleware.AuditRecord{ActorType: "platform_admin", Path: "/api/v1/plans/3"}, 0},
		{"admin export", middleware.AuditRecord{ActorType: "platform_admin", Path: "/api/v1/tenants/export"}, 0},
		{"staff keeps own tenant", middleware.AuditRecord{ActorType: "staff", TenantID: "4", Path: "/api/v1/tenants/me/logo"}, 4},
		{"staff path never names another tenant", middleware.AuditRecord{ActorType: "staff", Path: "/api/v1/tenants/9/logo"}, 0},
	}
	for _, c := range cases {
		got := tenantOf(c.rec)
		switch {
		case c.want == 0 && got != nil, c.want != 0 && (got == nil || *got != c.want):
			t.Errorf("%s: got %v, want %d", c.name, got, c.want)
		}
	}
}
