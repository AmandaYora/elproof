//go:build integration

// Tenant isolation suite.
//
// What it guards: a signed-in person of workspace A can never read or change
// anything that belongs to workspace B by guessing or reusing an id, whatever
// the route, and a client of B sees only what B's portal gives that client.
// It is a black-box test of the real server binary against a real MySQL (and
// real S3-compatible storage, for evidence files), so it exercises routing,
// the auth middleware chain, the role gates and the SQL together.
//
// How to run (a local MySQL and, for the evidence-file routes, a MinIO):
//
//	TEST_MYSQL_ADMIN_DSN='root:@tcp(127.0.0.1:3399)/' \
//	TEST_S3_ENDPOINT=127.0.0.1:9099 TEST_S3_ACCESS_KEY=... TEST_S3_SECRET_KEY=... \
//	go test -tags integration ./tests/isolation/ -v
//
// Without TEST_MYSQL_ADMIN_DSN the suite skips. Each run builds the server,
// creates and drops its own database, and needs no other setup.
//
// Shape of the checks:
//   - attack matrix: A's token against B's ids on every id-bearing route. The
//     answer must be 403/404 (never 2xx or 5xx), must not be the router's
//     "no such endpoint" answer (that would make a mistyped route pass
//     vacuously), and must not contain any of B's data. Where safe, B gets a
//     control request on the same route and must not be turned away.
//   - mixed ids: A's own project combined with B's sub-resource ids.
//   - foreign references: A writes bodies that name B's ids; each is paired
//     with a control using A's own ids that must succeed.
//   - lists and dashboards: A never sees B's rows or money.
//   - client principal: B's client reaches only its own project and only the
//     documents shared with it.
//   - afterwards B's data is intact.
package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

var routesCovered int

// leaks are strings that exist only in B's data; none may reach A.
func (w *world) leaks() []string {
	return []string{w.b.marker, fmt.Sprint(w.bContract), "4444444", "777777", "VREF-", "REF-RAHASIA"}
}

func (w *world) checkNoLeak(t *testing.T, what string, r reply) {
	t.Helper()
	body := string(r.body)
	for _, l := range w.leaks() {
		if strings.Contains(body, l) {
			t.Errorf("%s: response contains B's data (%q): %s", what, l, r)
		}
	}
}

// vacuous reports the router's own "no such endpoint" answer: a mistyped route
// would look like perfect isolation, so it must be told apart.
func vacuous(r reply) bool {
	msg, _ := r.dig("message").(string)
	return strings.Contains(msg, "Endpoint tidak ditemukan") || strings.Contains(msg, "Metode HTTP tidak diizinkan") || strings.Contains(msg, "Metode tidak diizinkan")
}

type attack struct {
	name   string
	method string
	path   string
	body   any
	// control: run the same request as B (against controlPath when set) and
	// require that B is not turned away. Left off for requests that would
	// disturb the fixtures (deletes, toggles, invitations).
	control     bool
	controlPath string
	// platform marks routes that belong to ElProof's own console: a workspace
	// token must get 403, and B has no control.
	platform bool
}

func TestTenantIsolation(t *testing.T) {
	s := startStack(t)
	w := buildWorld(t, s)

	// Order matters: read-only checks first, then the attacks (whose controls
	// may change B's data), then the integrity check.
	t.Run("client principal", func(t *testing.T) { w.clientIsolation(t, s) })
	t.Run("client portal (E4/E5/E6)", func(t *testing.T) { w.portalScenarios(t, s) })
	t.Run("lists and dashboards", func(t *testing.T) { w.lists(t, s) })
	t.Run("attack matrix", func(t *testing.T) { w.attackMatrix(t, s) })
	t.Run("mixed ids", func(t *testing.T) { w.mixedIDs(t, s) })
	t.Run("foreign references", func(t *testing.T) { w.foreignReferences(t, s) })
	t.Run("signed uploads, atomic payments and projects (E3/E7)", func(t *testing.T) { w.tahap4Scenarios(t, s) })
	t.Run("workspace B is intact", func(t *testing.T) { w.intact(t, s) })
	t.Logf("routes and requests covered: %d", routesCovered)
}

