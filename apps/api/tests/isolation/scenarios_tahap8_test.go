//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tahap 8 / E9 (ADR-0036): partial PATCH, optimistic concurrency with If-Match,
// and explicit state instead of toggles. One scenario signs up its own
// workspaces (X, whose rows are edited, and Y, who must never reach them) and
// drives the real server binary against a real MySQL.
//
// "Writes nothing" is proved with BEFORE UPDATE triggers on the tables that
// carry a state flag: a trigger fires for every row an UPDATE statement matches,
// even when the new values equal the old ones (MySQL then does not bump
// updated_at, so the version alone could not tell a redundant UPDATE from no
// UPDATE).

var updatedAtRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`)

// ---- helpers ----

// removedRoute asserts that a request to a deleted toggle route is answered exactly
// as any unknown route of its subtree is: 404 "Endpoint tidak ditemukan", not a
// 405/400/422 from some other branch of the router.
func removedRoute(t *testing.T, what string, r reply) {
	t.Helper()
	wantStatus(t, what+" (removed route)", r, http.StatusNotFound)
	if msg, _ := r.dig("message").(string); msg != "Endpoint tidak ditemukan" {
		t.Errorf("%s (removed route): want the router's own \"Endpoint tidak ditemukan\", got %s", what, r)
	}
}

// patch sends PATCH with an optional If-Match.
func (s *stack) patch(token, path string, body any, ifMatch string) reply {
	s.t.Helper()
	var h map[string]string
	if ifMatch != "" {
		h = map[string]string{"If-Match": ifMatch}
	}
	return s.doWith(token, "PATCH", path, body, h)
}

// patchRaw sends PATCH with a raw body.
func (s *stack) patchRaw(token, path, raw, ifMatch string) reply {
	s.t.Helper()
	var h map[string]string
	if ifMatch != "" {
		h = map[string]string{"If-Match": ifMatch}
	}
	return s.send(token, "PATCH", path, strings.NewReader(raw), true, "application/json", h)
}

// version is the item's `updatedAt` as a response carries it.
func version(r reply, keys ...string) string {
	return str(r, append([]string{"data"}, append(keys, "updatedAt")...)...)
}

// quoted is a version as an If-Match header value.
func quoted(v string) string { return `"` + v + `"` }

// watchWrites installs a BEFORE UPDATE trigger on each table that records every
// row an UPDATE touches, so a test can count writes exactly.
func (s *stack) watchWrites(tables ...string) {
	s.t.Helper()
	if _, err := s.db.Exec(`CREATE TABLE t8_writes (n BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, tbl VARCHAR(40) NOT NULL, row_id BIGINT UNSIGNED NOT NULL)`); err != nil {
		s.t.Fatal(err)
	}
	for _, tbl := range tables {
		q := fmt.Sprintf(`CREATE TRIGGER t8_%s_upd BEFORE UPDATE ON %s FOR EACH ROW INSERT INTO t8_writes (tbl, row_id) VALUES ('%s', OLD.id)`, tbl, tbl, tbl)
		if _, err := s.db.Exec(q); err != nil {
			s.t.Fatalf("trigger on %s: %v", tbl, err)
		}
	}
}

func (s *stack) writes(table string, id int64) int {
	s.t.Helper()
	return s.count(`SELECT COUNT(*) FROM t8_writes WHERE tbl = ? AND row_id = ?`, table, id)
}

// activity is how many entries the project's log has and the newest one's text.
func (s *stack) activity(projectID int64) (int, string) {
	s.t.Helper()
	n := s.count(`SELECT COUNT(*) FROM activity_log WHERE project_id = ?`, projectID)
	var desc string
	_ = s.db.QueryRow(`SELECT description FROM activity_log WHERE project_id = ? ORDER BY id DESC LIMIT 1`, projectID).Scan(&desc)
	return n, desc
}

func (s *stack) get(token, path string) reply {
	s.t.Helper()
	return s.mustOK("GET "+path, s.do(token, "GET", path, nil))
}

func data(r reply) map[string]any {
	m, _ := r.dig("data").(map[string]any)
	return m
}

func wantStatus(t *testing.T, what string, r reply, want int) {
	t.Helper()
	if r.status != want {
		t.Errorf("%s: want %d, got %s", what, want, r)
	}
}

// t8 is what the scenario edits.
type t8 struct {
	project, venue, engagement, engagement2, vmile, pmile, issue int64
	clientPay, vendorPay, venuePay                               int64
	vendor, category                                             int64
}

func (s *stack) t8Fixtures(x tenant) t8 {
	s.t.Helper()
	var f t8
	f.project = s.createProject(x, "T8 Proyek "+x.marker, 100_000_000)
	f.venue = s.mustOK("venue", s.do(x.token, "POST", "/api/v1/venues", venueBody(x, "Venue T8 "+x.marker))).id("data", "id")
	// Attach the venue through the new partial PATCH (its snapshot comes with it).
	s.mustOK("attach venue", s.patch(x.token, s.projectsPath(f.project, ""), map[string]any{"venueId": f.venue}, ""))
	f.vendor, f.category = s.newVendor(x, "Vendor T8 "+x.marker)
	f.engagement = s.engage(x, f.project, f.vendor, f.category, "Dekorasi", 5_000_000, "Booked", nil)
	f.engagement2 = s.engage(x, f.project, f.vendor, f.category, "Foto", 3_000_000, "Booked", nil)
	f.vmile = s.mustOK("vendor milestone", s.do(x.token, "POST", s.projectsPath(f.project, fmt.Sprintf("/vendors/%d/milestones", f.engagement)), map[string]any{
		"name": "Milestone T8", "description": "d", "targetDate": day(30), "picStaffId": x.staffID,
	})).id("data", "id")
	ms := s.get(x.token, s.projectsPath(f.project, "/milestones"))
	f.pmile = int64(rows(ms, "data")[0]["id"].(float64))
	f.issue = s.mustOK("issue", s.do(x.token, "POST", s.projectsPath(f.project, "/issues"), map[string]any{
		"projectVendorId": f.engagement, "vendorMilestoneId": f.vmile, "title": "Kendala T8", "description": "deskripsi", "impact": "Medium",
		"foundDate": day(-1), "resolutionPlan": "rencana", "picStaffId": x.staffID, "targetResolutionDate": day(10),
	})).id("data", "id")
	f.clientPay = s.clientPay(x, f.project, "DP", 10_000_000, "REF-T8")
	f.vendorPay = s.mustOK("vendor payment", s.do(x.token, "POST", s.projectsPath(f.project, "/payments"), map[string]any{
		"projectVendorId": f.engagement, "type": "DP", "amount": 1_000_000, "paymentDate": day(-2), "method": "Transfer", "referenceNumber": "VREF-T8",
	})).id("data", "id")
	f.venuePay = s.mustOK("venue payment", s.do(x.token, "POST", s.projectsPath(f.project, "/venue-payments"), map[string]any{
		"type": "DP", "amount": 777_777, "paymentDate": day(-2), "method": "Transfer", "referenceNumber": "VNREF-T8",
	})).id("data", "id")
	return f
}

// family is one resource the partial PATCH covers: where it lives, how to read it
// back, one valid change, and one change that is only wrong against the merged row.
type family struct {
	name string
	path string
	get  string // GET path whose data is the item, or a list the item is in
	list bool
	// patch is a valid single-field change; field/want say what must hold after it.
	patch        map[string]any
	field        string
	want         any
	untouched    []string // fields that must not move
	notNullable  string   // a required field: null is a 422 on that key
	badValue     map[string]any
	badKey       string
	mergedBad    map[string]any // valid on its own, wrong against the merged row
	mergedBadKey string
	// vary(n) is a valid change whose value differs for every n, so it is always a
	// real write; varyField is the field it sets.
	vary      func(n int) map[string]any
	varyField string
	// reset puts the row back in the state mergedBad is wrong against (earlier
	// scenarios move it around).
	reset map[string]any
}

func text(field string) func(n int) map[string]any {
	return func(n int) map[string]any { return map[string]any{field: fmt.Sprintf("nilai %s %d", field, n)} }
}

func (f family) read(t *testing.T, s *stack, tok string) map[string]any {
	t.Helper()
	r := s.get(tok, f.get)
	if !f.list {
		return data(r)
	}
	id := f.path[strings.LastIndex(f.path, "/")+1:]
	for _, m := range rows(r, "data") {
		if fmt.Sprint(int64(m["id"].(float64))) == id {
			return m
		}
	}
	t.Fatalf("%s: item %s not in %s", f.name, id, f.get)
	return nil
}

func TestTahap8(t *testing.T) {
	s := startStack(t)
	if _, err := s.db.Exec(`UPDATE trial_limits SET max_staff = 60, max_active_projects = 200, max_clients = 400 WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	x, y := s.signUp("t8x"), s.signUp("t8y")
	s.watchWrites("projects", "evidence", "vendors", "venues", "vendor_categories", "clients", "staff_members")
	f := s.t8Fixtures(x)

	pp := func(rest string) string { return s.projectsPath(f.project, rest) }
	projectFam := func() family {
		return family{name: "project", path: pp(""), get: pp(""), notNullable: "name", badValue: map[string]any{"eventDate": "kemarin"}, badKey: "eventDate",
			vary: text("description"), varyField: "description"}
	}
	families := []family{
		{
			name: "project milestone", path: pp(fmt.Sprintf("/milestones/%d", f.pmile)), get: pp("/milestones"), list: true,
			patch: map[string]any{"status": "In Progress"}, field: "status", want: "In Progress",
			untouched: []string{"name", "targetDate", "order"}, notNullable: "targetDate",
			badValue: map[string]any{"targetDate": "besok"}, badKey: "targetDate",
			vary: func(n int) map[string]any { return map[string]any{"targetDate": day(200 + n)} }, varyField: "targetDate",
		},
		{
			name: "project vendor", path: pp(fmt.Sprintf("/vendors/%d", f.engagement)), get: pp("/vendors"), list: true,
			patch: map[string]any{"scope": "Dekorasi baru"}, field: "scope", want: "Dekorasi baru",
			untouched: []string{"contractValue", "pricingTier", "engagementStatus", "dpAmount", "vendorId", "categoryId"}, notNullable: "scope",
			badValue: map[string]any{"eventStartTime": "25:99"}, badKey: "eventStartTime",
			vary: text("scope"), varyField: "scope",
		},
		{
			name: "vendor milestone", path: pp(fmt.Sprintf("/vendors/%d/milestones/%d", f.engagement, f.vmile)), get: pp(fmt.Sprintf("/vendors/%d/milestones", f.engagement)), list: true,
			patch: map[string]any{"notes": "catatan baru"}, field: "notes", want: "catatan baru",
			untouched: []string{"name", "description", "status", "targetDate", "picStaffId"}, notNullable: "status",
			badValue: map[string]any{"status": "Selesai banget"}, badKey: "status",
			vary: text("notes"), varyField: "notes",
		},
		{
			name: "kendala", path: pp(fmt.Sprintf("/issues/%d", f.issue)), get: pp("/issues"), list: true,
			patch: map[string]any{"impact": "High"}, field: "impact", want: "High",
			untouched: []string{"title", "description", "status", "resolutionPlan", "vendorMilestoneId", "projectVendorId", "targetResolutionDate"}, notNullable: "title",
			badValue: map[string]any{"impact": "Enormous"}, badKey: "impact",
			// Moving the kendala to another vendor while its milestone stays on the first one.
			mergedBad: map[string]any{"projectVendorId": f.engagement2}, mergedBadKey: "vendorMilestoneId",
			vary: text("description"), varyField: "description",
			reset: map[string]any{"projectVendorId": f.engagement, "vendorMilestoneId": f.vmile},
		},
		{
			name: "client payment", path: pp(fmt.Sprintf("/client-payments/%d", f.clientPay)), get: pp("/client-payments"), list: true,
			patch: map[string]any{"notes": "lunas sebagian"}, field: "notes", want: "lunas sebagian",
			untouched: []string{"type", "amount", "paymentDate", "method", "referenceNumber"}, notNullable: "amount",
			badValue: map[string]any{"type": "Hibah"}, badKey: "type",
			mergedBad: map[string]any{"amount": 100_000_001}, mergedBadKey: "amount",
			vary: text("notes"), varyField: "notes",
		},
		{
			name: "vendor payment", path: pp(fmt.Sprintf("/payments/%d", f.vendorPay)), get: pp("/payments"), list: true,
			patch: map[string]any{"method": "Tunai"}, field: "method", want: "Tunai",
			untouched: []string{"type", "amount", "paymentDate", "referenceNumber", "projectVendorId"}, notNullable: "type",
			badValue: map[string]any{"amount": 0}, badKey: "amount",
			mergedBad: map[string]any{"amount": 5_000_001}, mergedBadKey: "amount",
			vary: text("notes"), varyField: "notes",
		},
		{
			name: "venue payment", path: pp(fmt.Sprintf("/venue-payments/%d", f.venuePay)), get: pp("/venue-payments"), list: true,
			patch: map[string]any{"referenceNumber": "VN-BARU"}, field: "referenceNumber", want: "VN-BARU",
			untouched: []string{"type", "amount", "paymentDate", "method"}, notNullable: "method",
			badValue: map[string]any{"paymentDate": "2026/01/01"}, badKey: "paymentDate",
			mergedBad: map[string]any{"amount": 10_500_001}, mergedBadKey: "amount",
			vary: text("notes"), varyField: "notes",
		},
	}

	// ------------------------------------------------------------------
	t.Run("every item in scope carries updatedAt (RFC 3339, microseconds, UTC) and the ETag matches", func(t *testing.T) {
		single := s.get(x.token, pp(""))
		if v := version(single); !updatedAtRE.MatchString(v) || single.header.Get("ETag") != quoted(v) {
			t.Errorf("project updatedAt = %q, ETag = %q", v, single.header.Get("ETag"))
		}
		for _, l := range []struct {
			what, path string
		}{
			{"project list", "/api/v1/projects"},
			{"milestones", pp("/milestones")},
			{"vendors", pp("/vendors")},
			{"payments", pp("/payments")},
			{"client payments", pp("/client-payments")},
			{"venue payments", pp("/venue-payments")},
			{"issues", pp("/issues")},
		} {
			items := rows(s.get(x.token, l.path), "data")
			if len(items) == 0 {
				t.Errorf("%s: empty", l.what)
			}
			for _, it := range items {
				if v, _ := it["updatedAt"].(string); !updatedAtRE.MatchString(v) {
					t.Errorf("%s: item updatedAt = %q", l.what, v)
				}
			}
		}
		// Vendor milestones come nested in the vendor list and on their own route.
		for _, v := range rows(s.get(x.token, pp("/vendors")), "data") {
			ms, _ := v["milestones"].([]any)
			for _, m := range ms {
				if u, _ := m.(map[string]any)["updatedAt"].(string); !updatedAtRE.MatchString(u) {
					t.Errorf("nested vendor milestone updatedAt = %q", u)
				}
			}
		}
		for _, m := range rows(s.get(x.token, pp(fmt.Sprintf("/vendors/%d/milestones", f.engagement))), "data") {
			if u, _ := m["updatedAt"].(string); !updatedAtRE.MatchString(u) {
				t.Errorf("vendor milestone updatedAt = %q", u)
			}
		}
		// The answer of a PATCH carries the new version in the body and in ETag.
		for _, fam := range families {
			r := s.patch(x.token, fam.path, fam.vary(1000), "")
			if r.status != 200 {
				t.Errorf("%s: patch %s", fam.name, r)
				continue
			}
			v := version(r)
			if !updatedAtRE.MatchString(v) || r.header.Get("ETag") != quoted(v) {
				t.Errorf("%s: updatedAt %q, ETag %q", fam.name, v, r.header.Get("ETag"))
			}
		}
		// Microseconds round-trip through the driver: twenty edits in a row, each
		// guarded by the version the previous one returned, all succeed (a version
		// that lost its fraction would never match its own row).
		v := version(s.get(x.token, pp("")))
		for i := 0; i < 20; i++ {
			r := s.patch(x.token, pp(""), map[string]any{"description": fmt.Sprintf("urutan %d", i)}, quoted(v))
			if r.status != 200 {
				t.Fatalf("edit %d with the previous version: %s", i, r)
			}
			if nv := version(r); nv == v {
				t.Fatalf("edit %d left the version unchanged (%s)", i, nv)
			} else {
				v = nv
			}
		}
	})

	// ------------------------------------------------------------------
	t.Run("partial PATCH changes only what was sent (every family)", func(t *testing.T) {
		for _, fam := range families {
			before := fam.read(t, s, x.token)
			r := s.patch(x.token, fam.path, fam.patch, "")
			if r.status != 200 {
				t.Errorf("%s: %s", fam.name, r)
				continue
			}
			after := fam.read(t, s, x.token)
			if fmt.Sprint(after[fam.field]) != fmt.Sprint(fam.want) {
				t.Errorf("%s: %s = %v, want %v", fam.name, fam.field, after[fam.field], fam.want)
			}
			for _, k := range fam.untouched {
				if fmt.Sprint(before[k]) != fmt.Sprint(after[k]) {
					t.Errorf("%s: %s moved from %v to %v", fam.name, k, before[k], after[k])
				}
			}
			// The answer is the item itself, with the field changed.
			if got := r.dig("data", fam.field); fmt.Sprint(got) != fmt.Sprint(fam.want) {
				t.Errorf("%s: answer %s = %v", fam.name, fam.field, got)
			}
		}
	})

	t.Run("an empty body, unknown keys only and null on a required field are 422", func(t *testing.T) {
		all := append([]family{projectFam()}, families...)
		for _, fam := range all {
			r := s.patchRaw(x.token, fam.path, `{}`, "")
			wantStatus(t, fam.name+" {}", r, 422)
			if !hasKey(r, "body") {
				t.Errorf("%s {}: errors must be keyed `body`: %s", fam.name, r)
			}
			r = s.patchRaw(x.token, fam.path, `{"tidakDikenal": 1}`, "")
			wantStatus(t, fam.name+" unknown only", r, 422)
			r = s.patchRaw(x.token, fam.path, fmt.Sprintf(`{%q: null}`, fam.notNullable), "")
			wantStatus(t, fam.name+" null on "+fam.notNullable, r, 422)
			if !hasKey(r, fam.notNullable) {
				t.Errorf("%s: null must be keyed %s: %s", fam.name, fam.notNullable, r)
			}
			r = s.patch(x.token, fam.path, fam.badValue, "")
			wantStatus(t, fam.name+" bad value", r, 422)
			if !hasKey(r, fam.badKey) {
				t.Errorf("%s: bad value must be keyed %s: %s", fam.name, fam.badKey, r)
			}
			r = s.patchRaw(x.token, fam.path, `[1,2]`, "")
			wantStatus(t, fam.name+" not an object", r, 400)
			r = s.patchRaw(x.token, fam.path, `{"a":`, "")
			wantStatus(t, fam.name+" broken json", r, 400)
		}
	})

	t.Run("a rule that looks at several fields is judged on the merged row", func(t *testing.T) {
		for _, fam := range families {
			if fam.mergedBad == nil {
				continue
			}
			before := fam.read(t, s, x.token)
			r := s.patch(x.token, fam.path, fam.mergedBad, "")
			wantStatus(t, fam.name+" merged violation", r, 422)
			if !hasKey(r, fam.mergedBadKey) {
				t.Errorf("%s: want key %s: %s", fam.name, fam.mergedBadKey, r)
			}
			if after := fam.read(t, s, x.token); fmt.Sprint(before) != fmt.Sprint(after) {
				t.Errorf("%s: a refused patch changed the row: %v -> %v", fam.name, before, after)
			}
		}
		// A note-only patch on a payment is not blocked by the overpayment rule, and
		// the kendala's vendor and milestone can move together.
		r := s.patch(x.token, pp(fmt.Sprintf("/issues/%d", f.issue)), map[string]any{"projectVendorId": nil, "vendorMilestoneId": nil}, "")
		wantStatus(t, "kendala to general", r, 200)
		if r.dig("data", "projectVendorId") != nil || r.dig("data", "vendorMilestoneId") != nil {
			t.Errorf("both must be cleared: %s", r)
		}
		r = s.patch(x.token, pp(fmt.Sprintf("/issues/%d", f.issue)), map[string]any{"projectVendorId": f.engagement2}, "")
		wantStatus(t, "kendala to a vendor", r, 200)
		// Resolving stamps the date once.
		r = s.patch(x.token, pp(fmt.Sprintf("/issues/%d", f.issue)), map[string]any{"status": "Resolved"}, "")
		wantStatus(t, "resolve", r, 200)
		stamp := str(r, "data", "resolvedDate")
		if stamp == "" {
			t.Errorf("resolving must stamp resolvedDate: %s", r)
		}
		r = s.patch(x.token, pp(fmt.Sprintf("/issues/%d", f.issue)), map[string]any{"status": "Closed"}, "")
		if str(r, "data", "resolvedDate") != stamp {
			t.Errorf("resolvedDate must not be re-stamped: %s", r)
		}
		// completedDate can be cleared with null.
		pm := pp(fmt.Sprintf("/milestones/%d", f.pmile))
		s.mustOK("complete", s.patch(x.token, pm, map[string]any{"status": "Completed", "completedDate": day(0)}, ""))
		r = s.patch(x.token, pm, map[string]any{"completedDate": nil}, "")
		wantStatus(t, "clear completedDate", r, 200)
		if r.dig("data", "completedDate") != nil {
			t.Errorf("completedDate must be null: %s", r)
		}
	})

	t.Run("project: dates, venue snapshot, PIC and the money fields", func(t *testing.T) {
		before := data(s.get(x.token, pp("")))
		r := s.patch(x.token, pp(""), map[string]any{"name": "Nama Baru T8", "eventDate": day(150)}, "")
		wantStatus(t, "name + eventDate", r, 200)
		after := data(s.get(x.token, pp("")))
		for _, k := range []string{"brideName", "groomName", "venue", "prepStartDate", "packageName", "contractValue", "picStaffId", "description", "venueId", "venueRentalPrice", "venueCharge", "status", "isArchived"} {
			if fmt.Sprint(before[k]) != fmt.Sprint(after[k]) {
				t.Errorf("%s moved from %v to %v", k, before[k], after[k])
			}
		}
		if after["name"] != "Nama Baru T8" || after["eventDate"] != day(150) {
			t.Errorf("sent fields: %v %v", after["name"], after["eventDate"])
		}
		// status cannot be patched: it is not a field here.
		wantStatus(t, "status only", s.patch(x.token, pp(""), map[string]any{"status": "Completed"}, ""), 422)
		if data(s.get(x.token, pp("")))["status"] != before["status"] {
			t.Error("status must stay what it was")
		}
		// The venue was attached in the fixtures: its snapshot came from the venue.
		if after["venueRentalPrice"] != float64(10_000_000) || after["venueCharge"] != float64(500_000) {
			t.Errorf("venue snapshot: %v %v", after["venueRentalPrice"], after["venueCharge"])
		}
		r = s.patch(x.token, pp(""), map[string]any{"venueCharge": 600_000}, "")
		wantStatus(t, "snapshot charge", r, 200)
		if r.dig("data", "venueCharge") != float64(600_000) || r.dig("data", "venueRentalPrice") != float64(10_000_000) {
			t.Errorf("only the charge may change: %s", r)
		}
		// null detaches the venue and clears the snapshot with it.
		r = s.patch(x.token, pp(""), map[string]any{"venueId": nil}, "")
		wantStatus(t, "detach", r, 200)
		if r.dig("data", "venueId") != nil || r.dig("data", "venueRentalPrice") != nil || r.dig("data", "venueCharge") != nil {
			t.Errorf("detach must clear the snapshot: %s", r)
		}
		// A price with no venue is wrong against the merged row.
		r = s.patch(x.token, pp(""), map[string]any{"venueRentalPrice": 1}, "")
		wantStatus(t, "price without a venue", r, 422)
		if !hasKey(r, "venueId") {
			t.Errorf("want key venueId: %s", r)
		}
		// A venue of another workspace is refused; one's own re-attaches with its prices.
		yVenue := s.mustOK("Y venue", s.do(y.token, "POST", "/api/v1/venues", venueBody(y, "Venue Y"))).id("data", "id")
		r = s.patch(x.token, pp(""), map[string]any{"venueId": yVenue}, "")
		wantStatus(t, "foreign venue", r, 422)
		r = s.patch(x.token, pp(""), map[string]any{"venueId": f.venue}, "")
		wantStatus(t, "reattach", r, 200)
		if r.dig("data", "venueRentalPrice") != float64(10_000_000) || r.dig("data", "venue") != "Venue T8 "+x.marker {
			t.Errorf("reattach: %s", r)
		}
		// Money fields.
		wantStatus(t, "negative contract", s.patch(x.token, pp(""), map[string]any{"contractValue": -1}, ""), 422)
		// null clears the optional ones.
		r = s.patch(x.token, pp(""), map[string]any{"description": nil}, "")
		wantStatus(t, "description null", r, 200)
		if r.dig("data", "description") != "" {
			t.Errorf("description must be empty: %s", r)
		}
		// A PIC of another workspace is refused.
		r = s.patch(x.token, pp(""), map[string]any{"picStaffId": y.staffID}, "")
		wantStatus(t, "foreign PIC", r, 422)
		if !hasKey(r, "picStaffId") {
			t.Errorf("want key picStaffId: %s", r)
		}
	})

	t.Run("a patch that changes nothing writes nothing and logs nothing", func(t *testing.T) {
		cur := data(s.get(x.token, pp("")))
		w0 := s.writes("projects", f.project)
		a0, _ := s.activity(f.project)
		v0 := cur["updatedAt"].(string)
		r := s.patch(x.token, pp(""), map[string]any{"name": cur["name"], "contractValue": cur["contractValue"], "eventDate": cur["eventDate"]}, "")
		wantStatus(t, "no-op", r, 200)
		if version(r) != v0 || s.writes("projects", f.project) != w0 {
			t.Errorf("no-op wrote: version %s -> %s, writes %d -> %d", v0, version(r), w0, s.writes("projects", f.project))
		}
		if a1, _ := s.activity(f.project); a1 != a0 {
			t.Errorf("no-op logged %d entries", a1-a0)
		}
		// Even with a stale If-Match: nothing is written, so nothing can be lost.
		r = s.patch(x.token, pp(""), map[string]any{"name": cur["name"]}, quoted("2020-01-01T00:00:00.000000Z"))
		wantStatus(t, "no-op with a stale version", r, 200)
		// The same for a payment and a kendala.
		pay := pp(fmt.Sprintf("/client-payments/%d", f.clientPay))
		cp := rows(s.get(x.token, pp("/client-payments")), "data")[0]
		r = s.patch(x.token, pay, map[string]any{"amount": cp["amount"], "notes": cp["notes"]}, "")
		wantStatus(t, "payment no-op", r, 200)
		if version(r) != cp["updatedAt"] {
			t.Errorf("payment no-op moved the version: %s -> %v", version(r), cp["updatedAt"])
		}
	})

	t.Run("one activity entry names only the fields that changed", func(t *testing.T) {
		a0, _ := s.activity(f.project)
		wantStatus(t, "patch", s.patch(x.token, pp(""), map[string]any{"name": "Nama Log T8", "packageName": "Paket Log"}, ""), 200)
		a1, desc := s.activity(f.project)
		if a1 != a0+1 {
			t.Fatalf("a PATCH logs exactly one entry, got %d", a1-a0)
		}
		for _, want := range []string{"nama", "paket"} {
			if !strings.Contains(desc, want) {
				t.Errorf("%q must name %q", desc, want)
			}
		}
		for _, no := range []string{"deskripsi", "nilai kontrak", "tanggal acara", "PIC"} {
			if strings.Contains(desc, no) {
				t.Errorf("%q must not name %q (it did not change)", desc, no)
			}
		}
		// Sending the same values again is a no-op: no entry.
		wantStatus(t, "again", s.patch(x.token, pp(""), map[string]any{"name": "Nama Log T8", "packageName": "Paket Log"}, ""), 200)
		if a2, _ := s.activity(f.project); a2 != a1 {
			t.Errorf("a no-op PATCH logged %d entries", a2-a1)
		}
		// A payment edit: one entry, naming the field.
		a3, _ := s.activity(f.project)
		wantStatus(t, "payment", s.patch(x.token, pp(fmt.Sprintf("/payments/%d", f.vendorPay)), map[string]any{"notes": "dicek"}, ""), 200)
		a4, d4 := s.activity(f.project)
		if a4 != a3+1 || !strings.Contains(d4, "catatan") || strings.Contains(d4, "nominal") {
			t.Errorf("payment patch activity: %d new, %q", a4-a3, d4)
		}
	})

	// ------------------------------------------------------------------
	t.Run("If-Match: happy path, stale 409 with data.current, absent header, malformed, other workspace", func(t *testing.T) {
		n := 0
		for _, fam := range append([]family{projectFam()}, families...) {
			cur := fam.read(t, s, x.token)
			v1 := cur["updatedAt"].(string)

			// Happy path: the version the person saw.
			n++
			mine := fam.vary(n)
			a := s.patch(x.token, fam.path, mine, quoted(v1))
			wantStatus(t, fam.name+" with the current version", a, 200)
			v2 := version(a)
			if v2 == v1 {
				t.Errorf("%s: a write must change the version", fam.name)
			}

			// Stale: the same old version again, by someone else.
			n++
			b := s.patch(x.token, fam.path, fam.vary(n), quoted(v1))
			wantStatus(t, fam.name+" stale", b, 409)
			if b.header.Get("ETag") != "" {
				t.Errorf("%s: a 409 must not set an ETag", fam.name)
			}
			if got := b.dig("success"); got != false {
				t.Errorf("%s: success must be false: %s", fam.name, b)
			}
			if b.dig("message") != "Data ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru." {
				t.Errorf("%s: message %v", fam.name, b.dig("message"))
			}
			if c, _ := b.dig("errors", "_conflict").([]any); len(c) != 1 || c[0] != "stale" {
				t.Errorf("%s: errors._conflict = %v", fam.name, b.dig("errors"))
			}
			if got := version(b, "current"); got != v2 {
				t.Errorf("%s: data.current.updatedAt = %q, want the current %q", fam.name, got, v2)
			}
			if got := b.dig("data", "current", fam.varyField); fmt.Sprint(got) != fmt.Sprint(mine[fam.varyField]) {
				t.Errorf("%s: data.current must be the item as it is now (%s = %v): %s", fam.name, fam.varyField, got, b)
			}
			// Nothing of the refused write landed.
			after := fam.read(t, s, x.token)
			if after["updatedAt"] != v2 || fmt.Sprint(after[fam.varyField]) != fmt.Sprint(mine[fam.varyField]) {
				t.Errorf("%s: the stale write changed the row", fam.name)
			}

			// An unquoted value works too.
			n++
			wantStatus(t, fam.name+" unquoted version", s.patch(x.token, fam.path, fam.vary(n), v2), 200)
		}
	})

	t.Run("a stale version is answered before any rule is judged on data the caller never saw", func(t *testing.T) {
		for _, fam := range families {
			if fam.mergedBad == nil {
				continue
			}
			if fam.reset != nil {
				s.mustOK(fam.name+" reset", s.patch(x.token, fam.path, fam.reset, ""))
			}
			old := fam.read(t, s, x.token)["updatedAt"].(string)
			s.mustOK(fam.name+" moves on", s.patch(x.token, fam.path, fam.vary(9000), ""))
			// The same body is a 422 against the current row, but with the old version it is a 409.
			wantStatus(t, fam.name+" violation, current version", s.patch(x.token, fam.path, fam.mergedBad, ""), 422)
			wantStatus(t, fam.name+" violation, old version", s.patch(x.token, fam.path, fam.mergedBad, quoted(old)), 409)
		}
	})

	t.Run("If-Match absent: last write wins; malformed: 400 and nothing written", func(t *testing.T) {
		n := 5000
		for _, fam := range append([]family{projectFam()}, families...) {
			before := fam.read(t, s, x.token)
			n++
			body := fam.vary(n)
			// A stale write with no header still lands (the transitional rule of ADR-0036).
			wantStatus(t, fam.name+" without a header", s.patch(x.token, fam.path, body, ""), 200)
			if after := fam.read(t, s, x.token); fmt.Sprint(after[fam.varyField]) != fmt.Sprint(body[fam.varyField]) {
				t.Errorf("%s: last write must win: %v", fam.name, after[fam.varyField])
			}
			now := fam.read(t, s, x.token)
			for _, bad := range []string{"banana", `"2026-13-45T00:00:00Z"`, `W/"2026-11-10T08:15:30.123456Z"`, "*", `"a","b"`, `"unterminated`} {
				n++
				r := s.patch(x.token, fam.path, fam.vary(n), bad)
				wantStatus(t, fmt.Sprintf("%s If-Match %s", fam.name, bad), r, 400)
			}
			if after := fam.read(t, s, x.token); after["updatedAt"] != now["updatedAt"] {
				t.Errorf("%s: a malformed If-Match must write nothing", fam.name)
			}
			_ = before
		}
	})

	t.Run("another workspace gets 404, with or without a version", func(t *testing.T) {
		for _, fam := range append([]family{projectFam()}, families...) {
			wantStatus(t, fam.name, s.patch(y.token, fam.path, map[string]any{"name": "retas"}, ""), 404)
			wantStatus(t, fam.name+" with If-Match", s.patch(y.token, fam.path, map[string]any{"name": "retas"}, quoted("2026-01-01T00:00:00.000000Z")), 404)
		}
		// Nothing of X's changed.
		if data(s.get(x.token, pp("")))["name"] == "retas" {
			t.Error("Y changed X's project")
		}
	})

	t.Run("the 404 stays a 404 for a row that is gone, even with a stale version", func(t *testing.T) {
		wantStatus(t, "unknown milestone", s.patch(x.token, pp("/milestones/99999999"), map[string]any{"status": "Blocked"}, quoted("2026-01-01T00:00:00.000000Z")), 404)
		wantStatus(t, "unknown kendala", s.patch(x.token, pp("/issues/99999999"), map[string]any{"title": "x"}, ""), 404)
		wantStatus(t, "unknown payment", s.patch(x.token, pp("/payments/99999999"), map[string]any{"notes": "x"}, ""), 404)
		// A soft-deleted payment is gone for PATCH (restore is its own call).
		pay := s.mustOK("payment to delete", s.do(x.token, "POST", pp("/venue-payments"), map[string]any{
			"type": "Termin", "amount": 1000, "paymentDate": day(-1), "method": "Tunai", "referenceNumber": "DEL",
		})).id("data", "id")
		cur := rows(s.get(x.token, pp("/venue-payments")), "data")
		var v string
		for _, m := range cur {
			if int64(m["id"].(float64)) == pay {
				v, _ = m["updatedAt"].(string)
			}
		}
		s.mustOK("delete", s.do(x.token, "DELETE", pp(fmt.Sprintf("/venue-payments/%d", pay)), nil))
		wantStatus(t, "soft-deleted payment", s.patch(x.token, pp(fmt.Sprintf("/venue-payments/%d", pay)), map[string]any{"notes": "x"}, quoted(v)), 404)
	})

	// ------------------------------------------------------------------
	t.Run("concurrency: with the same version exactly one write wins and nothing is lost", func(t *testing.T) {
		type target struct {
			name   string
			path   string
			get    string
			list   bool
			fields func(round, i int) map[string]any
			check  string
		}
		targets := []target{
			{"project", pp(""), pp(""), false, func(round, i int) map[string]any {
				return map[string]any{"description": fmt.Sprintf("penulis-%d-%d", round, i)}
			}, "description"},
			{"kendala", pp(fmt.Sprintf("/issues/%d", f.issue)), pp("/issues"), true, func(round, i int) map[string]any {
				return map[string]any{"description": fmt.Sprintf("penulis-%d-%d", round, i)}
			}, "description"},
			{"vendor payment (project lock)", pp(fmt.Sprintf("/payments/%d", f.vendorPay)), pp("/payments"), true, func(round, i int) map[string]any {
				return map[string]any{"notes": fmt.Sprintf("penulis-%d-%d", round, i)}
			}, "notes"},
			{"client payment (project lock)", pp(fmt.Sprintf("/client-payments/%d", f.clientPay)), pp("/client-payments"), true, func(round, i int) map[string]any {
				return map[string]any{"notes": fmt.Sprintf("penulis-%d-%d", round, i)}
			}, "notes"},
		}
		for _, tg := range targets {
			fam := family{name: tg.name, path: tg.path, get: tg.get, list: tg.list}
			for round := 0; round < 6; round++ {
				const writers = 6
				v := fam.read(t, s, x.token)["updatedAt"].(string)
				statuses := make([]int, writers)
				replies := make([]reply, writers)
				start := make(chan struct{})
				var wg sync.WaitGroup
				for i := 0; i < writers; i++ {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						<-start
						replies[i] = s.patch(x.token, tg.path, tg.fields(round, i), quoted(v))
						statuses[i] = replies[i].status
					}(i)
				}
				close(start)
				wg.Wait()
				ok, stale, winner := 0, 0, -1
				for i, st := range statuses {
					switch st {
					case 200:
						ok++
						winner = i
					case 409:
						stale++
					default:
						t.Errorf("%s round %d writer %d: %s", tg.name, round, i, replies[i])
					}
				}
				if ok != 1 || stale != writers-1 {
					t.Fatalf("%s round %d: %d succeeded, %d got 409 (want exactly 1 and %d): %v", tg.name, round, ok, stale, writers-1, statuses)
				}
				// No lost update: the row holds the winner's value, and every loser was told
				// the current version.
				got := fam.read(t, s, x.token)
				if want := fmt.Sprintf("penulis-%d-%d", round, winner); got[tg.check] != want {
					t.Fatalf("%s round %d: the row holds %v, the winner wrote %s", tg.name, round, got[tg.check], want)
				}
				for i, rp := range replies {
					if statuses[i] == 409 && version(rp, "current") != got["updatedAt"] {
						t.Errorf("%s round %d: a loser's data.current is not the winner's version", tg.name, round)
					}
				}
			}
		}
	})

	t.Run("concurrency: two writers on different fields, the same version", func(t *testing.T) {
		v := data(s.get(x.token, pp("")))["updatedAt"].(string)
		a := s.patch(x.token, pp(""), map[string]any{"name": "Penulis A"}, quoted(v))
		b := s.patch(x.token, pp(""), map[string]any{"description": "Penulis B"}, quoted(v))
		wantStatus(t, "first writer", a, 200)
		wantStatus(t, "second writer", b, 409)
		cur := data(s.get(x.token, pp("")))
		if cur["name"] != "Penulis A" || cur["description"] == "Penulis B" {
			t.Errorf("the second writer must have been refused whole: %v / %v", cur["name"], cur["description"])
		}
		// Reading the 409's data.current and sending again with its version works.
		again := s.patch(x.token, pp(""), map[string]any{"description": "Penulis B"}, quoted(version(b, "current")))
		wantStatus(t, "retry on the current version", again, 200)
	})

	t.Run("without If-Match two writers on different fields both land (columns, not the whole row)", func(t *testing.T) {
		const rounds = 8
		for i := 0; i < rounds; i++ {
			name, desc := fmt.Sprintf("nama-%d", i), fmt.Sprintf("desk-%d", i)
			var wg sync.WaitGroup
			wg.Add(2)
			start := make(chan struct{})
			go func() { defer wg.Done(); <-start; s.patch(x.token, pp(""), map[string]any{"name": name}, "") }()
			go func() { defer wg.Done(); <-start; s.patch(x.token, pp(""), map[string]any{"description": desc}, "") }()
			close(start)
			wg.Wait()
			cur := data(s.get(x.token, pp("")))
			if cur["name"] != name || cur["description"] != desc {
				t.Fatalf("round %d: a field was overwritten with a stale value: name=%v description=%v", i, cur["name"], cur["description"])
			}
		}
	})

	// ------------------------------------------------------------------
	t.Run("project archive: explicit and idempotent, and the old toggle route is gone", func(t *testing.T) {
		P := s.createProject(x, "T8 Arsip "+x.marker, 1_000_000)
		path := s.projectsPath(P, "")
		w0 := s.writes("projects", P)
		r := s.patch(x.token, path, map[string]any{"isArchived": true}, "")
		wantStatus(t, "archive", r, 200)
		if r.dig("data", "isArchived") != true {
			t.Errorf("isArchived must be true: %s", r)
		}
		w1 := s.writes("projects", P)
		if w1 != w0+1 {
			t.Errorf("archiving is one write, got %d", w1-w0)
		}
		a1, _ := s.activity(P)
		v1 := version(r)
		// Again: 200, unchanged, nothing written or logged.
		r = s.patch(x.token, path, map[string]any{"isArchived": true}, "")
		wantStatus(t, "archive again", r, 200)
		if version(r) != v1 || s.writes("projects", P) != w1 {
			t.Errorf("archive again wrote: %s writes %d -> %d", version(r), w1, s.writes("projects", P))
		}
		if a2, _ := s.activity(P); a2 != a1 {
			t.Errorf("archive again logged %d entries", a2-a1)
		}
		// It leaves the default list and shows in the archived one.
		listed := func(q string) bool { // q is "" or "?archived=true"
			q = strings.TrimPrefix(q, "?")
			if q != "" {
				q = "?" + q
			}
			for _, m := range rows(s.get(x.token, withQuery("/api/v1/projects"+q, "limit", "100")), "data") {
				if int64(m["id"].(float64)) == P {
					return true
				}
			}
			return false
		}
		if listed("") || !listed("?archived=true") {
			t.Error("an archived project belongs to the archived view only")
		}
		// Unarchive with an explicit false, then archive again with an explicit true.
		r = s.patch(x.token, path, map[string]any{"isArchived": false}, "")
		wantStatus(t, "unarchive", r, 200)
		if r.dig("data", "isArchived") != false {
			t.Errorf("isArchived:false must have unarchived: %s", r)
		}
		if !listed("") || listed("?archived=true") {
			t.Error("an unarchived project is back in the default view only")
		}
		r = s.patch(x.token, path, map[string]any{"isArchived": true}, "")
		wantStatus(t, "archive again", r, 200)
		if r.dig("data", "isArchived") != true {
			t.Errorf("isArchived:true must have archived: %s", r)
		}
		// POST …/toggle-archive was removed (ADR-0036): the router's own 404, and it
		// does not flip the state on the way.
		removedRoute(t, "POST toggle-archive", s.do(x.token, "POST", path+"/toggle-archive", nil))
		if data(s.get(x.token, path))["isArchived"] != true {
			t.Error("the removed toggle-archive route must not change the archive state")
		}
		// The evidence toggle is removed whether or not the document exists (and
		// without object storage, which the evidence scenario below needs).
		removedRoute(t, "POST toggle-client-visible", s.do(x.token, "POST", path+"/evidence/999999/toggle-client-visible", nil))
		wantStatus(t, "unarchive by PATCH", s.patch(x.token, path, map[string]any{"isArchived": false}, ""), 200)
		wantStatus(t, "not a boolean", s.patchRaw(x.token, path, `{"isArchived": "ya"}`, ""), 422)
		wantStatus(t, "null", s.patchRaw(x.token, path, `{"isArchived": null}`, ""), 422)
		// Archive state and details together in one patch is allowed and is one write.
		w2 := s.writes("projects", P)
		r = s.patch(x.token, path, map[string]any{"isArchived": true, "description": "diarsipkan sambil edit"}, "")
		wantStatus(t, "archive + edit", r, 200)
		if s.writes("projects", P) != w2+1 {
			t.Error("one patch is one statement")
		}
	})

	t.Run("directory resources: isActive is explicit and idempotent, and the old toggle routes are gone", func(t *testing.T) {
		cat := s.mustOK("category", s.do(x.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": "Kategori T8 " + x.marker, "description": "x"})).id("data", "id")
		vendor := s.mustOK("vendor", s.do(x.token, "POST", "/api/v1/vendors", vendorBody(s, x, "Vendor Aktif T8 "+x.marker, cat))).id("data", "id")
		venue := s.mustOK("venue", s.do(x.token, "POST", "/api/v1/venues", venueBody(x, "Venue Aktif T8 "+x.marker))).id("data", "id")

		role := s.mustOK("role", s.do(x.token, "POST", "/api/v1/access/roles", map[string]any{
			"kind": "staff", "name": "Peran T8 " + x.marker, "description": "uji", "projectScope": "all", "permissions": []string{"project.edit"},
		})).id("data", "id")
		email := fmt.Sprintf("t8.staf.%d@example.com", time.Now().UnixNano())
		st := s.mustOK("staff", s.do(x.token, "POST", "/api/v1/staff", map[string]any{"name": "Staf T8", "title": "Uji", "roleId": role, "email": email, "phone": "081234567890"}))
		staff := st.id("data", "staff", "id")
		signIn := func() int {
			return s.do("", "POST", "/api/v1/auth/google", map[string]any{"credential": s.google.token(email, "Staf T8", "g-sub-t8-staf")}).status
		}
		s.mustOK("staff accepts", s.do("", "POST", "/api/v1/auth/links/"+str(st, "data", "invitation", "token")+"/google", map[string]any{
			"credential": s.google.token(email, "Staf T8", "g-sub-t8-staf"),
		}))
		if r := s.do("", "POST", "/api/v1/auth/google", map[string]any{"credential": s.google.token(email, "Staf T8", "g-sub-t8-staf")}); r.status != 200 {
			t.Fatalf("the invited staff must be able to sign in first: %s", r)
		}

		username := fmt.Sprintf("t8.klien.%d", time.Now().UnixNano()%1_000_000_000)
		cl := s.mustOK("client", s.do(x.token, "POST", "/api/v1/clients", map[string]any{
			"projectId": f.project, "role": "Bride", "name": "Klien T8", "phone": "081299999999", "username": username, "email": "",
		}))
		client := cl.id("data", "client", "id")
		s.mustOK("client password", s.do("", "POST", "/api/v1/auth/links/"+str(cl, "data", "accessLink", "token")+"/password", map[string]any{"password": "Kata-sandi-klien-123"}))
		clientLogin := func() int {
			return s.do("", "POST", "/api/v1/auth/login", map[string]any{"username": username, "password": "Kata-sandi-klien-123"}).status
		}
		if clientLogin() != 200 {
			t.Fatal("the client must be able to sign in first")
		}

		type res struct {
			name, path, table string
			id                int64
			extra             func(active bool) // observable side effect, checked after each change
		}
		list := []res{
			{"vendor", fmt.Sprintf("/api/v1/vendors/%d", vendor), "vendors", vendor, nil},
			{"venue", fmt.Sprintf("/api/v1/venues/%d", venue), "venues", venue, nil},
			{"vendor category", fmt.Sprintf("/api/v1/vendor-categories/%d", cat), "vendor_categories", cat, nil},
			{"client", fmt.Sprintf("/api/v1/clients/%d", client), "clients", client, func(active bool) {
				if got := clientLogin() == 200; got != active {
					t.Errorf("client: active=%v but sign-in works=%v", active, got)
				}
			}},
			{"staff", fmt.Sprintf("/api/v1/staff/%d", staff), "staff_members", staff, func(active bool) {
				if got := signIn() == 200; got != active {
					t.Errorf("staff: active=%v but sign-in works=%v", active, got)
				}
			}},
		}
		for _, rs := range list {
			// Deactivate: one write.
			w0 := s.writes(rs.table, rs.id)
			r := s.patch(x.token, rs.path, map[string]any{"isActive": false}, "")
			wantStatus(t, rs.name+" deactivate", r, 200)
			if r.dig("data", "isActive") != false || s.writes(rs.table, rs.id) != w0+1 {
				t.Errorf("%s: deactivate: %s (writes %d -> %d)", rs.name, r, w0, s.writes(rs.table, rs.id))
			}
			if rs.extra != nil {
				rs.extra(false)
			}
			// Again: 200, the same item, nothing written.
			w1 := s.writes(rs.table, rs.id)
			r2 := s.patch(x.token, rs.path, map[string]any{"isActive": false}, "")
			wantStatus(t, rs.name+" deactivate again", r2, 200)
			if r2.dig("data", "isActive") != false || s.writes(rs.table, rs.id) != w1 {
				t.Errorf("%s: a repeated state wrote: %s (writes %d -> %d)", rs.name, r2, w1, s.writes(rs.table, rs.id))
			}
			// Activate, twice.
			wantStatus(t, rs.name+" activate", s.patch(x.token, rs.path, map[string]any{"isActive": true}, ""), 200)
			if rs.extra != nil {
				rs.extra(true)
			}
			w2 := s.writes(rs.table, rs.id)
			r3 := s.patch(x.token, rs.path, map[string]any{"isActive": true}, "")
			wantStatus(t, rs.name+" activate again", r3, 200)
			if r3.dig("data", "isActive") != true || s.writes(rs.table, rs.id) != w2 {
				t.Errorf("%s: a repeated state wrote: %s", rs.name, r3)
			}
			// Bad bodies.
			wantStatus(t, rs.name+" isActive null", s.patchRaw(x.token, rs.path, `{"isActive": null}`, ""), 422)
			wantStatus(t, rs.name+" isActive string", s.patchRaw(x.token, rs.path, `{"isActive": "false"}`, ""), 422)
			wantStatus(t, rs.name+" isActive mixed with an edit", s.patchRaw(x.token, rs.path, `{"isActive": false, "name": "x"}`, ""), 422)
			wantStatus(t, rs.name+" not an object", s.patchRaw(x.token, rs.path, `[true]`, ""), 400)
			// POST …/toggle-active was removed (ADR-0036): the router's own 404, whichever
			// state the item is in, and it flips nothing on the way.
			for _, state := range []bool{true, false} {
				wantStatus(t, rs.name+" set state", s.patch(x.token, rs.path, map[string]any{"isActive": state}, ""), 200)
				removedRoute(t, rs.name+" POST toggle-active", s.do(x.token, "POST", rs.path+"/toggle-active", nil))
				if rs.extra != nil {
					rs.extra(state)
				}
				w3 := s.writes(rs.table, rs.id)
				if got := s.patch(x.token, rs.path, map[string]any{"isActive": state}, ""); got.dig("data", "isActive") != state || s.writes(rs.table, rs.id) != w3 {
					t.Errorf("%s: the removed toggle route changed the state (active=%v): %s", rs.name, state, got)
				}
			}
			wantStatus(t, rs.name+" back to active", s.patch(x.token, rs.path, map[string]any{"isActive": true}, ""), 200)
			if rs.extra != nil {
				rs.extra(true)
			}
			// Another workspace cannot reach it.
			wantStatus(t, rs.name+" from another workspace", s.patch(y.token, rs.path, map[string]any{"isActive": false}, ""), 404)
		}
		// The ordinary edit still works on the same route.
		r := s.patch(x.token, fmt.Sprintf("/api/v1/vendor-categories/%d", cat), map[string]any{"name": "Kategori T8 diganti " + x.marker, "description": "baru"}, "")
		wantStatus(t, "category edit", r, 200)
		if r.dig("data", "name") != "Kategori T8 diganti "+x.marker {
			t.Errorf("edit: %s", r)
		}
	})

	// ------------------------------------------------------------------
	t.Run("evidence: isClientVisible is explicit and idempotent, and the old toggle route is gone", func(t *testing.T) {
		if os.Getenv("TEST_S3_ENDPOINT") == "" {
			t.Skip("TEST_S3_ENDPOINT not set: evidence uploads need object storage")
		}
		doc := s.upload(x, f.project, "Dokumen T8", false)
		path := pp(fmt.Sprintf("/evidence/%d", doc))
		w0 := s.writes("evidence", doc)
		r := s.patch(x.token, path, map[string]any{"isClientVisible": true}, "")
		wantStatus(t, "show", r, 200)
		if r.dig("data", "isClientVisible") != true || !updatedAtRE.MatchString(version(r)) || r.header.Get("ETag") != quoted(version(r)) {
			t.Errorf("show: %s", r)
		}
		if s.writes("evidence", doc) != w0+1 {
			t.Error("showing is one write")
		}
		v1, w1 := version(r), s.writes("evidence", doc)
		r = s.patch(x.token, path, map[string]any{"isClientVisible": true}, "")
		wantStatus(t, "show again", r, 200)
		if version(r) != v1 || s.writes("evidence", doc) != w1 {
			t.Errorf("a repeated state wrote: %s", r)
		}
		// The list carries updatedAt too.
		var found bool
		for _, m := range rows(s.get(x.token, pp("/evidence")), "data") {
			if int64(m["id"].(float64)) == doc {
				found = true
				if u, _ := m["updatedAt"].(string); u != v1 {
					t.Errorf("evidence list updatedAt = %q, want %q", u, v1)
				}
			}
		}
		if !found {
			t.Error("the document is not in the list")
		}
		// If-Match: stale 409 with data.current, current works.
		stale := s.patch(x.token, path, map[string]any{"isClientVisible": false}, quoted("2020-01-01T00:00:00.000000Z"))
		wantStatus(t, "stale evidence", stale, 409)
		if version(stale, "current") != v1 || stale.dig("data", "current", "isClientVisible") != true {
			t.Errorf("data.current must be the document as it is: %s", stale)
		}
		wantStatus(t, "hide with the version", s.patch(x.token, path, map[string]any{"isClientVisible": false}, quoted(v1)), 200)
		wantStatus(t, "not a boolean", s.patchRaw(x.token, path, `{"isClientVisible": 1}`, ""), 422)
		wantStatus(t, "empty", s.patchRaw(x.token, path, `{}`, ""), 422)
		// The document is hidden now. POST …/toggle-client-visible was removed
		// (ADR-0036): the router's own 404, and it does not flip the flag.
		removedRoute(t, "POST toggle-client-visible", s.do(x.token, "POST", path+"/toggle-client-visible", nil))
		if got := s.patch(x.token, path, map[string]any{"isClientVisible": false}, ""); got.dig("data", "isClientVisible") != false {
			t.Errorf("the removed toggle-client-visible route must not change the flag: %s", got)
		}
		wantStatus(t, "show explicitly", s.patch(x.token, path, map[string]any{"isClientVisible": true}, ""), 200)
		// Another workspace: 404. A proof of a payment has no flag: 422.
		wantStatus(t, "other workspace", s.patch(y.token, path, map[string]any{"isClientVisible": true}, ""), 404)
		proof := s.attach(x, f.project, "clientPayment", f.clientPay, "Transfer Proof", "bukti T8", false)
		r = s.patch(x.token, pp(fmt.Sprintf("/evidence/%d", proof)), map[string]any{"isClientVisible": true}, "")
		wantStatus(t, "payment proof", r, 422)
		if !hasKey(r, "relatedKind") {
			t.Errorf("want key relatedKind: %s", r)
		}
	})

	// ------------------------------------------------------------------
	t.Run("permissions: archive, edit and evidence visibility are separate decisions", func(t *testing.T) {
		P := s.createProject(x, "T8 Izin "+x.marker, 1_000_000)
		path := s.projectsPath(P, "")
		archiver := s.staffWith(x, "Pengarsip", "project.archive")
		editor := s.staffWith(x, "Penyunting", "project.edit")
		wantStatus(t, "archiver archives", s.patch(archiver, path, map[string]any{"isArchived": true}, ""), 200)
		wantStatus(t, "archiver cannot edit details", s.patch(archiver, path, map[string]any{"name": "x"}, ""), 403)
		wantStatus(t, "archiver cannot mix", s.patch(archiver, path, map[string]any{"isArchived": false, "name": "x"}, ""), 403)
		wantStatus(t, "editor edits", s.patch(editor, path, map[string]any{"name": "Disunting"}, ""), 200)
		wantStatus(t, "editor cannot archive", s.patch(editor, path, map[string]any{"isArchived": false}, ""), 403)
		if data(s.get(x.token, path))["isArchived"] != true {
			t.Error("a refused patch must not change the archive state")
		}
		recorder := s.staffWith(x, "Pencatat", "finance.record")
		wantStatus(t, "finance.record cannot patch a project", s.patch(recorder, path, map[string]any{"name": "x"}, ""), 403)
		if os.Getenv("TEST_S3_ENDPOINT") != "" {
			doc := s.upload(x, P, "Dokumen izin", false)
			wantStatus(t, "finance.record cannot change a document's visibility", s.patch(recorder, s.projectsPath(P, fmt.Sprintf("/evidence/%d", doc)), map[string]any{"isClientVisible": true}, ""), 403)
			wantStatus(t, "editor can", s.patch(editor, s.projectsPath(P, fmt.Sprintf("/evidence/%d", doc)), map[string]any{"isClientVisible": true}, ""), 200)
		}
	})

	// ------------------------------------------------------------------
	t.Run("create inputs are unchanged: POST still wants the whole body", func(t *testing.T) {
		wantStatus(t, "milestone create without a date", s.do(x.token, "POST", pp("/milestones"), map[string]any{"name": "tanpa tanggal"}), 422)
		r := s.do(x.token, "POST", pp("/milestones"), map[string]any{"name": "Dengan tanggal", "targetDate": day(40)})
		wantStatus(t, "milestone create", r, 201)
		if !updatedAtRE.MatchString(version(r)) || r.header.Get("ETag") != quoted(version(r)) {
			t.Errorf("a created item carries its version too: %s (ETag %q)", r, r.header.Get("ETag"))
		}
	})
}

// mergedFor is a valid body for a stale write against fam (it never reaches the
// service on a stale version, so any valid one will do).
func mergedFor(fam family) map[string]any {
	if fam.patch != nil {
		return fam.patch
	}
	return map[string]any{"description": "basi"}
}

// ---- migration 000071 ----

func TestTahap8Migration(t *testing.T) {
	s := startStack(t)
	x := s.signUp("t8m")
	P := s.createProject(x, "Proyek migrasi "+x.marker, 1_000_000)
	tables := []string{"projects", "project_milestones", "project_vendors", "vendor_milestones", "vendor_issues", "client_payments", "vendor_payments", "venue_payments", "evidence"}

	precision := func(table string) (exists bool, p int) {
		var n *int
		err := s.db.QueryRow(`SELECT DATETIME_PRECISION FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = 'updated_at'`, table).Scan(&n)
		if err != nil {
			return false, 0
		}
		if n == nil {
			return true, 0
		}
		return true, *n
	}
	for _, tb := range tables {
		if ok, p := precision(tb); !ok || p != 6 {
			t.Errorf("after up: %s.updated_at exists=%v precision=%d, want TIMESTAMP(6)", tb, ok, p)
		}
	}
	var extra string
	if err := s.db.QueryRow(`SELECT EXTRA FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'projects' AND column_name = 'updated_at'`).Scan(&extra); err != nil || !strings.Contains(strings.ToLower(extra), "on update") {
		t.Errorf("projects.updated_at must update itself, EXTRA = %q (%v)", extra, err)
	}

	run := func(arg string) {
		cmd := exec.Command(s.bin, "migrate", arg)
		cmd.Dir, cmd.Env = s.dir, s.env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("migrate %s: %v\n%s", arg, err, out)
		}
	}
	// Down: the eight new columns go, projects keeps its column in whole seconds, data survives.
	run("down")
	for _, tb := range tables {
		ok, p := precision(tb)
		if tb == "projects" {
			if !ok || p != 0 {
				t.Errorf("after down: projects.updated_at exists=%v precision=%d, want TIMESTAMP", ok, p)
			}
			continue
		}
		if ok {
			t.Errorf("after down: %s still has updated_at", tb)
		}
	}
	if n := s.count(`SELECT COUNT(*) FROM projects WHERE id = ?`, P); n != 1 {
		t.Errorf("the project row must survive a down migration, found %d", n)
	}
	// Up again: everything is back and the server's PATCH works on the migrated data.
	run("up")
	for _, tb := range tables {
		if ok, p := precision(tb); !ok || p != 6 {
			t.Errorf("after up again: %s.updated_at exists=%v precision=%d", tb, ok, p)
		}
	}
	if n := s.count(`SELECT COUNT(*) FROM projects WHERE id = ? AND updated_at IS NOT NULL`, P); n != 1 {
		t.Errorf("the migrated project has no version")
	}
	// A down then up leaves the data readable and editable through the API.
	r := s.get(x.token, s.projectsPath(P, ""))
	if !updatedAtRE.MatchString(version(r)) {
		t.Errorf("migrated project updatedAt = %q", version(r))
	}
	wantStatus(t, "edit after the migration", s.patch(x.token, s.projectsPath(P, ""), map[string]any{"name": "Setelah migrasi"}, quoted(version(r))), http.StatusOK)
}
