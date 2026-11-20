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
		"/api/v1/projects/12/events",
		"/api/v1/projects/12/suppliers",
		// Tahap 3: "Layanan › Paket & rincian" (amounts redacted by PortalPolicy.Line).
		"/api/v1/projects/12/lines",
		"/api/v1/projects/12/supplier-payments",
		// Paket WO Tahap 4: the bills of the vendors the client pays directly.
		"/api/v1/projects/12/supplier-bills",
		"/api/v1/projects/12/client-payments",
		"/api/v1/projects/12/payment-terms",
		"/api/v1/tenants/me/contact",
		"/api/v1/projects/12/issues",
		"/api/v1/projects/12/evidence",
		"/api/v1/projects/12/evidence/7/file",
		"/api/v1/projects/12/evidence/7/link",
		"/api/v1/projects/12/documents",
		"/api/v1/projects/12/cover",
		"/api/v1/venues/3/attachment",
	}
	for _, p := range allowed {
		if !clientPortalAllowed(http.MethodGet, p) {
			t.Errorf("GET %s should be allowed for a client", p)
		}
	}

	denied := []string{
		// E6: these list every vendor and category name of the workspace; the
		// portal reads names inside /projects/{id}/suppliers and /issues instead.
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
		"/api/v1/projects/12/activity",
		// Routes of the old vendor model are gone and stay refused.
		"/api/v1/projects/12/payments",
		"/api/v1/projects/12/venue-payments",
		"/api/v1/projects/12/vendors",
		"/api/v1/projects/12/venue",
		// Tahap 5: the staff overview carries margin, vendor costs and internal actions.
		"/api/v1/projects/12/overview",
		// Tahap 5: "Kesepakatan awal" is the WO's frozen quotation (agreed costs, notes).
		"/api/v1/projects/12/agreement",
		"/api/v1/projects/from-quotation/5",
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
		"/api/v1/projects/12/closing-checklist",
		"/api/v1/projects/12/transition",
		"/api/v1/subscriptions",
		"/api/v1/dashboard",
		"/api/v1/dashboard/mine",
		// Keuangan: the WO's money across projects.
		"/api/v1/finance/summary",
		"/api/v1/finance/receivables",
		"/api/v1/finance/payables",
		"/api/v1/finance/verifications",
		// Malformed ids and deeper paths.
		"/api/v1/projects/abc/milestones",
		"/api/v1/projects/0",
		"/api/v1/projects/12/evidence/x/file",
		"/api/v1/projects/12/evidence/x/link",
		"/api/v1/projects/12/evidence/7",
		"/api/v1/projects/12/suppliers/4/milestones",
		"/api/v1/projects/12/suppliers/4",
		"/api/v1/projects/12/events/3",
		// The Rincian is read whole; no line, preview or reorder path is a portal read.
		"/api/v1/projects/12/lines/3",
		"/api/v1/projects/12/lines/preview",
		"/api/v1/projects/12/lines/reorder",
		"/api/v1/projects/12/lines/3/restore",
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
			"/api/v1/projects/12", "/api/v1/projects/12/evidence/7",
			// ADR-0047: recording a payment, the cover and editing a step stay staff writes.
			"/api/v1/projects/12/client-payments", "/api/v1/projects/12/client-payments/3", "/api/v1/projects/12/cover",
			"/api/v1/projects/12/milestones", "/api/v1/projects/12/milestones/3", "/api/v1/projects/12/evidence",
			"/api/v1/projects/12/supplier-payments", "/api/v1/projects/12/suppliers", "/api/v1/projects/12/suppliers/4",
			"/api/v1/projects/12/events", "/api/v1/projects/12/events/3", "/api/v1/projects/12/events/3/set-main",
			// Tahap 3: every Rincian write (and the preview) is staff only.
			"/api/v1/projects/12/lines", "/api/v1/projects/12/lines/preview", "/api/v1/projects/12/lines/reorder",
			"/api/v1/projects/12/lines/3", "/api/v1/projects/12/lines/3/restore",
			// Tahap 4: the bills are the WO's to keep, checking a proof is the WO's.
			"/api/v1/projects/12/supplier-bills", "/api/v1/projects/12/supplier-bills/terms", "/api/v1/projects/12/supplier-bills/terms/3",
			"/api/v1/projects/12/supplier-bills/terms/3/restore", "/api/v1/projects/12/supplier-payments/3/verify",
			"/api/v1/projects/12/supplier-payments/3/reject"} {
			if clientPortalAllowed(m, p) {
				t.Errorf("%s %s must be denied for a client", m, p)
			}
		}
	}
}

