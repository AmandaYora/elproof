//go:build integration

package isolation

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Tahap 6 (E11, E13 bulk, E14): server-side sorting, filters and derived columns
// on every list, the platform lists (transactions, tenants, audit log) and the
// bulk activate/deactivate of vendors and venues. One scenario signs up its own
// workspaces (X, with data, and Y, whose rows must never show up in X's lists)
// and a Platform Console Super Admin, so none of it depends on the attack matrix.
//
// Raised trial caps (staff, projects, clients) are set directly in the throwaway
// database: the scenario needs more rows than a trial workspace may hold.

// ---- helpers ----

// totp is the RFC 6238 code (SHA-1, 30 s, 6 digits) of a base32 secret.
func totp(secret string, at time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		panic(err)
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(at.Unix()/30))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	code := (uint32(sum[off]&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])) % 1_000_000
	return fmt.Sprintf("%06d", code)
}

// platformAdmin resets the database to the seed (one Super Admin, one plan),
// signs the admin in and enrols two-factor, which every Console route needs.
// It must run before any workspace is created: the seed truncates every table.
func (s *stack) platformAdmin() string {
	s.t.Helper()
	cmd := exec.Command(s.bin, "seed")
	cmd.Dir, cmd.Env = s.dir, s.env
	if out, err := cmd.CombinedOutput(); err != nil {
		s.t.Fatalf("api seed: %v\n%s", err, out)
	}
	login := s.mustOK("admin login", s.do("", "POST", "/api/v1/auth/login", map[string]any{"username": "superadmin", "password": "superadmin"}))
	tok := str(login, "data", "accessToken")
	setup := s.mustOK("mfa setup", s.do(tok, "POST", "/api/v1/auth/mfa/setup", nil))
	en := s.mustOK("mfa enable", s.do(tok, "POST", "/api/v1/auth/mfa/enable", map[string]any{"code": totp(str(setup, "data", "secret"), time.Now())}))
	tok = str(en, "data", "accessToken")
	if tok == "" {
		s.t.Fatalf("no two-factor admin token: %s", en)
	}
	return tok
}

// withQuery appends parameters to a path that may already carry some.
func withQuery(path string, kv ...string) string {
	q := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		q.Set(kv[i], kv[i+1])
	}
	if strings.Contains(path, "?") {
		return path + "&" + q.Encode()
	}
	return path + "?" + q.Encode()
}

// allPages walks every page of a list (limit 3, so there are several) and returns
// the rows in the order served, the total and the sort echoed in meta.
func (s *stack) allPages(token, path string) (items []map[string]any, total int64, sortEcho string) {
	s.t.Helper()
	for page := 1; page < 60; page++ {
		r := s.mustOK("page "+path, s.do(token, "GET", withQuery(path, "page", strconv.Itoa(page), "limit", "3"), nil))
		got := rows(r, "data")
		items = append(items, got...)
		total = num(r, "meta", "total")
		sortEcho = str(r, "meta", "sort")
		if page >= int(num(r, "meta", "total_pages")) {
			break
		}
	}
	return items, total, sortEcho
}

// sortKey is one comparable cell of a list (string, number, or absent).
type sortKey struct {
	s    string
	n    float64
	null bool
}

func strKey(v any) sortKey {
	s, ok := v.(string)
	if !ok || v == nil {
		return sortKey{null: true}
	}
	return sortKey{s: strings.ToLower(s)}
}

func numKey(v any) sortKey {
	f, ok := v.(float64)
	if !ok {
		return sortKey{null: true}
	}
	return sortKey{n: f}
}

func boolKey(v any) sortKey {
	if b, _ := v.(bool); b {
		return sortKey{n: 1}
	}
	return sortKey{n: 0}
}

func rankKey(order ...string) func(any) sortKey {
	return func(v any) sortKey {
		s, _ := v.(string)
		for i, o := range order {
			if o == s {
				return sortKey{n: float64(i)}
			}
		}
		return sortKey{n: -1}
	}
}

func cmpKey(a, b sortKey) int {
	switch {
	case a.null && b.null:
		return 0
	case a.null:
		return 1 // absent values are always last
	case b.null:
		return -1
	}
	switch {
	case a.s < b.s:
		return -1
	case a.s > b.s:
		return 1
	case a.n < b.n:
		return -1
	case a.n > b.n:
		return 1
	}
	return 0
}

// listSpec describes one paginated list for the generic checks.
type listSpec struct {
	name  string
	token string
	path  string
	// fields: sort name -> how a row's cell for that field is read.
	fields map[string]func(map[string]any) sortKey
	// def is the default sort ("field:dir") and legacy the order of ids it must
	// keep when no sort is sent (nil: only the echo is checked).
	def string
	// foreign are ids/markers of another workspace that must never show up.
	foreignIDs []int64
	foreignTxt string
	// idKey is the row's id field.
	idKey string
}

func (l listSpec) id(m map[string]any) int64 {
	k := l.idKey
	if k == "" {
		k = "id"
	}
	f, _ := m[k].(float64)
	return int64(f)
}

// checkList walks the list under every whitelisted sort in both directions.
func (s *stack) checkList(t *testing.T, l listSpec) {
	t.Helper()
	base, baseTotal, echo := s.allPages(l.token, l.path)
	if echo != l.def {
		t.Errorf("%s: default sort echoed as %q, want %q", l.name, echo, l.def)
	}
	if int64(len(base)) != baseTotal || baseTotal == 0 {
		t.Fatalf("%s: %d rows served, meta.total %d (an empty list proves nothing)", l.name, len(base), baseTotal)
	}
	want := map[int64]bool{}
	for _, m := range base {
		want[l.id(m)] = true
	}
	for _, fid := range l.foreignIDs {
		if want[fid] {
			t.Errorf("%s: lists the foreign row %d", l.name, fid)
		}
	}

	fields := make([]string, 0, len(l.fields))
	for f := range l.fields {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, field := range fields {
		for _, dir := range []string{"asc", "desc"} {
			label := fmt.Sprintf("%s sort=%s:%s", l.name, field, dir)
			routesCovered++
			items, total, echo := s.allPages(l.token, withQuery(l.path, "sort", field+":"+dir))
			if echo != field+":"+dir {
				t.Errorf("%s: meta.sort %q", label, echo)
			}
			if total != baseTotal || int64(len(items)) != baseTotal {
				t.Errorf("%s: %d rows over %d (meta.total %d), want %d", label, len(items), len(items), total, baseTotal)
			}
			seen := map[int64]bool{}
			for _, m := range items {
				id := l.id(m)
				if seen[id] {
					t.Errorf("%s: row %d appears on two pages (unstable paging)", label, id)
				}
				seen[id] = true
				if !want[id] {
					t.Errorf("%s: row %d is not in the unsorted list (leak or loss)", label, id)
				}
			}
			// The order really follows the column, ties by id in the same direction.
			read := l.fields[field]
			for i := 1; i < len(items); i++ {
				a, b := read(items[i-1]), read(items[i])
				c := cmpKey(a, b)
				if dir == "desc" && !(a.null || b.null) {
					c = -c
				} else if dir == "desc" && (a.null != b.null) {
					// nulls stay last in both directions: already encoded in cmpKey
				}
				if c > 0 {
					t.Errorf("%s: row %d (%+v) before row %d (%+v)", label, l.id(items[i-1]), a, l.id(items[i]), b)
					break
				}
				if c == 0 {
					p, q := l.id(items[i-1]), l.id(items[i])
					if (dir == "asc" && p > q) || (dir == "desc" && p < q) {
						t.Errorf("%s: tie between %d and %d not broken by id %s", label, p, q, dir)
						break
					}
				}
			}
			if l.foreignTxt != "" && strings.Contains(fmt.Sprint(items), l.foreignTxt) {
				t.Errorf("%s: carries the foreign marker %q", label, l.foreignTxt)
			}
		}
	}

	// No parameter, one direction word, and a sort only in the other case.
	r := s.mustOK(l.name+" asc default", s.do(l.token, "GET", withQuery(l.path, "sort", fields[0]), nil))
	if got := str(r, "meta", "sort"); got != fields[0]+":asc" {
		t.Errorf("%s: a bare field sorts ascending, meta says %q", l.name, got)
	}

	// Unknown fields and injection attempts: 422 on "sort", nothing executed.
	attempts := []string{
		"nope:asc", "password", fields[0] + ":sideways", fields[0] + ";DROP TABLE vendors", fields[0] + ":asc,id:desc",
		"(SELECT SLEEP(5))", fields[0] + ":asc;--", "1", "id", "`" + fields[0] + "`", fields[0] + "' OR '1'='1",
	}
	for _, a := range attempts {
		routesCovered++
		r := s.do(l.token, "GET", withQuery(l.path, "sort", a), nil)
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "sort") {
			t.Errorf("%s: sort=%q must be 422 with errors.sort, got %s", l.name, a, r)
			continue
		}
		if msgs, _ := r.dig("errors", "sort").([]any); len(msgs) != 1 {
			t.Errorf("%s: sort=%q: want one message, got %s", l.name, a, r)
		}
	}
	if r := s.do(l.token, "GET", withQuery(withQuery(l.path, "sort", fields[0]), "sort", fields[0]), nil); r.status != http.StatusUnprocessableEntity {
		t.Errorf("%s: two sort parameters must be refused, got %s", l.name, r)
	}
	// The injection attempts changed nothing: the list is whole and unchanged.
	again, total, _ := s.allPages(l.token, l.path)
	if total != baseTotal || len(again) != len(base) {
		t.Errorf("%s: the list changed after the attempts: %d/%d rows, want %d", l.name, len(again), total, baseTotal)
	}
	if r := s.do(l.token, "GET", withQuery(l.path, "sort", "nope"), nil); !strings.Contains(string(r.body), "Urutkan menurut salah satu dari: ") {
		t.Errorf("%s: the 422 should list the allowed fields: %s", l.name, r)
	} else {
		for _, f := range fields {
			if !strings.Contains(string(r.body), f) {
				t.Errorf("%s: the 422 should name %q: %s", l.name, f, r)
			}
		}
	}
}

