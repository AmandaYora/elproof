package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"elproof/internal/shared/middleware"
)

func TestClientPortalAllowed(t *testing.T) {
	allowed := []string{
		"/api/v1/auth/me",
		"/api/v1/tenants/me/branding",
		"/api/v1/tenants/me/logo",
		"/api/v1/projects/me",
		"/api/v1/projects/me/home",
		"/api/v1/projects/12",
		"/api/v1/projects/12/milestones",
		"/api/v1/projects/12/vendors",
		"/api/v1/projects/12/client-payments",
		"/api/v1/projects/12/payment-terms",
		"/api/v1/tenants/me/contact",
		"/api/v1/projects/12/issues",
		"/api/v1/projects/12/evidence",
		"/api/v1/projects/12/evidence/7/file",
		"/api/v1/projects/12/evidence/7/link",
		"/api/v1/projects/12/documents",
		"/api/v1/projects/12/venue",
		"/api/v1/venues/3/attachment",
	}
	for _, p := range allowed {
		if !clientPortalAllowed(http.MethodGet, p) {
			t.Errorf("GET %s should be allowed for a client", p)
		}
	}

	denied := []string{
		// E6: these list every vendor and category name of the workspace; the
		// portal reads names inside /projects/{id}/vendors and /issues instead.
		// vendors/summary itself is no longer a route (Tahap 9); the path stays
		// refused here so that it can never come back open to a client.
		"/api/v1/vendors/summary",
		"/api/v1/vendor-categories",
		"/api/v1/vendor-categories?all=true",
		// Only the exact self-service paths exist under projects/me.
		"/api/v1/projects/me/other",
		"/api/v1/projects/me/home/extra",
		// Verifying a payment is a staff write; GET on it is not a route either.
		"/api/v1/projects/12/client-payments/3/verify",
		// Internal finance and audit data of the client's own project.
		"/api/v1/projects/12/payments",
		"/api/v1/projects/12/venue-payments",
		"/api/v1/projects/12/activity",
		// Tahap 5: the staff overview carries margin, vendor costs and internal actions.
		"/api/v1/projects/12/overview",
		// Other tenants' / other couples' data and staff-only directories.
		"/api/v1/projects",
		"/api/v1/clients",
		"/api/v1/clients?projectId=12",
		"/api/v1/staff/summary",
		"/api/v1/vendors",
		"/api/v1/vendors/5",
		"/api/v1/venues/3",
		"/api/v1/tenants/me",
		"/api/v1/tenants/me/business-details",
		"/api/v1/tenants/me/brand-color",
		"/api/v1/projects/12/closing-checklist",
		"/api/v1/projects/12/transition",
		"/api/v1/subscriptions",
		"/api/v1/dashboard",
		"/api/v1/dashboard/mine",
		// Malformed ids and deeper paths.
		"/api/v1/projects/abc/milestones",
		"/api/v1/projects/0",
		"/api/v1/projects/12/evidence/x/file",
		"/api/v1/projects/12/evidence/x/link",
		"/api/v1/projects/12/evidence/7",
		"/api/v1/projects/12/vendors/4/milestones",
		"/api/v2/projects/me",
		"/other",
	}
	for _, p := range denied {
		if clientPortalAllowed(http.MethodGet, p) {
			t.Errorf("GET %s must be denied for a client", p)
		}
	}

	for _, m := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		for _, p := range []string{"/api/v1/projects/12/issues", "/api/v1/projects/12/client-payments/3/verify", "/api/v1/projects/me/home",
			// The explicit-state writes (archive a project, show a document) are staff writes too.
			"/api/v1/projects/12", "/api/v1/projects/12/evidence/7"} {
			if clientPortalAllowed(m, p) {
				t.Errorf("%s %s must be denied for a client", m, p)
			}
		}
	}
}

func TestRestrictPrincipalOnlyAffectsClients(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := middleware.RestrictPrincipal("client", clientPortalAllowed)(next)

	cases := []struct {
		principal string
		path      string
		want      int
	}{
		{"client", "/api/v1/projects/12/payments", http.StatusForbidden},
		{"client", "/api/v1/projects/12/vendors", http.StatusNoContent},
		{"staff", "/api/v1/projects/12/payments", http.StatusNoContent},
		{"platform_admin", "/api/v1/tenants", http.StatusNoContent},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), &middleware.Claims{PrincipalType: c.principal}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s GET %s: got %d, want %d", c.principal, c.path, rec.Code, c.want)
		}
	}
}
