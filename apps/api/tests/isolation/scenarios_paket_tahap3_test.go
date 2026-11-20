//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Paket WO, Tahap 3: the project's Rincian (GET/PUT/POST/PATCH/DELETE
// /projects/{id}/lines, preview, reorder, restore). Another workspace can
// neither list, read, write, delete nor preview a project's lines, nor point a
// line of its own at the other workspace's package, supplier or event; a client
// of one workspace cannot reach the other's; Keluarga sees the names and not
// one amount.
func TestPaketTahap3(t *testing.T) {
	s := startStack(t)
	a, b := s.signUp("L3A"), s.signUp("L3B")
	proj := func(id int64, rest string) string { return s.projectsPath(id, rest) }
	version := func(tn tenant, id int64) map[string]string {
		return map[string]string{"If-Match": quoted(s.linesVersion(tn, id))}
	}
	lineCount := func(projectID int64) int {
		return s.count("SELECT COUNT(*) FROM project_lines WHERE project_id = ? AND deleted_at IS NULL", projectID)
	}
	contractOf := func(tn tenant, projectID int64) any {
		return s.mustOK("project", s.do(tn.token, "GET", proj(projectID, ""), nil)).dig("data", "contractValue")
	}

	// ---- workspace A: a package, a project with a Resepsi of 600, a WO-paid
	// vendor, and a Rincian with every kind of line. Unique amounts so a leak
	// is searchable.
	resepsiA := s.kind(a, "Resepsi")
	aVendor, _ := s.newVendor(a, "Vendor A "+a.marker)
	aPackage := s.mustOK("A package", s.do(a.token, "POST", "/api/v1/packages", map[string]any{
		"name": "Paket Melati " + a.marker, "price": 150_000_000, "eventKindIds": []int64{resepsiA},
		"includedGuests": 500, "extraGuestStep": 50, "extraGuestStepPrice": 4_000_000,
		"components": []map[string]any{{"description": "Dekorasi pelaminan " + a.marker, "estimatedCost": 20_000_000}},
	})).id("data", "id")
	created := s.mustOK("A project", s.do(a.token, "POST", "/api/v1/projects",
		s.projectBody(a, "Proyek Rincian "+a.marker, 120, eventOf(resepsiA, day(120), 600, 0))))
	PA := created.id("data", "id")
	aEvents := rows(created, "data", "events")
	if len(aEvents) != 1 {
		t.Fatalf("A's project has no event: %s", created)
	}
	aEvent := int64(aEvents[0]["id"].(float64))
	aSup := s.engage(a, PA, aVendor, "Dekor "+a.marker, 20_000_000, "Booked", nil)
	s.mustOK("A rincian", s.doWith(a.token, "PUT", proj(PA, "/lines"), map[string]any{"lines": []map[string]any{
		{"key": "melati", "kind": "Package", "packageId": aPackage, "supplierRef": aSup},
		{"kind": "Item", "description": "Photobooth " + a.marker, "quantity": 1, "unit": "unit", "unitPrice": 7_777_777, "eventId": aEvent, "supplierRef": aSup},
		{"kind": "Discount", "description": "Potongan " + a.marker, "amount": 1_234_567},
	}}, version(a, PA)))
	// 150.000.000 + 8.000.000 (600 guests: 2 steps of 50 over 500) + 7.777.777 - 1.234.567; A is not PKP.
	const aTotal = 164_543_210
	aLines := rows(s.mustOK("A lines", s.do(a.token, "GET", proj(PA, "/lines"), nil)), "data", "lines")
	var aItem, aPkgLine, aExtra int64
	for _, l := range aLines {
		switch l["kind"] {
		case "Item":
			aItem = int64(l["id"].(float64))
		case "Package":
			aPkgLine = int64(l["id"].(float64))
		case "ExtraGuests":
			aExtra = int64(l["id"].(float64))
			if l["amount"] != float64(8_000_000) {
				t.Errorf("the extra-guest charge follows the Resepsi's 600 guests: %v", l)
			}
		}
	}
	if aItem == 0 || aPkgLine == 0 || aExtra == 0 {
		t.Fatalf("A's Rincian is incomplete: %v", aLines)
	}
	if c := contractOf(a, PA); c != float64(aTotal) {
		t.Fatalf("A's contract value is the Rincian's total, got %v want %d", c, aTotal)
	}
	aSecrets := []string{a.marker, "150000000", "7777777", "1234567", "164543210", "8000000"}
	noLeakOfA := func(t *testing.T, what string, r reply) {
		t.Helper()
		for _, secret := range aSecrets {
			if strings.Contains(string(r.body), secret) {
				t.Errorf("%s leaks %q: %s", what, secret, r)
			}
		}
	}

	// ---- workspace B: a project of its own with a vendor and an event.
	bVendor, _ := s.newVendor(b, "Vendor B "+b.marker)
	PB := s.createProject(b, "Proyek B "+b.marker, 50_000_000)
	bSup := s.engage(b, PB, bVendor, "Dekor B", 5_000_000, "Booked", nil)
	bEvents := rows(s.mustOK("B events", s.do(b.token, "GET", proj(PB, "/events"), nil)), "data")
	if len(bEvents) == 0 {
		t.Fatalf("B's project has no event")
	}
	bEvent := int64(bEvents[0]["id"].(float64))
	item := func(desc string) map[string]any {
		return map[string]any{"kind": "Item", "description": desc, "quantity": 1, "unit": "unit", "unitPrice": 1_000_000}
	}

	t.Run("WO B cannot list, read, write, delete or preview WO A's Rincian", func(t *testing.T) {
		before, contract := lineCount(PA), contractOf(a, PA)
		realVersion := quoted(s.linesVersion(a, PA))
		attacks := []attack{
			{name: "lines list", method: "GET", path: proj(PA, "/lines")},
			{name: "lines replace", method: "PUT", path: proj(PA, "/lines"), body: map[string]any{"lines": []map[string]any{item("Ganti " + b.marker)}}},
			{name: "line add", method: "POST", path: proj(PA, "/lines"), body: item("Tambah " + b.marker)},
			{name: "lines preview", method: "POST", path: proj(PA, "/lines/preview"), body: map[string]any{"lines": []map[string]any{item("Pratinjau " + b.marker)}}},
			{name: "lines reorder", method: "POST", path: proj(PA, "/lines/reorder"), body: map[string]any{"ids": []int64{aItem, aPkgLine}}},
			{name: "line update", method: "PATCH", path: proj(PA, fmt.Sprintf("/lines/%d", aItem)), body: map[string]any{"unitPrice": 1}},
			{name: "line delete", method: "DELETE", path: proj(PA, fmt.Sprintf("/lines/%d", aItem))},
			{name: "line restore", method: "POST", path: proj(PA, fmt.Sprintf("/lines/%d/restore", aItem))},
			{name: "payment terms recalculate", method: "POST", path: proj(PA, "/payment-terms/recalculate"), body: map[string]any{}},
		}
		for _, at := range attacks {
			at := at
			routesCovered++
			t.Run(at.name, func(t *testing.T) {
				// No version, a version no row carries, and A's real current
				// version: none of them opens A's project to B.
				for _, h := range []map[string]string{nil, {"If-Match": staleVersion}, {"If-Match": realVersion}} {
					r := s.doWith(b.token, at.method, at.path, at.body, h)
					if r.status != http.StatusNotFound && r.status != http.StatusForbidden {
						t.Errorf("B %s %s (%v): want 403 or 404, got %s", at.method, at.path, h, r)
					}
					if vacuous(r) {
						t.Errorf("B %s %s: router miss, not an ownership answer: %s", at.method, at.path, r)
					}
					noLeakOfA(t, "B "+at.method+" "+at.path, r)
				}
			})
		}
		if lineCount(PA) != before || contractOf(a, PA) != contract {
			t.Errorf("B's attempts changed A's Rincian: %d -> %d lines, contract %v -> %v", before, lineCount(PA), contract, contractOf(a, PA))
		}
		// B's own lines, read with A's line ids in a B path: not found either.
		for _, at := range []attack{
			{name: "B project + A line update", method: "PATCH", path: proj(PB, fmt.Sprintf("/lines/%d", aItem)), body: map[string]any{"unitPrice": 1}},
			{name: "B project + A line delete", method: "DELETE", path: proj(PB, fmt.Sprintf("/lines/%d", aItem))},
			{name: "B project + A line restore", method: "POST", path: proj(PB, fmt.Sprintf("/lines/%d/restore", aItem))},
			{name: "B project + A extra-guest line delete", method: "DELETE", path: proj(PB, fmt.Sprintf("/lines/%d", aExtra))},
		} {
			routesCovered++
			r := s.doWith(b.token, at.method, at.path, at.body, version(b, PB))
			if r.status != http.StatusNotFound {
				t.Errorf("%s: want 404, got %s", at.name, r)
			}
			noLeakOfA(t, at.name, r)
		}
		// A's line ids inside B's own replace and reorder: unknown here (422), nothing moves.
		routesCovered += 2
		bBefore := lineCount(PB)
		r := s.doWith(b.token, "PUT", proj(PB, "/lines"), map[string]any{"lines": []map[string]any{
			{"id": aItem, "kind": "Item", "description": "Curian " + b.marker, "quantity": 1, "unit": "unit", "unitPrice": 1},
		}}, version(b, PB))
		wantStatus(t, "B replace with A's line id", r, http.StatusUnprocessableEntity)
		if !hasKey(r, "lines.0.id") {
			t.Errorf("the refusal names the line: %s", r)
		}
		noLeakOfA(t, "B replace with A's line id", r)
		r = s.doWith(b.token, "POST", proj(PB, "/lines/reorder"), map[string]any{"ids": []int64{aItem, aPkgLine}}, version(b, PB))
		wantStatus(t, "B reorder with A's line ids", r, http.StatusUnprocessableEntity)
		noLeakOfA(t, "B reorder with A's line ids", r)
		if lineCount(PB) != bBefore || lineCount(PA) != before || contractOf(a, PA) != contract {
			t.Errorf("a refused write moved lines: B %d -> %d, A %d -> %d", bBefore, lineCount(PB), before, lineCount(PA))
		}
		if n := s.count("SELECT COUNT(*) FROM project_lines WHERE description LIKE ?", "Curian %"); n != 0 {
			t.Errorf("a refused line was stored: %d", n)
		}
	})

	t.Run("a line of B cannot point at A's package, supplier or event", func(t *testing.T) {
		bBefore, bContract := lineCount(PB), contractOf(b, PB)
		cases := map[string]struct {
			line map[string]any
			key  string
		}{
			"A's package":  {map[string]any{"kind": "Package", "packageId": aPackage}, "packageId"},
			"A's supplier": {func() map[string]any { m := item("Vendor curian"); m["supplierRef"] = aSup; return m }(), "supplierRef"},
			"A's event":    {func() map[string]any { m := item("Acara curian"); m["eventId"] = aEvent; return m }(), "eventId"},
		}
		for name, c := range cases {
			routesCovered += 3
			add := s.doWith(b.token, "POST", proj(PB, "/lines"), c.line, version(b, PB))
			wantStatus(t, "add with "+name, add, http.StatusUnprocessableEntity)
			if !hasKey(add, c.key) {
				t.Errorf("add with %s: the refusal names %s: %s", name, c.key, add)
			}
			noLeakOfA(t, "add with "+name, add)
			put := s.doWith(b.token, "PUT", proj(PB, "/lines"), map[string]any{"lines": []map[string]any{c.line}}, version(b, PB))
			wantStatus(t, "replace with "+name, put, http.StatusUnprocessableEntity)
			if !hasKey(put, "lines.0."+c.key) {
				t.Errorf("replace with %s: the refusal names lines.0.%s: %s", name, c.key, put)
			}
			noLeakOfA(t, "replace with "+name, put)
			pre := s.do(b.token, "POST", proj(PB, "/lines/preview"), map[string]any{"lines": []map[string]any{c.line}})
			wantStatus(t, "preview with "+name, pre, http.StatusUnprocessableEntity)
			noLeakOfA(t, "preview with "+name, pre)
		}
		// Patching B's own line onto A's supplier or event: refused too.
		// POST answers with the whole Rincian; the new line is found by its name.
		added := s.mustOK("B line", s.doWith(b.token, "POST", proj(PB, "/lines"), item("Lighting "+b.marker), version(b, PB)))
		var own int64
		for _, l := range rows(added, "data", "lines") {
			if l["description"] == "Lighting "+b.marker {
				own = int64(l["id"].(float64))
			}
		}
		if own == 0 {
			t.Fatalf("B's own line is not in the answer: %s", added)
		}
		bBefore, bContract = lineCount(PB), contractOf(b, PB)
		for key, ref := range map[string]int64{"supplierRef": aSup, "eventId": aEvent} {
			routesCovered++
			r := s.doWith(b.token, "PATCH", proj(PB, fmt.Sprintf("/lines/%d", own)), map[string]any{key: ref}, version(b, PB))
			wantStatus(t, "patch "+key+" to A's", r, http.StatusUnprocessableEntity)
			noLeakOfA(t, "patch "+key, r)
		}
		if lineCount(PB) != bBefore || contractOf(b, PB) != bContract {
			t.Errorf("a refused line changed B's Rincian")
		}
		// Control: B's own supplier and event are accepted.
		ok := item("Sound " + b.marker)
		ok["supplierRef"], ok["eventId"] = bSup, bEvent
		s.mustOK("B line with its own refs", s.doWith(b.token, "POST", proj(PB, "/lines"), ok, version(b, PB)))
		if n := s.count("SELECT COUNT(*) FROM project_lines WHERE project_id = ? AND (supplier_ref = ? OR event_id = ?)", PB, aSup, aEvent); n != 0 {
			t.Errorf("B's lines point at A's rows: %d", n)
		}
	})

	// A's clients: Pengantin (vendor + finance), Keluarga (vendor, no finance)
	// and a role without portal.vendor.view.
	bride := s.addClient(a, PA, "Bride", "Pengantin", fmt.Sprintf("pengantin.l3.%d", time.Now().UnixNano()%1_000_000_000))
	family := s.addClient(a, PA, "Family Representative", "Keluarga", fmt.Sprintf("kel.l3.%d", time.Now().UnixNano()%1_000_000_000))
	s.mustOK("client role", s.do(a.token, "POST", "/api/v1/access/roles", map[string]any{
		"kind": "client", "name": "Dokumen L3", "description": "Uji", "permissions": []string{"portal.documents.view"},
	}))
	docsOnly := s.addClient(a, PA, "Family Representative", "Dokumen L3", fmt.Sprintf("dok.l3.%d", time.Now().UnixNano()%1_000_000_000))

	t.Run("WO A's client cannot reach WO B", func(t *testing.T) {
		for _, c := range []struct{ who, tok string }{{"Pengantin", bride}, {"Keluarga", family}} {
			for _, at := range []attack{
				{name: "B lines", method: "GET", path: proj(PB, "/lines")},
				{name: "B project", method: "GET", path: proj(PB, "")},
				{name: "B lines preview", method: "POST", path: proj(PB, "/lines/preview"), body: map[string]any{"lines": []map[string]any{item("x")}}},
				{name: "B line add", method: "POST", path: proj(PB, "/lines"), body: item("x")},
			} {
				routesCovered++
				r := s.doWith(c.tok, at.method, at.path, at.body, map[string]string{"If-Match": staleVersion})
				if r.status != http.StatusNotFound && r.status != http.StatusForbidden {
					t.Errorf("%s %s %s: want 403 or 404, got %s", c.who, at.method, at.path, r)
				}
				if strings.Contains(string(r.body), b.marker) {
					t.Errorf("%s %s leaks B: %s", c.who, at.path, r)
				}
			}
		}
	})

	t.Run("the portal Rincian: Keluarga sees names only, Pengantin no component price", func(t *testing.T) {
		routesCovered += 4
		fam := s.mustOK("Keluarga lines", s.do(family, "GET", proj(PA, "/lines"), nil))
		body := string(fam.body)
		for _, secret := range []string{"150000000", "7777777", "1234567", "164543210", "8000000", "4000000", "20000000",
			`"amount"`, `"unitPrice"`, `"price"`, `"stepPrice"`, `"totals"`, `"supplierRef"`, `"margin"`, `"linesVersion"`, "Vendor A"} {
			if strings.Contains(body, secret) {
				t.Errorf("Keluarga's Rincian carries %q: %s", secret, fam)
			}
		}
		if !strings.Contains(body, "Paket Melati "+a.marker) || !strings.Contains(body, "Dekorasi pelaminan "+a.marker) {
			t.Errorf("Keluarga sees the package and what it contains: %s", fam)
		}
		famProject := string(s.mustOK("Keluarga project", s.do(family, "GET", proj(PA, ""), nil)).body)
		for _, secret := range []string{"164543210", `"contractValue"`, `"contractTax"`, `"contractSubtotal"`} {
			if strings.Contains(famProject, secret) {
				t.Errorf("Keluarga's project carries %q: %s", secret, famProject)
			}
		}

		pas := s.mustOK("Pengantin lines", s.do(bride, "GET", proj(PA, "/lines"), nil))
		if pas.dig("data", "totals", "total") != float64(aTotal) {
			t.Errorf("Pengantin sees the total: %s", pas)
		}
		for _, l := range rows(pas, "data", "lines") {
			if l["kind"] == "Component" {
				if _, has := l["amount"]; has {
					t.Errorf("a component's price is never shown: %v", l)
				}
			}
			if _, has := l["supplierRef"]; has {
				t.Errorf("the supplier behind a line never reaches the portal: %v", l)
			}
		}
		for _, secret := range []string{`"margin"`, `"linesVersion"`, `"termsMismatch"`, "Vendor A", "20000000"} {
			if strings.Contains(string(pas.body), secret) {
				t.Errorf("Pengantin's Rincian carries %q: %s", secret, pas)
			}
		}
		wantStatus(t, "no portal.vendor.view", s.do(docsOnly, "GET", proj(PA, "/lines"), nil), http.StatusForbidden)
	})

	t.Run("a client never writes the Rincian", func(t *testing.T) {
		before := lineCount(PA)
		for _, at := range []attack{
			{name: "replace", method: "PUT", path: proj(PA, "/lines"), body: map[string]any{"lines": []map[string]any{item("x")}}},
			{name: "add", method: "POST", path: proj(PA, "/lines"), body: item("x")},
			{name: "preview", method: "POST", path: proj(PA, "/lines/preview"), body: map[string]any{"lines": []map[string]any{item("x")}}},
			{name: "reorder", method: "POST", path: proj(PA, "/lines/reorder"), body: map[string]any{"ids": []int64{aItem}}},
			{name: "update", method: "PATCH", path: proj(PA, fmt.Sprintf("/lines/%d", aItem)), body: map[string]any{"unitPrice": 1}},
			{name: "delete", method: "DELETE", path: proj(PA, fmt.Sprintf("/lines/%d", aItem))},
			{name: "restore", method: "POST", path: proj(PA, fmt.Sprintf("/lines/%d/restore", aItem))},
		} {
			routesCovered++
			for _, h := range []map[string]string{nil, {"If-Match": quoted(s.linesVersion(a, PA))}} {
				if r := s.doWith(bride, at.method, at.path, at.body, h); r.status != http.StatusForbidden {
					t.Errorf("client %s (%v): want 403, got %s", at.name, h, r)
				}
			}
		}
		if lineCount(PA) != before || contractOf(a, PA) != float64(aTotal) {
			t.Errorf("a client's refused write changed the Rincian")
		}
	})
}