// staffAs signs in a staff member holding one of the workspace's built-in roles
// ("Wedding Planner", "Sales", "Admin") and returns the token and the staff id.
func (s *stack) staffAs(t tenant, roleName string) (string, int64) {
	s.t.Helper()
	var roleID int64
	roles := s.mustOK("roles", s.do(t.token, "GET", "/api/v1/access/roles?kind=staff", nil))
	for _, m := range rows(roles, "data") {
		if m["name"] == roleName {
			roleID = int64(m["id"].(float64))
		}
	}
	if roleID == 0 {
		s.t.Fatalf("no staff role %q: %s", roleName, roles)
	}
	email := fmt.Sprintf("%s.%d@example.com", strings.ReplaceAll(strings.ToLower(roleName), " ", "."), time.Now().UnixNano())
	st := s.mustOK("staff "+roleName, s.do(t.token, "POST", "/api/v1/staff", map[string]any{
		"name": roleName + " " + t.marker, "title": "Uji " + roleName, "roleId": roleID, "email": email, "phone": "081234567890",
	}))
	link := str(st, "data", "invitation", "token")
	sess := s.mustOK("staff accepts", s.do("", "POST", "/api/v1/auth/links/"+link+"/google", map[string]any{
		"credential": s.google.token(email, roleName, fmt.Sprintf("g-sub-t6-%d", time.Now().UnixNano())),
	}))
	tok := str(sess, "data", "accessToken")
	id, _ := strconv.ParseInt(fmt.Sprint(sess.dig("data", "principalId")), 10, 64)
	if tok == "" || id == 0 {
		s.t.Fatalf("no session for %s: %s", roleName, sess)
	}
	return tok, id
}

func (s *stack) projectFor(t tenant, name string, contract int64, eventIn int, picStaffID int64, status string) int64 {
	s.t.Helper()
	return s.mustOK("project "+name, s.do(t.token, "POST", "/api/v1/projects", map[string]any{
		"name": name, "brideName": "Bride " + name, "groomName": "Groom " + name, "eventDate": day(eventIn), "venue": "Gedung " + name,
		"prepStartDate": day(-30), "packageName": "Paket", "contractValue": contract, "status": status, "picStaffId": picStaffID,
	})).id("data", "id")
}

func (s *stack) vendorWith(t tenant, name string, category int64, city string, price *int64) int64 {
	s.t.Helper()
	body := vendorBody(s, t, name, category)
	body["city"] = city
	if price == nil {
		delete(body, "priceAkad")
	} else {
		body["priceAkad"] = *price
	}
	return s.mustOK("vendor "+name, s.do(t.token, "POST", "/api/v1/vendors", body)).id("data", "id")
}

func (s *stack) venueWith(t tenant, name, city string, capacity int, rental *int64) int64 {
	s.t.Helper()
	body := venueBody(t, name)
	body["city"], body["capacity"] = city, capacity
	if rental == nil {
		delete(body, "rentalPrice")
	} else {
		body["rentalPrice"] = *rental
	}
	return s.mustOK("venue "+name, s.do(t.token, "POST", "/api/v1/venues", body)).id("data", "id")
}

func i64(v int64) *int64 { return &v }

// ---- the scenario ----

