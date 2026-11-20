//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestFinance covers GET /finance/summary, /receivables, /payables and
// /verifications (Keuangan page) on a real database: the SQL allocation of each
// installment equals GET /payment-terms (the Go rule), the views and sorts, the
// headline figures, the cash flow equal to /dashboard's, the scope of a role
// limited to its own projects, that what the WO owes its suppliers needs
// finance.margin.view, and that another workspace never shows up.
func TestFinance(t *testing.T) {
	s := startStack(t)
	x, y := s.signUp("finx"), s.signUp("finy")
	wpTok, wpID := s.staffAs(x, "Wedding Planner")

	// ---- workspace Y: an overdue installment and a supplier balance that must never appear in X ----
	yP := s.projectAt(y, "Proyek "+y.marker, 70_000_000, 90, 0)
	s.setTerms(y, yP, term("DP "+y.marker, 20_000_000, -10))
	yVendor, _ := s.newVendor(y, "Vendor "+y.marker)
	s.engage(y, yP, yVendor, "Dekor "+y.marker, 9_000_000, "Booked", ptr(-1))

	// ---- workspace X ----
	// P1 (Owner's): DP 30 paid, Termin 1 overdue with 5 paid, Termin 2 due in 10 days.
	p1 := s.projectAt(x, "Proyek satu", 100_000_000, 120, 0)
	s.setTerms(x, p1, term("DP", 30_000_000, -20), term("Termin 1", 40_000_000, -3), term("Termin 2", 30_000_000, 10))
	pay := s.mustOK("client payment", s.do(x.token, "POST", s.projectsPath(p1, "/client-payments"), map[string]any{
		"type": "DP", "amount": 35_000_000, "paymentDate": day(-5), "method": "Transfer", "referenceNumber": "REF-FIN-1",
	})).id("data", "id")
	// A Tambahan is money received but does not pay an installment.
	s.mustOK("tambahan", s.do(x.token, "POST", s.projectsPath(p1, "/client-payments"), map[string]any{
		"type": "Tambahan", "amount": 2_000_000, "paymentDate": day(-40), "method": "Transfer", "referenceNumber": "REF-FIN-OLD",
	}))
	// Its proof makes it wait for verification.
	s.mustOK("proof", s.do(x.token, "POST", s.projectsPath(p1, "/evidence"), map[string]any{
		"name": "Bukti DP", "type": "Transfer Proof", "fileName": "bukti.pdf", "mimeType": "application/pdf", "base64Data": pdf,
		"documentDate": day(-5), "description": "", "relatedKind": "clientPayment", "relatedId": pay, "isClientVisible": false,
	}))
	vendor, _ := s.newVendor(x, "Vendor satu")
	eng := s.engage(x, p1, vendor, "Dekor", 5_000_000, "Booked", ptr(-2))
	s.supplierPay(x, p1, eng, "DP", 1_000_000, "VREF-FIN-1")
	// A supplier the client pays is never the WO's debt, whatever it is paid.
	clientPaid := s.engageBy(x, p1, vendor, "Dibayar klien", "Client", false)
	s.supplierPay(x, p1, clientPaid, "DP", 3_000_000, "CREF-FIN-1")
	// A supplier without an agreed cost is listed as incomplete (owed, amount
	// unknown); a cancelled one is not a payable.
	noCost := s.engage(x, p1, vendor, "Tanpa harga", 0, "Planned", nil)
	gone := s.engage(x, p1, vendor, "Batal", 7_000_000, "Booked", ptr(3))
	s.mustOK("cancel supplier", finPatchSupplier(s, x, p1, gone, map[string]any{"engagementStatus": "Cancelled"}))

	// P2 (the planner's, and the scoped finance person's as PIC Sales): its event
	// at a venue brings the venue in; the venue's agreed cost (rental + charge,
	// 10.5 jt, no due date) is what the WO owes it. Two installments not due yet.
	scopedTok, scopedID := finScopedStaff(s, x)
	venue := s.mustOK("venue", s.do(x.token, "POST", "/api/v1/venues", venueBody(x, "Venue dua"))).id("data", "id")
	p2Body := s.projectBody(x, "Proyek dua", 200, eventOf(s.kind(x, "Resepsi"), day(200), 200, venue))
	p2Body["brideName"], p2Body["groomName"], p2Body["picStaffId"], p2Body["picSalesStaffId"] = "Bride dua", "Groom dua", wpID, scopedID
	p2 := s.mustOK("project dua", s.do(x.token, "POST", "/api/v1/projects", p2Body)).id("data", "id")
	s.setContract(x, p2, 50_000_000)
	venueSup := s.venueSupplier(x, p2, venue)
	if venueSup == 0 {
		t.Fatalf("the event's venue is not among P2's suppliers")
	}
	s.mustOK("venue agreed cost", finPatchSupplier(s, x, p2, venueSup, map[string]any{"agreedCost": 10_500_000}))
	s.setTerms(x, p2, term("DP", 20_000_000, 5), term("Pelunasan", 30_000_000, 60))

	// P3 cancelled and P4 archived: both left out everywhere.
	p3 := s.projectAt(x, "Proyek batal", 10_000_000, 30, 0)
	s.setTerms(x, p3, term("DP", 10_000_000, -5))
	s.mustOK("cancel p3", s.do(x.token, "POST", s.projectsPath(p3, "/cancel"), map[string]any{}))
	p4 := s.projectAt(x, "Proyek arsip", 10_000_000, 30, 0)
	s.setTerms(x, p4, term("DP", 10_000_000, -5))
	s.mustOK("archive p4", s.do(x.token, "PATCH", s.projectsPath(p4, ""), map[string]any{"isArchived": true}))

	noLeak := func(t *testing.T, what string, r reply) {
		t.Helper()
		if strings.Contains(string(r.body), y.marker) {
			t.Errorf("%s shows workspace Y: %s", what, r)
		}
	}

	t.Run("receivables equal GET /payment-terms", func(t *testing.T) {
		r := s.mustOK("receivables", s.do(x.token, "GET", "/api/v1/finance/receivables?limit=100", nil))
		noLeak(t, "receivables", r)
		got := rows(r, "data")
		if len(got) != 5 || r.dig("meta", "total") != float64(5) {
			t.Fatalf("want the 5 installments of P1 and P2, got %d: %s", len(got), r)
		}
		byID := map[float64]map[string]any{}
		for _, m := range got {
			byID[m["id"].(float64)] = m
		}
		for _, p := range []int64{p1, p2} {
			terms := s.mustOK("terms", s.do(x.token, "GET", s.projectsPath(p, "/payment-terms"), nil))
			for _, want := range rows(terms, "data") {
				row := byID[want["id"].(float64)]
				if row == nil {
					t.Errorf("term %v missing from the finance list", want)
					continue
				}
				for _, k := range []string{"label", "amount", "dueDate", "paid", "remaining", "status"} {
					if !reflect.DeepEqual(row[k], want[k]) {
						t.Errorf("term %v %s: finance %v, payment-terms %v", want["id"], k, row[k], want[k])
					}
				}
				if row["projectId"] != float64(p) {
					t.Errorf("term %v projectId %v", want["id"], row["projectId"])
				}
			}
		}
		// Default order: earliest due date first.
		if got[0]["label"] != "DP" || got[0]["projectId"] != float64(p1) || got[1]["label"] != "Termin 1" {
			t.Errorf("default order: %v, %v", got[0], got[1])
		}
		if got[1]["status"] != "overdue" || got[1]["daysLate"] != float64(3) || got[1]["remaining"] != float64(35_000_000) {
			t.Errorf("Termin 1: %v", got[1])
		}
	})

	t.Run("receivable views, filters and sort", func(t *testing.T) {
		labels := func(path string) []string {
			r := s.mustOK(path, s.do(x.token, "GET", path, nil))
			noLeak(t, path, r)
			var out []string
			for _, m := range rows(r, "data") {
				out = append(out, fmt.Sprintf("%v:%v", m["projectId"], m["label"]))
			}
			return out
		}
		check := func(path string, want ...string) {
			t.Helper()
			if got := labels(path); !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got %v, want %v", path, got, want)
			}
		}
		one, two := fmt.Sprint(p1), fmt.Sprint(p2)
		check("/api/v1/finance/receivables?status=overdue", one+":Termin 1")
		check("/api/v1/finance/receivables?status=due_30d", two+":DP", one+":Termin 2")
		check("/api/v1/finance/receivables?status=paid", one+":DP")
		check("/api/v1/finance/receivables?status=unpaid&sort=remaining:desc", one+":Termin 1", two+":Pelunasan", one+":Termin 2", two+":DP")
		check("/api/v1/finance/receivables?q=dua", two+":DP", two+":Pelunasan")
		check(fmt.Sprintf("/api/v1/finance/receivables?picStaffId=%d", wpID), two+":DP", two+":Pelunasan")
		check(fmt.Sprintf("/api/v1/finance/receivables?dueFrom=%s&dueTo=%s", day(-4), day(6)), one+":Termin 1", two+":DP")
		page := s.mustOK("page 2", s.do(x.token, "GET", "/api/v1/finance/receivables?limit=2&page=2", nil))
		if len(rows(page, "data")) != 2 || page.dig("meta", "total_pages") != float64(3) || page.dig("meta", "sort") != "dueDate:asc" {
			t.Errorf("page 2: %s", page)
		}
		if r := s.do(x.token, "GET", "/api/v1/finance/receivables?status=lunas&sort=label", nil); r.status != 422 {
			t.Errorf("bad status: %s", r)
		}
	})

	t.Run("payables", func(t *testing.T) {
		r := s.mustOK("payables", s.do(x.token, "GET", "/api/v1/finance/payables", nil))
		noLeak(t, "payables", r)
		got := rows(r, "data")
		if len(got) != 3 {
			t.Fatalf("want the vendor, the venue and the one without a cost, got %s", r)
		}
		// Most still owed first: the venue (10.5 jt), then the vendor (4 jt,
		// overdue), then the supplier without an agreed cost (nothing known).
		venue, vend, incomplete := got[0], got[1], got[2]
		if incomplete["id"] != float64(noCost) || incomplete["incomplete"] != true || incomplete["status"] != "incomplete" ||
			incomplete["agreedCost"] != float64(0) || incomplete["remaining"] != float64(0) {
			t.Errorf("a WO-paid supplier without a cost is listed as incomplete: %v", incomplete)
		}
		if vend["incomplete"] != false || venue["incomplete"] != false {
			t.Errorf("rows with a cost are complete: %v / %v", venue, vend)
		}
		if k := rows(s.mustOK("incomplete", s.do(x.token, "GET", "/api/v1/finance/payables?status=incomplete", nil)), "data"); len(k) != 1 || k[0]["id"] != float64(noCost) {
			t.Errorf("status=incomplete: %v", k)
		}
		if k := rows(s.mustOK("open", s.do(x.token, "GET", "/api/v1/finance/payables?status=open", nil)), "data"); len(k) != 3 {
			t.Errorf("status=open includes the incomplete one: %v", k)
		}
		if venue["kind"] != "venue" || venue["remaining"] != float64(10_500_000) || venue["status"] != "unpaid" || venue["dueDate"] != nil || venue["projectId"] != float64(p2) {
			t.Errorf("venue row: %v", venue)
		}
		if vend["kind"] != "vendor" || vend["payeeName"] != "Vendor satu" || vend["paid"] != float64(1_000_000) || vend["remaining"] != float64(4_000_000) ||
			vend["status"] != "overdue" || vend["daysLate"] != float64(2) || vend["id"] != float64(eng) {
			t.Errorf("vendor row: %v", vend)
		}
		if k := rows(s.mustOK("venue only", s.do(x.token, "GET", "/api/v1/finance/payables?kind=venue", nil)), "data"); len(k) != 1 || k[0]["kind"] != "venue" {
			t.Errorf("kind=venue: %v", k)
		}
		if k := rows(s.mustOK("overdue", s.do(x.token, "GET", "/api/v1/finance/payables?status=overdue", nil)), "data"); len(k) != 1 || k[0]["kind"] != "vendor" {
			t.Errorf("status=overdue: %v", k)
		}
		if k := rows(s.mustOK("paid", s.do(x.token, "GET", "/api/v1/finance/payables?status=paid", nil)), "data"); len(k) != 0 {
			t.Errorf("status=paid: %v", k)
		}
	})

	t.Run("summary and cash flow", func(t *testing.T) {
		r := s.mustOK("summary", s.do(x.token, "GET", "/api/v1/finance/summary", nil))
		noLeak(t, "summary", r)
		want := map[string]map[string]any{
			"overdue":             {"amount": 35_000_000.0, "count": 1.0, "projects": 1.0},
			"dueSoon":             {"amount": 50_000_000.0, "count": 2.0, "projects": 2.0},
			"received":            {"amount": 35_000_000.0, "count": 1.0, "projects": 0.0},
			"payables":            {"amount": 14_500_000.0, "count": 2.0, "overdueAmount": 4_000_000.0, "overdueCount": 1.0, "incompleteCount": 1.0},
			"pendingVerification": {"amount": 35_000_000.0, "count": 1.0, "projects": 1.0},
		}
		for key, fields := range want {
			for f, v := range fields {
				if got := r.dig("data", key, f); got != v {
					t.Errorf("summary %s.%s = %v, want %v", key, f, got, v)
				}
			}
		}
		// The same weeks as the dashboard (no completed project here, so the two scopes match).
		dash := s.mustOK("dashboard", s.do(x.token, "GET", "/api/v1/dashboard", nil))
		if !reflect.DeepEqual(r.dig("data", "cashFlow"), dash.dig("data", "cashFlow")) {
			t.Errorf("cash flow differs from /dashboard:\n%v\n%v", r.dig("data", "cashFlow"), dash.dig("data", "cashFlow"))
		}
	})

	t.Run("verifications", func(t *testing.T) {
		r := s.mustOK("verifications", s.do(x.token, "GET", "/api/v1/finance/verifications", nil))
		got := rows(r, "data")
		if len(got) != 1 || got[0]["id"] != float64(pay) || got[0]["projectId"] != float64(p1) || got[0]["referenceNumber"] != "REF-FIN-1" {
			t.Fatalf("verifications: %s", r)
		}
		s.mustOK("verify", s.do(x.token, "POST", s.projectsPath(p1, fmt.Sprintf("/client-payments/%d/verify", pay)), nil))
		if again := rows(s.mustOK("verifications", s.do(x.token, "GET", "/api/v1/finance/verifications", nil)), "data"); len(again) != 0 {
			t.Errorf("a verified payment still waits: %v", again)
		}
	})

	t.Run("a role limited to its own projects", func(t *testing.T) {
		r := s.mustOK("wp receivables", s.do(wpTok, "GET", "/api/v1/finance/receivables", nil))
		for _, m := range rows(r, "data") {
			if m["projectId"] != float64(p2) {
				t.Errorf("planner sees a project they do not hold: %v", m)
			}
		}
		if len(rows(r, "data")) != 2 {
			t.Errorf("planner: %s", r)
		}
		// Asking for the Owner's projects narrows further, never widens.
		if mine := rows(s.mustOK("wp pic", s.do(wpTok, "GET", fmt.Sprintf("/api/v1/finance/receivables?picStaffId=%d", x.staffID), nil)), "data"); len(mine) != 0 {
			t.Errorf("planner with the Owner's picStaffId: %v", mine)
		}
		// A role with finance.margin.view and the same narrow scope sees only the venue of P2.
		sum := s.mustOK("scoped summary", s.do(scopedTok, "GET", "/api/v1/finance/summary", nil))
		if sum.dig("data", "overdue", "amount") != 0.0 || sum.dig("data", "payables", "amount") != 10_500_000.0 {
			t.Errorf("scoped summary: %s", sum)
		}
		pays := rows(s.mustOK("scoped payables", s.do(scopedTok, "GET", "/api/v1/finance/payables", nil)), "data")
		if len(pays) != 1 || pays[0]["kind"] != "venue" || pays[0]["projectId"] != float64(p2) {
			t.Errorf("scoped payables: %v", pays)
		}
	})

	t.Run("what the WO owes its suppliers needs finance.margin.view", func(t *testing.T) {
		// The Wedding Planner records payments (finance.record) but has no finance.margin.view.
		r := s.do(wpTok, "GET", "/api/v1/finance/payables", nil)
		wantStatus(t, "planner payables", r, http.StatusForbidden)
		if r.dig("errors", "code") != "forbidden_permission" {
			t.Errorf("planner payables: want forbidden_permission: %s", r)
		}
		if strings.Contains(string(r.body), "Venue dua") || strings.Contains(string(r.body), "10500000") {
			t.Errorf("refusal carries the payable: %s", r)
		}
		sum := s.mustOK("wp summary", s.do(wpTok, "GET", "/api/v1/finance/summary", nil))
		if _, present := data(sum)["payables"]; present {
			t.Errorf("summary without finance.margin.view carries payables: %s", sum)
		}
		for _, wk := range rows(sum, "data", "cashFlow") {
			if out, _ := wk["expectedOut"].(float64); out != 0 {
				t.Errorf("summary without finance.margin.view shows money out: %v", wk)
			}
		}
		if sum.dig("data", "dueSoon", "amount") != 20_000_000.0 {
			t.Errorf("planner still sees what is owed to the WO: %s", sum)
		}
		// The same narrow scope with the permission does see it (the refusal is the permission, not the scope).
		if r := s.do(scopedTok, "GET", "/api/v1/finance/payables", nil); r.status != http.StatusOK {
			t.Errorf("scoped role with finance.margin.view: %s", r)
		}
		// A role with every finance permission but finance.margin.view, across all projects.
		noMargin := s.staffWith(x, "Kasir", "finance.record", "finance.payment.delete")
		wantStatus(t, "no-margin payables", s.do(noMargin, "GET", "/api/v1/finance/payables", nil), http.StatusForbidden)
		ns := s.mustOK("no-margin summary", s.do(noMargin, "GET", "/api/v1/finance/summary", nil))
		if _, present := data(ns)["payables"]; present {
			t.Errorf("no-margin summary carries payables: %s", ns)
		}
		if ns.dig("data", "overdue", "amount") != 35_000_000.0 {
			t.Errorf("no-margin summary lost the receivables: %s", ns)
		}
	})

	t.Run("workspace Y sees only its own", func(t *testing.T) {
		r := s.mustOK("y receivables", s.do(y.token, "GET", "/api/v1/finance/receivables", nil))
		if got := rows(r, "data"); len(got) != 1 || got[0]["projectId"] != float64(yP) {
			t.Errorf("Y receivables: %s", r)
		}
		for _, path := range []string{"/api/v1/finance/summary", "/api/v1/finance/payables", "/api/v1/finance/verifications",
			fmt.Sprintf("/api/v1/finance/receivables?picStaffId=%d", y.staffID), "/api/v1/finance/receivables?q=" + y.marker} {
			noLeak(t, path, s.mustOK(path, s.do(x.token, "GET", path, nil)))
		}
	})
}