// ADR-0047: the portal is read-only except for exactly five writes.
func TestClientPortalWrites(t *testing.T) {
	allowed := [][2]string{
		{http.MethodPost, "/api/v1/projects/12/uploads"},
		{http.MethodPut, "/api/v1/projects/12/uploads/eyJ0.sig"},
		{http.MethodPost, "/api/v1/projects/12/client-payments/submissions"},
		{http.MethodPost, "/api/v1/projects/12/supplier-bills/submissions"},
		{http.MethodPost, "/api/v1/projects/12/milestones/3/complete"},
	}
	for _, c := range allowed {
		if !clientPortalAllowed(c[0], c[1]) {
			t.Errorf("%s %s should be allowed for a client", c[0], c[1])
		}
	}
	denied := [][2]string{
		// Right path, wrong method.
		{http.MethodPut, "/api/v1/projects/12/uploads"},
		{http.MethodPost, "/api/v1/projects/12/uploads/eyJ0.sig"},
		{http.MethodDelete, "/api/v1/projects/12/uploads/eyJ0.sig"},
		{http.MethodPut, "/api/v1/projects/12/client-payments/submissions"},
		{http.MethodPatch, "/api/v1/projects/12/milestones/3/complete"},
		{http.MethodDelete, "/api/v1/projects/12/milestones/3/complete"},
		// GET on a write-only route is not a route either.
		{http.MethodGet, "/api/v1/projects/12/uploads"},
		{http.MethodGet, "/api/v1/projects/12/client-payments/submissions"},
		{http.MethodGet, "/api/v1/projects/12/milestones/3/complete"},
		// Malformed ids and deeper or shallower paths.
		{http.MethodPost, "/api/v1/projects/abc/uploads"},
		{http.MethodPost, "/api/v1/projects/0/client-payments/submissions"},
		{http.MethodPost, "/api/v1/projects/12/milestones/x/complete"},
		{http.MethodPost, "/api/v1/projects/12/milestones/3/complete/again"},
		{http.MethodPost, "/api/v1/projects/12/uploads/a/b"},
		{http.MethodPut, "/api/v1/projects/12/uploads/"},
		{http.MethodPost, "/api/v1/projects/me/uploads"},
		// "Jadikan proyek" and its down-payment upload are staff only (Tahap 5).
		{http.MethodPost, "/api/v1/projects/from-quotation/5"},
		{http.MethodPost, "/api/v1/projects/from-quotation/5/uploads"},
		{http.MethodPut, "/api/v1/projects/from-quotation/5/uploads/eyJ0.sig"},
		{http.MethodPost, "/api/v2/projects/12/uploads"},
		{http.MethodPost, "/api/v1/projects/12/client-payments/submission"},
		{http.MethodPut, "/api/v1/projects/12/supplier-bills/submissions"},
		{http.MethodGet, "/api/v1/projects/12/supplier-bills/submissions"},
		{http.MethodPost, "/api/v1/projects/0/supplier-bills/submissions"},
		{http.MethodPost, "/api/v1/projects/12/supplier-bills/submissions/3"},
		// Outside projects.
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodPost, "/api/v1/finance/verifications"},
		{http.MethodPut, "/api/v1/tenants/me/logo"},
	}
	for _, c := range denied {
		if clientPortalAllowed(c[0], c[1]) {
			t.Errorf("%s %s must be denied for a client", c[0], c[1])
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
		{"client", "/api/v1/projects/12/activity", http.StatusForbidden},
		{"client", "/api/v1/projects/12/suppliers", http.StatusNoContent},
		{"client", "/api/v1/projects/12/lines", http.StatusNoContent},
		{"staff", "/api/v1/projects/12/activity", http.StatusNoContent},
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
