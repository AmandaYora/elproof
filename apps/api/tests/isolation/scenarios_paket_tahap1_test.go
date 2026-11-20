//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Paket WO, Tahap 1 (docs/plan/paket-wo/PLAN.md §8): the catalog of event
// kinds and packages, the Penawaran section of the business profile, and the
// venue's two capacities. Two workspaces sign up: A owns the rows, B must
// never see, change, delete, or use them (ADR-0004).
func TestPaketTahap1(t *testing.T) {
	s := startStack(t)
	a := s.signUp("PA")
	b := s.signUp("PB")

	kindIDs := func(tok string) map[string]int64 {
		out := map[string]int64{}
		for _, k := range rows(s.mustOK("event kinds", s.do(tok, "GET", "/api/v1/event-kinds", nil)), "data") {
			out[fmt.Sprint(k["name"])] = int64(k["id"].(float64))
		}
		return out
	}

	t.Run("a new WO starts with six event kinds and a document code", func(t *testing.T) {
		kinds := kindIDs(a.token)
		for _, name := range []string{"Lamaran", "Pengajian", "Siraman", "Akad", "Resepsi", "Ngunduh mantu"} {
			if kinds[name] == 0 {
				t.Errorf("missing built-in kind %q: %v", name, kinds)
			}
		}
		if len(kinds) != 6 {
			t.Errorf("want 6 kinds, got %v", kinds)
		}
		q := s.mustOK("quote settings", s.do(a.token, "GET", "/api/v1/tenants/me/quote-settings", nil))
		code, _ := q.dig("data", "documentCode").(string)
		if !regexp.MustCompile(`^[A-Z0-9]{2,5}$`).MatchString(code) {
			t.Errorf("document code %q", code)
		}
		if q.dig("data", "isPkp") != false || q.dig("data", "defaultValidityDays") != float64(14) {
			t.Errorf("defaults: %s", q)
		}
		if terms, _ := q.dig("data", "defaultTermsText").(string); !strings.Contains(terms, "1.") {
			t.Errorf("default terms: %q", terms)
		}
	})

	t.Run("a venue without capacity cannot be saved", func(t *testing.T) {
		body := venueBody(a, "Tanpa kapasitas "+a.marker)
		delete(body, "capacityStanding")
		r := s.do(a.token, "POST", "/api/v1/venues", body)
		wantStatus(t, "venue without capacity", r, http.StatusUnprocessableEntity)
	})

	aKinds := kindIDs(a.token)
	bKinds := kindIDs(b.token)
	venueBodyA := venueBody(a, "Gedung Melati "+a.marker)
	venueBodyA["capacityStanding"], venueBodyA["capacitySeated"] = 600, 400
	aVenue := s.mustOK("A venue", s.do(a.token, "POST", "/api/v1/venues", venueBodyA)).id("data", "id")
	aVendor := s.mustOK("A vendor", s.do(a.token, "POST", "/api/v1/vendors", vendorBody(s, a, "Dapur "+a.marker, 0))).id("data", "id")

	pkg := func(name string, included int, venue, vendor int64, kinds ...int64) map[string]any {
		body := map[string]any{
			"name": name, "price": 120000000, "eventKindIds": kinds,
			"includedGuests": included, "extraGuestStep": 50, "extraGuestStepPrice": 4000000,
			"components": []map[string]any{{"description": "Katering " + name, "estimatedCost": 60000000}},
		}
		if venue != 0 {
			body["defaultVenueId"] = venue
		}
		if vendor != 0 {
			body["components"].([]map[string]any)[0]["vendorId"] = vendor
		}
		return body
	}

	var aPackage int64
	t.Run("default venue capacity bounds the included guests", func(t *testing.T) {
		r := s.do(a.token, "POST", "/api/v1/packages", pkg("Paket 650 "+a.marker, 650, aVenue, aVendor, aKinds["Akad"], aKinds["Resepsi"]))
		wantStatus(t, "650 in a 600 venue", r, http.StatusUnprocessableEntity)
		if msg := fmt.Sprint(r.dig("errors", "includedGuests")); !strings.Contains(msg, "600") || !strings.Contains(msg, "650") {
			t.Errorf("message should name both numbers: %s", r)
		}
		ok := s.mustOK("A package", s.do(a.token, "POST", "/api/v1/packages", pkg("Paket Resepsi "+a.marker, 500, aVenue, aVendor, aKinds["Akad"], aKinds["Resepsi"])))
		aPackage = ok.id("data", "id")
		if ok.dig("data", "components") == nil || ok.dig("data", "defaultVenue", "name") == nil {
			t.Errorf("package response: %s", ok)
		}
	})

	t.Run("preview prices extra guests with the same rule", func(t *testing.T) {
		r := s.mustOK("preview", s.do(a.token, "POST", "/api/v1/packages/preview", map[string]any{
			"price": 120000000, "includedGuests": 500, "extraGuestStep": 50, "extraGuestStepPrice": 4000000, "guests": 620,
		}))
		if r.dig("data", "extra", "amount") != float64(12000000) || r.dig("data", "total") != float64(132000000) {
			t.Errorf("preview: %s", r)
		}
	})

	t.Run("workspace B cannot see, change, or delete A's package", func(t *testing.T) {
		list := s.mustOK("B packages", s.do(b.token, "GET", "/api/v1/packages", nil))
		if strings.Contains(string(list.body), a.marker) {
			t.Errorf("B's list shows A's package: %s", list)
		}
		path := fmt.Sprintf("/api/v1/packages/%d", aPackage)
		wantStatus(t, "B GET", s.do(b.token, "GET", path, nil), http.StatusNotFound)
		wantStatus(t, "B PATCH state", s.do(b.token, "PATCH", path, map[string]any{"isActive": false}), http.StatusNotFound)
		wantStatus(t, "B PATCH edit", s.do(b.token, "PATCH", path, pkg("Curian", 100, 0, 0, bKinds["Akad"])), http.StatusNotFound)
		wantStatus(t, "B DELETE", s.do(b.token, "DELETE", path, nil), http.StatusNotFound)
		s.mustOK("A still has it", s.do(a.token, "GET", path, nil))
	})

	t.Run("workspace B cannot use A's venue, vendor, or event kind", func(t *testing.T) {
		r := s.do(b.token, "POST", "/api/v1/packages", pkg("B venue "+b.marker, 100, aVenue, 0, bKinds["Akad"]))
		wantStatus(t, "A's venue as B's default venue", r, http.StatusUnprocessableEntity)
		if r.dig("errors", "defaultVenueId") == nil {
			t.Errorf("want a defaultVenueId error: %s", r)
		}
		if strings.Contains(string(r.body), "Gedung Melati") {
			t.Errorf("refusal leaks A's venue name: %s", r)
		}
		r = s.do(b.token, "POST", "/api/v1/packages", pkg("B vendor "+b.marker, 100, 0, aVendor, bKinds["Akad"]))
		wantStatus(t, "A's vendor in B's package", r, http.StatusUnprocessableEntity)
		r = s.do(b.token, "POST", "/api/v1/packages", pkg("B kind "+b.marker, 100, 0, 0, aKinds["Akad"]))
		wantStatus(t, "A's event kind in B's package", r, http.StatusUnprocessableEntity)
	})

	t.Run("workspace B cannot change or delete A's event kinds", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/event-kinds/%d", aKinds["Lamaran"])
		wantStatus(t, "B PATCH state", s.do(b.token, "PATCH", path, map[string]any{"isActive": false}), http.StatusNotFound)
		wantStatus(t, "B rename", s.do(b.token, "PATCH", path, map[string]any{"name": "Curian"}), http.StatusNotFound)
		wantStatus(t, "B DELETE", s.do(b.token, "DELETE", path, nil), http.StatusNotFound)
		if kindIDs(a.token)["Lamaran"] == 0 {
			t.Error("A lost its kind")
		}
	})

	t.Run("a directory row a package refers to cannot be deleted permanently", func(t *testing.T) {
		r := s.do(a.token, "DELETE", fmt.Sprintf("/api/v1/vendors/%d", aVendor), nil)
		wantStatus(t, "delete referenced vendor", r, http.StatusConflict)
		if r.dig("errors", "code") != "vendor_in_offers" {
			t.Errorf("want vendor_in_offers: %s", r)
		}
		r = s.do(a.token, "DELETE", fmt.Sprintf("/api/v1/venues/%d", aVenue), nil)
		wantStatus(t, "delete referenced venue", r, http.StatusConflict)
		if r.dig("errors", "code") != "venue_in_offers" {
			t.Errorf("want venue_in_offers: %s", r)
		}
	})

	t.Run("an event kind a package covers cannot be deleted", func(t *testing.T) {
		r := s.do(a.token, "DELETE", fmt.Sprintf("/api/v1/event-kinds/%d", aKinds["Akad"]), nil)
		wantStatus(t, "delete used kind", r, http.StatusConflict)
		s.mustOK("delete unused kind", s.do(a.token, "DELETE", fmt.Sprintf("/api/v1/event-kinds/%d", aKinds["Siraman"]), nil))
	})

	t.Run("editing a package needs If-Match", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/packages/%d", aPackage)
		body := pkg("Paket Resepsi "+a.marker, 450, aVenue, aVendor, aKinds["Resepsi"])
		wantStatus(t, "no If-Match", s.do(a.token, "PATCH", path, body), http.StatusPreconditionRequired)
		wantStatus(t, "stale If-Match", s.doWith(a.token, "PATCH", path, body, map[string]string{"If-Match": `"2001-01-01T00:00:00.000000Z"`}), http.StatusConflict)
		current, _ := s.mustOK("read", s.do(a.token, "GET", path, nil)).dig("data", "updatedAt").(string)
		r := s.mustOK("fresh If-Match", s.doWith(a.token, "PATCH", path, body, map[string]string{"If-Match": `"` + current + `"`}))
		if r.dig("data", "includedGuests") != float64(450) {
			t.Errorf("edit not saved: %s", r)
		}
	})
}
