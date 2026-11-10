//go:build integration

package isolation

import (
	"bytes"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Tahap 9: GET /vendors/export and GET /venues/export honour exactly the filters
// of their list (q or its alias search, categoryId and city for vendors, city,
// isActive), so a filtered list and its export agree.

// exportNames reads the first column (the name) of an export workbook, header
// row skipped, sorted.
func exportNames(t *testing.T, s *stack, token, path string) []string {
	t.Helper()
	r := s.do(token, "GET", path, nil)
	if r.status != http.StatusOK {
		t.Fatalf("GET %s: %s", path, r)
	}
	if ct := r.header.Get("Content-Type"); !strings.Contains(ct, "spreadsheetml") {
		t.Fatalf("GET %s: content type %q", path, ct)
	}
	f, err := excelize.OpenReader(bytes.NewReader(r.body))
	if err != nil {
		t.Fatalf("GET %s: not a workbook: %v", path, err)
	}
	defer f.Close()
	grid, err := f.GetRows(f.GetSheetName(0))
	if err != nil || len(grid) == 0 {
		t.Fatalf("GET %s: no rows (%v)", path, err)
	}
	names := []string{}
	for _, row := range grid[1:] {
		if len(row) > 0 {
			names = append(names, row[0])
		}
	}
	sort.Strings(names)
	return names
}

// listNames reads the names of a paginated list answer (limit 100), sorted.
func listNames(t *testing.T, s *stack, token, path string) []string {
	t.Helper()
	names := []string{}
	for _, row := range rows(s.mustOK("GET "+path, s.do(token, "GET", path, nil)), "data") {
		names = append(names, row["name"].(string))
	}
	sort.Strings(names)
	return names
}

func exportQuery(base string, kv ...string) string {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return base + "?" + v.Encode()
}

func TestExportFilters(t *testing.T) {
	s := startStack(t)
	x := s.signUp("expf")
	other := s.signUp("expo")
	m := x.marker

	// ---- vendors: five rows, two cities, one inactive --------------------
	type spec struct{ name, city string }
	vendorSpecs := []spec{
		{"Melati Dekor " + m, "Kota Bandung"}, {"Mawar Foto " + m, "Kota Bandung"}, {"Anggrek Rias " + m, "Kota Surabaya"},
		{"Mawar Musik " + m, "Kota Surabaya"}, {"Teratai Katering " + m, "Kota Bandung"},
	}
	vendorIDs := map[string]int64{}
	var category int64
	for _, sp := range vendorSpecs {
		body := vendorBody(s, x, sp.name, 0)
		body["city"] = sp.city
		r := s.mustOK("vendor "+sp.name, s.do(x.token, "POST", "/api/v1/vendors", body))
		vendorIDs[sp.name] = r.id("data", "id")
		category = r.id("data", "categoryId")
	}
	// Another workspace's vendor must never show up in this one's export.
	s.mustOK("other vendor", s.do(other.token, "POST", "/api/v1/vendors", vendorBody(s, other, "Mawar Asing "+other.marker, 0)))

	inactive := "Anggrek Rias " + m
	s.mustOK("deactivate vendor", s.do(x.token, "POST", "/api/v1/vendors/bulk", map[string]any{"ids": []int64{vendorIDs[inactive]}, "action": "deactivate"}))

	all := []string{}
	for _, sp := range vendorSpecs {
		all = append(all, sp.name)
	}
	sort.Strings(all)
	without := func(drop string) []string {
		out := []string{}
		for _, n := range all {
			if n != drop {
				out = append(out, n)
			}
		}
		return out
	}

	t.Run("vendors: unfiltered export is unchanged (every row of the workspace, none of another)", func(t *testing.T) {
		if got := exportNames(t, s, x.token, "/api/v1/vendors/export"); !reflect.DeepEqual(got, all) {
			t.Errorf("unfiltered export = %v, want %v", got, all)
		}
		if got := listNames(t, s, x.token, "/api/v1/vendors?limit=100"); !reflect.DeepEqual(got, all) {
			t.Errorf("unfiltered list = %v, want %v", got, all)
		}
	})
	t.Run("vendors: isActive", func(t *testing.T) {
		if got := exportNames(t, s, x.token, "/api/v1/vendors/export?isActive=false"); !reflect.DeepEqual(got, []string{inactive}) {
			t.Errorf("isActive=false export = %v, want only %q", got, inactive)
		}
		if got := exportNames(t, s, x.token, "/api/v1/vendors/export?isActive=true"); !reflect.DeepEqual(got, without(inactive)) {
			t.Errorf("isActive=true export = %v, want %v", got, without(inactive))
		}
		wantStatus(t, "list isActive=maybe", s.do(x.token, "GET", "/api/v1/vendors?isActive=maybe", nil), http.StatusUnprocessableEntity)
		wantStatus(t, "export isActive=maybe", s.do(x.token, "GET", "/api/v1/vendors/export?isActive=maybe", nil), http.StatusUnprocessableEntity)
		wantStatus(t, "export categoryId=abc", s.do(x.token, "GET", "/api/v1/vendors/export?categoryId=abc", nil), http.StatusBadRequest)
	})
	t.Run("vendors: every filter agrees with the list", func(t *testing.T) {
		cat := strconv.FormatInt(category, 10)
		for name, kv := range map[string][]string{
			"q":                  {"q", "Mawar"},
			"search alias":       {"search", "Mawar"},
			"q wins over search": {"q", "Mawar", "search", "Melati"},
			"q is trimmed":       {"q", "  mawar  "},
			"q on the pic name":  {"q", "PIC " + m},
			"q and isActive":     {"q", "Anggrek", "isActive", "false"},
			"q matches nothing":  {"q", "tidak-ada-" + m},
			"city":               {"city", "Kota Surabaya"},
			"city and isActive":  {"city", "Kota Surabaya", "isActive", "true"},
			"categoryId":         {"categoryId", cat},
			"all of them":        {"q", "Mawar", "categoryId", cat, "city", "Kota Bandung", "isActive", "true"},
		} {
			kv := kv
			t.Run(name, func(t *testing.T) {
				listed := listNames(t, s, x.token, exportQuery("/api/v1/vendors", append(kv, "limit", "100")...))
				exported := exportNames(t, s, x.token, exportQuery("/api/v1/vendors/export", kv...))
				if !reflect.DeepEqual(listed, exported) {
					t.Errorf("list %v != export %v", listed, exported)
				}
			})
		}
		// The filters really narrow (the comparisons above would also pass if both ignored them).
		if got := exportNames(t, s, x.token, "/api/v1/vendors/export?q=Mawar"); len(got) != 2 {
			t.Errorf("q=Mawar export = %v, want the two Mawar vendors", got)
		}
		if got := exportNames(t, s, x.token, "/api/v1/vendors/export?search=Mawar"); len(got) != 2 {
			t.Errorf("search=Mawar export = %v, want the two Mawar vendors", got)
		}
		if got := exportNames(t, s, x.token, "/api/v1/vendors/export?q=Mawar&city=Kota+Surabaya&isActive=true"); !reflect.DeepEqual(got, []string{"Mawar Musik " + m}) {
			t.Errorf("combined export = %v", got)
		}
	})

	// ---- venues ----------------------------------------------------------
	venueSpecs := []spec{
		{"Gedung Melati " + m, "Kota Bandung"}, {"Aula Mawar " + m, "Kota Bandung"}, {"Ballroom Anggrek " + m, "Kota Surabaya"},
		{"Taman Mawar " + m, "Kota Surabaya"},
	}
	venueIDs := map[string]int64{}
	for _, sp := range venueSpecs {
		body := venueBody(x, sp.name)
		body["city"] = sp.city
		venueIDs[sp.name] = s.mustOK("venue "+sp.name, s.do(x.token, "POST", "/api/v1/venues", body)).id("data", "id")
	}
	s.mustOK("other venue", s.do(other.token, "POST", "/api/v1/venues", venueBody(other, "Aula Asing "+other.marker)))
	inactiveVenue := "Ballroom Anggrek " + m
	s.mustOK("deactivate venue", s.do(x.token, "POST", "/api/v1/venues/bulk", map[string]any{"ids": []int64{venueIDs[inactiveVenue]}, "action": "deactivate"}))

	allVenues := []string{}
	for _, sp := range venueSpecs {
		allVenues = append(allVenues, sp.name)
	}
	sort.Strings(allVenues)

	t.Run("venues: unfiltered export is unchanged", func(t *testing.T) {
		if got := exportNames(t, s, x.token, "/api/v1/venues/export"); !reflect.DeepEqual(got, allVenues) {
			t.Errorf("unfiltered export = %v, want %v", got, allVenues)
		}
	})
	t.Run("venues: isActive=false returns only the inactive row", func(t *testing.T) {
		if got := exportNames(t, s, x.token, "/api/v1/venues/export?isActive=false"); !reflect.DeepEqual(got, []string{inactiveVenue}) {
			t.Errorf("isActive=false export = %v, want only %q", got, inactiveVenue)
		}
		if got := exportNames(t, s, x.token, "/api/v1/venues/export?isActive=true"); len(got) != len(allVenues)-1 {
			t.Errorf("isActive=true export = %v", got)
		}
		wantStatus(t, "export isActive=maybe", s.do(x.token, "GET", "/api/v1/venues/export?isActive=maybe", nil), http.StatusUnprocessableEntity)
	})
	t.Run("venues: every filter agrees with the list", func(t *testing.T) {
		for name, kv := range map[string][]string{
			"q":                  {"q", "Mawar"},
			"search alias":       {"search", "Mawar"},
			"q wins over search": {"q", "Mawar", "search", "Melati"},
			"q and isActive":     {"q", "Anggrek", "isActive", "false"},
			"q matches nothing":  {"q", "tidak-ada-" + m},
			"city":               {"city", "Kota Surabaya"},
			"all of them":        {"q", "Mawar", "city", "Kota Surabaya", "isActive", "true"},
		} {
			kv := kv
			t.Run(name, func(t *testing.T) {
				listed := listNames(t, s, x.token, exportQuery("/api/v1/venues", append(kv, "limit", "100")...))
				exported := exportNames(t, s, x.token, exportQuery("/api/v1/venues/export", kv...))
				if !reflect.DeepEqual(listed, exported) {
					t.Errorf("list %v != export %v", listed, exported)
				}
			})
		}
		if got := exportNames(t, s, x.token, "/api/v1/venues/export?q=Mawar"); len(got) != 2 {
			t.Errorf("q=Mawar export = %v, want the two Mawar venues", got)
		}
		if got := exportNames(t, s, x.token, "/api/v1/venues/export?q=Mawar&city=Kota+Surabaya"); !reflect.DeepEqual(got, []string{"Taman Mawar " + m}) {
			t.Errorf("combined export = %v", got)
		}
	})
}