func ptr(v int) *int { return &v }

// finPatchSupplier edits one of the project's suppliers as the list shows it
// now (its updatedAt as If-Match).
func finPatchSupplier(s *stack, t tenant, projectID, supplierID int64, body map[string]any) reply {
	s.t.Helper()
	var version string
	for _, m := range rows(s.mustOK("suppliers", s.do(t.token, "GET", s.projectsPath(projectID, "/suppliers"), nil)), "data") {
		if m["id"] == float64(supplierID) {
			version, _ = m["updatedAt"].(string)
		}
	}
	if version == "" {
		s.t.Fatalf("supplier %d not on project %d", supplierID, projectID)
	}
	return s.doWith(t.token, "PATCH", s.projectsPath(projectID, fmt.Sprintf("/suppliers/%d", supplierID)), body,
		map[string]string{"If-Match": `"` + version + `"`})
}

// finScopedStaff signs in a staff member whose role holds finance.record and
// finance.margin.view but reaches only the projects they hold.
func finScopedStaff(s *stack, t tenant) (string, int64) {
	s.t.Helper()
	role := s.mustOK("scoped role", s.do(t.token, "POST", "/api/v1/access/roles", map[string]any{
		"kind": "staff", "name": "Keuangan proyek " + t.marker, "description": "uji", "projectScope": "assigned",
		"permissions": []string{"finance.record", "finance.margin.view"},
	})).id("data", "id")
	email := fmt.Sprintf("keuangan.proyek.%d@example.com", time.Now().UnixNano())
	st := s.mustOK("scoped staff", s.do(t.token, "POST", "/api/v1/staff", map[string]any{
		"name": "Keuangan proyek", "title": "Uji", "roleId": role, "email": email, "phone": "081234567890",
	}))
	sess := s.mustOK("scoped staff accepts", s.do("", "POST", "/api/v1/auth/links/"+str(st, "data", "invitation", "token")+"/google", map[string]any{
		"credential": s.google.token(email, "Keuangan proyek", fmt.Sprintf("g-sub-fin-%d", time.Now().UnixNano())),
	}))
	id, _ := strconv.ParseInt(fmt.Sprint(sess.dig("data", "principalId")), 10, 64)
	tok := str(sess, "data", "accessToken")
	if tok == "" || id == 0 {
		s.t.Fatalf("no session for the scoped finance staff: %s", sess)
	}
	return tok, id
}
