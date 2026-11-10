//go:build integration

package isolation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConcurrentSignups: workspaces registered at the same moment must all
// succeed. The provisioning saga used to deadlock in MySQL (1213) when two
// new tenants seeded their defaults at once, and the losing sign-up got a 500.
func TestConcurrentSignups(t *testing.T) {
	s := startStack(t)
	const n = 8
	// Two rounds: with the Owner that is the three staff a trial workspace allows.
	for round := 0; round < 2; round++ {
		var wg sync.WaitGroup
		results := make([]string, n)
		start := make(chan struct{})
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				name := fmt.Sprintf("r%dw%d", round, i)
				email := fmt.Sprintf("owner.%s.%d@example.com", name, time.Now().UnixNano())
				sub := fmt.Sprintf("g-sub-%s-%d", name, time.Now().UnixNano())
				body, _ := json.Marshal(map[string]any{
					"credential": s.google.token(email, "Owner "+name, sub), "businessName": "WO Bersamaan " + name, "acceptTerms": true,
				})
				req, _ := http.NewRequest("POST", s.base+"/api/v1/public/signup/workspace", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.99.%d.%d", round, i+1))
				<-start
				resp, err := s.client.Do(req)
				if err != nil {
					results[i] = "transport: " + err.Error()
					return
				}
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				if resp.StatusCode < 200 || resp.StatusCode > 299 {
					results[i] = fmt.Sprintf("%d %s", resp.StatusCode, tail(string(b), 200))
				}
			}(i)
		}
		close(start)
		wg.Wait()
		for i, r := range results {
			if r != "" {
				t.Errorf("round %d sign-up %d failed: %s", round, i, r)
			}
		}
	}
	var incomplete int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM signup_requests WHERE status <> 'completed'`).Scan(&incomplete); err != nil {
		t.Fatal(err)
	}
	if incomplete != 0 {
		t.Errorf("%d sign-ups left half-provisioned", incomplete)
	}
}

// burst sends n requests at the same moment and returns, per request, "" on a
// 2xx or what went wrong. It never calls t.Fatal (it runs off the test goroutine).
func (s *stack) burst(n int, req func(i int) (token, method, path string, body any)) []string {
	results := make([]string, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		token, method, path, body := req(i)
		b, _ := json.Marshal(body)
		r, _ := http.NewRequest(method, s.base+path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("10.98.%d.%d", (i/250)%250, i%250+1))
		wg.Add(1)
		go func(i int, r *http.Request) {
			defer wg.Done()
			<-start
			resp, err := s.client.Do(r)
			if err != nil {
				results[i] = "transport: " + err.Error()
				return
			}
			defer resp.Body.Close()
			out, _ := io.ReadAll(resp.Body)
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				results[i] = fmt.Sprintf("%d %s", resp.StatusCode, tail(string(out), 200))
			}
		}(i, r)
	}
	close(start)
	wg.Wait()
	return results
}

// TestConcurrentStaffInvites: several workspaces inviting a team member at the
// same moment all succeed. Issuing the invitation link used to deadlock (1213)
// on the auth_tokens gap lock, and one invitation got a 500. Re-sending one
// person's invitation several times at once still leaves exactly one open link.
func TestConcurrentStaffInvites(t *testing.T) {
	s := startStack(t)
	const n = 6
	owners := make([]tenant, n)
	roles := make([]int64, n)
	for i := range owners {
		owners[i] = s.signUp(fmt.Sprintf("undang%d", i))
		roles[i] = s.mustOK("role", s.do(owners[i].token, "POST", "/api/v1/access/roles", map[string]any{
			"kind": "staff", "name": "Peran " + owners[i].marker, "description": "Uji", "projectScope": "all", "permissions": []string{"project.edit"},
		})).id("data", "id")
	}
	// Two rounds: with the Owner that is the three staff a trial workspace allows.
	for round := 0; round < 2; round++ {
		for i, r := range s.burst(n, func(i int) (string, string, string, any) {
			return owners[i].token, "POST", "/api/v1/staff", map[string]any{
				"name": fmt.Sprintf("Staf %d-%d", round, i), "title": "Koordinator", "roleId": roles[i], "phone": "081234567890",
				"email": fmt.Sprintf("staf.%d.%d.%d@example.com", round, i, time.Now().UnixNano()),
			}
		}) {
			if r != "" {
				t.Errorf("round %d invitation %d failed: %s", round, i, r)
			}
		}
	}

	// One person, several re-sends at once.
	o := owners[0]
	staff := s.mustOK("staff list", s.do(o.token, "GET", "/api/v1/staff?limit=50", nil))
	var staffID int64
	for _, row := range rows(staff, "data") {
		if row["loginStatus"] == "invited" {
			staffID = int64(row["id"].(float64))
			break
		}
	}
	if staffID == 0 {
		t.Fatalf("no invited staff member to re-invite: %s", staff)
	}
	for i, r := range s.burst(5, func(int) (string, string, string, any) {
		return o.token, "POST", fmt.Sprintf("/api/v1/staff/%d/invite", staffID), map[string]any{}
	}) {
		if r != "" {
			t.Errorf("re-send %d failed: %s", i, r)
		}
	}
	var open int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM auth_tokens t JOIN credentials c ON c.id = t.credential_id
		WHERE c.principal_type = 'staff' AND c.principal_id = ? AND t.used_at IS NULL AND t.revoked_at IS NULL`, fmt.Sprint(staffID)).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Errorf("after concurrent re-sends: %d open invitation links, want 1", open)
	}
}

