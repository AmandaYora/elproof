//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Paket WO, Tahap 2: events, the project's vendors and venues ("Vendor &
// venue") and what the WO or the client pays them. The rules here are inside
// one workspace (who of the WO's staff or the client may see or do what);
// the cross-workspace side is in the attack matrix (isolation_test.go).
func TestPaketTahap2(t *testing.T) {
	s := startStack(t)
	tn := s.signUp("P2")
	proj := func(id int64, rest string) string { return s.projectsPath(id, rest) }
	akad := s.kind(tn, "Akad")

	category := s.mustOK("category", s.do(tn.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": "Dekorasi " + tn.marker})).id("data", "id")
	vendorWO := s.mustOK("WO vendor", s.do(tn.token, "POST", "/api/v1/vendors", vendorBody(s, tn, "Vendor Rahasia "+tn.marker, category))).id("data", "id")
	vendorClient := s.mustOK("client vendor", s.do(tn.token, "POST", "/api/v1/vendors", vendorBody(s, tn, "Vendor Klien "+tn.marker, category))).id("data", "id")
	bigVenue := s.mustOK("big venue", s.do(tn.token, "POST", "/api/v1/venues", venueBody(tn, "Gedung Besar "+tn.marker))).id("data", "id")
	small := venueBody(tn, "Gedung Kecil "+tn.marker)
	small["capacityStanding"] = 100
	smallVenue := s.mustOK("small venue", s.do(tn.token, "POST", "/api/v1/venues", small)).id("data", "id")

	wpToken, wpID := s.staffAs(tn, "Wedding Planner")
	P := s.projectFor(tn, "Proyek P2 "+tn.marker, 100_000_000, 120, wpID, "Preparation")
	supWO := s.engage(tn, P, vendorWO, "Dekor WO "+tn.marker, 5_000_000, "Booked", nil)
	supClient := s.engageBy(tn, P, vendorClient, "Dekor klien "+tn.marker, "Client", false)
	payWO := s.supplierPay(tn, P, supWO, "DP", 1_000_000, "WOPAY-"+tn.marker)
	payClient := s.supplierPay(tn, P, supClient, "DP", 2_000_000, "CLPAY-"+tn.marker)
	issueWO := s.mustOK("issue on the hidden vendor", s.do(tn.token, "POST", proj(P, "/issues"), map[string]any{
		"projectSupplierId": supWO, "title": "Kendala dekor " + tn.marker, "description": "d", "impact": "Low", "foundDate": day(-1),
		"picStaffId": tn.staffID, "targetResolutionDate": day(5), "isClientVisible": true,
	})).id("data", "id")
	payBody := func(sup int64, ref string) map[string]any {
		return map[string]any{"projectSupplierId": sup, "type": "Termin", "amount": 100_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": ref}
	}

	// A client of the project, with every portal permission (Pengantin).
	bride := s.addClient(tn, P, "Bride", "Pengantin", fmt.Sprintf("pengantin.p2.%d", time.Now().UnixNano()%1_000_000_000))

	t.Run("a client never sees a hidden vendor's name nor the WO's payments", func(t *testing.T) {
		routesCovered += 3
		sl := s.mustOK("client suppliers", s.do(bride, "GET", proj(P, "/suppliers"), nil))
		if e := rowByID(rows(sl, "data"), supWO); e == nil || e["name"] != "Dekorasi "+tn.marker {
			t.Errorf("the hidden vendor is named by its category: %s", sl)
		}
		if e := rowByID(rows(sl, "data"), supClient); e == nil || e["name"] != "Vendor Klien "+tn.marker {
			t.Errorf("a vendor the client pays is named: %s", sl)
		}
		il := s.mustOK("client issues", s.do(bride, "GET", proj(P, "/issues"), nil))
		if i := rowByID(rows(il, "data"), issueWO); i == nil || i["supplierName"] != "Dekorasi "+tn.marker {
			t.Errorf("the issue names the service, not the vendor: %s", il)
		}
		pl := s.mustOK("client supplier payments", s.do(bride, "GET", proj(P, "/supplier-payments"), nil))
		if rowByID(rows(pl, "data"), payWO) != nil || rowByID(rows(pl, "data"), payClient) == nil {
			t.Errorf("the client sees its own supplier payments only: %s", pl)
		}
		for what, r := range map[string]reply{"suppliers": sl, "issues": il, "supplier payments": pl} {
			for _, leak := range []string{"Vendor Rahasia", "WOPAY-", "5000000", "agreedCost\":5"} {
				if strings.Contains(string(r.body), leak) {
					t.Errorf("client %s leaks %q: %s", what, leak, r)
				}
			}
		}
		// Control: the owner sees the real name.
		if e := rowByID(rows(s.mustOK("owner suppliers", s.do(tn.token, "GET", proj(P, "/suppliers"), nil)), "data"), supWO); e == nil || e["name"] != "Vendor Rahasia "+tn.marker {
			t.Errorf("the owner sees the vendor's name: %v", e)
		}
	})

	t.Run("portal routes follow the client's portal permissions", func(t *testing.T) {
		role := func(name string, perms ...string) {
			s.mustOK("client role "+name, s.do(tn.token, "POST", "/api/v1/access/roles", map[string]any{
				"kind": "client", "name": name, "description": "Uji", "permissions": perms,
			}))
		}
		role("Dokumen saja", "portal.documents.view")
		role("Jadwal saja", "portal.timeline.view")
		docsOnly := s.addClient(tn, P, "Family Representative", "Dokumen saja", fmt.Sprintf("dok.p2.%d", time.Now().UnixNano()%1_000_000_000))
		timelineOnly := s.addClient(tn, P, "Family Representative", "Jadwal saja", fmt.Sprintf("jadwal.p2.%d", time.Now().UnixNano()%1_000_000_000))
		family := s.addClient(tn, P, "Family Representative", "Keluarga", fmt.Sprintf("kel.p2.%d", time.Now().UnixNano()%1_000_000_000))
		for _, c := range []struct {
			who              string
			tok              string
			events, supplier int
		}{
			{"Pengantin", bride, http.StatusOK, http.StatusOK},
			{"Keluarga", family, http.StatusOK, http.StatusOK},
			{"timeline only", timelineOnly, http.StatusOK, http.StatusForbidden},
			{"documents only", docsOnly, http.StatusForbidden, http.StatusForbidden},
		} {
			routesCovered += 2
			wantStatus(t, c.who+" GET events", s.do(c.tok, "GET", proj(P, "/events"), nil), c.events)
			wantStatus(t, c.who+" GET suppliers", s.do(c.tok, "GET", proj(P, "/suppliers"), nil), c.supplier)
		}
		// Keluarga has no finance: the supplier payments are closed.
		wantStatus(t, "Keluarga GET supplier-payments", s.do(family, "GET", proj(P, "/supplier-payments"), nil), http.StatusForbidden)

		// A client's view of an event's venue is its name and address only.
		ev := rows(s.mustOK("client events", s.do(bride, "GET", proj(P, "/events"), nil)), "data")
		if len(ev) == 0 {
			t.Fatalf("the client sees no event")
		}

		// Writes are the WO's.
		eid := int64(ev[0]["id"].(float64))
		for _, w := range []attack{
			{name: "event create", method: "POST", path: proj(P, "/events"), body: eventOf(akad, day(121), 10, 0)},
			{name: "event update", method: "PUT", path: proj(P, fmt.Sprintf("/events/%d", eid)), body: eventOf(akad, day(120), 10, bigVenue)},
			{name: "event delete", method: "DELETE", path: proj(P, fmt.Sprintf("/events/%d", eid))},
			{name: "event set-main", method: "POST", path: proj(P, fmt.Sprintf("/events/%d/set-main", eid)), body: map[string]any{}},
			{name: "event move", method: "POST", path: proj(P, fmt.Sprintf("/events/%d/move", eid)), body: map[string]any{"direction": "down"}},
			{name: "event duplicate", method: "POST", path: proj(P, fmt.Sprintf("/events/%d/duplicate", eid)), body: map[string]any{}},
			{name: "supplier create", method: "POST", path: proj(P, "/suppliers"), body: map[string]any{"vendorId": vendorClient, "paidBy": "Client", "engagementStatus": "Planned", "picStaffId": tn.staffID}},
			{name: "supplier update", method: "PATCH", path: proj(P, fmt.Sprintf("/suppliers/%d", supWO)), body: map[string]any{"showName": true}},
			{name: "supplier delete", method: "DELETE", path: proj(P, fmt.Sprintf("/suppliers/%d", supClient))},
			{name: "supplier restore", method: "POST", path: proj(P, fmt.Sprintf("/suppliers/%d/restore", supClient))},
			{name: "supplier milestone", method: "POST", path: proj(P, fmt.Sprintf("/suppliers/%d/milestones", supClient)), body: map[string]any{"name": "x", "targetDate": day(5)}},
			{name: "supplier payment", method: "POST", path: proj(P, "/supplier-payments"), body: payBody(supClient, "CL-SELF")},
			{name: "supplier payment update", method: "PATCH", path: proj(P, fmt.Sprintf("/supplier-payments/%d", payClient)), body: payBody(supClient, "CL-SELF")},
			{name: "supplier payment delete", method: "DELETE", path: proj(P, fmt.Sprintf("/supplier-payments/%d", payClient))},
		} {
			routesCovered++
			for _, h := range []map[string]string{nil, {"If-Match": staleVersion}} {
				if r := s.doWith(bride, w.method, w.path, w.body, h); r.status != http.StatusForbidden {
					t.Errorf("client %s (%v): want 403, got %s", w.name, h, r)
				}
			}
		}
		if s.count("SELECT COUNT(*) FROM supplier_payments WHERE reference_number = 'CL-SELF'") != 0 {
			t.Error("a refused client write was stored")
		}
	})

	t.Run("choosing a venue for an event adds it to the suppliers at once", func(t *testing.T) {
		routesCovered += 3
		venueRow := func(venue int64) map[string]any {
			for _, sup := range rows(s.mustOK("suppliers", s.do(tn.token, "GET", proj(P, "/suppliers"), nil)), "data") {
				if sup["kind"] == "venue" && sup["supplierId"] == float64(venue) {
					return sup
				}
			}
			return nil
		}
		if venueRow(bigVenue) != nil {
			t.Fatalf("the venue is not a supplier before it is chosen")
		}
		ev := rows(s.mustOK("events", s.do(tn.token, "GET", proj(P, "/events"), nil)), "data")
		eid := int64(ev[0]["id"].(float64))
		s.mustOK("choose the venue", s.doFresh(tn.token, "PUT", proj(P, fmt.Sprintf("/events/%d", eid)), eventOf(akad, day(120), 250, bigVenue)))
		v := venueRow(bigVenue)
		if v == nil || v["paidBy"] != "WO" || v["name"] != "Gedung Besar "+tn.marker || v["capacityStanding"] != float64(300) || v["usedByEvents"] != float64(1) {
			t.Errorf("the chosen venue is a supplier right away: %v", v)
		}
		// So is a venue on a new event.
		s.mustOK("new event at the small venue", s.do(tn.token, "POST", proj(P, "/events"), eventOf(akad, day(122), 50, smallVenue)))
		if venueRow(smallVenue) == nil {
			t.Errorf("a new event's venue is a supplier right away")
		}
		// And the client sees the venue named, with nothing of the WO's.
		cl := s.mustOK("client suppliers", s.do(bride, "GET", proj(P, "/suppliers"), nil))
		if !strings.Contains(string(cl.body), "Gedung Besar "+tn.marker) {
			t.Errorf("the client sees the venue: %s", cl)
		}
		ce := s.mustOK("client events", s.do(bride, "GET", proj(P, "/events"), nil))
		for _, e := range rows(ce, "data") {
			if v, ok := e["venue"].(map[string]any); ok {
				for _, k := range []string{"supplierId", "venueId", "capacityStanding", "capacitySeated"} {
					if _, has := v[k]; has {
						t.Errorf("the client's event venue carries %q: %v", k, v)
					}
				}
			}
		}
	})

	t.Run("a session above the venue's capacity is refused, and so is a smaller venue", func(t *testing.T) {
		routesCovered += 3
		before := s.count("SELECT COUNT(*) FROM project_events WHERE project_id = ?", P)
		r := s.do(tn.token, "POST", proj(P, "/events"), eventOf(akad, day(123), 301, bigVenue))
		wantStatus(t, "session above capacity", r, http.StatusUnprocessableEntity)
		if !hasKey(r, "sessions.0.guests") {
			t.Errorf("the refusal names the session: %s", r)
		}
		if s.count("SELECT COUNT(*) FROM project_events WHERE project_id = ?", P) != before {
			t.Errorf("a refused event was stored")
		}
		// The main event holds 250 at the big venue: the small one (100) cannot take it.
		ev := rows(s.mustOK("events", s.do(tn.token, "GET", proj(P, "/events"), nil)), "data")
		var eid int64
		for _, e := range ev {
			if v, ok := e["venue"].(map[string]any); ok && v["venueId"] == float64(bigVenue) {
				eid = int64(e["id"].(float64))
			}
		}
		if eid == 0 {
			t.Fatalf("no event at the big venue: %v", ev)
		}
		r = s.doFresh(tn.token, "PUT", proj(P, fmt.Sprintf("/events/%d", eid)), eventOf(akad, day(120), 250, smallVenue))
		wantStatus(t, "smaller venue", r, http.StatusUnprocessableEntity)
		if !hasKey(r, "sessions.0.guests") {
			t.Errorf("the refusal names the session: %s", r)
		}
		after := rows(s.mustOK("events", s.do(tn.token, "GET", proj(P, "/events"), nil)), "data")
		if e := rowByID(after, eid); e == nil || e["venue"].(map[string]any)["venueId"] != float64(bigVenue) {
			t.Errorf("the event keeps its venue: %v", e)
		}
		// A seated event at a venue with no seated capacity: refused on the venue.
		seated := eventOf(akad, day(124), 10, bigVenue)
		seated["layout"] = "Seated"
		r = s.do(tn.token, "POST", proj(P, "/events"), seated)
		wantStatus(t, "no seated capacity", r, http.StatusUnprocessableEntity)
		if !hasKey(r, "venueId") {
			t.Errorf("the refusal names the venue: %s", r)
		}
		// Control: within capacity is fine.
		s.mustOK("within capacity", s.do(tn.token, "POST", proj(P, "/events"), eventOf(akad, day(123), 300, bigVenue)))
	})

	// Last: until the list stops failing for staff without finance.margin.view,
	// a dropped connection here ends the whole test.
	t.Run("a Wedding Planner neither sees nor records what the WO pays a supplier", func(t *testing.T) {
		routesCovered += 6
		// Staff without finance.margin.view still see the real names and notes:
		// hiding a vendor's name is a rule for the client only.
		sl := s.do(wpToken, "GET", proj(P, "/suppliers"), nil)
		wantStatus(t, "WP suppliers", sl, http.StatusOK)
		if e := rowByID(rows(sl, "data"), supWO); e == nil || e["agreedCost"] != nil || e["paidWo"] != nil {
			t.Errorf("the WP sees the WO's cost or what it paid: %s", sl)
		} else if e["name"] != "Vendor Rahasia "+tn.marker || e["picStaffId"] != float64(tn.staffID) {
			t.Errorf("the WP gets the client's redacted view of the supplier (name %v, picStaffId %v): %s", e["name"], e["picStaffId"], sl)
		}
		list := s.do(wpToken, "GET", proj(P, "/supplier-payments"), nil)
		wantStatus(t, "WP supplier payments", list, http.StatusOK)
		if rowByID(rows(list, "data"), payWO) != nil || strings.Contains(string(list.body), "WOPAY-") {
			t.Errorf("the WP's list shows the WO's payment: %s", list)
		}
		if rowByID(rows(list, "data"), payClient) == nil {
			t.Errorf("the WP's list should show the client-paid supplier's payment: %s", list)
		}
		// The WO's payment as an item: not there for the WP. (A WP may not delete
		// any payment: that is a 403 before the lookup, whatever the id.)
		if r := s.do(wpToken, "PATCH", proj(P, fmt.Sprintf("/supplier-payments/%d", payWO)), payBody(supWO, "x")); r.status != http.StatusNotFound {
			t.Errorf("WP PATCH the WO's payment: want 404, got %s", r)
		}
		if r := s.do(wpToken, "DELETE", proj(P, fmt.Sprintf("/supplier-payments/%d", payWO)), nil); r.status != http.StatusNotFound && r.status != http.StatusForbidden {
			t.Errorf("WP DELETE the WO's payment: want 403 or 404, got %s", r)
		}
		if r := s.doWith(wpToken, "PATCH", proj(P, fmt.Sprintf("/supplier-payments/%d", payWO)), payBody(supWO, "x"), map[string]string{"If-Match": staleVersion}); r.status != http.StatusNotFound {
			t.Errorf("WP PATCH the WO's payment with a version: want 404, got %s", r)
		}
		// Recording one: refused.
		r := s.do(wpToken, "POST", proj(P, "/supplier-payments"), payBody(supWO, "WP-WO"))
		wantStatus(t, "WP records a WO payment", r, http.StatusForbidden)
		if n := s.count("SELECT COUNT(*) FROM supplier_payments WHERE reference_number = 'WP-WO'"); n != 0 {
			t.Errorf("a refused payment was stored: %d", n)
		}
		// The client-paid supplier: recorded and corrected by the WP.
		created := s.mustOK("WP records a client payment", s.do(wpToken, "POST", proj(P, "/supplier-payments"), payBody(supClient, "WP-CL")))
		s.mustOK("WP corrects it", s.doWith(wpToken, "PATCH", proj(P, fmt.Sprintf("/supplier-payments/%d", created.id("data", "id"))), payBody(supClient, "WP-CL2"),
			map[string]string{"If-Match": quoted(str(created, "data", "updatedAt"))}))
		// Control: the owner sees and changes both.
		own := s.mustOK("owner supplier payments", s.do(tn.token, "GET", proj(P, "/supplier-payments"), nil))
		if rowByID(rows(own, "data"), payWO) == nil || rowByID(rows(own, "data"), payClient) == nil {
			t.Errorf("the owner sees every payment: %s", own)
		}
	})
}
