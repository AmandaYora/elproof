//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Paket WO, Tahap 5: "Jadikan proyek" (/api/v1/projects/from-quotation/**) and
// "Kesepakatan awal" (/api/v1/projects/{id}/agreement). Another workspace can
// neither convert, preview nor upload against an accepted quotation, nor read
// the agreement of a project made from one; and a conversion of its own cannot
// name the other workspace's staff.
func TestPaketTahap5Convert(t *testing.T) {
	s := startStack(t)
	a, b := s.signUp("C5A"), s.signUp("C5B")
	qPath := func(id int64, rest string) string { return fmt.Sprintf("/api/v1/quotations/%d%s", id, rest) }
	convPath := func(id int64, rest string) string {
		return fmt.Sprintf("/api/v1/projects/from-quotation/%d%s", id, rest)
	}
	projectsOf := func(quotationID int64) int {
		return s.count("SELECT COUNT(*) FROM projects WHERE quotation_id = ?", quotationID)
	}

	// acceptedQuotation builds, in workspace tn, a quotation with one dated
	// Resepsi at a directory venue, an item of a unique amount, one revision,
	// and marks it Accepted.
	acceptedQuotation := func(tn tenant, amount int64) int64 {
		resepsi := s.kind(tn, "Resepsi")
		venue := s.mustOK(tn.name+" venue", s.do(tn.token, "POST", "/api/v1/venues", venueBody(tn, "Gedung "+tn.marker))).id("data", "id")
		q := s.mustOK(tn.name+" quotation", s.do(tn.token, "POST", "/api/v1/quotations", map[string]any{
			"coupleNames": "Ayu " + tn.marker + " & Bima", "contactWhatsapp": "081299990000",
		})).id("data", "quotation", "id")
		if q == 0 {
			t.Fatalf("%s's quotation has no id", tn.name)
		}
		s.mustOK(tn.name+" event", s.do(tn.token, "POST", qPath(q, "/events"), map[string]any{
			"eventKindId": resepsi, "eventDate": day(120), "venueId": venue, "layout": "Standing",
			"sessions": []map[string]any{{"label": "Sesi 1", "guests": 200}},
		}))
		linesVersion, _ := s.mustOK(tn.name+" detail", s.do(tn.token, "GET", qPath(q, ""), nil)).dig("data", "quotation", "linesUpdatedAt").(string)
		s.mustOK(tn.name+" lines", s.doWith(tn.token, "PUT", qPath(q, "/lines"), map[string]any{"lines": []map[string]any{
			{"kind": "Item", "description": "Dokumentasi " + tn.marker, "quantity": 1, "unit": "paket", "unitPrice": amount},
		}}, map[string]string{"If-Match": quoted(linesVersion)}))
		if r := s.do(tn.token, "POST", qPath(q, "/revisions"), map[string]any{}); r.status != http.StatusCreated {
			t.Fatalf("%s's revision: want 201, got %s", tn.name, r)
		}
		s.mustOK(tn.name+" accept", s.do(tn.token, "POST", qPath(q, "/status"), map[string]any{
			"status": "Accepted", "revisionN": 1, "acceptedOn": day(0), "channel": "WhatsApp",
		}))
		return q
	}
	convertBody := func(tn tenant) map[string]any {
		return map[string]any{"events": []map[string]any{{"eventDate": day(120)}}, "prepStartDate": day(1), "picStaffId": tn.staffID}
	}

	QA := acceptedQuotation(a, 86_420_975)
	aSecrets := []string{a.marker, "86420975"}
	noLeakOfA := func(t *testing.T, what string, r reply) {
		t.Helper()
		for _, secret := range aSecrets {
			if strings.Contains(string(r.body), secret) {
				t.Errorf("%s leaks %q: %s", what, secret, r)
			}
		}
	}

	t.Run("WO B cannot convert, preview or upload against WO A's quotation", func(t *testing.T) {
		attacks := []attack{
			{name: "convert", method: "POST", path: convPath(QA, ""), body: convertBody(b)},
			{name: "convert with a down payment", method: "POST", path: convPath(QA, ""), body: func() map[string]any {
				body := convertBody(b)
				body["downPayment"] = map[string]any{"amount": 1_000_000, "paymentDate": day(0), "method": "Transfer"}
				return body
			}()},
			{name: "preview", method: "POST", path: convPath(QA, "/preview"), body: convertBody(b)},
			{name: "upload grant", method: "POST", path: convPath(QA, "/uploads"), body: map[string]any{
				"fileName": "bukti.pdf", "mimeType": "application/pdf", "size": 1024}},
			{name: "upload bytes", method: "PUT", path: convPath(QA, "/uploads/eyJ0.sig"), body: map[string]any{}},
		}
		for _, at := range attacks {
			at := at
			routesCovered++
			t.Run(at.name, func(t *testing.T) {
				r := s.do(b.token, at.method, at.path, at.body)
				if r.status != http.StatusNotFound {
					t.Errorf("B %s %s: want 404, got %s", at.method, at.path, r)
				}
				if vacuous(r) {
					t.Errorf("B %s %s: router miss, not an ownership answer: %s", at.method, at.path, r)
				}
				noLeakOfA(t, "B "+at.method+" "+at.path, r)
			})
		}
		if n := projectsOf(QA); n != 0 {
			t.Fatalf("B's attempts made %d project(s) from A's quotation", n)
		}
	})

	// A converts its own quotation: the project carries the quotation.
	routesCovered++
	made := s.do(a.token, "POST", convPath(QA, ""), convertBody(a))
	wantStatus(t, "A converts its quotation", made, http.StatusCreated)
	PA := made.id("data", "id")
	if PA == 0 || made.id("data", "quotation", "id") != QA {
		t.Fatalf("A's conversion: %s", made)
	}
	routesCovered++
	again := s.do(a.token, "POST", convPath(QA, ""), convertBody(a))
	wantStatus(t, "A converts again", again, http.StatusOK)
	if again.id("data", "id") != PA || again.dig("data", "created") != false {
		t.Errorf("a second conversion answers the first project: %s", again)
	}
	if n := projectsOf(QA); n != 1 {
		t.Fatalf("A's quotation became %d projects, want 1", n)
	}
	s.mustOK("A reads its agreement", s.do(a.token, "GET", s.projectsPath(PA, "/agreement"), nil))

	t.Run("WO B cannot read the agreement of WO A's project", func(t *testing.T) {
		for _, path := range []string{s.projectsPath(PA, "/agreement"), s.projectsPath(PA, "")} {
			routesCovered++
			r := s.do(b.token, "GET", path, nil)
			wantStatus(t, "B GET "+path, r, http.StatusNotFound)
			if vacuous(r) {
				t.Errorf("B GET %s: router miss, not an ownership answer: %s", path, r)
			}
			noLeakOfA(t, "B GET "+path, r)
		}
		// After the conversion, B's attempts still answer 404, never the
		// "already a project" 200.
		routesCovered++
		r := s.do(b.token, "POST", convPath(QA, ""), convertBody(b))
		wantStatus(t, "B converts A's converted quotation", r, http.StatusNotFound)
		noLeakOfA(t, "B converts A's converted quotation", r)
		routesCovered++
		r = s.do(b.token, "POST", convPath(QA, "/preview"), convertBody(b))
		wantStatus(t, "B previews A's converted quotation", r, http.StatusNotFound)
		noLeakOfA(t, "B previews A's converted quotation", r)
	})

	t.Run("WO B's own conversion cannot name WO A's staff", func(t *testing.T) {
		QB := acceptedQuotation(b, 13_579_246)
		for _, key := range []string{"picStaffId", "picSalesStaffId"} {
			routesCovered++
			body := convertBody(b)
			body[key] = a.staffID
			r := s.do(b.token, "POST", convPath(QB, ""), body)
			wantStatus(t, "B conversion with A's staff as "+key, r, http.StatusUnprocessableEntity)
			if !hasKey(r, "picStaffId") && !hasKey(r, key) {
				t.Errorf("the refusal names the PIC: %s", r)
			}
			noLeakOfA(t, "B conversion with A's staff", r)
		}
		if n := projectsOf(QB); n != 0 {
			t.Errorf("a refused conversion was stored: %d", n)
		}
		// B's quotation converts with B's own staff, and A cannot read its agreement.
		routesCovered++
		mine := s.do(b.token, "POST", convPath(QB, ""), convertBody(b))
		wantStatus(t, "B converts its quotation", mine, http.StatusCreated)
		PB := mine.id("data", "id")
		routesCovered++
		wantStatus(t, "A reads B's agreement", s.do(a.token, "GET", s.projectsPath(PB, "/agreement"), nil), http.StatusNotFound)
	})

	// A's project lost nothing to B's attempts.
	if ag := s.mustOK("A agreement", s.do(a.token, "GET", s.projectsPath(PA, "/agreement"), nil)); ag.id("data", "quotationId") != QA ||
		ag.dig("data", "revisionN") != float64(1) {
		t.Errorf("A's agreement moved: %s", ag)
	}
}