// TestCreatedItemVersion: the `updatedAt` a create answers is the row's real
// version (ADR-0036), so editing straight from that answer with it as If-Match
// succeeds. Every versioned item of a project used to answer the zero time
// there, and that edit was a 409.
func TestCreatedItemVersion(t *testing.T) {
	s := startStack(t)
	o := s.signUp("Versi")

	// check creates an item, compares the version it answers with the one the
	// item's list (or detail) shows, edits it with that version and returns its id.
	check := func(what, createPath string, body map[string]any, itemPath func(id int64) string, edit map[string]any) int64 {
		t.Helper()
		created := s.mustOK("create "+what, s.do(o.token, "POST", createPath, body))
		id := created.id("data", "id")
		v, _ := created.dig("data", "updatedAt").(string)
		if v == "" || strings.HasPrefix(v, "0001-") {
			t.Errorf("%s: create answered updatedAt %q, want the row's version", what, v)
			return id
		}
		path := itemPath(id)
		if listed := s.versionOf(o.token, path); listed != v {
			t.Errorf("%s: create answered updatedAt %q, a read says %q", what, v, listed)
		}
		if r := s.patch(o.token, path, edit, quoted(v)); r.status != http.StatusOK {
			t.Errorf("%s: edit with the version from create: want 200, got %s", what, r)
		}
		return id
	}

	project := check("project", "/api/v1/projects", map[string]any{
		"name": "Proyek " + o.marker, "brideName": "Bride", "groomName": "Groom", "eventDate": day(120), "venue": "Gedung",
		"prepStartDate": day(1), "packageName": "Paket", "contractValue": 50_000_000, "status": "Preparation", "picStaffId": o.staffID,
	}, func(id int64) string { return fmt.Sprintf("/api/v1/projects/%d", id) }, map[string]any{"description": "Diubah langsung"})
	in := func(rest string) string { return fmt.Sprintf("/api/v1/projects/%d/%s", project, rest) }
	item := func(kind string) func(int64) string {
		return func(id int64) string { return in(fmt.Sprintf("%s/%d", kind, id)) }
	}

	check("project milestone", in("milestones"), map[string]any{
		"name": "Timeline tambahan", "targetDate": day(40),
	}, item("milestones"), map[string]any{"targetDate": day(41)})

	category := s.mustOK("category", s.do(o.token, "POST", "/api/v1/vendor-categories", map[string]any{
		"name": "Kategori " + o.marker, "description": "Uji",
	})).id("data", "id")
	vendor := s.mustOK("vendor", s.do(o.token, "POST", "/api/v1/vendors", vendorBody(s, o, "Vendor "+o.marker, category))).id("data", "id")
	engagement := check("project vendor", in("vendors"), map[string]any{
		"vendorId": vendor, "categoryId": category, "scope": "Dekorasi", "contractValue": 5_000_000,
		"pricingTier": "Akad", "engagementStatus": "Booked", "eventDate": day(120), "dpAmount": 1_000_000, "dueDate": day(60),
		"picStaffId": o.staffID,
	}, item("vendors"), map[string]any{"notes": "Diubah langsung"})
	check("vendor milestone", in(fmt.Sprintf("vendors/%d/milestones", engagement)), map[string]any{
		"name": "Milestone vendor", "targetDate": day(30), "picStaffId": o.staffID,
	}, item(fmt.Sprintf("vendors/%d/milestones", engagement)), map[string]any{"notes": "Diubah langsung"})
	check("issue", in("issues"), map[string]any{
		"projectVendorId": engagement, "title": "Kendala", "description": "Uji", "impact": "Medium", "foundDate": day(-1),
		"resolutionPlan": "Rencana", "picStaffId": o.staffID, "targetResolutionDate": day(10),
	}, item("issues"), map[string]any{"description": "Diubah langsung"})
	check("client payment", in("client-payments"), map[string]any{
		"type": "DP", "amount": 4_000_000, "paymentDate": day(-2), "method": "Transfer",
	}, item("client-payments"), map[string]any{"notes": "Diubah langsung"})
	check("vendor payment", in("payments"), map[string]any{
		"projectVendorId": engagement, "type": "DP", "amount": 1_000_000, "paymentDate": day(-2), "method": "Transfer",
	}, item("payments"), map[string]any{"notes": "Diubah langsung"})
	check("venue payment", in("venue-payments"), map[string]any{
		"type": "DP", "amount": 700_000, "paymentDate": day(-2), "method": "Transfer",
	}, item("venue-payments"), map[string]any{"notes": "Diubah langsung"})
}

// TestClientUnknownRelation: a client relation outside the three the product
// knows is a 422 on `role`, not a 500 from the database enum.
func TestClientUnknownRelation(t *testing.T) {
	s := startStack(t)
	o := s.signUp("Relasi")
	project := s.createProject(o, "Proyek "+o.marker, 50_000_000)
	for _, role := range []string{"Family", "", "bride"} {
		r := s.do(o.token, "POST", "/api/v1/clients", map[string]any{
			"projectId": project, "role": role, "name": "Klien " + o.marker, "phone": "081298765432",
			"username": fmt.Sprintf("klien.rel.%d", time.Now().UnixNano()%1_000_000_000),
		})
		if r.status != http.StatusUnprocessableEntity {
			t.Errorf("role %q: want 422, got %s", role, r)
			continue
		}
		if errs, _ := r.dig("errors", "role").([]any); len(errs) == 0 {
			t.Errorf("role %q: want a message under errors.role, got %s", role, r)
		}
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM clients`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("a refused client left %d rows behind", n)
	}
	// A known relation still works.
	s.mustOK("valid client", s.do(o.token, "POST", "/api/v1/clients", map[string]any{
		"projectId": project, "role": "Family Representative", "name": "Klien " + o.marker, "phone": "081298765432",
		"username": fmt.Sprintf("klien.rel.%d", time.Now().UnixNano()%1_000_000_000),
	}))
}