// ---------------------------------------------------------------------------

func (w *world) attackMatrix(t *testing.T, s *stack) {
	P, P2, SAC := w.Project, w.Project2, w.Sacrifice
	proj := func(id int64, rest string) string { return fmt.Sprintf("/api/v1/projects/%d%s", id, rest) }
	b := w.b

	projectBody := map[string]any{
		"name": "Proyek B " + b.marker, "brideName": "Bride " + b.marker, "groomName": "Groom " + b.marker,
		"eventDate": day(120), "venue": "Gedung " + b.marker, "prepStartDate": day(1), "packageName": "Paket " + b.marker,
		"contractValue": w.bContract, "status": "Preparation", "picStaffId": b.staffID, "description": "Catatan " + b.marker,
	}
	// Update controls resend each row's original values, so the integrity
	// check at the end can still tell attacks from controls.
	payment := map[string]any{"projectVendorId": w.Engagement, "type": "DP", "amount": 1000000, "paymentDate": day(-2), "method": "Transfer", "referenceNumber": "VREF-" + b.marker, "notes": "Rahasia " + b.marker}
	cpay := map[string]any{"type": "DP", "amount": 4444444, "paymentDate": day(-2), "method": "Transfer", "referenceNumber": "REF-" + b.marker, "notes": "Rahasia " + b.marker}
	vpay := map[string]any{"type": "DP", "amount": 777777, "paymentDate": day(-2), "method": "Transfer", "referenceNumber": "VNREF-" + b.marker, "notes": "Rahasia " + b.marker}
	issue := map[string]any{"projectVendorId": w.Engagement, "title": "Kendala " + b.marker, "description": "d", "impact": "Medium", "foundDate": day(-1), "resolutionPlan": "r", "picStaffId": b.staffID, "targetResolutionDate": day(10)}
	engagement := map[string]any{"vendorId": w.Vendor, "categoryId": w.Category, "scope": "Tambahan " + b.marker, "contractValue": 100000, "pricingTier": "Akad", "engagementStatus": "Planned", "eventDate": day(120), "picStaffId": b.staffID}
	engagementOrig := map[string]any{"vendorId": w.Vendor, "categoryId": w.Category, "scope": "Dekorasi " + b.marker, "contractValue": 5000000, "pricingTier": "Akad", "engagementStatus": "Booked", "eventDate": day(120), "dpAmount": 1000000, "dueDate": day(60), "picStaffId": b.staffID, "notes": "Rahasia " + b.marker}
	vmilestone := map[string]any{"name": "Baru " + b.marker, "description": "d", "targetDate": day(20), "picStaffId": b.staffID}
	upload := map[string]any{"name": "Baru " + b.marker, "type": "Document", "fileName": "x.pdf", "mimeType": "application/pdf", "base64Data": pdf, "documentDate": day(-1), "relatedKind": "general"}
	vendorBodyB := vendorBody(s, b, "Vendor B "+b.marker, w.Category)
	venueBodyB := venueBody(b, "Venue B "+b.marker)
	file := map[string]any{"fileName": "lampiran.pdf", "mimeType": "application/pdf", "base64Data": pdf}
	roleBody := map[string]any{"kind": "staff", "name": "Peran " + b.marker, "description": "Rahasia", "projectScope": "all", "permissions": []string{"project.edit"}}

	attacks := []attack{
		// ---- projects and everything inside them ----
		{name: "project get", method: "GET", path: proj(P, ""), control: true},
		{name: "project update", method: "PATCH", path: proj(P, ""), body: projectBody, control: true},
		{name: "project delete", method: "DELETE", path: proj(P, "")},
		{name: "project cancel", method: "POST", path: proj(P, "/cancel"), body: map[string]any{}, control: true, controlPath: proj(SAC, "/cancel")},
		{name: "project transition", method: "POST", path: proj(P, "/transition"), body: map[string]any{"action": "no_such_action"}, control: true},
		{name: "project toggle-archive", method: "POST", path: proj(P, "/toggle-archive"), control: true, controlPath: proj(SAC, "/toggle-archive")},
		{name: "project duplicate", method: "POST", path: proj(P, "/duplicate"), body: map[string]any{"name": "Salinan " + b.marker, "eventDate": day(200)}, control: true},
		{name: "payment-terms get", method: "GET", path: proj(P, "/payment-terms"), control: true},
		{name: "payment-terms put", method: "PUT", path: proj(P, "/payment-terms"), body: map[string]any{"terms": []map[string]any{{"label": "Termin " + b.marker, "amount": 1000000, "dueDate": day(30)}}}, control: true},
		{name: "closing-checklist", method: "GET", path: proj(P, "/closing-checklist"), control: true},
		{name: "milestones list", method: "GET", path: proj(P, "/milestones"), control: true},
		{name: "milestone create", method: "POST", path: proj(P, "/milestones"), body: map[string]any{"name": "M " + b.marker, "targetDate": day(15)}, control: true},
		{name: "milestone reorder", method: "PATCH", path: proj(P, "/milestones"), body: map[string]any{"orderedIds": []int64{w.ProjectMilestone}}, control: true},
		{name: "milestone update", method: "PATCH", path: proj(P, fmt.Sprintf("/milestones/%d", w.ProjectMilestone)), body: map[string]any{"status": "In Progress", "targetDate": day(15)}, control: true},
		{name: "project overview", method: "GET", path: proj(P, "/overview"), control: true},
		{name: "engagements list", method: "GET", path: proj(P, "/vendors"), control: true},
		// Tahap 5: soft delete and restore. No controls: a restore of something that is
		// not deleted is a 404 for its own workspace too.
		{name: "engagement delete", method: "DELETE", path: proj(P, fmt.Sprintf("/vendors/%d", w.Engagement))},
		{name: "engagement restore", method: "POST", path: proj(P, fmt.Sprintf("/vendors/%d/restore", w.Engagement))},
		{name: "vendor payment restore", method: "POST", path: proj(P, fmt.Sprintf("/payments/%d/restore", w.VendorPay))},
		{name: "client payment restore", method: "POST", path: proj(P, fmt.Sprintf("/client-payments/%d/restore", w.ClientPayment))},
		{name: "venue payment restore", method: "POST", path: proj(P, fmt.Sprintf("/venue-payments/%d/restore", w.VenuePay))},
		{name: "evidence delete", method: "DELETE", path: proj(P, fmt.Sprintf("/evidence/%d", w.EvidenceInternal))},
		{name: "client restore", method: "POST", path: fmt.Sprintf("/api/v1/clients/%d/restore", w.Client)},
		{name: "engagement create", method: "POST", path: proj(P, "/vendors"), body: engagement, control: true},
		{name: "engagement update", method: "PATCH", path: proj(P, fmt.Sprintf("/vendors/%d", w.Engagement)), body: engagementOrig, control: true},
		{name: "engagement cancel", method: "POST", path: proj(P, fmt.Sprintf("/vendors/%d/cancel", w.Engagement)), body: map[string]any{}},
		{name: "vendor milestones list", method: "GET", path: proj(P, fmt.Sprintf("/vendors/%d/milestones", w.Engagement)), control: true},
		{name: "vendor milestone create", method: "POST", path: proj(P, fmt.Sprintf("/vendors/%d/milestones", w.Engagement)), body: vmilestone, control: true},
		{name: "vendor milestone update", method: "PATCH", path: proj(P, fmt.Sprintf("/vendors/%d/milestones/%d", w.Engagement, w.VendorMilestone)), body: map[string]any{"status": "In Progress", "targetDate": day(30)}, control: true},
		{name: "vendor payments list", method: "GET", path: proj(P, "/payments"), control: true},
		{name: "vendor payment create", method: "POST", path: proj(P, "/payments"), body: payment, control: true},
		{name: "vendor payment update", method: "PATCH", path: proj(P, fmt.Sprintf("/payments/%d", w.VendorPay)), body: payment, control: true},
		{name: "vendor payment delete", method: "DELETE", path: proj(P, fmt.Sprintf("/payments/%d", w.VendorPay))},
		{name: "client payments list", method: "GET", path: proj(P, "/client-payments"), control: true},
		{name: "client payment create", method: "POST", path: proj(P, "/client-payments"), body: cpay, control: true},
		{name: "client payment update", method: "PATCH", path: proj(P, fmt.Sprintf("/client-payments/%d", w.ClientPayment)), body: cpay, control: true},
		{name: "client payment delete", method: "DELETE", path: proj(P, fmt.Sprintf("/client-payments/%d", w.ClientPayment))},
		{name: "venue payments list", method: "GET", path: proj(P, "/venue-payments"), control: true},
		{name: "venue payment create", method: "POST", path: proj(P, "/venue-payments"), body: vpay, control: true},
		{name: "venue payment update", method: "PATCH", path: proj(P, fmt.Sprintf("/venue-payments/%d", w.VenuePay)), body: vpay, control: true},
		{name: "venue payment delete", method: "DELETE", path: proj(P, fmt.Sprintf("/venue-payments/%d", w.VenuePay))},
		{name: "issues list", method: "GET", path: proj(P, "/issues"), control: true},
		{name: "issue create", method: "POST", path: proj(P, "/issues"), body: issue, control: true},
		{name: "issue update", method: "PATCH", path: proj(P, fmt.Sprintf("/issues/%d", w.Issue)), body: map[string]any{
			"projectVendorId": w.Engagement, "title": "Kendala " + b.marker, "description": "d", "impact": "Medium", "foundDate": day(-1), "status": "Open",
			"resolutionPlan": "r", "picStaffId": b.staffID, "targetResolutionDate": day(10)}, control: true},
		{name: "evidence list", method: "GET", path: proj(P, "/evidence"), control: true},
		{name: "evidence upload", method: "POST", path: proj(P, "/evidence"), body: upload, control: true},
		{name: "evidence file", method: "GET", path: proj(P, fmt.Sprintf("/evidence/%d/file", w.EvidenceInternal)), control: true},
		{name: "evidence link", method: "GET", path: proj(P, fmt.Sprintf("/evidence/%d/link", w.EvidenceInternal)), control: true},
		{name: "evidence link (download)", method: "GET", path: proj(P, fmt.Sprintf("/evidence/%d/link?disposition=attachment", w.EvidenceInternal)), control: true},
		{name: "evidence toggle-client-visible", method: "POST", path: proj(P, fmt.Sprintf("/evidence/%d/toggle-client-visible", w.EvidenceInternal))},
		{name: "upload create", method: "POST", path: proj(P, "/uploads"), body: map[string]any{"fileName": "a.pdf", "mimeType": "application/pdf", "size": 10}, control: true},
		{name: "upload put", method: "PUT", path: proj(P, "/uploads/abc.def"), control: true},
		{name: "documents", method: "GET", path: proj(P, "/documents"), control: true},
		{name: "activity", method: "GET", path: proj(P, "/activity"), control: true},
		{name: "activity filtered and paged", method: "GET", path: proj(P, "/activity?entityType=client_payment&entityId=1&page=1&limit=5"), control: true},
		{name: "venue", method: "GET", path: proj(P, "/venue"), control: true},
		{name: "second project get", method: "GET", path: proj(P2, ""), control: true},

		// ---- vendors, venues, categories ----
		{name: "vendor update", method: "PATCH", path: fmt.Sprintf("/api/v1/vendors/%d", w.Vendor), body: vendorBodyB, control: true},
		{name: "vendor toggle-active", method: "POST", path: fmt.Sprintf("/api/v1/vendors/%d/toggle-active", w.Vendor)},
		{name: "vendor project-history", method: "GET", path: fmt.Sprintf("/api/v1/vendors/%d/project-history", w.Vendor), control: true},
		{name: "vendor attachment put", method: "PUT", path: fmt.Sprintf("/api/v1/vendors/%d/attachment", w.Vendor), body: file, control: true},
		{name: "vendor attachment get", method: "GET", path: fmt.Sprintf("/api/v1/vendors/%d/attachment", w.Vendor)},
		{name: "vendor delete-impact", method: "GET", path: fmt.Sprintf("/api/v1/vendors/%d/delete-impact", w.Vendor), control: true},
		{name: "vendor delete", method: "DELETE", path: fmt.Sprintf("/api/v1/vendors/%d", w.Vendor)},
		{name: "venue get", method: "GET", path: fmt.Sprintf("/api/v1/venues/%d", w.Venue), control: true},
		{name: "venue update", method: "PATCH", path: fmt.Sprintf("/api/v1/venues/%d", w.Venue), body: venueBodyB, control: true},
		{name: "venue toggle-active", method: "POST", path: fmt.Sprintf("/api/v1/venues/%d/toggle-active", w.Venue)},
		{name: "venue attachment put", method: "PUT", path: fmt.Sprintf("/api/v1/venues/%d/attachment", w.Venue), body: file, control: true},
		{name: "venue attachment get", method: "GET", path: fmt.Sprintf("/api/v1/venues/%d/attachment", w.Venue)},
		{name: "venue delete-impact", method: "GET", path: fmt.Sprintf("/api/v1/venues/%d/delete-impact", w.Venue), control: true},
		{name: "venue delete", method: "DELETE", path: fmt.Sprintf("/api/v1/venues/%d", w.Venue)},
		{name: "category update", method: "PATCH", path: fmt.Sprintf("/api/v1/vendor-categories/%d", w.Category), body: map[string]any{"name": "Kategori " + b.marker, "description": "Rahasia " + b.marker}, control: true},
		{name: "category toggle-active", method: "POST", path: fmt.Sprintf("/api/v1/vendor-categories/%d/toggle-active", w.Category)},
		{name: "category delete", method: "DELETE", path: fmt.Sprintf("/api/v1/vendor-categories/%d", w.Category)},

		// ---- clients, staff, roles, templates ----
		{name: "client update", method: "PATCH", path: fmt.Sprintf("/api/v1/clients/%d", w.Client), body: map[string]any{"name": "Klien " + b.marker, "phone": "081298765432", "relationNote": "x", "role": "Bride"}, control: true},
		{name: "client delete", method: "DELETE", path: fmt.Sprintf("/api/v1/clients/%d", w.Client)},
		{name: "client toggle-active", method: "POST", path: fmt.Sprintf("/api/v1/clients/%d/toggle-active", w.Client)},
		{name: "client access-link", method: "POST", path: fmt.Sprintf("/api/v1/clients/%d/access-link", w.Client)},
		{name: "client replace-representative", method: "POST", path: fmt.Sprintf("/api/v1/clients/%d/replace-representative", w.Client), body: map[string]any{}},
		{name: "staff update", method: "PATCH", path: fmt.Sprintf("/api/v1/staff/%d", w.Staff2), body: map[string]any{"name": "Staf " + b.marker, "title": "Koordinator", "roleId": w.Role, "email": "staf@example.com", "phone": "081234567890"}, control: true},
		{name: "staff toggle-active", method: "POST", path: fmt.Sprintf("/api/v1/staff/%d/toggle-active", w.Staff2)},
		{name: "staff invite", method: "POST", path: fmt.Sprintf("/api/v1/staff/%d/invite", w.Staff2)},
		{name: "staff delete-impact", method: "GET", path: fmt.Sprintf("/api/v1/staff/%d/delete-impact", w.Staff2), control: true},
		{name: "staff delete", method: "DELETE", path: fmt.Sprintf("/api/v1/staff/%d", w.Staff2)},
		{name: "owner staff update", method: "PATCH", path: fmt.Sprintf("/api/v1/staff/%d", w.b.staffID), body: map[string]any{"name": "Dibajak", "title": "Owner", "roleId": w.Role, "email": "x@example.com", "phone": "0812"}},
		{name: "role get", method: "GET", path: fmt.Sprintf("/api/v1/access/roles/%d", w.Role), control: true},
		{name: "role update", method: "PATCH", path: fmt.Sprintf("/api/v1/access/roles/%d", w.Role), body: roleBody, control: true},
		{name: "role delete", method: "DELETE", path: fmt.Sprintf("/api/v1/access/roles/%d", w.Role)},
		{name: "template update", method: "PATCH", path: fmt.Sprintf("/api/v1/milestone-templates/%d", w.Template), body: map[string]any{"name": "Template " + b.marker, "daysBeforeEvent": 45}, control: true},
		{name: "template delete", method: "DELETE", path: fmt.Sprintf("/api/v1/milestone-templates/%d", w.Template)},

		// ---- billing: one workspace's ledger ----
		{name: "subscription transaction", method: "GET", path: fmt.Sprintf("/api/v1/subscription-transactions/%d", w.Tx), control: true},

		// ---- ElProof's own console: a workspace token is never enough ----
		{name: "tenant get", method: "GET", path: fmt.Sprintf("/api/v1/tenants/%d", w.b.tenant), platform: true},
		{name: "tenant update", method: "PATCH", path: fmt.Sprintf("/api/v1/tenants/%d", w.b.tenant), body: map[string]any{"businessName": "Dibajak"}, platform: true},
		{name: "tenant logo get", method: "GET", path: fmt.Sprintf("/api/v1/tenants/%d/logo", w.b.tenant), platform: true},
		{name: "tenant logo put", method: "PUT", path: fmt.Sprintf("/api/v1/tenants/%d/logo", w.b.tenant), body: file, platform: true},
		{name: "tenant toggle-suspension", method: "POST", path: fmt.Sprintf("/api/v1/tenants/%d/toggle-suspension", w.b.tenant), platform: true},
		{name: "tenant owner-invite", method: "POST", path: fmt.Sprintf("/api/v1/tenants/%d/owner-invite", w.b.tenant), platform: true},
		{name: "tenant activate-subscription", method: "POST", path: fmt.Sprintf("/api/v1/tenants/%d/activate-subscription", w.b.tenant), body: map[string]any{"planId": 1, "reason": "other", "proofReference": "xxx", "note": "xxxxx"}, platform: true},
		{name: "tenants list", method: "GET", path: "/api/v1/tenants", platform: true},
		{name: "tenants export", method: "GET", path: "/api/v1/tenants/export", platform: true},
		{name: "tenant names", method: "GET", path: fmt.Sprintf("/api/v1/tenants/names?ids=%d", w.b.tenant), platform: true},
		{name: "console overview", method: "GET", path: fmt.Sprintf("/api/v1/console/tenants/%d/overview", w.b.tenant), platform: true},
		{name: "console notes", method: "GET", path: fmt.Sprintf("/api/v1/console/tenants/%d/notes", w.b.tenant), platform: true},
		{name: "console note add", method: "POST", path: fmt.Sprintf("/api/v1/console/tenants/%d/notes", w.b.tenant), body: map[string]any{"body": "catatan"}, platform: true},
		{name: "console data-requests", method: "GET", path: fmt.Sprintf("/api/v1/console/tenants/%d/data-requests", w.b.tenant), platform: true},
		{name: "console support", method: "GET", path: fmt.Sprintf("/api/v1/console/tenants/%d/support", w.b.tenant), platform: true},
		{name: "console impersonate", method: "POST", path: fmt.Sprintf("/api/v1/console/tenants/%d/impersonate", w.b.tenant), body: map[string]any{"reason": "alasan yang cukup panjang"}, platform: true},
		{name: "console all data-requests", method: "GET", path: "/api/v1/console/data-requests", platform: true},
		{name: "console announcements", method: "GET", path: "/api/v1/console/announcements", platform: true},
		{name: "console announcement create", method: "POST", path: "/api/v1/console/announcements", body: map[string]any{"title": "x"}, platform: true},
		{name: "console policy get", method: "GET", path: "/api/v1/console/subscription-policy", platform: true},
		{name: "console policy put", method: "PUT", path: "/api/v1/console/subscription-policy", body: map[string]any{"trialDays": 1, "expiringSoonDays": 1, "graceDays": 1, "readOnlyDays": 1}, platform: true},
		{name: "console session end", method: "POST", path: "/api/v1/console/support-sessions/1/end", platform: true},
		{name: "console login slide", method: "PUT", path: "/api/v1/console/login-slides/0", body: map[string]any{"mimeType": "image/png", "base64Data": "AA=="}, platform: true},
		{name: "platform metrics", method: "GET", path: "/api/v1/platform/metrics", platform: true},
		{name: "platform trends", method: "GET", path: "/api/v1/platform/trends", platform: true},
		{name: "platform admins list", method: "GET", path: "/api/v1/platform-admins", platform: true},
		{name: "platform admin get", method: "GET", path: "/api/v1/platform-admins/1", platform: true},
		{name: "platform admin reset-password", method: "POST", path: "/api/v1/platform-admins/1/reset-password", body: map[string]any{"password": "Password-baru-123"}, platform: true},
		{name: "audit log", method: "GET", path: "/api/v1/audit-log", platform: true},
		{name: "audit log export", method: "GET", path: "/api/v1/audit-log/export", platform: true},
		{name: "plans write", method: "POST", path: "/api/v1/plans", body: map[string]any{"name": "x", "durationMonths": 1, "price": 1, "features": []string{"x"}}, platform: true},
		{name: "trial limits write", method: "PUT", path: "/api/v1/trial-limits", body: map[string]any{"maxStaff": 99}, platform: true},
		{name: "payment gateway config", method: "GET", path: "/api/v1/payment/gateway-config", platform: true},
		{name: "payment apps", method: "GET", path: "/api/v1/payment/apps", platform: true},
	}

	for _, a := range attacks {
		a := a
		routesCovered++
		t.Run(a.name, func(t *testing.T) {
			label := fmt.Sprintf("%s %s", a.method, a.path)
			r := s.do(w.a.token, a.method, a.path, a.body)
			if r.status != http.StatusForbidden && r.status != http.StatusNotFound {
				t.Errorf("A -> %s: want 403 or 404, got %s", label, r)
			}
			if !a.platform && vacuous(r) {
				t.Errorf("A -> %s: answered with the router's \"no such endpoint\" (a mistyped route would pass vacuously): %s", label, r)
			}
			w.checkNoLeak(t, "A -> "+label, r)

			if !a.control || a.platform {
				return
			}
			cp := a.path
			if a.controlPath != "" {
				cp = a.controlPath
			}
			c := s.do(w.b.token, a.method, cp, a.body)
			if c.status == http.StatusNotFound || c.status == http.StatusForbidden || c.status == http.StatusUnauthorized || c.status >= 500 {
				t.Errorf("control: B -> %s %s must reach its own data, got %s", a.method, cp, c)
			}
		})
	}
}