func TestTahap6(t *testing.T) {
	s := startStack(t)
	admin := s.platformAdmin()
	if _, err := s.db.Exec(`UPDATE trial_limits SET max_staff = 60, max_active_projects = 200, max_clients = 400 WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	x, y := s.signUp("t6x"), s.signUp("t6y")

	// The four built-in staff roles of X and one person for each scope.
	wpTok, wpID := s.staffAs(x, "Wedding Planner")
	salesTok, salesID := s.staffAs(x, "Sales")
	adminTok, _ := s.staffAs(x, "Admin")
	_ = salesID

	// ---------------- workspace Y: rows that sort first AND last, so any leak is visible ----------------
	yCat := s.mustOK("Y category", s.do(y.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": "AAA Kategori " + y.marker, "description": "x"})).id("data", "id")
	yVendors := []int64{
		s.vendorWith(y, "AAA Vendor "+y.marker, yCat, "Kota Bandung", i64(1)),
		s.vendorWith(y, "ZZZ Vendor "+y.marker, yCat, "Kota Bandung", i64(999999999)),
	}
	yVenue := s.venueWith(y, "AAA Venue "+y.marker, "Kota Bandung", 1, i64(1))
	yProject := s.projectFor(y, "AAA Proyek "+y.marker, 1, 5, y.staffID, "Preparation")
	yClient := s.mustOK("Y client", s.do(y.token, "POST", "/api/v1/clients", map[string]any{
		"projectId": yProject, "role": "Bride", "name": "AAA Klien " + y.marker, "phone": "081200000001", "username": fmt.Sprintf("y.klien.%d", time.Now().UnixNano()%1_000_000_000), "email": "",
	})).id("data", "client", "id")

	// ---------------- workspace X ----------------
	xCat := map[string]int64{}
	for _, name := range []string{"Dekorasi t6", "catering t6", "Foto t6"} {
		xCat[name] = s.mustOK("X category", s.do(x.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": name + " " + x.marker, "description": "x"})).id("data", "id")
	}
	catName := map[int64]string{}
	for name, id := range xCat {
		catName[id] = name + " " + x.marker
	}

	vendorSpecs := []struct {
		name   string
		cat    string
		city   string
		price  *int64
		active bool
	}{
		{"delta", "Dekorasi t6", "Kota Bandung", i64(3_000_000), true},
		{"Alpha", "catering t6", "Jakarta Selatan", i64(1_000_000), true},
		{"charlie", "Foto t6", "Kota Bandung", nil, false},
		{"Bravo", "Dekorasi t6", "Kota Bekasi", i64(2_000_000), true},
		{"echo", "catering t6", "", i64(2_000_000), false},
		{"Foxtrot", "Foto t6", "Jakarta Selatan", i64(500_000), true},
		{"golf", "Dekorasi t6", "Kota Bandung", i64(2_000_000), true},
	}
	var xVendors []int64
	vendorCat := map[int64]string{}
	for _, v := range vendorSpecs {
		id := s.vendorWith(x, v.name+" "+x.marker, xCat[v.cat], v.city, v.price)
		xVendors = append(xVendors, id)
		vendorCat[id] = catName[xCat[v.cat]]
		if !v.active {
			s.mustOK("deactivate vendor", s.do(x.token, "PATCH", fmt.Sprintf("/api/v1/vendors/%d", id), map[string]any{"isActive": false}))
		}
	}

	venueSpecs := []struct {
		name   string
		city   string
		cap    int
		rental *int64
		active bool
	}{
		{"Gedung Mawar", "Kota Bandung", 300, i64(10_000_000), true},
		{"aula Melati", "Jakarta Selatan", 150, i64(5_000_000), false},
		{"Ballroom Dahlia", "Kota Bekasi", 800, nil, true},
		{"Pendopo Anggrek", "Kota Bandung", 200, i64(5_000_000), true},
		{"Taman Kenanga", "Kota Bandung", 200, i64(20_000_000), false},
	}
	var xVenues []int64
	for _, v := range venueSpecs {
		id := s.venueWith(x, v.name+" "+x.marker, v.city, v.cap, v.rental)
		xVenues = append(xVenues, id)
		if !v.active {
			s.mustOK("deactivate venue", s.do(x.token, "PATCH", fmt.Sprintf("/api/v1/venues/%d", id), map[string]any{"isActive": false}))
		}
	}

	// Projects: derived columns, filters, statuses, two PICs.
	pDays := map[string]int{"P-juli": 30, "P-agustus": 60, "P-sept": 90, "P-okt": 120, "P-nov": 150, "P-des": 180, "P-satu": 45}
	pID := map[string]int64{}
	pName := map[int64]string{}
	pEvent := map[int64]string{}
	for _, name := range []string{"P-juli", "P-agustus", "P-sept", "P-okt", "P-nov", "P-des", "P-satu"} {
		pic := x.staffID
		if name == "P-okt" || name == "P-nov" {
			pic = wpID
		}
		id := s.projectFor(x, name+" "+x.marker, int64(100_000_000+len(name)*1_000_000), pDays[name], pic, "Draft")
		pID[name], pName[id], pEvent[id] = id, name+" "+x.marker, day(pDays[name])
		// Every project starts as Draft: P-sept stays there, P-agustus goes on to Ready, the others to Preparation.
		if name != "P-sept" {
			s.mustOK("start preparation", s.do(x.token, "POST", s.projectsPath(id, "/transition"), map[string]any{"action": "start_preparation"}))
		}
		if name == "P-agustus" {
			s.mustOK("mark ready", s.do(x.token, "POST", s.projectsPath(id, "/transition"), map[string]any{"action": "mark_ready"}))
		}
	}
	// P-juli: contract 100M+6M; DP 30M paid, Termin 10M paid, a Tambahan, and a deleted Termin.
	pj := pID["P-juli"]
	s.setTerms(x, pj, term("DP", 30_000_000, 10), term("Termin 1", 30_000_000, 40), term("Pelunasan", 46_000_000, 70))
	s.clientPay(x, pj, "DP", 30_000_000, "T6-DP")
	s.clientPay(x, pj, "Termin", 10_000_000, "T6-T1")
	s.clientPay(x, pj, "Tambahan", 5_000_000, "T6-TAMBAHAN")
	hidden := s.clientPay(x, pj, "Termin", 20_000_000, "T6-HIDDEN")
	s.mustOK("soft delete payment", s.do(x.token, "DELETE", s.projectsPath(pj, fmt.Sprintf("/client-payments/%d", hidden)), nil))
	// P-agustus: schedule but nothing paid. P-sept: paid in full, schedule too. P-okt: no schedule, part paid. P-nov: nothing.
	pa := pID["P-agustus"]
	s.setTerms(x, pa, term("DP", 50_000_000, 5), term("Pelunasan", 57_000_000, 25))
	ps := pID["P-sept"]
	s.setTerms(x, ps, term("Lunas", 106_000_000+0, 3))
	s.clientPay(x, ps, "Pelunasan", 106_000_000, "T6-FULL")
	// P-okt: no schedule, 41M paid. Chosen so that the order by remaining differs from what a
	// wrong rule would give: 64M left here against 66M on P-juli (a deleted payment or a Tambahan
	// counted by mistake would put P-juli ahead).
	po := pID["P-okt"]
	s.clientPay(x, po, "DP", 41_000_000, "T6-OKT")
	// P-satu: a single installment due before P-juli's next one, nothing paid. Its next due date is
	// earlier than P-juli's (30 < 40 days), so an "unpaid" rule that skips a first installment would
	// reverse the two.
	s.setTerms(x, pID["P-satu"], term("Satu", 106_000_000, 30))

	// Clients: invited, active, inactive and "none" (no login at all), two roles, three projects.
	clientIDs := map[string]int64{}
	mkClient := func(label, role string, project int64, accept bool) int64 {
		name := label + " " + x.marker
		username := fmt.Sprintf("t6.%s.%d", strings.ToLower(label), time.Now().UnixNano()%1_000_000_000)
		cl := s.mustOK("client "+label, s.do(x.token, "POST", "/api/v1/clients", map[string]any{
			"projectId": project, "role": role, "name": name, "phone": "0812" + fmt.Sprint(time.Now().UnixNano()%100000000),
			"username": username, "email": "",
		}))
		id := cl.id("data", "client", "id")
		if accept {
			link := str(cl, "data", "accessLink", "token")
			s.mustOK("client password", s.do("", "POST", "/api/v1/auth/links/"+link+"/password", map[string]any{"password": "Kata-sandi-klien-123"}))
		}
		clientIDs[label] = id
		return id
	}
	mkClient("Rina", "Bride", pj, true)
	mkClient("Dewa", "Groom", pj, true)
	mkClient("Citra", "Family Representative", pa, false)
	mkClient("Banu", "Bride", pa, true)
	mkClient("Eko", "Groom", ps, false)
	mkClient("Fajar", "Bride", po, true)
	s.mustOK("deactivate Banu", s.do(x.token, "PATCH", fmt.Sprintf("/api/v1/clients/%d", clientIDs["Banu"]), map[string]any{"isActive": false}))
	// "none": the person has no login at all (links and sessions go first, they point at the login).
	for _, q := range []string{
		`DELETE FROM auth_tokens WHERE credential_id IN (SELECT id FROM credentials WHERE principal_type = 'client' AND principal_id = ?)`,
		`DELETE FROM refresh_tokens WHERE credential_id IN (SELECT id FROM credentials WHERE principal_type = 'client' AND principal_id = ?)`,
		`DELETE FROM credentials WHERE principal_type = 'client' AND principal_id = ?`,
	} {
		if _, err := s.db.Exec(q, strconv.FormatInt(clientIDs["Fajar"], 10)); err != nil {
			t.Fatal(err)
		}
	}
	wantLogin := map[string]string{"Rina": "active", "Dewa": "active", "Citra": "invited", "Banu": "inactive", "Eko": "invited", "Fajar": "none"}

	// Platform data for the console lists.
	s.mustOK("plan beta", s.do(admin, "POST", "/api/v1/plans", map[string]any{"appId": "platform-billing", "name": "beta plan", "durationMonths": 1, "price": 100000, "features": []string{"x"}}))
	s.mustOK("plan gamma", s.do(admin, "POST", "/api/v1/plans", map[string]any{"appId": "platform-billing", "name": "Gamma Plan", "durationMonths": 6, "price": 500000, "features": []string{"x"}}))
	plans := map[int64]string{}
	for _, m := range rows(s.mustOK("plans", s.do(admin, "GET", "/api/v1/plans?all=true", nil)), "data") {
		plans[int64(m["id"].(float64))] = fmt.Sprint(m["name"])
	}
	var extraTenants []int64
	for i, name := range []string{"Zaza WO", "bima WO", "Citra WO", "dodo WO"} {
		r := s.mustOK("register tenant", s.do(admin, "POST", "/api/v1/tenants", map[string]any{
			"businessName": name + " t6", "ownerName": "Owner " + name, "email": fmt.Sprintf("owner.t6.%d.%d@example.com", i, time.Now().UnixNano()), "phone": "0812", "city": "Bandung",
		}))
		extraTenants = append(extraTenants, r.id("data", "tenant", "id"))
	}
	planIDs := make([]int64, 0, len(plans))
	for id := range plans {
		planIDs = append(planIDs, id)
	}
	sort.Slice(planIDs, func(i, j int) bool { return planIDs[i] < planIDs[j] })
	allTenants := append([]int64{x.tenant, y.tenant}, extraTenants...)
	for i, id := range allTenants {
		if i == 3 {
			continue // one tenant without a plan: it must sort last
		}
		if _, err := s.db.Exec(`UPDATE tenants SET plan_id = ? WHERE id = ?`, planIDs[i%len(planIDs)], id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(`UPDATE tenants SET trial_ends_at = NULL WHERE id = ?`, extraTenants[0]); err != nil {
		t.Fatal(err)
	}

	// Transactions on fixed days (Indonesian time), several tenants, statuses and references.
	type txRow struct {
		tenant    int64
		amount    int64
		status    string
		ref       string
		createdAt string // WIB
	}
	txRows := []txRow{
		{x.tenant, 2_000_000, "paid", "ELP-X-001", "2026-02-10 23:59:00"},
		{x.tenant, 2_500_000, "unpaid", "ELP-X-002", "2026-02-11 00:01:00"},
		{x.tenant, 3_000_000, "expired", "ELP-X-003", "2026-03-01 09:00:00"},
		{y.tenant, 1_000_000, "paid", "ELP-Y-001", "2026-02-10 12:00:00"},
		{y.tenant, 9_000_000, "pending", "ELP-Y-002", "2026-04-01 12:00:00"},
		{extraTenants[0], 2_000_000, "paid", "ELP-Z-001", "2026-02-12 12:00:00"},
		{extraTenants[1], 4_000_000, "granted", "ELP-B-001", "2026-02-13 12:00:00"},
		{extraTenants[2], 4_000_000, "paid", "ELP-C-001", "2026-02-14 12:00:00"},
		{extraTenants[3], 500_000, "cancelled", "ELP-D-001", "2026-02-15 12:00:00"},
	}
	var txIDs []int64
	for _, r := range txRows {
		res, err := s.db.Exec(`INSERT INTO subscription_transactions (tenant_id, type, amount, payment_method, payment_reference, status, created_at, paid_at)
			VALUES (?, 'renewal', ?, 'QRIS', ?, ?, CONVERT_TZ(?, '+07:00', 'SYSTEM'), NULL)`, r.tenant, r.amount, r.ref, r.status, r.createdAt)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		txIDs = append(txIDs, id)
	}
	tenantNameOf := map[int64]string{}
	for _, m := range rows(s.mustOK("tenants", s.do(admin, "GET", "/api/v1/tenants?limit=100", nil)), "data") {
		tenantNameOf[int64(m["id"].(float64))] = fmt.Sprint(m["businessName"])
	}

	// Audit rows with fixed actions, actors and days.
	type auditRow struct {
		actor, role, action, at string
		id                      string
	}
	auditRows := []auditRow{
		{"7", "Super Admin", "POST /t6/suspend", "2026-02-10 10:00:00", "7"},
		{"9", "Billing", "POST /t6/plans", "2026-02-11 10:00:00", "9"},
		{"10", "Billing", "POST /t6/plans", "2026-02-12 10:00:00", "10"},
		{"7", "Super Admin", "PUT /t6/policy-x", "2026-02-13 10:00:00", "7"},
		{"11", "Support", "POST /t6/notes", "2026-02-14 10:00:00", "11"},
	}
	for _, r := range auditRows {
		if _, err := s.db.Exec(`INSERT INTO audit_log (occurred_at, actor_type, actor_id, actor_role, action, method, path, status_code, ip, user_agent)
			VALUES (CONVERT_TZ(?, '+07:00', 'SYSTEM'), 'platform_admin', ?, ?, ?, 'POST', '/api/v1/t6/audit', 200, '10.0.0.1', 'test')`, r.at, r.actor, r.role, r.action); err != nil {
			t.Fatal(err)
		}
	}

	// ======================= (a) every list: stable paging, tenant scope, whitelist =======================
	name := func(k string) func(map[string]any) sortKey {
		return func(m map[string]any) sortKey { return strKey(m[k]) }
	}
	number := func(k string) func(map[string]any) sortKey {
		return func(m map[string]any) sortKey { return numKey(m[k]) }
	}
	flag := func(k string) func(map[string]any) sortKey {
		return func(m map[string]any) sortKey { return boolKey(m[k]) }
	}
	rank := func(k string, order ...string) func(map[string]any) sortKey {
		f := rankKey(order...)
		return func(m map[string]any) sortKey { return f(m[k]) }
	}

	t.Run("a: sort and paging are stable and tenant-scoped on every list", func(t *testing.T) {
		specs := []listSpec{
			{name: "projects", token: x.token, path: "/api/v1/projects", def: "eventDate:desc", foreignIDs: []int64{yProject}, foreignTxt: y.marker, fields: map[string]func(map[string]any) sortKey{
				"name": name("name"), "eventDate": name("eventDate"), "contractValue": number("contractValue"), "createdAt": func(m map[string]any) sortKey { return sortKey{} },
				"status":    rank("status", "Draft", "Preparation", "Ready", "Completed", "Cancelled"),
				"remaining": number("remaining"), "nextDueDate": name("nextDueDate"),
			}},
			{name: "vendors", token: x.token, path: "/api/v1/vendors", def: "createdAt:asc", foreignIDs: yVendors, foreignTxt: y.marker, fields: map[string]func(map[string]any) sortKey{
				"name": name("name"), "categoryName": name("categoryName"), "city": name("city"), "priceAkad": number("priceAkad"),
				"isActive": flag("isActive"), "createdAt": name("createdAt"),
			}},
			{name: "venues", token: x.token, path: "/api/v1/venues", def: "createdAt:asc", foreignIDs: []int64{yVenue}, foreignTxt: y.marker, fields: map[string]func(map[string]any) sortKey{
				"name": name("name"), "city": name("city"), "capacity": number("capacity"), "rentalPrice": number("rentalPrice"),
				"isActive": flag("isActive"), "createdAt": name("createdAt"),
			}},
			{name: "vendor-categories", token: x.token, path: "/api/v1/vendor-categories", def: "createdAt:asc", foreignIDs: []int64{yCat}, foreignTxt: y.marker, fields: map[string]func(map[string]any) sortKey{
				"name": name("name"), "createdAt": func(m map[string]any) sortKey { return sortKey{} },
			}},
			{name: "clients", token: x.token, path: "/api/v1/clients", def: "createdAt:asc", foreignIDs: []int64{yClient}, foreignTxt: y.marker, fields: map[string]func(map[string]any) sortKey{
				"name": name("name"), "role": rank("role", "Bride", "Groom", "Family Representative"), "projectName": name("projectName"),
				"projectEventDate": name("projectEventDate"), "loginStatus": name("loginStatus"), "createdAt": func(m map[string]any) sortKey { return sortKey{} },
			}},
			{name: "staff", token: x.token, path: "/api/v1/staff", def: "createdAt:asc", foreignTxt: y.marker, fields: map[string]func(map[string]any) sortKey{
				"name": name("name"), "title": name("title"), "roleName": name("roleName"), "isActive": flag("isActive"),
				"createdAt": func(m map[string]any) sortKey { return sortKey{} },
			}},
			// The WO's own transaction history, through the same endpoint.
			{name: "subscription-transactions (owner)", token: x.token, path: "/api/v1/subscription-transactions", def: "createdAt:desc", foreignTxt: "ELP-Y-", fields: map[string]func(map[string]any) sortKey{
				"createdAt": name("createdAt"), "amount": number("amount"), "tenantName": name("tenantName"),
				"status": rank("status", "unpaid", "pending", "paid", "expired", "granted", "cancelled"),
			}},
			{name: "subscription-transactions (console)", token: admin, path: "/api/v1/subscription-transactions", def: "createdAt:desc", fields: map[string]func(map[string]any) sortKey{
				"createdAt": name("createdAt"), "amount": number("amount"), "tenantName": name("tenantName"),
				"status": rank("status", "unpaid", "pending", "paid", "expired", "granted", "cancelled"),
			}},
			{name: "tenants", token: admin, path: "/api/v1/tenants", def: "createdAt:desc", fields: map[string]func(map[string]any) sortKey{
				"name": name("businessName"),
				// The list orders by the stored (cached) status, which the sweeper refreshes; read it from the table.
				"status": func(m map[string]any) sortKey {
					var st string
					if err := s.db.QueryRow(`SELECT subscription_status FROM tenants WHERE id = ?`, int64(m["id"].(float64))).Scan(&st); err != nil {
						t.Fatal(err)
					}
					return rankKey("trialing", "active", "expiring_soon", "past_due", "read_only", "suspended", "pending_payment", "expired")(st)
				},
				"createdAt": name("joinedAt"), "trialEndsAt": name("trialEndsAt"),
				"planName": func(m map[string]any) sortKey {
					if f, ok := m["planId"].(float64); ok {
						return strKey(plans[int64(f)])
					}
					return sortKey{null: true}
				},
			}},
			{name: "audit-log", token: admin, path: "/api/v1/audit-log", def: "createdAt:desc", fields: map[string]func(map[string]any) sortKey{
				"createdAt": name("occurredAt"), "action": name("action"),
				"actor": func(m map[string]any) sortKey {
					id, _ := strconv.Atoi(fmt.Sprint(m["actorId"]))
					return sortKey{s: fmt.Sprintf("%s/%s/%020d", m["actorType"], strings.ToLower(fmt.Sprint(m["actorRole"])), id)}
				},
			}},
		}
		for _, l := range specs {
			l := l
			t.Run(l.name, func(t *testing.T) { s.checkList(t, l) })
		}
	})

	t.Run("a: the order really is the whitelisted one, in default and chosen sorts", func(t *testing.T) {
		// Default projects: latest event first, as before.
		r := s.mustOK("projects default", s.do(x.token, "GET", "/api/v1/projects?limit=100", nil))
		got := ids(rows(r, "data"))
		if len(got) != 7 || got[0] != pID["P-des"] || got[4] != pID["P-agustus"] || got[5] != pID["P-satu"] || got[6] != pID["P-juli"] {
			t.Errorf("default projects order (event date, newest first): %v", got)
		}
		// Default vendors: creation order, as before.
		r = s.mustOK("vendors default", s.do(x.token, "GET", "/api/v1/vendors?limit=100", nil))
		if got := ids(rows(r, "data")); !reflect.DeepEqual(got, xVendors) {
			t.Errorf("default vendors order: %v want %v", got, xVendors)
		}
		// Nulls last in both directions (a vendor without a price).
		for _, dir := range []string{"asc", "desc"} {
			r = s.mustOK("vendors by price", s.do(x.token, "GET", "/api/v1/vendors?limit=100&sort=priceAkad:"+dir, nil))
			list := rows(r, "data")
			if last := list[len(list)-1]; last["priceAkad"] != nil {
				t.Errorf("priceAkad:%s: the vendor without a price must come last, got %v", dir, last["priceAkad"])
			}
		}
		// categoryName is the category's name, from the vendors module's own table.
		for _, m := range rows(s.mustOK("vendors", s.do(x.token, "GET", "/api/v1/vendors?limit=100", nil)), "data") {
			if want := vendorCat[int64(m["id"].(float64))]; m["categoryName"] != want {
				t.Errorf("vendor %v: categoryName %v, want %q", m["id"], m["categoryName"], want)
			}
		}
	})

	// ======================= (b) projects: derived columns and filters =======================
	t.Run("b: remaining and nextDueDate", func(t *testing.T) {
		list := rows(s.mustOK("projects", s.do(x.token, "GET", "/api/v1/projects?limit=100", nil)), "data")
		check := func(label string, id int64, remaining int64, next any) {
			routesCovered++
			m := rowByID(list, id)
			if m == nil {
				t.Fatalf("%s: project %d not listed", label, id)
			}
			if int64(m["remaining"].(float64)) != remaining {
				t.Errorf("%s: remaining %v, want %d", label, m["remaining"], remaining)
			}
			if m["nextDueDate"] != next {
				t.Errorf("%s: nextDueDate %v, want %v", label, m["nextDueDate"], next)
			}
		}
		contract := func(name string) int64 { return int64(100_000_000 + len(name)*1_000_000) }
		// 106M contract: DP 30M + Termin 10M applied (Tambahan out, the deleted Termin out) -> 66M left;
		// 40M covers the DP term and 10M of "Termin 1": that one is the first not fully paid.
		check("P-juli", pj, contract("P-juli")-40_000_000, day(40))
		check("P-agustus (nothing paid)", pa, contract("P-agustus"), day(5))
		check("P-sept (paid in full)", ps, 0, nil)
		check("P-okt (no schedule)", po, contract("P-okt")-41_000_000, nil)
		check("P-satu (one unpaid installment)", pID["P-satu"], contract("P-satu"), day(30))
		check("P-nov (nothing at all)", pID["P-nov"], contract("P-nov"), nil)

		// Restoring the deleted payment changes the number; it was excluded for being deleted.
		s.mustOK("restore", s.do(x.token, "POST", s.projectsPath(pj, fmt.Sprintf("/client-payments/%d/restore", hidden)), nil))
		list = rows(s.mustOK("projects", s.do(x.token, "GET", "/api/v1/projects?limit=100", nil)), "data")
		check("P-juli after restore", pj, contract("P-juli")-60_000_000, day(70))
		s.mustOK("delete again", s.do(x.token, "DELETE", s.projectsPath(pj, fmt.Sprintf("/client-payments/%d", hidden)), nil))

		// The list agrees with what the overview of each project says is left.
		for name, id := range pID {
			ov := s.mustOK("overview", s.do(x.token, "GET", s.projectsPath(id, "/overview"), nil))
			m := rowByID(rows(s.mustOK("projects", s.do(x.token, "GET", "/api/v1/projects?limit=100", nil)), "data"), id)
			if num(ov, "data", "finance", "remaining") != int64(m["remaining"].(float64)) {
				t.Errorf("%s: list remaining %v vs overview %d", name, m["remaining"], num(ov, "data", "finance", "remaining"))
			}
			nt, _ := ov.dig("data", "finance", "nextTerm", "dueDate").(string)
			if nd, _ := m["nextDueDate"].(string); nd != nt {
				t.Errorf("%s: list nextDueDate %q vs overview next term %q", name, nd, nt)
			}
		}
	})

	t.Run("b: sorting by the derived columns", func(t *testing.T) {
		order := func(sortParam string) []string {
			routesCovered++
			var got []string
			for _, m := range rows(s.mustOK("projects", s.do(x.token, "GET", "/api/v1/projects?limit=100&sort="+sortParam, nil)), "data") {
				got = append(got, strings.TrimSuffix(fmt.Sprint(m["name"]), " "+x.marker))
			}
			return got
		}
		// remaining: 0, 64M, 66M, 105M (two, by id), 106M, 109M.
		if got, want := order("remaining:asc"), []string{"P-sept", "P-okt", "P-juli", "P-nov", "P-des", "P-satu", "P-agustus"}; !reflect.DeepEqual(got, want) {
			t.Errorf("remaining:asc: %v, want %v", got, want)
		}
		if got, want := order("remaining:desc"), []string{"P-agustus", "P-satu", "P-des", "P-nov", "P-juli", "P-okt", "P-sept"}; !reflect.DeepEqual(got, want) {
			t.Errorf("remaining:desc (ties by id, descending too): %v, want %v", got, want)
		}
		// nextDueDate: 5, 30, 40 days; the four projects without an unpaid installment come last in
		// both directions, by id.
		if got, want := order("nextDueDate:asc"), []string{"P-agustus", "P-satu", "P-juli", "P-sept", "P-okt", "P-nov", "P-des"}; !reflect.DeepEqual(got, want) {
			t.Errorf("nextDueDate:asc: %v, want %v", got, want)
		}
		if got, want := order("nextDueDate:desc"), []string{"P-juli", "P-satu", "P-agustus", "P-des", "P-nov", "P-okt", "P-sept"}; !reflect.DeepEqual(got, want) {
			t.Errorf("nextDueDate:desc: %v, want %v", got, want)
		}
	})

	t.Run("b: filters eventFrom, eventTo, picStaffId", func(t *testing.T) {
		count := func(path string) []int64 {
			routesCovered++
			return ids(rows(s.mustOK(path, s.do(x.token, "GET", path, nil)), "data"))
		}
		one := func(got []int64, want ...int64) bool {
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
			return reflect.DeepEqual(got, want)
		}
		if got := count(fmt.Sprintf("/api/v1/projects?eventFrom=%s&eventTo=%s", day(60), day(60))); !one(got, pID["P-agustus"]) {
			t.Errorf("a single inclusive day: %v", got)
		}
		if got := count("/api/v1/projects?eventFrom=" + day(120)); !one(got, pID["P-okt"], pID["P-nov"], pID["P-des"]) {
			t.Errorf("eventFrom is inclusive: %v", got)
		}
		if got := count("/api/v1/projects?eventTo=" + day(60)); !one(got, pID["P-juli"], pID["P-agustus"], pID["P-satu"]) {
			t.Errorf("eventTo is inclusive: %v", got)
		}
		if got := count(fmt.Sprintf("/api/v1/projects?picStaffId=%d", wpID)); !one(got, pID["P-okt"], pID["P-nov"]) {
			t.Errorf("picStaffId: %v", got)
		}
		if got := count(fmt.Sprintf("/api/v1/projects?picStaffId=%d&eventFrom=%s&q=nov&status=Preparation", wpID, day(100))); !one(got, pID["P-nov"]) {
			t.Errorf("filters combine with q and status: %v", got)
		}
		if got := count(fmt.Sprintf("/api/v1/projects?picStaffId=%d", y.staffID)); len(got) != 0 {
			t.Errorf("another workspace's staff id filters X's list to nothing, got %v", got)
		}
		// Reversed range and malformed values are 422 under their own keys.
		r := s.do(x.token, "GET", fmt.Sprintf("/api/v1/projects?eventFrom=%s&eventTo=%s", day(90), day(30)), nil)
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "eventTo") {
			t.Errorf("reversed range: %s", r)
		}
		r = s.do(x.token, "GET", "/api/v1/projects?eventFrom=besok&eventTo=2027-13-45&picStaffId=abc", nil)
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "eventFrom") || !hasKey(r, "eventTo") || !hasKey(r, "picStaffId") {
			t.Errorf("malformed values name every field: %s", r)
		}
		// q and search are the same search.
		if a, b := count("/api/v1/projects?q=agustus"), count("/api/v1/projects?search=agustus"); !one(a, pID["P-agustus"]) || !one(b, pID["P-agustus"]) {
			t.Errorf("q/search: %v %v", a, b)
		}
	})

	t.Run("b: a Wedding Planner never sees another planner's projects through any new filter", func(t *testing.T) {
		mine := []int64{pID["P-okt"], pID["P-nov"]}
		// The planner holds exactly P-okt and P-nov (PIC); every other project belongs to the Owner.
		list := func(path string) []int64 {
			routesCovered++
			return ids(rows(s.mustOK(path, s.do(wpTok, "GET", path, nil)), "data"))
		}
		same := func(got, want []int64) bool {
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
			return reflect.DeepEqual(got, want)
		}
		if got := list("/api/v1/projects?limit=100"); !same(got, append([]int64(nil), mine...)) {
			t.Fatalf("the planner's own list: %v", got)
		}
		for _, p := range []string{
			fmt.Sprintf("/api/v1/projects?picStaffId=%d", x.staffID),       // the Owner's projects
			"/api/v1/projects?eventFrom=" + day(1) + "&eventTo=" + day(60), // dates of the Owner's projects
			"/api/v1/projects?sort=remaining:desc&limit=100",               // derived columns
			"/api/v1/projects?sort=nextDueDate:asc&limit=100",
			"/api/v1/projects?q=juli",
			"/api/v1/projects?q=agustus&status=Preparation",
		} {
			for _, id := range list(p) {
				if id != mine[0] && id != mine[1] {
					t.Errorf("planner GET %s shows project %d, which is not theirs", p, id)
				}
			}
		}
		if got := list(fmt.Sprintf("/api/v1/projects?picStaffId=%d", x.staffID)); len(got) != 0 {
			t.Errorf("the Owner's PIC filter must give the planner nothing, got %v", got)
		}
		// Their own PIC filter and sorts still work, and carry the derived columns of their projects.
		got := rows(s.mustOK("planner sorted", s.do(wpTok, "GET", fmt.Sprintf("/api/v1/projects?picStaffId=%d&sort=remaining:asc", wpID), nil)), "data")
		if len(got) != 2 || int64(got[0]["remaining"].(float64)) > int64(got[1]["remaining"].(float64)) {
			t.Errorf("planner sorted by remaining: %v", got)
		}
		// Sales (PIC Sales of nothing, PIC of nothing) sees none of X's projects by any filter.
		if got := ids(rows(s.mustOK("sales", s.do(salesTok, "GET", "/api/v1/projects?sort=remaining:asc&limit=100", nil)), "data")); len(got) != 0 {
			t.Errorf("Sales holds no project: %v", got)
		}
	})

	// ======================= (c) clients =======================
	t.Run("c: clients list: filters, sort, project labels", func(t *testing.T) {
		all := rows(s.mustOK("clients", s.do(x.token, "GET", "/api/v1/clients?page=1&limit=100", nil)), "data")
		if len(all) != 6 {
			t.Fatalf("X has 6 clients, the list shows %d", len(all))
		}
		for _, m := range all {
			label := strings.TrimSuffix(fmt.Sprint(m["name"]), " "+x.marker)
			pid := int64(m["projectId"].(float64))
			if m["projectName"] != pName[pid] || m["projectEventDate"] != pEvent[pid] {
				t.Errorf("%s: project %q on %q, want %q on %q", label, m["projectName"], m["projectEventDate"], pName[pid], pEvent[pid])
			}
			if m["loginStatus"] != wantLogin[label] {
				t.Errorf("%s: loginStatus %v, want %s", label, m["loginStatus"], wantLogin[label])
			}
		}
		names := func(path string) []string {
			routesCovered++
			var out []string
			for _, m := range rows(s.mustOK(path, s.do(x.token, "GET", path, nil)), "data") {
				out = append(out, strings.TrimSuffix(fmt.Sprint(m["name"]), " "+x.marker))
			}
			sort.Strings(out)
			return out
		}
		eq := func(got []string, want ...string) bool { sort.Strings(want); return reflect.DeepEqual(got, want) }
		if got := names("/api/v1/clients?page=1&role=Bride"); !eq(got, "Rina", "Banu", "Fajar") {
			t.Errorf("role=Bride: %v", got)
		}
		if got := names("/api/v1/clients?page=1&role=Groom"); !eq(got, "Dewa", "Eko") {
			t.Errorf("role=Groom: %v", got)
		}
		for status, want := range map[string][]string{"active": {"Rina", "Dewa"}, "invited": {"Citra", "Eko"}, "inactive": {"Banu"}, "none": {"Fajar"}} {
			if got := names("/api/v1/clients?page=1&loginStatus=" + status); !eq(got, want...) {
				t.Errorf("loginStatus=%s: %v want %v", status, got, want)
			}
		}
		if got := names(fmt.Sprintf("/api/v1/clients?page=1&projectId=%d", pa)); !eq(got, "Citra", "Banu") {
			t.Errorf("projectId: %v", got)
		}
		if got := names(fmt.Sprintf("/api/v1/clients?page=1&projectId=%d&role=Bride&loginStatus=inactive", pa)); !eq(got, "Banu") {
			t.Errorf("filters combine: %v", got)
		}
		if got := names("/api/v1/clients?page=1&q=eko"); !eq(got, "Eko") {
			t.Errorf("q: %v", got)
		}
		if got := names("/api/v1/clients?page=1&search=eko"); !eq(got, "Eko") {
			t.Errorf("search (the older name of q): %v", got)
		}
		// The loginStatus filter pages correctly (limit 1, page 2) and keeps its total.
		r := s.mustOK("paged filter", s.do(x.token, "GET", "/api/v1/clients?loginStatus=active&limit=1&page=2&sort=name:desc", nil))
		if num(r, "meta", "total") != 2 || num(r, "meta", "total_pages") != 2 || len(rows(r, "data")) != 1 || str(r, "meta", "sort") != "name:desc" {
			t.Errorf("paging a derived-status filter: %s", r)
		}
		if got := rows(r, "data")[0]["name"]; got != "Dewa "+x.marker {
			t.Errorf("page 2 of name:desc among Rina and Dewa should be Dewa, got %v", got)
		}
		// Foreign project ids and unknown vocabulary.
		if got := names(fmt.Sprintf("/api/v1/clients?page=1&projectId=%d", yProject)); len(got) != 0 {
			t.Errorf("another workspace's project id lists %v", got)
		}
		for _, bad := range []string{"role=Cousin", "loginStatus=unknown", "projectId=abc", "projectId=0"} {
			routesCovered++
			r := s.do(x.token, "GET", "/api/v1/clients?page=1&"+bad, nil)
			if r.status != http.StatusUnprocessableEntity {
				t.Errorf("%s: want 422, got %s", bad, r)
			}
		}
		// Sorting by the project's name orders by the project, not by the client.
		list := rows(s.mustOK("by project", s.do(x.token, "GET", "/api/v1/clients?limit=100&sort=projectName:asc", nil)), "data")
		var projectNames []string
		for _, m := range list {
			projectNames = append(projectNames, strings.ToLower(fmt.Sprint(m["projectName"])))
		}
		if !sort.StringsAreSorted(projectNames) {
			t.Errorf("projectName:asc: %v", projectNames)
		}
	})

	t.Run("c: the clients rules for scoped roles and the older shapes are unchanged", func(t *testing.T) {
		// all=true and ?projectId= alone are still plain arrays (no meta), exactly as before.
		r := s.mustOK("all=true", s.do(x.token, "GET", "/api/v1/clients?all=true", nil))
		if _, ok := r.dig("data").([]any); !ok || r.dig("meta") != nil || len(rows(r, "data")) != 7 && len(rows(r, "data")) != 6 {
			t.Errorf("all=true keeps its shape: %s", r)
		}
		if got := len(rows(r, "data")); got != 6 {
			t.Errorf("all=true lists X's 6 clients, got %d", got)
		}
		r = s.mustOK("projectId", s.do(x.token, "GET", fmt.Sprintf("/api/v1/clients?projectId=%d", pa), nil))
		if r.dig("meta") != nil || len(rows(r, "data")) != 2 {
			t.Errorf("projectId alone keeps its shape (the project tab): %s", r)
		}
		if r.id("data") != 0 {
			t.Errorf("unexpected shape: %s", r)
		}
		// A Wedding Planner holds P-okt: its clients (paged or not), never the others'.
		planner := func(path string) reply { routesCovered++; return s.do(wpTok, "GET", path, nil) }
		if r := planner(fmt.Sprintf("/api/v1/clients?projectId=%d", po)); r.status != http.StatusOK || len(rows(r, "data")) != 1 {
			t.Errorf("planner, own project (array): %s", r)
		}
		if r := planner(fmt.Sprintf("/api/v1/clients?projectId=%d&page=1&sort=projectName", po)); r.status != http.StatusOK || len(rows(r, "data")) != 1 || str(r, "meta", "sort") != "projectName:asc" {
			t.Errorf("planner, own project (paged): %s", r)
		}
		for _, path := range []string{
			fmt.Sprintf("/api/v1/clients?projectId=%d", pa),
			fmt.Sprintf("/api/v1/clients?projectId=%d&page=1", pa),
			fmt.Sprintf("/api/v1/clients?projectId=%d&loginStatus=active&sort=name", pj),
			"/api/v1/clients?page=1", "/api/v1/clients?sort=name", "/api/v1/clients?role=Bride", "/api/v1/clients?loginStatus=active",
			"/api/v1/clients?q=Rina", "/api/v1/clients?all=true", "/api/v1/clients",
		} {
			if r := planner(path); r.status != http.StatusForbidden {
				t.Errorf("planner GET %s: want 403, got %s", path, r)
			}
		}
		// Sales holds no project: the same.
		for _, path := range []string{fmt.Sprintf("/api/v1/clients?projectId=%d&page=1", pa), "/api/v1/clients?page=1&sort=name"} {
			routesCovered++
			if r := s.do(salesTok, "GET", path, nil); r.status != http.StatusForbidden {
				t.Errorf("sales GET %s: want 403, got %s", path, r)
			}
		}
		// Y's Owner sees only Y's client, with Y's project label.
		r = s.mustOK("Y clients", s.do(y.token, "GET", "/api/v1/clients?page=1&sort=projectName", nil))
		if got := ids(rows(r, "data")); !reflect.DeepEqual(got, []int64{yClient}) {
			t.Errorf("Y's list: %v", got)
		}
		if strings.Contains(string(r.body), x.marker) {
			t.Errorf("Y's list carries X's data: %s", r)
		}
	})

	// ======================= (d) vendors, venues, staff =======================
	t.Run("d: isActive, categoryName, roleId", func(t *testing.T) {
		activeNames := func(token, path string) []string {
			routesCovered++
			var out []string
			for _, m := range rows(s.mustOK(path, s.do(token, "GET", path, nil)), "data") {
				out = append(out, fmt.Sprint(m["isActive"]))
			}
			return out
		}
		for _, want := range []struct {
			path string
			n    int
		}{
			{"/api/v1/vendors?isActive=true&limit=100", 5}, {"/api/v1/vendors?isActive=false&limit=100", 2},
			{"/api/v1/venues?isActive=true&limit=100", 3}, {"/api/v1/venues?isActive=false&limit=100", 2},
		} {
			got := activeNames(x.token, want.path)
			if len(got) != want.n {
				t.Errorf("%s: %d rows, want %d", want.path, len(got), want.n)
			}
			flag := strings.Contains(want.path, "isActive=true")
			for _, v := range got {
				if v != fmt.Sprint(flag) {
					t.Errorf("%s: a row with isActive %s", want.path, v)
				}
			}
		}
		// The filter composes with categoryId and search.
		r := s.mustOK("category+active", s.do(x.token, "GET", fmt.Sprintf("/api/v1/vendors?categoryId=%d&isActive=true&limit=100", xCat["Dekorasi t6"]), nil))
		if len(rows(r, "data")) != 3 {
			t.Errorf("Dekorasi active: %s", r)
		}
		for _, bad := range []string{"/api/v1/vendors?isActive=yes", "/api/v1/venues?isActive=1", "/api/v1/staff?isActive=maybe", "/api/v1/staff?roleId=abc", "/api/v1/staff?roleId=0"} {
			routesCovered++
			if r := s.do(x.token, "GET", bad, nil); r.status != http.StatusUnprocessableEntity {
				t.Errorf("%s: want 422, got %s", bad, r)
			}
		}
		// Venue and vendor lists never show Y's rows, whatever the filter.
		for _, path := range []string{"/api/v1/vendors?isActive=true&sort=name&limit=100", "/api/v1/venues?isActive=true&sort=capacity:asc&limit=100"} {
			if r := s.mustOK(path, s.do(x.token, "GET", path, nil)); strings.Contains(string(r.body), y.marker) {
				t.Errorf("%s leaks Y: %s", path, r)
			}
		}

		// Staff: filter by role and by activity, and the role name is the current one.
		staffList := rows(s.mustOK("staff", s.do(x.token, "GET", "/api/v1/staff?limit=100", nil)), "data")
		if len(staffList) != 4 { // Owner, Wedding Planner, Sales, Admin
			t.Fatalf("X's team: %d people", len(staffList))
		}
		var wpRole int64
		for _, m := range staffList {
			if m["id"].(float64) == float64(wpID) {
				wpRole = int64(m["roleId"].(float64))
				if m["roleName"] != "Wedding Planner" || m["role"] != "Wedding Planner" {
					t.Errorf("planner row: %v", m)
				}
			}
		}
		if got := ids(rows(s.mustOK("by role", s.do(x.token, "GET", fmt.Sprintf("/api/v1/staff?roleId=%d", wpRole), nil)), "data")); !reflect.DeepEqual(got, []int64{wpID}) {
			t.Errorf("roleId: %v", got)
		}
		if got := ids(rows(s.mustOK("old role param", s.do(x.token, "GET", fmt.Sprintf("/api/v1/staff?role=%d", wpRole), nil)), "data")); !reflect.DeepEqual(got, []int64{wpID}) {
			t.Errorf("role (the older name): %v", got)
		}
		s.mustOK("deactivate planner", s.do(x.token, "PATCH", fmt.Sprintf("/api/v1/staff/%d", wpID), map[string]any{"isActive": false}))
		if got := ids(rows(s.mustOK("inactive staff", s.do(x.token, "GET", "/api/v1/staff?isActive=false", nil)), "data")); !reflect.DeepEqual(got, []int64{wpID}) {
			t.Errorf("staff isActive=false: %v", got)
		}
		if got := rows(s.mustOK("active staff", s.do(x.token, "GET", "/api/v1/staff?isActive=true&limit=100", nil)), "data"); len(got) != 3 {
			t.Errorf("staff isActive=true: %d", len(got))
		}
		s.mustOK("reactivate planner", s.do(x.token, "PATCH", fmt.Sprintf("/api/v1/staff/%d", wpID), map[string]any{"isActive": true}))
		// Ordered by the role's name: Admin, Owner, Sales, Wedding Planner.
		var roleNames []string
		for _, m := range rows(s.mustOK("by role name", s.do(x.token, "GET", "/api/v1/staff?sort=roleName:asc&limit=100", nil)), "data") {
			roleNames = append(roleNames, fmt.Sprint(m["roleName"]))
		}
		if !reflect.DeepEqual(roleNames, []string{"Admin", "Owner", "Sales", "Wedding Planner"}) {
			t.Errorf("roleName:asc: %v", roleNames)
		}
		// A role without team.manage cannot list the team at all (unchanged gate).
		if r := s.do(adminTok, "GET", "/api/v1/staff?sort=name", nil); r.status != http.StatusForbidden {
			t.Errorf("Admin has no team.manage: want 403, got %s", r)
		}
		if r := s.do(y.token, "GET", "/api/v1/staff?sort=name&limit=100", nil); strings.Contains(string(r.body), x.marker) {
			t.Errorf("Y's team list shows X's people: %s", r)
		}
	})

	// ======================= (e) transactions =======================
	t.Run("e: subscription transactions: tenantName, search, dates, status, access", func(t *testing.T) {
		list := func(token, path string) []map[string]any {
			routesCovered++
			return rows(s.mustOK(path, s.do(token, "GET", path, nil)), "data")
		}
		refs := func(l []map[string]any) []string {
			var out []string
			for _, m := range l {
				out = append(out, fmt.Sprint(m["paymentReference"]))
			}
			sort.Strings(out)
			return out
		}
		eq := func(got []string, want ...string) bool { sort.Strings(want); return reflect.DeepEqual(got, want) }
		const base = "/api/v1/subscription-transactions?limit=100"

		all := list(admin, base)
		if len(all) != len(txRows) {
			t.Fatalf("the console sees %d transactions, want %d", len(all), len(txRows))
		}
		for _, m := range all {
			tenantID := int64(m["tenantId"].(float64))
			if m["tenantName"] != tenantNameOf[tenantID] || m["tenantName"] == "" {
				t.Errorf("transaction %v: tenantName %v, want %q", m["id"], m["tenantName"], tenantNameOf[tenantID])
			}
		}
		// Search: by tenant name (any case, a fragment) and by payment reference.
		if got := refs(list(admin, withQuery(base, "search", "ZAZA"))); !eq(got, "ELP-Z-001") {
			t.Errorf("search by tenant name: %v", got)
		}
		if got := refs(list(admin, withQuery(base, "q", "bima wo"))); !eq(got, "ELP-B-001") {
			t.Errorf("q by tenant name: %v", got)
		}
		if got := refs(list(admin, withQuery(base, "search", "elp-y-"))); !eq(got, "ELP-Y-001", "ELP-Y-002") {
			t.Errorf("search by reference: %v", got)
		}
		if got := refs(list(admin, withQuery(base, "search", "t6x"))); !eq(got, "ELP-X-001", "ELP-X-002", "ELP-X-003") {
			t.Errorf("search by the marker in X's tenant name: %v", got)
		}
		if got := list(admin, withQuery(base, "search", "no-such-thing")); len(got) != 0 {
			t.Errorf("no match: %v", got)
		}
		// A search with % or _ is a literal, not a wildcard.
		if got := list(admin, withQuery(base, "search", "%")); len(got) != 0 {
			t.Errorf("a lone %% must not match everything: %v", got)
		}
		// Dates are Indonesian days, both ends inclusive.
		if got := refs(list(admin, withQuery(base, "from", "2026-02-10", "to", "2026-02-10"))); !eq(got, "ELP-X-001", "ELP-Y-001") {
			t.Errorf("a single day (23:59 WIB still counts, 00:01 the next day does not): %v", got)
		}
		if got := refs(list(admin, withQuery(base, "from", "2026-02-11", "to", "2026-02-12"))); !eq(got, "ELP-X-002", "ELP-Z-001") {
			t.Errorf("a range: %v", got)
		}
		if got := refs(list(admin, withQuery(base, "from", "2026-04-01"))); !eq(got, "ELP-Y-002") {
			t.Errorf("from only: %v", got)
		}
		if got := refs(list(admin, withQuery(base, "to", "2026-02-10"))); !eq(got, "ELP-X-001", "ELP-Y-001") {
			t.Errorf("to only: %v", got)
		}
		if got := refs(list(admin, withQuery(base, "status", "paid", "from", "2026-02-10", "to", "2026-02-14"))); !eq(got, "ELP-X-001", "ELP-Y-001", "ELP-Z-001", "ELP-C-001") {
			t.Errorf("status with dates: %v", got)
		}
		if got := refs(list(admin, withQuery(base, "search", "ELP-", "status", "granted"))); !eq(got, "ELP-B-001") {
			t.Errorf("status with search: %v", got)
		}
		for _, bad := range [][]string{{"from", "kemarin"}, {"to", "2026-99-99"}, {"from", "2026-03-01", "to", "2026-02-01"}} {
			routesCovered++
			r := s.do(admin, "GET", withQuery(base, bad...), nil)
			if r.status != http.StatusUnprocessableEntity || (!hasKey(r, "from") && !hasKey(r, "to")) {
				t.Errorf("%v: want 422 on from/to, got %s", bad, r)
			}
		}
		// Sorted by tenant name in memory: ascending, ties by id, all rows once.
		byName := list(admin, withQuery(base, "sort", "tenantName:asc"))
		var names []string
		for _, m := range byName {
			names = append(names, strings.ToLower(fmt.Sprint(m["tenantName"])))
		}
		if !sort.StringsAreSorted(names) || len(byName) != len(txRows) {
			t.Errorf("tenantName:asc: %v", names)
		}
		// With filters and a page past the first.
		r := s.mustOK("tenantName page 2", s.do(admin, "GET", withQuery("/api/v1/subscription-transactions", "sort", "tenantName:desc", "search", "ELP-", "limit", "4", "page", "2"), nil))
		if len(rows(r, "data")) != 4 || num(r, "meta", "total") != int64(len(txRows)) || str(r, "meta", "sort") != "tenantName:desc" {
			t.Errorf("tenantName page 2: %s", r)
		}

		// Workspace Owner: only their own tenant, whatever they ask for.
		own := list(x.token, base)
		if len(own) != 3 {
			t.Errorf("X's history: %d rows", len(own))
		}
		for _, m := range own {
			if int64(m["tenantId"].(float64)) != x.tenant {
				t.Errorf("X sees tenant %v's transaction", m["tenantId"])
			}
		}
		for _, q := range [][]string{{"tenantId", strconv.FormatInt(y.tenant, 10)}, {"search", "ELP-Y-"}, {"search", "t6y"}, {"sort", "tenantName:desc"}} {
			got := list(x.token, withQuery(base, q...))
			for _, m := range got {
				if int64(m["tenantId"].(float64)) != x.tenant || strings.Contains(fmt.Sprint(m), "ELP-Y-") {
					t.Errorf("X asking %v sees %v", q, m)
				}
			}
		}
		// Roles that may not read it, and the receipt of another tenant.
		for name, tok := range map[string]string{"Wedding Planner": wpTok, "Sales": salesTok, "Admin": adminTok} {
			routesCovered++
			if r := s.do(tok, "GET", base, nil); r.status != http.StatusForbidden {
				t.Errorf("%s GET transactions: want 403, got %s", name, r)
			}
			if r := s.do(tok, "GET", withQuery(base, "search", "t6y", "sort", "tenantName"), nil); r.status != http.StatusForbidden {
				t.Errorf("%s GET transactions with the new parameters: want 403, got %s", name, r)
			}
		}
		clientTok := s.addClient(x, pj, "Bride", "Pengantin", fmt.Sprintf("t6.klien.%d", time.Now().UnixNano()%1_000_000_000))
		routesCovered++
		if r := s.do(clientTok, "GET", base, nil); r.status != http.StatusForbidden {
			t.Errorf("a client GET transactions: want 403, got %s", r)
		}
		if r := s.do("", "GET", base, nil); r.status != http.StatusUnauthorized {
			t.Errorf("no token: want 401, got %s", r)
		}
		if r := s.do(x.token, "GET", fmt.Sprintf("/api/v1/subscription-transactions/%d", txIDs[3]), nil); r.status != http.StatusNotFound {
			t.Errorf("X reading Y's receipt: want 404, got %s", r)
		}
		rc := s.mustOK("receipt", s.do(x.token, "GET", fmt.Sprintf("/api/v1/subscription-transactions/%d", txIDs[0]), nil))
		if str(rc, "data", "tenantName") != tenantNameOf[x.tenant] {
			t.Errorf("the receipt names its tenant: %s", rc)
		}
		// The names endpoint the web used before keeps working.
		nm := s.mustOK("names", s.do(admin, "GET", fmt.Sprintf("/api/v1/tenants/names?ids=%d,%d", x.tenant, y.tenant), nil))
		if str(nm, "data", strconv.FormatInt(x.tenant, 10)) != tenantNameOf[x.tenant] {
			t.Errorf("/tenants/names: %s", nm)
		}
		routesCovered++
		if r := s.do(x.token, "GET", "/api/v1/tenants/names?ids=1", nil); r.status != http.StatusForbidden {
			t.Errorf("a workspace token on /tenants/names: want 403, got %s", r)
		}
	})

	t.Run("e: tenants and audit log: filters", func(t *testing.T) {
		list := func(path string) []map[string]any {
			routesCovered++
			return rows(s.mustOK(path, s.do(admin, "GET", path, nil)), "data")
		}
		if got := list("/api/v1/tenants?limit=100"); len(got) != 6 {
			t.Errorf("tenants: %d", len(got))
		}
		if got := list(withQuery("/api/v1/tenants?limit=100", "q", "zaza")); len(got) != 1 {
			t.Errorf("tenants q: %d", len(got))
		}
		if got := list(withQuery("/api/v1/tenants?limit=100", "source", "platform")); len(got) != 4 {
			t.Errorf("tenants source=platform (the four registered by the console): %d", len(got))
		}
		if got := list(withQuery("/api/v1/tenants?limit=100", "source", "self_service")); len(got) != 2 {
			t.Errorf("tenants source=self_service: %d", len(got))
		}
		if got := list(withQuery("/api/v1/tenants?limit=100", "status", "trialing")); len(got) < 1 {
			t.Errorf("tenants status=trialing: %d", len(got))
		}
		// Sorting by plan name puts the tenant without a plan last in both directions.
		for _, dir := range []string{"asc", "desc"} {
			got := list(withQuery("/api/v1/tenants?limit=100", "sort", "planName:"+dir))
			if last := got[len(got)-1]; last["planId"] != nil {
				t.Errorf("planName:%s: the tenant without a plan must be last, got plan %v", dir, last["planId"])
			}
		}
		// The audit log filters.
		audit := func(path string) []map[string]any { return list(path) }
		if got := audit(withQuery("/api/v1/audit-log?limit=100", "action", "POST /t6/plans")); len(got) != 2 {
			t.Errorf("audit action (exact): %d", len(got))
		}
		if got := audit(withQuery("/api/v1/audit-log?limit=100", "action", "POST /t6/plan")); len(got) != 0 {
			t.Errorf("audit action is an exact match, not a prefix: %d", len(got))
		}
		if got := audit(withQuery("/api/v1/audit-log?limit=100", "search", "policy-x")); len(got) != 1 {
			t.Errorf("audit search: %d", len(got))
		}
		if got := audit(withQuery("/api/v1/audit-log?limit=100", "q", "policy-x")); len(got) != 1 {
			t.Errorf("audit q: %d", len(got))
		}
		if got := audit(withQuery("/api/v1/audit-log?limit=100", "from", "2026-02-11", "to", "2026-02-12", "actorType", "platform_admin")); len(got) != 2 {
			t.Errorf("audit dates: %d", len(got))
		}
		if got := audit(withQuery("/api/v1/audit-log?limit=100", "from", "2026-02-11", "to", "2026-02-12", "action", "POST /t6/plans", "sort", "action:asc")); len(got) != 2 {
			t.Errorf("audit filters combine: %d", len(got))
		}
		// Actor sort: type, role, then the id as a number (9 before 10 before 11).
		var ids []string
		for _, m := range audit(withQuery("/api/v1/audit-log?limit=100", "action", "POST /t6/plans", "sort", "actor:asc")) {
			ids = append(ids, fmt.Sprint(m["actorId"]))
		}
		if !reflect.DeepEqual(ids, []string{"9", "10"}) {
			t.Errorf("actor:asc must order ids numerically: %v", ids)
		}
		// Only a Super Admin reaches it (unchanged), a workspace never does.
		routesCovered++
		if r := s.do(x.token, "GET", "/api/v1/audit-log?sort=action", nil); r.status != http.StatusForbidden {
			t.Errorf("a workspace token on the audit log: want 403, got %s", r)
		}
		if r := s.do(x.token, "GET", "/api/v1/tenants?sort=name", nil); r.status != http.StatusForbidden {
			t.Errorf("a workspace token on the tenant list: want 403, got %s", r)
		}
	})

	// ======================= (f) bulk activate / deactivate =======================
	t.Run("f: bulk vendors and venues", func(t *testing.T) {
		cases := []struct {
			name        string
			path        string
			own         []int64
			foreign     []int64
			isActive    func(id int64) bool
			canManage   string
			activeNow   map[int64]bool
			wantRefused string
		}{
			{name: "vendors", path: "/api/v1/vendors/bulk", own: xVendors, foreign: yVendors},
			{name: "venues", path: "/api/v1/venues/bulk", own: xVenues, foreign: []int64{yVenue}},
		}
		table := map[string]string{"vendors": "vendors", "venues": "venues"}
		for _, c := range cases {
			c := c
			state := func(id int64) (active, found bool) {
				var v bool
				err := s.db.QueryRow("SELECT is_active FROM "+table[c.name]+" WHERE id = ?", id).Scan(&v)
				return v, err == nil
			}
			t.Run(c.name, func(t *testing.T) {
				// Current state: own[0] active, own[1] active (vendors) / inactive (venues) — read it, do not assume.
				a, b := c.own[0], c.own[1]
				_, _ = state(a)
				stateB, _ := state(b)

				// Deactivate [a, b, a (duplicate), a foreign row, a missing id]: the foreign and the
				// missing id are only reported, the foreign row is untouched.
				foreign0Before, _ := state(c.foreign[0])
				routesCovered++
				r := s.mustOK("bulk deactivate", s.do(x.token, "POST", c.path, map[string]any{
					"ids": []int64{a, b, a, c.foreign[0], 9_999_999}, "action": "deactivate",
				}))
				wantUpdated := 1
				var wantUnchanged []any
				if stateB {
					wantUpdated = 2
				} else {
					wantUnchanged = []any{float64(b)}
				}
				if num(r, "data", "updated") != int64(wantUpdated) {
					t.Errorf("updated: %s", r)
				}
				gotUnchanged, _ := r.dig("data", "unchanged").([]any)
				if len(gotUnchanged) != len(wantUnchanged) || (len(wantUnchanged) > 0 && gotUnchanged[0] != wantUnchanged[0]) {
					t.Errorf("unchanged: %s", r)
				}
				notFound, _ := r.dig("data", "notFound").([]any)
				if !reflect.DeepEqual(notFound, []any{float64(c.foreign[0]), float64(9_999_999)}) {
					t.Errorf("notFound must list the foreign id and the missing one, in order: %s", r)
				}
				if v, _ := state(a); v {
					t.Errorf("the first row should now be inactive")
				}
				if v, _ := state(c.foreign[0]); v != foreign0Before {
					t.Errorf("another workspace's row was touched by X's bulk request")
				}
				if strings.Contains(string(r.body), y.marker) {
					t.Errorf("the answer leaks the foreign row: %s", r)
				}
				// The same request again changes nothing.
				r = s.mustOK("bulk deactivate again", s.do(x.token, "POST", c.path, map[string]any{"ids": []int64{a, b}, "action": "deactivate"}))
				if num(r, "data", "updated") != 0 || len(r.dig("data", "unchanged").([]any)) != 2 {
					t.Errorf("a repeat is a no-op: %s", r)
				}
				// Activate them back, with a mix of states.
				r = s.mustOK("bulk activate", s.do(x.token, "POST", c.path, map[string]any{"ids": []int64{a, b, c.own[2]}, "action": "activate"}))
				if got := num(r, "data", "updated") + int64(len(r.dig("data", "unchanged").([]any))); got != 3 {
					t.Errorf("activate: every own id is updated or unchanged: %s", r)
				}
				for _, id := range []int64{a, b, c.own[2]} {
					if v, _ := state(id); !v {
						t.Errorf("row %d should be active", id)
					}
				}
				// Never leaks through the other workspace's own request either.
				routesCovered++
				r = s.mustOK("Y bulk with X's ids", s.do(y.token, "POST", c.path, map[string]any{"ids": []int64{a, b}, "action": "deactivate"}))
				if num(r, "data", "updated") != 0 || len(r.dig("data", "notFound").([]any)) != 2 {
					t.Errorf("Y asking about X's ids: %s", r)
				}
				if v, _ := state(a); !v {
					t.Errorf("Y deactivated X's row")
				}

				// Validation: per-field keys, nothing changed.
				before := map[int64]bool{}
				for _, id := range c.own {
					before[id], _ = state(id)
				}
				tooMany := make([]int64, 101)
				for i := range tooMany {
					tooMany[i] = int64(i + 1)
				}
				for label, body := range map[string]map[string]any{
					"no ids":       {"ids": []int64{}, "action": "activate"},
					"missing ids":  {"action": "activate"},
					"too many ids": {"ids": tooMany, "action": "deactivate"},
					"zero id":      {"ids": []int64{a, 0}, "action": "deactivate"},
					"bad action":   {"ids": []int64{a}, "action": "delete"},
					"no action":    {"ids": []int64{a}},
					"both wrong":   {"ids": []int64{}, "action": "toggle"},
				} {
					routesCovered++
					r := s.do(x.token, "POST", c.path, body)
					if r.status != http.StatusUnprocessableEntity {
						t.Errorf("%s: want 422, got %s", label, r)
						continue
					}
					switch label {
					case "no ids", "missing ids", "too many ids", "zero id":
						if !hasKey(r, "ids") {
							t.Errorf("%s: want errors.ids: %s", label, r)
						}
					case "bad action", "no action":
						if !hasKey(r, "action") {
							t.Errorf("%s: want errors.action: %s", label, r)
						}
					case "both wrong":
						if !hasKey(r, "ids") || !hasKey(r, "action") {
							t.Errorf("%s: want both keys: %s", label, r)
						}
					}
				}
				routesCovered++
				if r := s.do(x.token, "POST", c.path, map[string]any{"ids": "abc", "action": "activate"}); r.status != http.StatusUnprocessableEntity || !hasKey(r, "ids") {
					t.Errorf("ids of the wrong type: %s", r)
				}
				maxOK := make([]int64, 100)
				for i := range maxOK {
					maxOK[i] = int64(9_000_000 + i)
				}
				routesCovered++
				if r := s.do(x.token, "POST", c.path, map[string]any{"ids": maxOK, "action": "activate"}); r.status != http.StatusOK || len(r.dig("data", "notFound").([]any)) != 100 {
					t.Errorf("exactly 100 ids are allowed: %s", r)
				}
				for _, id := range c.own {
					if v, _ := state(id); v != before[id] {
						t.Errorf("a refused request changed row %d", id)
					}
				}

				// Permissions: the permission of a single state change (vendor.manage). A planner has none, a client neither.
				for label, tok := range map[string]string{"Wedding Planner": wpTok, "Sales": salesTok} {
					routesCovered++
					if r := s.do(tok, "POST", c.path, map[string]any{"ids": []int64{a}, "action": "deactivate"}); r.status != http.StatusForbidden {
						t.Errorf("%s: want 403, got %s", label, r)
					}
				}
				// A client principal never reaches a staff list or a bulk route (the portal allowlist).
				clientTok := s.addClient(x, pj, "Groom", "Keluarga", fmt.Sprintf("t6.bulk.%s.%d", c.name, time.Now().UnixNano()%1_000_000_000))
				for _, req := range []struct{ method, path string }{
					{"POST", c.path}, {"GET", "/api/v1/clients?page=1&sort=name"}, {"GET", "/api/v1/" + c.name + "?sort=name"}, {"GET", "/api/v1/projects?sort=remaining"},
				} {
					routesCovered++
					if r := s.do(clientTok, req.method, req.path, map[string]any{"ids": []int64{a}, "action": "deactivate"}); r.status != http.StatusForbidden {
						t.Errorf("client principal %s %s: want 403, got %s", req.method, req.path, r)
					}
				}
				routesCovered++
				if r := s.do("", "POST", c.path, map[string]any{"ids": []int64{a}, "action": "deactivate"}); r.status != http.StatusUnauthorized {
					t.Errorf("no token: %s", r)
				}
				routesCovered++
				if r := s.do(x.token, "GET", c.path, nil); r.status != http.StatusMethodNotAllowed {
					t.Errorf("GET on a POST-only route: %s", r)
				}
				if v, _ := state(a); !v {
					t.Errorf("a forbidden request changed row %d", a)
				}
				// Admin holds vendor.manage, like a single state change.
				routesCovered++
				if r := s.do(adminTok, "POST", c.path, map[string]any{"ids": []int64{a}, "action": "activate"}); r.status != http.StatusOK {
					t.Errorf("Admin (vendor.manage): %s", r)
				}
			})
		}

		// Atomic: one failing row makes the whole request change nothing. A trigger refuses the
		// deactivation of the third vendor; the first two must be rolled back with it.
		t.Run("atomic", func(t *testing.T) {
			a, b, c := xVendors[0], xVendors[1], xVendors[2]
			s.mustOK("activate all three", s.do(x.token, "POST", "/api/v1/vendors/bulk", map[string]any{"ids": []int64{a, b, c}, "action": "activate"}))
			if _, err := s.db.Exec(fmt.Sprintf(`CREATE TRIGGER t6_refuse BEFORE UPDATE ON vendors FOR EACH ROW
				BEGIN IF NEW.id = %d AND NEW.is_active = 0 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'refused'; END IF; END`, c)); err != nil {
				t.Fatal(err)
			}
			defer s.db.Exec(`DROP TRIGGER IF EXISTS t6_refuse`)
			routesCovered++
			r := s.do(x.token, "POST", "/api/v1/vendors/bulk", map[string]any{"ids": []int64{a, b, c}, "action": "deactivate"})
			if r.status < 500 {
				t.Errorf("the failing statement should surface as an error, got %s", r)
			}
			for _, id := range []int64{a, b, c} {
				var v bool
				if err := s.db.QueryRow("SELECT is_active FROM vendors WHERE id = ?", id).Scan(&v); err != nil || !v {
					t.Errorf("vendor %d must still be active after the failed bulk request (not atomic)", id)
				}
			}
		})
	})

	// The older shapes of every list keep working with no parameters at all.
	t.Run("the lists with no new parameters answer as before", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/projects", "/api/v1/projects?all=true", "/api/v1/vendors", "/api/v1/vendors?all=true", "/api/v1/venues", "/api/v1/venues?all=true",
			"/api/v1/vendor-categories", "/api/v1/vendor-categories?all=true", "/api/v1/clients", "/api/v1/clients?all=true", "/api/v1/staff", "/api/v1/staff?all=true",
			"/api/v1/subscription-transactions",
		} {
			routesCovered++
			r := s.mustOK(path, s.do(x.token, "GET", path, nil))
			paged := !strings.Contains(path, "all=true")
			if paged != (r.dig("meta") != nil) {
				t.Errorf("GET %s: meta present = %v, want %v", path, r.dig("meta") != nil, paged)
			}
			if paged {
				for _, k := range []string{"page", "limit", "total", "total_pages", "sort"} {
					if r.dig("meta", k) == nil {
						t.Errorf("GET %s: meta.%s missing: %s", path, k, r)
					}
				}
			}
		}
	})
	t.Logf("routes and requests covered: %d", routesCovered)
}
