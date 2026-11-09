//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func (w *world) checkNoLeakVia(t *testing.T) func(string, reply) {
	return func(what string, r reply) { w.checkNoLeak(t, what, r) }
}

// ---------------------------------------------------------------------------
// lists and dashboards

func (w *world) lists(t *testing.T, s *stack) {
	endpoints := []string{
		"/api/v1/projects", "/api/v1/projects?all=true", "/api/v1/projects?archived=true", "/api/v1/projects?search=RAHASIA",
		"/api/v1/vendors", "/api/v1/vendors?all=true", "/api/v1/vendors?search=RAHASIA", "/api/v1/vendors/summary",
		"/api/v1/venues", "/api/v1/venues?all=true", "/api/v1/venues?search=RAHASIA",
		"/api/v1/vendor-categories", "/api/v1/vendor-categories?all=true", "/api/v1/vendor-categories?search=RAHASIA",
		"/api/v1/clients", "/api/v1/clients?all=true", "/api/v1/clients?search=RAHASIA",
		fmt.Sprintf("/api/v1/clients?projectId=%d", w.Project),
		"/api/v1/staff", "/api/v1/staff?all=true", "/api/v1/staff?search=RAHASIA", "/api/v1/staff/summary",
		"/api/v1/access/roles", "/api/v1/access/roles?kind=staff", "/api/v1/access/roles?kind=client", "/api/v1/access/me",
		"/api/v1/milestone-templates",
		"/api/v1/dashboard", "/api/v1/dashboard/mine",
		"/api/v1/subscription-transactions", "/api/v1/subscription-transactions?all=true",
		"/api/v1/tenants/me", "/api/v1/tenants/me/contact", "/api/v1/tenants/me/branding", "/api/v1/tenants/me/subscription",
		"/api/v1/tenants/me/support-access", "/api/v1/trial-limits", "/api/v1/announcements/active",
	}
	for _, p := range endpoints {
		p := p
		routesCovered++
		t.Run(strings.TrimPrefix(p, "/api/v1/"), func(t *testing.T) {
			r := s.do(w.a.token, "GET", p, nil)
			// Some of these are forbidden to this role or only answer other
			// principals; what matters is that nothing of B's is in the answer.
			if r.status >= 500 {
				t.Errorf("GET %s: server error %s", p, r)
			}
			w.checkNoLeak(t, "A GET "+p, r)
			// Ids of B must not appear as ids in A's rows.
			for _, id := range []int64{w.Project, w.Project2, w.Sacrifice} {
				if strings.Contains(p, "projects") && strings.Contains(string(r.body), fmt.Sprintf(`"id":%d,`, id)) && strings.Contains(string(r.body), "Proyek B") {
					t.Errorf("GET %s lists B's project %d: %s", p, id, r)
				}
			}
		})
	}

	// A sees exactly its own project, and its own contract value on the dashboard.
	t.Run("A project list is only A's", func(t *testing.T) {
		r := s.mustOK("A projects", s.do(w.a.token, "GET", "/api/v1/projects?all=true", nil))
		list, _ := r.dig("data").([]any)
		if len(list) != 1 {
			t.Errorf("A has 1 project, list shows %d: %s", len(list), r)
		}
	})
	t.Run("B list control", func(t *testing.T) {
		r := s.mustOK("B projects", s.do(w.b.token, "GET", "/api/v1/projects?all=true", nil))
		if list, _ := r.dig("data").([]any); len(list) < 3 {
			t.Errorf("B should see its own projects (the leak checks above are vacuous otherwise): %s", r)
		}
	})
}

// ---------------------------------------------------------------------------
// foreign references in write bodies

type foreign struct {
	name   string
	method string
	path   string
	attack map[string]any // names one of B's ids
	own    map[string]any // the same request with A's own ids: must succeed
	// verify, when set, runs after a refused attack and fails the test if the
	// reference was stored anyway.
	verify func(t *testing.T)
}