// mixedIDs combines A's own project with B's sub-resource ids: a lookup that
// checks the project but not the child's owner would let these through.
func (w *world) mixedIDs(t *testing.T, s *stack) {
	// A needs its own engagement so that A's project has something to be mixed with.
	cat := vendorBody(s, w.a, "x", 0)["categoryId"]
	ownEng := s.mustOK("A engagement", s.do(w.a.token, "POST", fmt.Sprintf("/api/v1/projects/%d/vendors", w.aProject), map[string]any{
		"vendorId": w.aVendor, "categoryId": cat, "scope": "Dekor A", "contractValue": 1000000, "pricingTier": "Akad",
		"engagementStatus": "Booked", "eventDate": day(120), "picStaffId": w.a.staffID,
	})).id("data", "id")
	ap := func(rest string) string { return fmt.Sprintf("/api/v1/projects/%d%s", w.aProject, rest) }
	cases := []attack{
		{name: "A project + B milestone", method: "PATCH", path: ap(fmt.Sprintf("/milestones/%d", w.ProjectMilestone)), body: map[string]any{"status": "Completed", "targetDate": day(15)}},
		{name: "A project + B engagement update", method: "PATCH", path: ap(fmt.Sprintf("/vendors/%d", w.Engagement)), body: map[string]any{"vendorId": w.aVendor, "categoryId": cat, "scope": "x", "pricingTier": "Akad", "engagementStatus": "Planned", "eventDate": day(120), "picStaffId": w.a.staffID}},
		{name: "A project + B engagement cancel", method: "POST", path: ap(fmt.Sprintf("/vendors/%d/cancel", w.Engagement)), body: map[string]any{}},
		{name: "A project + B engagement milestones", method: "GET", path: ap(fmt.Sprintf("/vendors/%d/milestones", w.Engagement))},
		{name: "A project + B engagement milestone create", method: "POST", path: ap(fmt.Sprintf("/vendors/%d/milestones", w.Engagement)), body: map[string]any{"name": "x", "targetDate": day(5), "picStaffId": w.a.staffID}},
		{name: "A engagement + B vendor milestone", method: "PATCH", path: ap(fmt.Sprintf("/vendors/%d/milestones/%d", ownEng, w.VendorMilestone)), body: map[string]any{"status": "Completed", "targetDate": day(15)}},
		{name: "A project + B vendor payment update", method: "PATCH", path: ap(fmt.Sprintf("/payments/%d", w.VendorPay)), body: map[string]any{"projectVendorId": ownEng, "type": "DP", "amount": 1, "paymentDate": day(-1), "method": "Transfer"}},
		{name: "A project + B vendor payment delete", method: "DELETE", path: ap(fmt.Sprintf("/payments/%d", w.VendorPay))},
		{name: "A project + B client payment update", method: "PATCH", path: ap(fmt.Sprintf("/client-payments/%d", w.ClientPayment)), body: map[string]any{"type": "DP", "amount": 1, "paymentDate": day(-1), "method": "Transfer"}},
		{name: "A project + B client payment delete", method: "DELETE", path: ap(fmt.Sprintf("/client-payments/%d", w.ClientPayment))},
		{name: "A project + B venue payment update", method: "PATCH", path: ap(fmt.Sprintf("/venue-payments/%d", w.VenuePay)), body: map[string]any{"type": "DP", "amount": 1, "paymentDate": day(-1), "method": "Transfer"}},
		{name: "A project + B venue payment delete", method: "DELETE", path: ap(fmt.Sprintf("/venue-payments/%d", w.VenuePay))},
		{name: "A project + B issue update", method: "PATCH", path: ap(fmt.Sprintf("/issues/%d", w.Issue)), body: map[string]any{"projectVendorId": ownEng, "title": "x", "description": "d", "impact": "Low", "foundDate": day(-1), "status": "Open", "picStaffId": w.a.staffID}},
		{name: "A project + B engagement delete", method: "DELETE", path: ap(fmt.Sprintf("/vendors/%d", w.Engagement))},
		{name: "A project + B engagement restore", method: "POST", path: ap(fmt.Sprintf("/vendors/%d/restore", w.Engagement))},
		{name: "A project + B vendor payment restore", method: "POST", path: ap(fmt.Sprintf("/payments/%d/restore", w.VendorPay))},
		{name: "A project + B client payment restore", method: "POST", path: ap(fmt.Sprintf("/client-payments/%d/restore", w.ClientPayment))},
		{name: "A project + B venue payment restore", method: "POST", path: ap(fmt.Sprintf("/venue-payments/%d/restore", w.VenuePay))},
		{name: "A project + B evidence delete", method: "DELETE", path: ap(fmt.Sprintf("/evidence/%d", w.EvidenceVisible))},
		{name: "A project + B evidence file", method: "GET", path: ap(fmt.Sprintf("/evidence/%d/file", w.EvidenceVisible))},
		{name: "A project + B evidence link", method: "GET", path: ap(fmt.Sprintf("/evidence/%d/link", w.EvidenceVisible))},
		{name: "A project + B evidence toggle", method: "POST", path: ap(fmt.Sprintf("/evidence/%d/toggle-client-visible", w.EvidenceVisible))},
	}
	for _, a := range cases {
		a := a
		routesCovered++
		t.Run(a.name, func(t *testing.T) {
			r := s.do(w.a.token, a.method, a.path, a.body)
			if r.status != http.StatusNotFound && r.status != http.StatusForbidden {
				t.Errorf("%s %s: want 403 or 404, got %s", a.method, a.path, r)
			}
			if vacuous(r) {
				t.Errorf("%s %s: router miss, not an ownership answer: %s", a.method, a.path, r)
			}
			w.checkNoLeak(t, a.method+" "+a.path, r)
		})
	}
}
