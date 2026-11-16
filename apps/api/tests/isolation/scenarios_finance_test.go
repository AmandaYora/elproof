//go:build integration

package isolation

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestFinance covers GET /finance/summary, /receivables, /payables and
// /verifications (Keuangan page) on a real database: the SQL allocation of each
// installment equals GET /payment-terms (the Go rule), the views and sorts, the
// headline figures, the cash flow equal to /dashboard's, the scope of a role
// limited to its own projects, and that another workspace never shows up.
func TestFinance(t *testing.T) {
	s := startStack(t)
	x, y := s.signUp("finx"), s.signUp("finy")
	wpTok, wpID := s.staffAs(x, "Wedding Planner")

	// ---- workspace Y: an overdue installment and a vendor balance that must never appear in X ----
	yP := s.projectAt(y, "Proyek "+y.marker, 70_000_000, 90, 3_000_000, 0)
	s.setTerms(y, yP, term("DP "+y.marker, 20_000_000, -10))
	yVendor, yCat := s.newVendor(y, "Vendor "+y.marker)
	s.engage(y, yP, yVendor, yCat, "Dekor "+y.marker, 9_000_000, "Booked", ptr(-1))

	// ---- workspace X ----
	// P1 (Owner's): DP 30 paid, Termin 1 overdue with 5 paid, Termin 2 due in 10 days.
	p1 := s.projectAt(x, "Proyek satu", 100_000_000, 120, 0, 0)
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
	vendor, cat := s.newVendor(x, "Vendor satu")
	eng := s.engage(x, p1, vendor, cat, "Dekor", 5_000_000, "Booked", ptr(-2))
	s.mustOK("vendor payment", s.do(x.token, "POST", s.projectsPath(p1, "/payments"), map[string]any{
		"projectVendorId": eng, "type": "DP", "amount": 1_000_000, "paymentDate": day(-2), "method": "Transfer",
	}))
	// An engagement without a price is not a payable; a cancelled one neither.
	s.engage(x, p1, vendor, cat, "Tanpa harga", 0, "Planned", nil)
	gone := s.engage(x, p1, vendor, cat, "Batal", 7_000_000, "Booked", ptr(3))
	s.mustOK("cancel engagement", s.do(x.token, "POST", s.projectsPath(p1, fmt.Sprintf("/vendors/%d/cancel", gone)), map[string]any{}))

	// P2 (the planner's): a venue cost and two installments not due yet.
	venue := s.mustOK("venue", s.do(x.token, "POST", "/api/v1/venues", venueBody(x, "Venue dua"))).id("data", "id")
	p2 := s.mustOK("project dua", s.do(x.token, "POST", "/api/v1/projects", map[string]any{
		"name": "Proyek dua", "brideName": "Bride dua", "groomName": "Groom dua", "eventDate": day(200), "venue": "Gedung dua",
		"prepStartDate": day(-30), "packageName": "Paket", "contractValue": 50_000_000, "status": "Preparation", "picStaffId": wpID,
		"venueId": venue, "venueRentalPrice": 10_000_000, "venueCharge": 500_000,
	})).id("data", "id")
	s.setTerms(x, p2, term("DP", 20_000_000, 5), term("Pelunasan", 30_000_000, 60))

	// P3 cancelled and P4 archived: both left out everywhere.
	p3 := s.projectAt(x, "Proyek batal", 10_000_000, 30, 0, 0)
	s.setTerms(x, p3, term("DP", 10_000_000, -5))
	s.mustOK("cancel p3", s.do(x.token, "POST", s.projectsPath(p3, "/cancel"), map[string]any{}))
	p4 := s.projectAt(x, "Proyek arsip", 10_000_000, 30, 0, 0)
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
		if len(got) != 2 {
			t.Fatalf("want the vendor and the venue, got %s", r)
		}
		// Most still owed first: the venue (10.5 jt), then the vendor (4 jt, overdue).
		venue, vend := got[0], got[1]
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
			"payables":            {"amount": 14_500_000.0, "count": 2.0, "overdueAmount": 4_000_000.0, "overdueCount": 1.0},
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
		r := s.do(wpTok, "GET", "/api/v1/finance/receivables", nil)
		if r.status == 403 {
			t.Skip("the Wedding Planner template has no finance.record in this build")
		}
		s.mustOK("wp receivables", r)
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
		sum := s.mustOK("wp summary", s.do(wpTok, "GET", "/api/v1/finance/summary", nil))
		if sum.dig("data", "overdue", "amount") != 0.0 || sum.dig("data", "payables", "amount") != 10_500_000.0 {
			t.Errorf("planner summary: %s", sum)
		}
		pays := rows(s.mustOK("wp payables", s.do(wpTok, "GET", "/api/v1/finance/payables", nil)), "data")
		if len(pays) != 1 || pays[0]["kind"] != "venue" {
			t.Errorf("planner payables: %v", pays)
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