func (w *world) foreignReferences(t *testing.T, s *stack) {
	a, b := w.a, w.b
	catA := vendorBody(s, a, "x", 0)["categoryId"]
	ownRole := s.mustOK("A role", s.do(a.token, "POST", "/api/v1/access/roles", map[string]any{
		"kind": "staff", "name": "Peran A", "description": "x", "projectScope": "all", "permissions": []string{"project.edit"},
	})).id("data", "id")
	ownEng := s.mustOK("A engagement", s.do(a.token, "POST", fmt.Sprintf("/api/v1/projects/%d/vendors", w.aProject), map[string]any{
		"vendorId": w.aVendor, "categoryId": catA, "scope": "Dekor A2", "contractValue": 1000000, "pricingTier": "Akad",
		"engagementStatus": "Booked", "eventDate": day(120), "picStaffId": a.staffID,
	})).id("data", "id")
	clientRoles := s.mustOK("A client roles", s.do(a.token, "GET", "/api/v1/access/roles?kind=client", nil))
	var ownClientRole int64
	if list, ok := clientRoles.dig("data").([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			if f, ok := m["id"].(float64); ok {
				ownClientRole = int64(f)
			}
		}
	}
	var ownTemplates []int64
	tpl := s.mustOK("A templates", s.do(a.token, "GET", "/api/v1/milestone-templates", nil))
	if list, ok := tpl.dig("data").([]any); ok {
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				if f, ok := m["id"].(float64); ok {
					ownTemplates = append(ownTemplates, int64(f))
				}
			}
		}
	}
	proj := func(id int64, rest string) string { return fmt.Sprintf("/api/v1/projects/%d%s", id, rest) }
	projectInput := func(pic, sales int64, venue any) map[string]any {
		m := map[string]any{
			"name": "Proyek A baru", "brideName": "A", "groomName": "B", "eventDate": day(150), "venue": "Gedung", "prepStartDate": day(1),
			"packageName": "Paket", "contractValue": 1000, "status": "Preparation", "picStaffId": pic, "picSalesStaffId": sales,
		}
		if venue != nil {
			m["venueId"] = venue
		}
		return m
	}
	projectVenueID := func() any {
		r := s.mustOK("A project", s.do(a.token, "GET", proj(w.aProject, ""), nil))
		return r.dig("data", "venueId")
	}

	cases := []foreign{
		{name: "project create with B's staff as PIC", method: "POST", path: "/api/v1/projects",
			attack: projectInput(w.Staff2, 0, nil), own: projectInput(a.staffID, 0, nil)},
		{name: "project create with B's owner as PIC", method: "POST", path: "/api/v1/projects",
			attack: projectInput(b.staffID, 0, nil), own: projectInput(a.staffID, 0, nil)},
		{name: "project create with B's staff as PIC Sales", method: "POST", path: "/api/v1/projects",
			attack: projectInput(a.staffID, w.Staff2, nil), own: projectInput(a.staffID, a.staffID, nil)},
		{name: "project update with B's venue", method: "PATCH", path: proj(w.aProject, ""),
			attack: projectInput(a.staffID, 0, w.Venue), own: projectInput(a.staffID, 0, w.aVenue),
			verify: func(t *testing.T) {
				if v := projectVenueID(); v == float64(w.Venue) {
					t.Errorf("A's project now points at B's venue %d", w.Venue)
				}
			}},
		{name: "project update with B's staff as PIC", method: "PATCH", path: proj(w.aProject, ""),
			attack: projectInput(w.Staff2, 0, nil), own: projectInput(a.staffID, 0, nil)},
		// The date and category are left out on purpose: the server derives them (E7), so the
		// attack is refused for the foreign id and the control succeeds whatever the project's date now is.
		{name: "engagement with B's vendor", method: "POST", path: proj(w.aProject, "/vendors"),
			attack: map[string]any{"vendorId": w.Vendor, "categoryId": catA, "scope": "x", "contractValue": 1, "pricingTier": "Akad", "engagementStatus": "Planned", "picStaffId": a.staffID},
			own:    map[string]any{"vendorId": w.aVendor, "categoryId": catA, "scope": "y", "contractValue": 1, "pricingTier": "Akad", "engagementStatus": "Planned", "picStaffId": a.staffID}},
		{name: "engagement with B's category", method: "POST", path: proj(w.aProject, "/vendors"),
			attack: map[string]any{"vendorId": w.aVendor, "categoryId": w.Category, "scope": "x", "contractValue": 1, "pricingTier": "Akad", "engagementStatus": "Planned", "picStaffId": a.staffID},
			own:    map[string]any{"vendorId": w.aVendor, "categoryId": catA, "scope": "z", "contractValue": 1, "pricingTier": "Akad", "engagementStatus": "Planned", "picStaffId": a.staffID}},
		{name: "engagement with B's staff as PIC", method: "POST", path: proj(w.aProject, "/vendors"),
			attack: map[string]any{"vendorId": w.aVendor, "categoryId": catA, "scope": "x", "contractValue": 1, "pricingTier": "Akad", "engagementStatus": "Planned", "picStaffId": w.Staff2},
			own:    map[string]any{"vendorId": w.aVendor, "categoryId": catA, "scope": "w", "contractValue": 1, "pricingTier": "Akad", "engagementStatus": "Planned", "picStaffId": a.staffID}},
		{name: "vendor milestone with B's staff as PIC", method: "POST", path: proj(w.aProject, fmt.Sprintf("/vendors/%d/milestones", ownEng)),
			attack: map[string]any{"name": "x", "targetDate": day(5), "picStaffId": w.Staff2},
			own:    map[string]any{"name": "x", "targetDate": day(5), "picStaffId": a.staffID}},
		{name: "vendor payment with B's engagement", method: "POST", path: proj(w.aProject, "/payments"),
			attack: map[string]any{"projectVendorId": w.Engagement, "type": "DP", "amount": 1, "paymentDate": day(-1), "method": "Transfer"},
			own:    map[string]any{"projectVendorId": ownEng, "type": "DP", "amount": 1, "paymentDate": day(-1), "method": "Transfer"}},
		{name: "issue with B's engagement", method: "POST", path: proj(w.aProject, "/issues"),
			attack: map[string]any{"projectVendorId": w.Engagement, "title": "x", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": a.staffID, "targetResolutionDate": day(5)},
			own:    map[string]any{"projectVendorId": ownEng, "title": "x", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": a.staffID, "targetResolutionDate": day(5)}},
		{name: "issue with B's staff as PIC", method: "POST", path: proj(w.aProject, "/issues"),
			attack: map[string]any{"projectVendorId": ownEng, "title": "x", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": w.Staff2, "targetResolutionDate": day(5)},
			own:    map[string]any{"projectVendorId": ownEng, "title": "y", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": a.staffID, "targetResolutionDate": day(5)}},
		{name: "evidence attached to B's milestone", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: map[string]any{"name": "x", "type": "Document", "fileName": "x.pdf", "mimeType": "application/pdf", "base64Data": pdf, "documentDate": day(-1), "relatedKind": "projectMilestone", "relatedId": w.ProjectMilestone},
			own:    map[string]any{"name": "x", "type": "Document", "fileName": "x.pdf", "mimeType": "application/pdf", "base64Data": pdf, "documentDate": day(-1), "relatedKind": "general"}},
		{name: "evidence attached to B's issue", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: map[string]any{"name": "x", "type": "Document", "fileName": "x.pdf", "mimeType": "application/pdf", "base64Data": pdf, "documentDate": day(-1), "relatedKind": "issue", "relatedId": w.Issue},
			own:    map[string]any{"name": "x2", "type": "Document", "fileName": "x.pdf", "mimeType": "application/pdf", "base64Data": pdf, "documentDate": day(-1), "relatedKind": "general"}},
		{name: "client with B's portal role", method: "POST", path: "/api/v1/clients",
			attack: map[string]any{"projectId": w.aProject, "role": "Bride", "name": "Klien A", "phone": "081200000001", "username": "klien.a.satu", "accessRoleId": w.ClientRole},
			own:    map[string]any{"projectId": w.aProject, "role": "Bride", "name": "Klien A", "phone": "081200000001", "username": "klien.a.dua", "accessRoleId": ownClientRole}},
		{name: "client in B's project", method: "POST", path: "/api/v1/clients",
			attack: map[string]any{"projectId": w.Project2, "role": "Bride", "name": "Klien A", "phone": "081200000002", "username": "klien.a.tiga", "accessRoleId": ownClientRole},
			own:    map[string]any{"projectId": w.aProject, "role": "Groom", "name": "Klien A", "phone": "081200000002", "username": "klien.a.empat", "accessRoleId": ownClientRole}},
		{name: "staff with B's role", method: "POST", path: "/api/v1/staff",
			attack: map[string]any{"name": "Staf A", "title": "x", "roleId": w.Role, "email": "staf.a.satu@example.com", "phone": "081200000003"},
			own:    map[string]any{"name": "Staf A", "title": "x", "roleId": ownRole, "email": "staf.a.dua@example.com", "phone": "081200000003"}},
		{name: "milestone template reorder with B's templates", method: "PATCH", path: "/api/v1/milestone-templates",
			attack: map[string]any{"orderedIds": append(append([]int64{}, ownTemplates...), w.Template)},
			own:    map[string]any{"orderedIds": ownTemplates}},
		{name: "vendor with B's category", method: "POST", path: "/api/v1/vendors",
			attack: vendorBody(s, a, "Vendor A salah", w.Category), own: vendorBody(s, a, "Vendor A benar", 0)},
	}
	for _, c := range cases {
		c := c
		routesCovered++
		t.Run(c.name, func(t *testing.T) {
			r := s.do(a.token, c.method, c.path, c.attack)
			if r.status < 400 || r.status >= 500 {
				t.Errorf("%s %s naming B's id must be refused with a 4xx, got %s", c.method, c.path, r)
			}
			w.checkNoLeak(t, c.name, r)
			if c.verify != nil {
				c.verify(t)
			}
			ctl := s.do(a.token, c.method, c.path, c.own)
			if ctl.status < 200 || ctl.status > 299 {
				t.Errorf("control: the same request with A's own ids should succeed, got %s", ctl)
			}
		})
	}

	// None of A's writes above may have created anything pointing into B.
	t.Run("nothing of A references B", func(t *testing.T) {
		for _, p := range []string{"/api/v1/staff?all=true", "/api/v1/clients?all=true", "/api/v1/projects?all=true"} {
			r := s.mustOK(p, s.do(a.token, "GET", p, nil))
			w.checkNoLeak(t, "A GET "+p, r)
		}
	})
}

// ---------------------------------------------------------------------------
// the client principal of workspace B

func (w *world) clientIsolation(t *testing.T, s *stack) {
	c := w.clientToken
	denied := []attack{
		{name: "A's project", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d", w.aProject)},
		{name: "A's project documents", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/documents", w.aProject)},
		{name: "B's other project", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d", w.Project2)},
		{name: "B's other project milestones", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/milestones", w.Project2)},
		{name: "B's sacrificial project", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d", w.Sacrifice)},
		{name: "internal vendor payments", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/payments", w.Project)},
		{name: "project activity log", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/activity", w.Project)},
		{name: "closing checklist", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/closing-checklist", w.Project)},
		{name: "edit the project", method: "PATCH", path: fmt.Sprintf("/api/v1/projects/%d", w.Project), body: map[string]any{"name": "x"}},
		{name: "create a project", method: "POST", path: "/api/v1/projects", body: map[string]any{"name": "x"}},
		{name: "record a payment", method: "POST", path: fmt.Sprintf("/api/v1/projects/%d/client-payments", w.Project), body: map[string]any{"type": "DP", "amount": 1}},
		{name: "toggle a document", method: "POST", path: fmt.Sprintf("/api/v1/projects/%d/evidence/%d/toggle-client-visible", w.Project, w.EvidenceInternal)},
		{name: "staff list", method: "GET", path: "/api/v1/staff"},
		{name: "client list", method: "GET", path: "/api/v1/clients"},
		{name: "roles", method: "GET", path: "/api/v1/access/roles"},
		{name: "vendors write", method: "POST", path: "/api/v1/vendors", body: map[string]any{"name": "x"}},
		{name: "dashboard", method: "GET", path: "/api/v1/dashboard"},
		{name: "subscription transactions", method: "GET", path: "/api/v1/subscription-transactions"},
		{name: "own tenant record", method: "GET", path: "/api/v1/tenants/me"},
		{name: "console", method: "GET", path: fmt.Sprintf("/api/v1/console/tenants/%d/overview", w.b.tenant)},
		{name: "internal document file", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/evidence/%d/file", w.Project, w.EvidenceInternal)},
		{name: "internal document link", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/evidence/%d/link", w.Project, w.EvidenceInternal)},
		{name: "internal document download link", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/evidence/%d/link?disposition=attachment", w.Project, w.EvidenceInternal)},
	}
	for _, d := range denied {
		d := d
		routesCovered++
		t.Run("client refused: "+d.name, func(t *testing.T) {
			r := s.do(c, d.method, d.path, d.body)
			if r.status != http.StatusForbidden && r.status != http.StatusNotFound {
				t.Errorf("client -> %s %s: want 403 or 404, got %s", d.method, d.path, r)
			}
			if strings.Contains(string(r.body), "Dokumen internal") {
				t.Errorf("client -> %s %s leaks the internal document: %s", d.method, d.path, r)
			}
		})
	}

	// What the client may do, so the refusals above are not vacuous.
	allowed := []struct{ name, path string }{
		{"own project", fmt.Sprintf("/api/v1/projects/%d", w.Project)},
		{"own milestones", fmt.Sprintf("/api/v1/projects/%d/milestones", w.Project)},
		{"shared document file", fmt.Sprintf("/api/v1/projects/%d/evidence/%d/file", w.Project, w.EvidenceVisible)},
		{"shared document link", fmt.Sprintf("/api/v1/projects/%d/evidence/%d/link", w.Project, w.EvidenceVisible)},
		{"documents tab", fmt.Sprintf("/api/v1/projects/%d/documents", w.Project)},
	}
	for _, a := range allowed {
		a := a
		routesCovered++
		t.Run("client allowed: "+a.name, func(t *testing.T) {
			r := s.do(c, "GET", a.path, nil)
			if r.status != http.StatusOK {
				t.Errorf("client -> GET %s should work, got %s", a.path, r)
			}
		})
	}
	t.Run("client document list hides internal documents", func(t *testing.T) {
		for _, p := range []string{fmt.Sprintf("/api/v1/projects/%d/documents", w.Project), fmt.Sprintf("/api/v1/projects/%d/evidence", w.Project)} {
			r := s.do(c, "GET", p, nil)
			if strings.Contains(string(r.body), "Dokumen internal") {
				t.Errorf("client GET %s shows the internal document: %s", p, r)
			}
		}
	})
	t.Run("A's staff cannot use B's client portal", func(t *testing.T) {
		for _, p := range []string{
			fmt.Sprintf("/api/v1/projects/%d", w.Project), fmt.Sprintf("/api/v1/projects/%d/documents", w.Project),
			fmt.Sprintf("/api/v1/projects/%d/evidence/%d/link", w.Project, w.EvidenceVisible),
		} {
			r := s.do(w.a.token, "GET", p, nil)
			if r.status != http.StatusNotFound && r.status != http.StatusForbidden {
				t.Errorf("A -> GET %s: want 403 or 404, got %s", p, r)
			}
		}
	})
	t.Run("no token and a forged token get nothing", func(t *testing.T) {
		for _, tok := range []string{"", "not-a-token", c + "x"} {
			r := s.do(tok, "GET", fmt.Sprintf("/api/v1/projects/%d", w.Project), nil)
			if r.status != http.StatusUnauthorized {
				t.Errorf("token %q: want 401, got %s", tok, r)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// after everything: B's data is untouched

func (w *world) intact(t *testing.T, s *stack) {
	b := w.b
	check := func(what, path string, want ...string) {
		routesCovered++
		t.Run(what, func(t *testing.T) {
			r := s.do(b.token, "GET", path, nil)
			if r.status != http.StatusOK {
				t.Errorf("B GET %s: %s", path, r)
				return
			}
			for _, x := range want {
				if !strings.Contains(string(r.body), x) {
					t.Errorf("B GET %s lost %q: %s", path, x, r)
				}
			}
		})
	}
	P := w.Project
	check("project", fmt.Sprintf("/api/v1/projects/%d", P), "Proyek B "+b.marker, fmt.Sprint(w.bContract))
	check("second project", fmt.Sprintf("/api/v1/projects/%d", w.Project2), "Proyek B2 "+b.marker)
	check("vendors", "/api/v1/vendors?all=true", "Vendor B "+b.marker)
	check("venue", fmt.Sprintf("/api/v1/venues/%d", w.Venue), "Venue B "+b.marker)
	check("categories", "/api/v1/vendor-categories?all=true", "Kategori "+b.marker)
	check("engagement", fmt.Sprintf("/api/v1/projects/%d/vendors", P), "Dekorasi "+b.marker)
	check("vendor milestone", fmt.Sprintf("/api/v1/projects/%d/vendors/%d/milestones", P, w.Engagement), "Milestone vendor "+b.marker)
	check("vendor payments", fmt.Sprintf("/api/v1/projects/%d/payments", P), "VREF-"+b.marker)
	check("client payments", fmt.Sprintf("/api/v1/projects/%d/client-payments", P), "REF-"+b.marker)
	check("venue payments", fmt.Sprintf("/api/v1/projects/%d/venue-payments", P), "VNREF-"+b.marker)
	check("issues", fmt.Sprintf("/api/v1/projects/%d/issues", P), "Kendala "+b.marker)
	check("evidence", fmt.Sprintf("/api/v1/projects/%d/evidence", P), "Dokumen internal "+b.marker, "Dokumen klien "+b.marker)
	check("payment terms", fmt.Sprintf("/api/v1/projects/%d/payment-terms", P), "Termin "+b.marker)
	check("clients", "/api/v1/clients?all=true", "Klien "+b.marker)
	check("staff", "/api/v1/staff?all=true", "Staf "+b.marker)
	check("role", fmt.Sprintf("/api/v1/access/roles/%d", w.Role), "Peran "+b.marker)
	check("templates", "/api/v1/milestone-templates", "Template "+b.marker)
	check("transaction", fmt.Sprintf("/api/v1/subscription-transactions/%d", w.Tx), "ELP-"+b.marker)

	t.Run("B's owner is still the owner and signed in", func(t *testing.T) {
		if r := s.do(b.token, "GET", "/api/v1/tenants/me", nil); r.status != http.StatusOK || !strings.Contains(string(r.body), b.marker) {
			t.Errorf("tenant record changed: %s", r)
		}
	})
	t.Run("B's client can still sign in", func(t *testing.T) {
		if r := s.do(w.clientToken, "GET", fmt.Sprintf("/api/v1/projects/%d", P), nil); r.status != http.StatusOK {
			t.Errorf("client lost access (the attacks must not have deactivated it): %s", r)
		}
	})
}
