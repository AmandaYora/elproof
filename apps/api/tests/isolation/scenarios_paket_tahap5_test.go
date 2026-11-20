//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Paket WO, Tahap 5: quotations (/api/v1/quotations/**). Another workspace can
// neither list, read, write, revise, render, share, accept, copy nor delete a
// quotation, its events, vendors, lines, terms, revisions or acceptance
// attachment; and a quotation of its own cannot point at the other workspace's
// staff, event kind, venue, vendor, package, event or quotation vendor.
func TestPaketTahap5(t *testing.T) {
	s := startStack(t)
	a, b := s.signUp("Q5A"), s.signUp("Q5B")
	qPath := func(id int64, rest string) string { return fmt.Sprintf("/api/v1/quotations/%d%s", id, rest) }
	detail := func(tn tenant, id int64) reply {
		return s.mustOK("quotation detail", s.do(tn.token, "GET", qPath(id, ""), nil))
	}
	versionOf := func(tn tenant, id int64, field string) string {
		v, _ := detail(tn, id).dig("data", "quotation", field).(string)
		if v == "" {
			t.Fatalf("quotation %d has no %s", id, field)
		}
		return v
	}
	ifMatch := func(v string) map[string]string { return map[string]string{"If-Match": quoted(v)} }
	countIn := func(table string, quotationID int64) int {
		return s.count("SELECT COUNT(*) FROM "+table+" WHERE quotation_id = ?", quotationID)
	}

	// ---- workspace A: a venue, a vendor, a package for Resepsi, and a
	// quotation with an event at that venue, the vendor at an agreed cost, a
	// package line and an item, and one revision. Unique amounts so a leak is
	// searchable.
	resepsiA := s.kind(a, "Resepsi")
	aVenue := s.mustOK("A venue", s.do(a.token, "POST", "/api/v1/venues", venueBody(a, "Gedung "+a.marker))).id("data", "id")
	aVendor, _ := s.newVendor(a, "Vendor "+a.marker)
	aPackage := s.mustOK("A package", s.do(a.token, "POST", "/api/v1/packages", map[string]any{
		"name": "Paket Kenanga " + a.marker, "price": 135_791_113, "eventKindIds": []int64{resepsiA},
		"includedGuests": 500, "extraGuestStep": 50, "extraGuestStepPrice": 4_000_000,
		"components": []map[string]any{{"description": "Dekorasi " + a.marker, "estimatedCost": 20_000_000}},
	})).id("data", "id")
	QA := s.mustOK("A quotation", s.do(a.token, "POST", "/api/v1/quotations", map[string]any{
		"coupleNames": "Pasangan " + a.marker, "contactWhatsapp": "081299990000", "notes": "Catatan " + a.marker,
	})).id("data", "quotation", "id")
	if QA == 0 {
		t.Fatal("A's quotation has no id")
	}
	withEvent := s.mustOK("A event", s.do(a.token, "POST", qPath(QA, "/events"), map[string]any{
		"eventKindId": resepsiA, "eventDate": day(120), "venueId": aVenue, "layout": "Standing",
		"sessions": []map[string]any{{"label": "Sesi 1", "guests": 200}},
	}))
	aEvents := rows(withEvent, "data", "events")
	if len(aEvents) != 1 {
		t.Fatalf("A's quotation has no event: %s", withEvent)
	}
	aEvent := int64(aEvents[0]["id"].(float64))
	withVendor := s.mustOK("A vendor", s.do(a.token, "POST", qPath(QA, "/suppliers"), map[string]any{
		"vendorId": aVendor, "paidBy": "WO", "agreedCost": 7_654_321, "notes": "Rahasia " + a.marker,
	}))
	var aSup, aVenueSup int64
	for _, sp := range rows(withVendor, "data", "suppliers") {
		switch sp["kind"] {
		case "vendor":
			aSup = int64(sp["id"].(float64))
		case "venue":
			aVenueSup = int64(sp["id"].(float64))
		}
	}
	if aSup == 0 || aVenueSup == 0 {
		t.Fatalf("A's quotation misses its vendor or venue: %s", withVendor)
	}
	s.mustOK("A lines", s.doWith(a.token, "PUT", qPath(QA, "/lines"), map[string]any{"lines": []map[string]any{
		{"kind": "Package", "packageId": aPackage, "eventId": aEvent},
		{"kind": "Item", "description": "Photobooth " + a.marker, "quantity": 1, "unit": "unit", "unitPrice": 7_777_777, "supplierRef": aSup},
	}}, ifMatch(versionOf(a, QA, "linesUpdatedAt"))))
	if r := s.do(a.token, "POST", qPath(QA, "/revisions"), map[string]any{}); r.status != http.StatusCreated {
		t.Fatalf("A's first revision: want 201, got %s", r)
	}
	aSecrets := []string{a.marker, "135791113", "7777777", "7654321"}
	noLeakOfA := func(t *testing.T, what string, r reply) {
		t.Helper()
		for _, secret := range aSecrets {
			if strings.Contains(string(r.body), secret) {
				t.Errorf("%s leaks %q: %s", what, secret, r)
			}
		}
	}
	snapshotOfA := func() string {
		d := detail(a, QA)
		return fmt.Sprintf("%v|%v|%v|%d|%d|%d", d.dig("data", "quotation", "status"), d.dig("data", "quotation", "revisionCount"),
			d.dig("data", "totals", "total"), countIn("quotation_events", QA), countIn("quotation_suppliers", QA), countIn("quotation_lines", QA))
	}

	t.Run("WO B cannot reach WO A's quotation on any route", func(t *testing.T) {
		before := snapshotOfA()
		realVersion := quoted(versionOf(a, QA, "updatedAt"))
		item := map[string]any{"kind": "Item", "description": "Curian " + b.marker, "quantity": 1, "unit": "unit", "unitPrice": 1}
		attacks := []attack{
			{name: "detail", method: "GET", path: qPath(QA, "")},
			{name: "header update", method: "PUT", path: qPath(QA, ""), body: map[string]any{"coupleNames": "Curian " + b.marker}},
			{name: "delete", method: "DELETE", path: qPath(QA, "")},
			{name: "terms text", method: "PUT", path: qPath(QA, "/terms-text"), body: map[string]any{"termsText": "Curian"}},
			{name: "event add", method: "POST", path: qPath(QA, "/events"), body: map[string]any{"eventKindId": resepsiA, "layout": "Standing"}},
			{name: "event order", method: "POST", path: qPath(QA, "/events/order"), body: map[string]any{"ids": []int64{aEvent}}},
			{name: "event update", method: "PUT", path: qPath(QA, fmt.Sprintf("/events/%d", aEvent)), body: map[string]any{"eventKindId": resepsiA, "layout": "Seated"}},
			{name: "event delete", method: "DELETE", path: qPath(QA, fmt.Sprintf("/events/%d", aEvent))},
			{name: "vendor add", method: "POST", path: qPath(QA, "/suppliers"), body: map[string]any{"vendorId": aVendor, "paidBy": "WO"}},
			{name: "vendor update", method: "PUT", path: qPath(QA, fmt.Sprintf("/suppliers/%d", aSup)), body: map[string]any{"paidBy": "Client"}},
			{name: "vendor delete", method: "DELETE", path: qPath(QA, fmt.Sprintf("/suppliers/%d", aSup))},
			{name: "vendor refresh", method: "POST", path: qPath(QA, fmt.Sprintf("/suppliers/%d/refresh", aSup))},
			{name: "lines replace", method: "PUT", path: qPath(QA, "/lines"), body: map[string]any{"lines": []map[string]any{item}}},
			{name: "lines preview", method: "POST", path: qPath(QA, "/lines/preview"), body: map[string]any{"lines": []map[string]any{item}}},
			{name: "terms", method: "PUT", path: qPath(QA, "/terms"), body: map[string]any{"terms": []map[string]any{{"label": "Lunas", "percent": 100, "dueBasis": "Accepted"}}}},
			{name: "revisions list", method: "GET", path: qPath(QA, "/revisions")},
			{name: "revision create", method: "POST", path: qPath(QA, "/revisions"), body: map[string]any{}},
			{name: "revision read", method: "GET", path: qPath(QA, "/revisions/1")},
			{name: "revision pdf", method: "GET", path: qPath(QA, "/revisions/1/pdf")},
			{name: "revision share", method: "POST", path: qPath(QA, "/revisions/1/share"), body: map[string]any{}},
			{name: "compare", method: "GET", path: qPath(QA, "/compare?from=1&to=0")},
			{name: "accept", method: "POST", path: qPath(QA, "/status"), body: map[string]any{"status": "Accepted", "revisionN": 1, "acceptedOn": day(0), "channel": "WhatsApp"}},
			{name: "reject", method: "POST", path: qPath(QA, "/status"), body: map[string]any{"status": "Rejected", "reason": "Curian"}},
			{name: "copy", method: "POST", path: qPath(QA, "/copy"), body: map[string]any{}},
			{name: "alternative", method: "POST", path: qPath(QA, "/alternatives"), body: map[string]any{}},
			{name: "extend", method: "POST", path: qPath(QA, "/extend"), body: map[string]any{"validUntil": day(60)}},
			{name: "preview pdf", method: "GET", path: qPath(QA, "/preview.pdf")},
			{name: "acceptance attachment upload", method: "PUT", path: qPath(QA, "/acceptance-attachment"), body: map[string]any{
				"fileName": "bukti.pdf", "mimeType": "application/pdf", "base64Data": "JVBERi0xLjQK"}},
			{name: "acceptance attachment read", method: "GET", path: qPath(QA, "/acceptance-attachment")},
			{name: "activity", method: "GET", path: qPath(QA, "/activity")},
		}
		for _, at := range attacks {
			at := at
			routesCovered++
			t.Run(at.name, func(t *testing.T) {
				// No version, a version no row carries, and A's real current
				// version: none of them opens A's quotation to B.
				for _, h := range []map[string]string{nil, {"If-Match": staleVersion}, {"If-Match": realVersion}} {
					r := s.doWith(b.token, at.method, at.path, at.body, h)
					if r.status != http.StatusNotFound {
						t.Errorf("B %s %s (%v): want 404, got %s", at.method, at.path, h, r)
					}
					if vacuous(r) {
						t.Errorf("B %s %s: router miss, not an ownership answer: %s", at.method, at.path, r)
					}
					noLeakOfA(t, "B "+at.method+" "+at.path, r)
				}
			})
		}
		if after := snapshotOfA(); after != before {
			t.Errorf("B's attempts changed A's quotation: %s -> %s", before, after)
		}
	})

	t.Run("WO B's lists and counts show nothing of WO A", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/quotations", "/api/v1/quotations?search=" + a.marker, "/api/v1/quotations?tab=Draft",
			fmt.Sprintf("/api/v1/quotations?picStaffId=%d", a.staffID), "/api/v1/quotations/summary",
		} {
			routesCovered++
			r := s.mustOK("B "+path, s.do(b.token, "GET", path, nil))
			noLeakOfA(t, "B GET "+path, r)
			if !strings.HasSuffix(path, "/summary") {
				if list := rows(r, "data"); len(list) != 0 {
					t.Errorf("B GET %s lists A's quotations: %v", path, list)
				}
			} else if all := r.dig("data", "all"); all != float64(0) {
				t.Errorf("B's summary counts A's quotations: %s", r)
			}
		}
	})

	t.Run("WO B's own quotation cannot point at WO A's records", func(t *testing.T) {
		resepsiB := s.kind(b, "Resepsi")
		bQuotes := func() int { return s.count("SELECT COUNT(*) FROM quotations WHERE tenant_id = ?", b.tenant) }
		before := bQuotes()

		// A's staff as the PIC of B's quotation.
		routesCovered++
		r := s.do(b.token, "POST", "/api/v1/quotations", map[string]any{"coupleNames": "Pasangan " + b.marker, "picStaffId": a.staffID})
		wantStatus(t, "B quotation with A's staff as PIC", r, http.StatusUnprocessableEntity)
		if !hasKey(r, "picStaffId") {
			t.Errorf("the refusal names picStaffId: %s", r)
		}
		noLeakOfA(t, "B quotation with A's staff", r)
		if bQuotes() != before {
			t.Fatal("a refused quotation was stored")
		}

		QB := s.mustOK("B quotation", s.do(b.token, "POST", "/api/v1/quotations", map[string]any{"coupleNames": "Pasangan " + b.marker})).id("data", "quotation", "id")
		bEvent := rows(s.mustOK("B event", s.do(b.token, "POST", qPath(QB, "/events"), map[string]any{
			"eventKindId": resepsiB, "eventDate": day(90), "layout": "Seated", "sessions": []map[string]any{{"label": "Sesi 1", "guests": 100}},
		})), "data", "events")
		if len(bEvent) != 1 {
			t.Fatal("B's quotation has no event")
		}
		bEventID := int64(bEvent[0]["id"].(float64))
		events, suppliers, lines := countIn("quotation_events", QB), countIn("quotation_suppliers", QB), countIn("quotation_lines", QB)

		refused := func(what string, r reply, key string) {
			t.Helper()
			routesCovered++
			wantStatus(t, what, r, http.StatusUnprocessableEntity)
			if !hasKey(r, key) {
				t.Errorf("%s: the refusal names %s: %s", what, key, r)
			}
			noLeakOfA(t, what, r)
		}
		refused("event with A's event kind", s.do(b.token, "POST", qPath(QB, "/events"), map[string]any{
			"eventKindId": resepsiA, "eventDate": day(91), "layout": "Seated"}), "eventKindId")
		refused("event at A's venue", s.do(b.token, "POST", qPath(QB, "/events"), map[string]any{
			"eventKindId": resepsiB, "eventDate": day(91), "venueId": aVenue, "layout": "Standing"}), "venueId")
		refused("vendor from A's directory", s.do(b.token, "POST", qPath(QB, "/suppliers"), map[string]any{
			"vendorId": aVendor, "paidBy": "WO"}), "vendorId")

		linesVersion := ifMatch(versionOf(b, QB, "linesUpdatedAt"))
		refused("line with A's package", s.doWith(b.token, "PUT", qPath(QB, "/lines"), map[string]any{"lines": []map[string]any{
			{"kind": "Package", "packageId": aPackage, "eventId": bEventID}}}, linesVersion), "lines.0.packageId")
		refused("line on A's event", s.doWith(b.token, "PUT", qPath(QB, "/lines"), map[string]any{"lines": []map[string]any{
			{"kind": "Item", "description": "Curian " + b.marker, "quantity": 1, "unit": "unit", "unitPrice": 1, "eventId": aEvent}}}, linesVersion), "lines.0.eventId")
		refused("line with A's quotation vendor", s.doWith(b.token, "PUT", qPath(QB, "/lines"), map[string]any{"lines": []map[string]any{
			{"kind": "Item", "description": "Curian " + b.marker, "quantity": 1, "unit": "unit", "unitPrice": 1, "supplierRef": aSup}}}, linesVersion), "lines.0.supplierRef")
		refused("preview with A's package", s.do(b.token, "POST", qPath(QB, "/lines/preview"), map[string]any{"lines": []map[string]any{
			{"kind": "Package", "packageId": aPackage, "eventId": bEventID}}}), "lines.0.packageId")
		refused("terms due on A's event", s.doWith(b.token, "PUT", qPath(QB, "/terms"), map[string]any{"terms": []map[string]any{
			{"label": "Lunas", "percent": 100, "dueBasis": "Event", "dueOffsetDays": -7, "dueEventId": aEvent}}},
			ifMatch(versionOf(b, QB, "termsUpdatedAt"))), "terms.0.dueEventId")

		// A's child ids under B's own quotation: not found.
		for _, at := range []attack{
			{name: "B quotation + A event update", method: "PUT", path: qPath(QB, fmt.Sprintf("/events/%d", aEvent)), body: map[string]any{"eventKindId": resepsiB, "layout": "Seated"}},
			{name: "B quotation + A event delete", method: "DELETE", path: qPath(QB, fmt.Sprintf("/events/%d", aEvent))},
			{name: "B quotation + A vendor update", method: "PUT", path: qPath(QB, fmt.Sprintf("/suppliers/%d", aSup)), body: map[string]any{"paidBy": "Client"}},
			{name: "B quotation + A venue delete", method: "DELETE", path: qPath(QB, fmt.Sprintf("/suppliers/%d", aVenueSup))},
			{name: "B quotation + A vendor refresh", method: "POST", path: qPath(QB, fmt.Sprintf("/suppliers/%d/refresh", aSup))},
		} {
			routesCovered++
			r := s.doWith(b.token, at.method, at.path, at.body, map[string]string{"If-Match": staleVersion})
			wantStatus(t, at.name, r, http.StatusNotFound)
			noLeakOfA(t, at.name, r)
		}
		// A's event ids in B's event order: unknown here.
		routesCovered++
		r = s.do(b.token, "POST", qPath(QB, "/events/order"), map[string]any{"ids": []int64{aEvent, bEventID}})
		if r.status < 400 || r.status >= 500 {
			t.Errorf("B ordering A's event: want a 4xx refusal, got %s", r)
		}
		noLeakOfA(t, "B ordering A's event", r)

		if e, sp, l := countIn("quotation_events", QB), countIn("quotation_suppliers", QB), countIn("quotation_lines", QB); e != events || sp != suppliers || l != lines {
			t.Errorf("a refused write stored rows on B: events %d -> %d, vendors %d -> %d, lines %d -> %d", events, e, suppliers, sp, lines, l)
		}
		if n := s.count("SELECT COUNT(*) FROM quotation_lines WHERE description LIKE ?", "Curian %"); n != 0 {
			t.Errorf("a refused line was stored: %d", n)
		}
		if n := s.count("SELECT COUNT(*) FROM quotation_suppliers WHERE tenant_id = ? AND supplier_id = ? AND supplier_kind = 'vendor'", b.tenant, aVendor); n != 0 {
			t.Errorf("A's vendor was added to B's quotation: %d", n)
		}
		if n := s.count("SELECT COUNT(*) FROM quotation_events WHERE tenant_id = ? AND event_kind_id = ?", b.tenant, resepsiA); n != 0 {
			t.Errorf("A's event kind is on B's quotation: %d", n)
		}
	})

	if after := s.mustOK("A detail", s.do(a.token, "GET", qPath(QA, ""), nil)); after.dig("data", "quotation", "status") != "Draft" ||
		after.dig("data", "quotation", "revisionCount") != float64(1) {
		t.Errorf("A's quotation moved: %s", after)
	}
}
