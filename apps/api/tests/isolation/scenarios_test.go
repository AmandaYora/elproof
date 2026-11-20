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
		"/api/v1/vendors", "/api/v1/vendors?all=true", "/api/v1/vendors?search=RAHASIA",
		// The removed directory route (Tahap 9): still never anything of B's, whatever the router answers.
		"/api/v1/vendors/summary",
		"/api/v1/venues", "/api/v1/venues?all=true", "/api/v1/venues?search=RAHASIA",
		"/api/v1/vendor-categories", "/api/v1/vendor-categories?all=true", "/api/v1/vendor-categories?search=RAHASIA",
		"/api/v1/clients", "/api/v1/clients?all=true", "/api/v1/clients?search=RAHASIA",
		fmt.Sprintf("/api/v1/clients?projectId=%d", w.Project),
		"/api/v1/staff", "/api/v1/staff?all=true", "/api/v1/staff?search=RAHASIA", "/api/v1/staff/summary",
		"/api/v1/access/roles", "/api/v1/access/roles?kind=staff", "/api/v1/access/roles?kind=client", "/api/v1/access/me",
		"/api/v1/milestone-templates",
		"/api/v1/dashboard", "/api/v1/dashboard/mine",
		"/api/v1/subscription-transactions", "/api/v1/subscription-transactions?all=true",
		// Tahap 6: sorting, filters and derived columns never widen the tenant scope.
		"/api/v1/projects?sort=remaining:desc", "/api/v1/projects?sort=nextDueDate:asc&eventFrom=2020-01-01&eventTo=2099-12-31",
		fmt.Sprintf("/api/v1/projects?picStaffId=%d", w.b.staffID), "/api/v1/projects?sort=name&q=RAHASIA",
		"/api/v1/clients?page=1&sort=projectName", "/api/v1/clients?page=1&loginStatus=active&sort=loginStatus:desc", "/api/v1/clients?page=1&role=Bride&q=RAHASIA",
		fmt.Sprintf("/api/v1/clients?projectId=%d&page=1&sort=name", w.Project), fmt.Sprintf("/api/v1/clients?projectId=%d&page=1", w.Project2),
		"/api/v1/vendors?sort=categoryName:desc&isActive=true", "/api/v1/vendors?sort=priceAkad&q=RAHASIA", "/api/v1/venues?sort=capacitySeated:desc&isActive=false",
		"/api/v1/vendor-categories?sort=name:desc", "/api/v1/staff?sort=roleName&isActive=true", fmt.Sprintf("/api/v1/staff?roleId=%d", w.Role),
		"/api/v1/subscription-transactions?sort=tenantName", "/api/v1/subscription-transactions?sort=amount:desc&search=RAHASIA",
		"/api/v1/subscription-transactions?from=2020-01-01&to=2099-12-31&search=ELP-",
		"/api/v1/tenants/me", "/api/v1/tenants/me/contact", "/api/v1/tenants/me/branding", "/api/v1/tenants/me/subscription",
		"/api/v1/tenants/me/support-access", "/api/v1/trial-limits", "/api/v1/announcements/active",
		// Paket WO: event kinds, packages and the money views across projects.
		"/api/v1/event-kinds", "/api/v1/event-kinds?all=true", "/api/v1/packages",
		"/api/v1/finance/summary", "/api/v1/finance/receivables", "/api/v1/finance/payables", "/api/v1/finance/verifications",
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
	ownRole := s.mustOK("A role", s.do(a.token, "POST", "/api/v1/access/roles", map[string]any{
		"kind": "staff", "name": "Peran A", "description": "x", "projectScope": "all", "permissions": []string{"project.edit"},
	})).id("data", "id")
	ownSup := s.engage(a, w.aProject, w.aVendor, "Dekor A2", 1000000, "Booked", nil)
	ownStep := s.mustOK("A supplier milestone", s.do(a.token, "POST", fmt.Sprintf("/api/v1/projects/%d/suppliers/%d/milestones", w.aProject, ownSup),
		map[string]any{"name": "Langkah A", "description": "d", "targetDate": day(5), "picStaffId": a.staffID})).id("data", "id")
	ownPay := s.supplierPay(a, w.aProject, ownSup, "DP", 1000, "AREF")
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
	n := 0
	projectInput := func(pic, sales int64, events ...map[string]any) map[string]any {
		n++
		m := s.projectBody(a, fmt.Sprintf("Proyek A baru %d", n), 150, events...)
		m["picStaffId"], m["picSalesStaffId"] = pic, sales
		return m
	}
	// A project the control created stays out of the trial cap (5 active projects).
	archiveCreated := func(t *testing.T, r reply) {
		if r.status != http.StatusCreated {
			return
		}
		id := r.id("data", "id")
		s.mustOK("archive control project", s.doFresh(a.token, "PATCH", proj(id, ""), map[string]any{"isArchived": true}))
	}
	aEvents := rows(s.mustOK("A events", s.do(a.token, "GET", proj(w.aProject, "/events"), nil)), "data")
	if len(aEvents) == 0 {
		t.Fatalf("A's project has no event")
	}
	aEvent := int64(aEvents[0]["id"].(float64))
	aEventVenue := func() any {
		for _, e := range rows(s.mustOK("A events", s.do(a.token, "GET", proj(w.aProject, "/events"), nil)), "data") {
			if int64(e["id"].(float64)) == aEvent {
				return e["venueId"]
			}
		}
		return nil
	}
	issueBody := func(sup int64, pic int64, step any) map[string]any {
		m := map[string]any{"projectSupplierId": sup, "title": "x", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": pic, "targetResolutionDate": day(5)}
		if step != nil {
			m["vendorMilestoneId"] = step
		}
		return m
	}
	evidence := func(kind string, id int64) map[string]any {
		m := map[string]any{"name": "x", "type": "Document", "fileName": "x.pdf", "mimeType": "application/pdf", "base64Data": pdf, "documentDate": day(-1), "relatedKind": kind}
		if id != 0 {
			m["relatedId"] = id
		}
		return m
	}
	supplierBody := func(vendor, pic int64, scope string) map[string]any {
		return map[string]any{"vendorId": vendor, "paidBy": "Client", "scope": scope, "engagementStatus": "Planned", "picStaffId": pic}
	}
	payBody := func(sup int64) map[string]any {
		return map[string]any{"projectSupplierId": sup, "type": "DP", "amount": 1, "paymentDate": day(-1), "method": "Transfer"}
	}

	cases := []foreign{
		{name: "project create with B's staff as PIC", method: "POST", path: "/api/v1/projects",
			attack: projectInput(w.Staff2, 0), own: projectInput(a.staffID, 0)},
		{name: "project create with B's owner as PIC", method: "POST", path: "/api/v1/projects",
			attack: projectInput(b.staffID, 0), own: projectInput(a.staffID, 0)},
		{name: "project create with B's staff as PIC Sales", method: "POST", path: "/api/v1/projects",
			attack: projectInput(a.staffID, w.Staff2), own: projectInput(a.staffID, a.staffID)},
		{name: "project create with B's venue on an event", method: "POST", path: "/api/v1/projects",
			attack: projectInput(a.staffID, 0, eventOf(w.aKind, day(150), 50, w.Venue)), own: projectInput(a.staffID, 0, eventOf(w.aKind, day(150), 50, w.aVenue))},
		{name: "project create with B's event kind", method: "POST", path: "/api/v1/projects",
			attack: projectInput(a.staffID, 0, eventOf(w.Kind, day(150), 50, 0)), own: projectInput(a.staffID, 0, eventOf(w.aKind, day(150), 50, 0))},
		{name: "project update with B's staff as PIC", method: "PATCH", path: proj(w.aProject, ""),
			attack: projectInput(w.Staff2, 0), own: projectInput(a.staffID, 0)},
		{name: "event create with B's venue", method: "POST", path: proj(w.aProject, "/events"),
			attack: eventOf(w.aKind, day(151), 50, w.Venue), own: eventOf(w.aKind, day(151), 50, 0)},
		{name: "event create with B's event kind", method: "POST", path: proj(w.aProject, "/events"),
			attack: eventOf(w.Kind, day(152), 50, 0), own: eventOf(w.aKind, day(152), 50, 0)},
		{name: "event update with B's venue", method: "PUT", path: proj(w.aProject, fmt.Sprintf("/events/%d", aEvent)),
			attack: eventOf(w.aKind, day(120), 100, w.Venue), own: eventOf(w.aKind, day(120), 100, w.aVenue),
			verify: func(t *testing.T) {
				if v := aEventVenue(); v == float64(w.Venue) {
					t.Errorf("A's event now points at B's venue %d", w.Venue)
				}
			}},
		{name: "event update with B's event kind", method: "PUT", path: proj(w.aProject, fmt.Sprintf("/events/%d", aEvent)),
			attack: eventOf(w.Kind, day(120), 100, w.aVenue), own: eventOf(w.aKind, day(120), 100, w.aVenue)},
		{name: "supplier with B's vendor", method: "POST", path: proj(w.aProject, "/suppliers"),
			attack: supplierBody(w.Vendor, a.staffID, "x"), own: supplierBody(w.aVendor, a.staffID, "y")},
		{name: "supplier with B's venue as vendor", method: "POST", path: proj(w.aProject, "/suppliers"),
			attack: supplierBody(w.Venue, a.staffID, "x"), own: supplierBody(w.aVendor, a.staffID, "v")},
		{name: "supplier with B's staff as PIC", method: "POST", path: proj(w.aProject, "/suppliers"),
			attack: supplierBody(w.aVendor, w.Staff2, "x"), own: supplierBody(w.aVendor, a.staffID, "w")},
		{name: "supplier update with B's staff as PIC", method: "PATCH", path: proj(w.aProject, fmt.Sprintf("/suppliers/%d", ownSup)),
			attack: map[string]any{"picStaffId": w.Staff2}, own: map[string]any{"picStaffId": a.staffID}},
		{name: "supplier milestone with B's staff as PIC", method: "POST", path: proj(w.aProject, fmt.Sprintf("/suppliers/%d/milestones", ownSup)),
			attack: map[string]any{"name": "x", "targetDate": day(5), "picStaffId": w.Staff2},
			own:    map[string]any{"name": "x", "targetDate": day(5), "picStaffId": a.staffID}},
		{name: "supplier payment with B's supplier", method: "POST", path: proj(w.aProject, "/supplier-payments"),
			attack: payBody(w.Supplier), own: payBody(ownSup)},
		{name: "supplier payment with B's venue supplier", method: "POST", path: proj(w.aProject, "/supplier-payments"),
			attack: payBody(w.VenueSupplier), own: payBody(ownSup)},
		{name: "issue with B's supplier", method: "POST", path: proj(w.aProject, "/issues"),
			attack: issueBody(w.Supplier, a.staffID, nil), own: issueBody(ownSup, a.staffID, nil)},
		{name: "issue with B's supplier milestone", method: "POST", path: proj(w.aProject, "/issues"),
			attack: issueBody(ownSup, a.staffID, w.VendorMilestone), own: issueBody(ownSup, a.staffID, ownStep)},
		{name: "issue with B's staff as PIC", method: "POST", path: proj(w.aProject, "/issues"),
			attack: issueBody(ownSup, w.Staff2, nil), own: issueBody(ownSup, a.staffID, nil)},
		{name: "evidence attached to B's milestone", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: evidence("projectMilestone", w.ProjectMilestone), own: evidence("general", 0)},
		{name: "evidence attached to B's issue", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: evidence("issue", w.Issue), own: evidence("general", 0)},
		{name: "evidence attached to B's supplier", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: evidence("projectSupplier", w.Supplier), own: evidence("projectSupplier", ownSup)},
		{name: "evidence attached to B's supplier milestone", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: evidence("vendorMilestone", w.VendorMilestone), own: evidence("vendorMilestone", ownStep)},
		{name: "evidence attached to B's supplier payment", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: evidence("supplierPayment", w.SupplierPay), own: evidence("supplierPayment", ownPay)},
		{name: "evidence attached to B's client payment", method: "POST", path: proj(w.aProject, "/evidence"),
			attack: evidence("clientPayment", w.ClientPayment), own: evidence("general", 0)},
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
			r := s.doFresh(a.token, c.method, c.path, c.attack)
			if r.status < 400 || r.status >= 500 {
				t.Errorf("%s %s naming B's id must be refused with a 4xx, got %s", c.method, c.path, r)
			}
			archiveCreated(t, r)
			w.checkNoLeak(t, c.name, r)
			if c.verify != nil {
				c.verify(t)
			}
			ctl := s.doFresh(a.token, c.method, c.path, c.own)
			if ctl.status < 200 || ctl.status > 299 {
				t.Errorf("control: the same request with A's own ids should succeed, got %s", ctl)
			}
			if c.method == "POST" && c.path == "/api/v1/projects" {
				archiveCreated(t, ctl)
			}
		})
	}

	// A payment's supplier is fixed once recorded (PATCH ignores
	// projectSupplierId): naming B's supplier cannot move A's payment there.
	t.Run("supplier payment cannot be moved to B's supplier", func(t *testing.T) {
		routesCovered++
		r := s.doFresh(a.token, "PATCH", proj(w.aProject, fmt.Sprintf("/supplier-payments/%d", ownPay)), payBody(w.Supplier))
		if r.status >= 500 {
			t.Errorf("PATCH naming B's supplier: %s", r)
		}
		w.checkNoLeak(t, "PATCH naming B's supplier", r)
		got := rowByID(rows(s.mustOK("A supplier payments", s.do(a.token, "GET", proj(w.aProject, "/supplier-payments"), nil)), "data"), ownPay)
		if got == nil || got["projectSupplierId"] != float64(ownSup) {
			t.Errorf("A's payment %d left its supplier %d: %v", ownPay, ownSup, got)
		}
		b := rows(s.mustOK("B supplier payments", s.do(w.b.token, "GET", proj(w.Project, "/supplier-payments"), nil)), "data")
		if rowByID(b, ownPay) != nil {
			t.Errorf("A's payment %d shows up in B's project", ownPay)
		}
	})

	// None of A's writes above may have created anything pointing into B.
	t.Run("nothing of A references B", func(t *testing.T) {
		for _, p := range []string{
			"/api/v1/staff?all=true", "/api/v1/clients?all=true", "/api/v1/projects?all=true", "/api/v1/projects?archived=true",
			proj(w.aProject, "/events"), proj(w.aProject, "/suppliers"), proj(w.aProject, "/supplier-payments"),
			proj(w.aProject, "/issues"), proj(w.aProject, "/evidence"), "/api/v1/finance/payables",
		} {
			r := s.mustOK(p, s.do(a.token, "GET", p, nil))
			w.checkNoLeak(t, "A GET "+p, r)
		}
		for _, sup := range rows(s.mustOK("A suppliers", s.do(a.token, "GET", proj(w.aProject, "/suppliers"), nil)), "data") {
			if id, _ := sup["supplierId"].(float64); int64(id) == w.Vendor || int64(id) == w.Venue {
				t.Errorf("A's project lists B's vendor/venue %d as a supplier: %v", int64(id), sup)
			}
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
		{name: "B's other project suppliers", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/suppliers", w.Project2)},
		{name: "B's other project events", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/events", w.Project2)},
		{name: "B's other project supplier payments", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/supplier-payments", w.Project2)},
		{name: "a WO supplier payment", method: "PATCH", path: fmt.Sprintf("/api/v1/projects/%d/supplier-payments/%d", w.Project, w.SupplierPay), body: map[string]any{"amount": 1}},
		{name: "record a supplier payment", method: "POST", path: fmt.Sprintf("/api/v1/projects/%d/supplier-payments", w.Project), body: map[string]any{"projectSupplierId": w.Supplier, "type": "DP", "amount": 1}},
		{name: "add a supplier", method: "POST", path: fmt.Sprintf("/api/v1/projects/%d/suppliers", w.Project), body: map[string]any{"vendorId": w.Vendor, "paidBy": "Client"}},
		{name: "add an event", method: "POST", path: fmt.Sprintf("/api/v1/projects/%d/events", w.Project), body: eventOf(w.Kind, day(121), 10, 0)},
		{name: "payables", method: "GET", path: "/api/v1/finance/payables"},
		{name: "project activity log", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/activity", w.Project)},
		{name: "closing checklist", method: "GET", path: fmt.Sprintf("/api/v1/projects/%d/closing-checklist", w.Project)},
		{name: "edit the project", method: "PATCH", path: fmt.Sprintf("/api/v1/projects/%d", w.Project), body: map[string]any{"name": "x"}},
		{name: "create a project", method: "POST", path: "/api/v1/projects", body: map[string]any{"name": "x"}},
		{name: "record a payment", method: "POST", path: fmt.Sprintf("/api/v1/projects/%d/client-payments", w.Project), body: map[string]any{"type": "DP", "amount": 1}},
		{name: "show a document to the client", method: "PATCH", path: fmt.Sprintf("/api/v1/projects/%d/evidence/%d", w.Project, w.EvidenceInternal), body: map[string]any{"isClientVisible": true}},
		{name: "archive the project", method: "PATCH", path: fmt.Sprintf("/api/v1/projects/%d", w.Project), body: map[string]any{"isArchived": true}},
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
		{"own events", fmt.Sprintf("/api/v1/projects/%d/events", w.Project)},
		{"own suppliers", fmt.Sprintf("/api/v1/projects/%d/suppliers", w.Project)},
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
	// The client of B's project reads its suppliers and their payments, but the
	// WO's own costs (what the WO pays the vendor and the venue) stay internal.
	t.Run("client never sees WO supplier payments", func(t *testing.T) {
		r := s.do(c, "GET", fmt.Sprintf("/api/v1/projects/%d/supplier-payments", w.Project), nil)
		if r.status != http.StatusOK && r.status != http.StatusForbidden {
			t.Fatalf("client GET supplier-payments: %s", r)
		}
		for _, x := range []string{"VREF-" + w.b.marker, "VNREF-" + w.b.marker, "Rahasia " + w.b.marker, "777777"} {
			if strings.Contains(string(r.body), x) {
				t.Errorf("client sees the WO's supplier payment (%q): %s", x, r)
			}
		}
		if rowByID(rows(r, "data"), w.SupplierPay) != nil || rowByID(rows(r, "data"), w.VenuePay) != nil {
			t.Errorf("client list carries a WO payment row: %s", r)
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
	check("supplier", fmt.Sprintf("/api/v1/projects/%d/suppliers", P), "Dekorasi "+b.marker, "Venue B "+b.marker)
	check("supplier milestone (embedded in the suppliers)", fmt.Sprintf("/api/v1/projects/%d/suppliers", P), "Milestone vendor "+b.marker)
	check("supplier and venue payments", fmt.Sprintf("/api/v1/projects/%d/supplier-payments", P), "VREF-"+b.marker, "VNREF-"+b.marker)
	check("events", fmt.Sprintf("/api/v1/projects/%d/events", P), "Venue B "+b.marker, fmt.Sprintf(`"id":%d`, w.Event))
	check("client payments", fmt.Sprintf("/api/v1/projects/%d/client-payments", P), "REF-"+b.marker)
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
