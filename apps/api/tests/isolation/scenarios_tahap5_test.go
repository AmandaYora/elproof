//go:build integration

package isolation

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Tahap 5 (E1, E2, E8, E12, E13): the project overview, vendor/activity data
// with names, a fresh project progress on every mutation, soft delete with
// restore and purge, kendala without a vendor and with client visibility, and
// the three new dashboard sections. Every scenario signs up its own workspace
// (a trial workspace may have at most 5 active projects), so none depends on or
// disturbs the fixtures of the attack matrix. They need real object storage for
// the proof files and skip without TEST_S3_ENDPOINT.

// ---- helpers ----

// staffWith creates a staff login whose role holds exactly perms (all projects)
// and returns its signed-in token.
func (s *stack) staffWith(t tenant, roleName string, perms ...string) string {
	s.t.Helper()
	role := s.mustOK("role "+roleName, s.do(t.token, "POST", "/api/v1/access/roles", map[string]any{
		"kind": "staff", "name": roleName + " " + t.marker, "description": "uji", "projectScope": "all", "permissions": perms,
	})).id("data", "id")
	email := fmt.Sprintf("%s.%d@example.com", strings.ReplaceAll(roleName, " ", "."), time.Now().UnixNano())
	st := s.mustOK("staff "+roleName, s.do(t.token, "POST", "/api/v1/staff", map[string]any{
		"name": "Staf " + roleName, "title": "Uji", "roleId": role, "email": email, "phone": "081234567890",
	}))
	link := str(st, "data", "invitation", "token")
	// A staff invitation is accepted with the Google account of the invited email.
	sess := s.mustOK("staff accepts", s.do("", "POST", "/api/v1/auth/links/"+link+"/google", map[string]any{
		"credential": s.google.token(email, "Staf "+roleName, fmt.Sprintf("g-sub-staf-%d", time.Now().UnixNano())),
	}))
	tok := str(sess, "data", "accessToken")
	if tok == "" {
		s.t.Fatalf("no token for staff %s: %s", roleName, sess)
	}
	return tok
}

// projectAt creates a project whose event is `eventIn` days away. venueRental
// and venueCharge, when non-zero, attach a venue with those prices.
func (s *stack) projectAt(t tenant, name string, contract int64, eventIn int, venueRental, venueCharge int64) int64 {
	s.t.Helper()
	body := map[string]any{
		"name": name, "brideName": "Bride " + t.marker, "groomName": "Groom " + t.marker,
		"eventDate": day(eventIn), "venue": "Gedung " + t.marker, "prepStartDate": day(-30),
		"packageName": "Paket", "contractValue": contract, "status": "Preparation", "picStaffId": t.staffID,
	}
	if venueRental > 0 || venueCharge > 0 {
		venue := s.mustOK("venue", s.do(t.token, "POST", "/api/v1/venues", venueBody(t, "Venue "+name))).id("data", "id")
		body["venueId"], body["venueRentalPrice"], body["venueCharge"] = venue, venueRental, venueCharge
	}
	return s.mustOK("project "+name, s.do(t.token, "POST", "/api/v1/projects", body)).id("data", "id")
}

func (s *stack) engage(t tenant, projectID, vendorID, categoryID int64, scope string, contract int64, status string, dueIn *int) int64 {
	s.t.Helper()
	body := map[string]any{
		"vendorId": vendorID, "categoryId": categoryID, "scope": scope, "contractValue": contract, "pricingTier": "Akad",
		"engagementStatus": status, "picStaffId": t.staffID,
	}
	if dueIn != nil {
		body["dueDate"] = day(*dueIn)
	}
	return s.mustOK("engage "+scope, s.do(t.token, "POST", s.projectsPath(projectID, "/vendors"), body)).id("data", "id")
}

func (s *stack) newVendor(t tenant, name string) (id, category int64) {
	s.t.Helper()
	v := s.mustOK("vendor "+name, s.do(t.token, "POST", "/api/v1/vendors", vendorBody(s, t, name, 0)))
	return v.id("data", "id"), v.id("data", "categoryId")
}

func (s *stack) setTerms(t tenant, projectID int64, terms ...map[string]any) {
	s.t.Helper()
	s.mustOK("terms", s.do(t.token, "PUT", s.projectsPath(projectID, "/payment-terms"), map[string]any{"terms": terms}))
}

func term(label string, amount int64, dueIn int) map[string]any {
	return map[string]any{"label": label, "amount": amount, "dueDate": day(dueIn)}
}

func (s *stack) clientPay(t tenant, projectID int64, typ string, amount int64, ref string) int64 {
	s.t.Helper()
	return s.mustOK("client payment "+ref, s.do(t.token, "POST", s.projectsPath(projectID, "/client-payments"), map[string]any{
		"type": typ, "amount": amount, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": ref,
	})).id("data", "id")
}

func (s *stack) projectProgress(t tenant, projectID int64) any {
	s.t.Helper()
	return s.mustOK("project", s.do(t.token, "GET", s.projectsPath(projectID, ""), nil)).dig("data", "progress")
}

// progressMatches asserts that a mutation's answer carries the progress a fresh
// GET /projects/{id} reports (E8: a header must never go stale).
func (s *stack) progressMatches(t *testing.T, tn tenant, projectID int64, label string, r reply) {
	t.Helper()
	if r.status < 200 || r.status > 299 {
		t.Errorf("%s: %s", label, r)
		return
	}
	got := r.dig("data", "projectProgress")
	if got == nil {
		t.Errorf("%s: no projectProgress in the answer: %s", label, r)
		return
	}
	if want := s.projectProgress(tn, projectID); !reflect.DeepEqual(got, want) {
		t.Errorf("%s: projectProgress %v differs from a fresh GET /projects/{id} progress %v", label, got, want)
	}
	routesCovered++
}

func (s *stack) purgeNow() string {
	s.t.Helper()
	cmd := exec.Command(s.bin, "purge")
	cmd.Dir, cmd.Env = s.dir, s.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.t.Fatalf("api purge: %v\n%s", err, out)
	}
	return string(out)
}

// ageDeleted pretends a soft-deleted row was deleted `by` ago.
func (s *stack) ageDeleted(table string, id int64, by time.Duration) {
	s.t.Helper()
	res, err := s.db.Exec("UPDATE "+table+" SET deleted_at = DATE_SUB(deleted_at, INTERVAL ? SECOND) WHERE id = ? AND deleted_at IS NOT NULL", int(by.Seconds()), id)
	if err != nil {
		s.t.Fatal(err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		s.t.Fatalf("ageDeleted %s %d: %d rows", table, id, n)
	}
}

func objectExists(t *testing.T, key string) bool {
	t.Helper()
	raw, err := minio.New(os.Getenv("TEST_S3_ENDPOINT"), &minio.Options{Creds: credentials.NewStaticV4(os.Getenv("TEST_S3_ACCESS_KEY"), os.Getenv("TEST_S3_SECRET_KEY"), "")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.StatObject(context.Background(), "isolation", key, minio.StatObjectOptions{})
	return err == nil
}

func (s *stack) storageKey(evidenceID int64) string {
	s.t.Helper()
	var key string
	if err := s.db.QueryRow("SELECT storage_path FROM evidence WHERE id = ?", evidenceID).Scan(&key); err != nil {
		s.t.Fatal(err)
	}
	return key
}

func ids(list []map[string]any) []int64 {
	var out []int64
	for _, m := range list {
		if f, ok := m["id"].(float64); ok {
			out = append(out, int64(f))
		}
	}
	return out
}

func contains(list []int64, id int64) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func within(t *testing.T, label string, got, want time.Time, slack time.Duration) {
	t.Helper()
	if d := got.Sub(want); d < -slack || d > slack {
		t.Errorf("%s: %v is not within %v of %v", label, got, slack, want)
	}
}

// ---- the scenarios ----

func TestTahap5(t *testing.T) {
	s := startStack(t)
	needStorage(t)
	tn, other := s.signUp("t5a"), s.signUp("t5b")
	vendorID, category := s.newVendor(tn, "Vendor T5 "+tn.marker)
	proj := func(id int64, rest string) string { return s.projectsPath(id, rest) }
	ownerName := "Owner " + tn.marker

	// Staff with narrower roles: no margin, no payment deletion, no revenue.
	noMargin := s.staffWith(tn, "tanpa margin", "project.edit", "finance.record", "dashboard.view")
	noDelete := s.staffWith(tn, "tanpa hapus", "project.edit", "finance.record", "finance.margin.view")

	t.Run("(a) overview: shape, numbers, margin gated, a client is refused, another workspace gets 404", func(t *testing.T) {
		P := s.projectAt(tn, "Proyek T5 overview", 100_000_000, 20, 3_000_000, 1_000_000)
		s.setTerms(tn, P, term("DP", 30_000_000, -5), term("Pelunasan", 70_000_000, 40))
		s.clientPay(tn, P, "DP", 10_000_000, "OV-1")
		planned := s.engage(tn, P, vendorID, category, "Dekor planned", 20_000_000, "Planned", nil)
		s.engage(tn, P, vendorID, category, "Foto booked", 5_000_000, "Booked", nil)
		cancelled := s.engage(tn, P, vendorID, category, "Batal", 7_000_000, "Booked", nil)
		s.mustOK("cancel", s.do(tn.token, "POST", proj(P, fmt.Sprintf("/vendors/%d/cancel", cancelled)), map[string]any{}))
		late := s.mustOK("late milestone", s.do(tn.token, "POST", proj(P, "/milestones"), map[string]any{"name": "Fitting lewat", "targetDate": day(-3)})).id("data", "id")
		highIssue := s.mustOK("issue", s.do(tn.token, "POST", proj(P, "/issues"), map[string]any{
			"projectVendorId": planned, "title": "Tenda bocor", "description": "d", "impact": "High", "foundDate": day(-1), "picStaffId": tn.staffID,
		})).id("data", "id")
		s.mustOK("general hidden issue", s.do(tn.token, "POST", proj(P, "/issues"), map[string]any{
			"title": "Parkir sempit", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": tn.staffID, "isClientVisible": false,
		}))
		brideToken := s.addClient(tn, P, "Bride", "Pengantin", fmt.Sprintf("ov.bride.%d", time.Now().UnixNano()%1_000_000_000))
		s.mustOK("invited client", s.do(tn.token, "POST", "/api/v1/clients", map[string]any{
			"projectId": P, "role": "Family Representative", "name": "Ibu Undangan", "phone": "081200000001", "username": fmt.Sprintf("ov.fam.%d", time.Now().UnixNano()%1_000_000_000), "email": "",
		}))

		// Statement budget: the same authenticated call without a handler body is the
		// baseline (token, session, grants); what the overview adds must stay bounded
		// no matter how many vendors, terms or milestones the project has.
		selects := func() int {
			var name string
			var n int
			if err := s.db.QueryRow("SHOW GLOBAL STATUS LIKE 'Com_select'").Scan(&name, &n); err != nil {
				t.Fatal(err)
			}
			return n
		}
		b0 := selects()
		s.mustOK("baseline", s.do(tn.token, "GET", "/api/v1/auth/me", nil))
		baseline := selects() - b0
		o0 := selects()
		r := s.mustOK("overview", s.do(tn.token, "GET", proj(P, "/overview"), nil))
		used := selects() - o0 - baseline
		t.Logf("overview: %d SELECT statements beyond the authentication baseline (%d)", used, baseline)
		if used > 14+2+1 { // 14 fixed + one login lookup per client (2) + slack for a background query
			t.Errorf("the overview used %d statements; it must not grow with the size of the project", used)
		}
		routesCovered++
		for _, k := range []string{"project", "finance", "actions", "nextMilestones", "unconfirmedVendors", "openIssueCount", "team", "clients"} {
			if r.dig("data", k) == nil {
				t.Errorf("overview has no %q: %s", k, r)
			}
		}
		// `project` is exactly what GET /projects/{id} answers, progress and picName included.
		single := s.mustOK("project", s.do(tn.token, "GET", proj(P, ""), nil))
		if got, want := r.dig("data", "project"), single.dig("data"); !reflect.DeepEqual(got, want) {
			t.Errorf("overview.project differs from GET /projects/{id}:\n%v\n%v", got, want)
		}
		terms := s.mustOK("terms", s.do(tn.token, "GET", proj(P, "/payment-terms"), nil))
		fin := r.dig("data", "finance").(map[string]any)
		if !reflect.DeepEqual(fin["terms"], terms.dig("data")) {
			t.Errorf("finance.terms must be what GET /payment-terms returns:\n%v\n%v", fin["terms"], terms.dig("data"))
		}
		want := map[string]float64{"contractValue": 100_000_000, "paid": 10_000_000, "remaining": 90_000_000, "overdueTotal": 20_000_000, "vendorCost": 25_000_000, "venueCost": 4_000_000, "margin": 71_000_000}
		for k, v := range want {
			if fin[k] != v {
				t.Errorf("finance.%s = %v, want %v (a cancelled engagement costs nothing)", k, fin[k], v)
			}
		}
		next, _ := fin["nextTerm"].(map[string]any)
		if next["label"] != "DP" || next["status"] != "overdue" || next["remaining"] != float64(20_000_000) {
			t.Errorf("nextTerm: %v", fin["nextTerm"])
		}

		var kinds []string
		seenWarning := false
		for _, a := range rows(r, "data", "actions") {
			kinds = append(kinds, fmt.Sprint(a["kind"]))
			if a["tone"] == "warning" {
				seenWarning = true
			} else if seenWarning {
				t.Errorf("a danger action after a warning: %v", rows(r, "data", "actions"))
			}
			for _, k := range []string{"id", "title", "detail", "tab"} {
				if a[k] == nil || a[k] == "" {
					t.Errorf("action without %s: %v", k, a)
				}
			}
		}
		for _, k := range []string{"term_overdue", "issue_open", "vendor_unconfirmed", "milestone_late"} {
			if !contains2(kinds, k) {
				t.Errorf("no %s action in %v", k, kinds)
			}
		}
		for _, a := range rows(r, "data", "actions") {
			switch a["kind"] {
			case "vendor_unconfirmed":
				if a["title"] != "Vendor T5 "+tn.marker+" belum terkonfirmasi" || a["tab"] != "vendor" || a["tone"] != "warning" {
					t.Errorf("vendor action: %v", a)
				}
			case "term_overdue":
				if a["tone"] != "danger" || a["tab"] != "pembayaran/client" {
					t.Errorf("term action: %v", a)
				}
			case "issue_open":
				if a["id"] == fmt.Sprintf("issue-%d", highIssue) && a["tone"] != "danger" {
					t.Errorf("a High kendala is danger: %v", a)
				}
			case "milestone_late":
				if a["id"] == fmt.Sprintf("ms-%d", late) && a["tab"] != "milestone" {
					t.Errorf("milestone action: %v", a)
				}
			}
		}
		nm := rows(r, "data", "nextMilestones")
		if len(nm) == 0 || len(nm) > 5 {
			t.Errorf("nextMilestones: %d", len(nm))
		}
		for i := 1; i < len(nm); i++ {
			if fmt.Sprint(nm[i-1]["targetDate"]) > fmt.Sprint(nm[i]["targetDate"]) {
				t.Errorf("nextMilestones not in date order: %v", nm)
			}
		}
		if um := rows(r, "data", "unconfirmedVendors"); len(um) != 1 || um[0]["vendorName"] != "Vendor T5 "+tn.marker || um[0]["engagementStatus"] != "Planned" {
			t.Errorf("unconfirmedVendors: %v", um)
		}
		if n := num(r, "data", "openIssueCount"); n != 2 {
			t.Errorf("staff see every open kendala (2), got %d", n)
		}
		if r.dig("data", "team", "picName") != ownerName || num(r, "data", "team", "picStaffId") != tn.staffID || r.dig("data", "team", "picSalesName") != "" {
			t.Errorf("team: %v", r.dig("data", "team"))
		}
		cl := rows(r, "data", "clients")
		if len(cl) != 2 {
			t.Fatalf("clients: %v", cl)
		}
		logins := map[string]bool{}
		for _, c := range cl {
			logins[fmt.Sprint(c["role"])] = c["hasLogin"] == true
			if c["name"] == "" || c["id"] == nil {
				t.Errorf("client: %v", c)
			}
		}
		if !logins["Bride"] || logins["Family Representative"] {
			t.Errorf("hasLogin: a bride who set a password can sign in, an invited representative cannot yet: %v", logins)
		}

		// finance.margin.view gates `margin`; the rest of the overview is unchanged.
		nm2 := s.mustOK("overview without margin", s.do(noMargin, "GET", proj(P, "/overview"), nil))
		fin2 := nm2.dig("data", "finance").(map[string]any)
		if _, present := fin2["margin"]; present {
			t.Errorf("margin must be absent without finance.margin.view: %v", fin2)
		}
		if fin2["vendorCost"] != float64(25_000_000) {
			t.Errorf("the rest of finance is unchanged: %v", fin2)
		}
		// A client principal and another workspace.
		if r := s.do(brideToken, "GET", proj(P, "/overview"), nil); r.status != http.StatusForbidden {
			t.Errorf("a client: want 403, got %s", r)
		}
		if r := s.do(other.token, "GET", proj(P, "/overview"), nil); r.status != http.StatusNotFound || vacuous(r) {
			t.Errorf("another workspace: want 404, got %s", r)
		} else {
			for _, leak := range []string{tn.marker, "100000000", "71000000"} {
				if strings.Contains(string(r.body), leak) {
					t.Errorf("the 404 leaks %q: %s", leak, r)
				}
			}
		}
		if r := s.do("", "GET", proj(P, "/overview"), nil); r.status != http.StatusUnauthorized {
			t.Errorf("no token: want 401, got %s", r)
		}
		routesCovered += 4
	})

	var (
		Pb, X, E, M, I int64
		bride          string
	)
	t.Run("(b) soft delete and restore: payments, engagements; totals follow", func(t *testing.T) {
		Pb = s.projectAt(tn, "Proyek T5 hapus", 50_000_000, 100, 0, 0)
		s.setTerms(tn, Pb, term("DP", 20_000_000, -2), term("Pelunasan", 30_000_000, 60))
		bride = s.addClient(tn, Pb, "Bride", "Pengantin", fmt.Sprintf("hapus.bride.%d", time.Now().UnixNano()%1_000_000_000))
		X = s.clientPay(tn, Pb, "DP", 20_000_000, "HAPUS-X")
		proofX := s.attach(tn, Pb, "clientPayment", X, "Transfer Proof", "bukti-x", false)
		s.mustOK("verify", s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/verify", X)), nil))
		home := func() reply { return s.mustOK("home", s.do(bride, "GET", "/api/v1/projects/me/home", nil)) }

		if num(home(), "data", "finance", "paid") != 20_000_000 {
			t.Fatalf("setup: the client sees the payment: %s", home())
		}
		ownerOverview := func() reply { return s.mustOK("overview", s.do(tn.token, "GET", proj(Pb, "/overview"), nil)) }
		dash := func() reply { return s.mustOK("dashboard", s.do(tn.token, "GET", "/api/v1/dashboard", nil)) }
		overdueFor := func(d reply, project int64) int {
			n := 0
			for _, row := range rows(d, "data", "overdueTerms") {
				if int64(row["projectId"].(float64)) == project {
					n++
				}
			}
			return n
		}
		if overdueFor(dash(), Pb) != 0 {
			t.Fatal("setup: the DP is paid, so it is not overdue")
		}

		// ---- delete a verified client payment ----
		del := s.do(tn.token, "DELETE", proj(Pb, fmt.Sprintf("/client-payments/%d", X)), nil)
		routesCovered++
		s.mustOK("delete client payment", del)
		if num(del, "data", "id") != X {
			t.Errorf("delete answers the id: %s", del)
		}
		until, err := time.Parse(time.RFC3339, str(del, "data", "restorableUntil"))
		if err != nil {
			t.Fatalf("restorableUntil: %v (%s)", err, del)
		}
		within(t, "restorableUntil", until, time.Now().Add(24*time.Hour), time.Minute)
		if del.dig("data", "projectProgress") == nil || del.dig("data", "ledgerSummary") == nil {
			t.Errorf("a delete carries projectProgress and ledgerSummary: %s", del)
		}
		if num(del, "data", "ledgerSummary", "paid") != 0 || num(del, "data", "ledgerSummary", "remaining") != 50_000_000 {
			t.Errorf("ledger after delete: %v", del.dig("data", "ledgerSummary"))
		}
		if terms := rows(del, "data", "ledgerSummary", "terms"); len(terms) != 2 || terms[0]["status"] != "overdue" {
			t.Errorf("the schedule is re-allocated: %v", terms)
		}
		s.progressMatches(t, tn, Pb, "client payment delete", del)

		list := s.mustOK("list", s.do(tn.token, "GET", proj(Pb, "/client-payments"), nil))
		if contains(ids(rows(list, "data")), X) {
			t.Error("a deleted payment must leave the list")
		}
		if num(ownerOverview(), "data", "finance", "paid") != 0 {
			t.Error("overview: a deleted payment does not count as paid")
		}
		if h := home(); num(h, "data", "finance", "paid") != 0 || len(rows(h, "data", "recentUpdates")) != 0 {
			t.Errorf("the portal home must forget the payment and its verification news: %s", h)
		}
		if overdueFor(dash(), Pb) != 1 {
			t.Error("dashboard: the DP is unpaid again, so it is overdue")
		}
		// Its proof is hidden from everyone until restored.
		ev := s.mustOK("evidence list", s.do(tn.token, "GET", proj(Pb, "/evidence"), nil))
		if contains(ids(rows(ev, "data")), proofX) {
			t.Error("the proof of a deleted payment must not be listed")
		}
		for _, who := range []struct{ name, token string }{{"staff", tn.token}, {"client", bride}} {
			for _, tail := range []string{"file", "link"} {
				if r := s.do(who.token, "GET", proj(Pb, fmt.Sprintf("/evidence/%d/%s", proofX, tail)), nil); r.status != http.StatusNotFound {
					t.Errorf("%s GET evidence/%s of a deleted payment: want 404, got %s", who.name, tail, r)
				}
			}
		}
		if r := s.do(tn.token, "DELETE", proj(Pb, fmt.Sprintf("/client-payments/%d", X)), nil); r.status != http.StatusNotFound {
			t.Errorf("deleting twice: want 404, got %s", r)
		}
		if r := s.do(tn.token, "PATCH", proj(Pb, fmt.Sprintf("/client-payments/%d", X)), clientPayBody(1, "x")); r.status != http.StatusNotFound {
			t.Errorf("editing a deleted payment: want 404, got %s", r)
		}
		if r := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/evidence")), map[string]any{
			"name": "x", "type": "Transfer Proof", "fileName": "x.pdf", "mimeType": "application/pdf", "base64Data": pdf, "relatedKind": "clientPayment", "relatedId": X,
		}); r.status != http.StatusUnprocessableEntity {
			t.Errorf("attaching a file to a deleted payment: want 422, got %s", r)
		}
		// Permissions: a role without finance.payment.delete may neither delete nor restore.
		if r := s.do(noDelete, "DELETE", proj(Pb, fmt.Sprintf("/client-payments/%d", X)), nil); r.status != http.StatusForbidden {
			t.Errorf("delete without finance.payment.delete: want 403, got %s", r)
		}
		if r := s.do(noDelete, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/restore", X)), nil); r.status != http.StatusForbidden {
			t.Errorf("restore without finance.payment.delete: want 403, got %s", r)
		}
		if r := s.do(bride, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/restore", X)), nil); r.status != http.StatusForbidden {
			t.Errorf("a client may not restore: want 403, got %s", r)
		}
		if r := s.do(other.token, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/restore", X)), nil); r.status != http.StatusNotFound {
			t.Errorf("another workspace: want 404, got %s", r)
		}

		// ---- restore ----
		res := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/restore", X)), nil)
		routesCovered++
		s.mustOK("restore", res)
		if num(res, "data", "id") != X || res.dig("data", "verifiedAt") == nil || len(rows(res, "data", "evidence")) != 1 {
			t.Errorf("a restored payment is the list item again, verification and proof included: %s", res)
		}
		if num(res, "data", "ledgerSummary", "paid") != 20_000_000 || res.dig("data", "projectProgress") == nil {
			t.Errorf("restore carries ledgerSummary and projectProgress: %s", res)
		}
		s.progressMatches(t, tn, Pb, "client payment restore", res)
		if h := home(); num(h, "data", "finance", "paid") != 20_000_000 || len(rows(h, "data", "recentUpdates")) != 1 {
			t.Errorf("home after restore: %s", h)
		}
		if r := s.do(tn.token, "GET", proj(Pb, fmt.Sprintf("/evidence/%d/file", proofX)), nil); r.status != http.StatusOK {
			t.Errorf("the proof opens again after restore: %s", r)
		}
		if overdueFor(dash(), Pb) != 0 {
			t.Error("dashboard after restore: the DP is paid again")
		}
		if r := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/restore", X)), nil); r.status != http.StatusNotFound {
			t.Errorf("restoring a payment that is not deleted: want 404, got %s", r)
		}

		// ---- restore re-checks the overpayment rule ----
		s.mustOK("delete again", s.do(tn.token, "DELETE", proj(Pb, fmt.Sprintf("/client-payments/%d", X)), nil))
		Y := s.clientPay(tn, Pb, "Pelunasan", 35_000_000, "HAPUS-Y") // 35M fits only because X (20M) is gone
		bad := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/restore", X)), nil)
		if bad.status != http.StatusUnprocessableEntity || !hasKey(bad, "amount") {
			t.Errorf("restore past the contract: want 422 on amount, got %s", bad)
		}
		if bad.dig("errors", "code") != "payment_exceeds_contract" {
			t.Errorf("the usual code: %s", bad)
		}
		s.mustOK("make room", s.do(tn.token, "DELETE", proj(Pb, fmt.Sprintf("/client-payments/%d", Y)), nil))
		s.mustOK("restore now fits", s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/client-payments/%d/restore", X)), nil))

		// ---- vendor payment, venue payment ----
		E = s.engage(tn, Pb, vendorID, category, "Dekor hapus", 20_000_000, "Booked", nil)
		M = s.mustOK("vendor milestone", s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/vendors/%d/milestones", E)), map[string]any{
			"name": "Pasang tenda", "description": "d", "targetDate": day(30), "picStaffId": tn.staffID,
		})).id("data", "id")
		I = s.mustOK("issue of the engagement", s.do(tn.token, "POST", proj(Pb, "/issues"), map[string]any{
			"projectVendorId": E, "vendorMilestoneId": M, "title": "Tenda kecil", "description": "d", "impact": "Medium", "foundDate": day(-1), "picStaffId": tn.staffID,
		})).id("data", "id")
		vp := s.mustOK("vendor payment", s.do(tn.token, "POST", proj(Pb, "/payments"), map[string]any{
			"projectVendorId": E, "type": "DP", "amount": 5_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "VP-HAPUS",
		})).id("data", "id")
		vProof := s.attach(tn, Pb, "payment", vp, "Transfer Proof", "bukti-vp", false)
		paid := func() float64 {
			for _, row := range rows(s.mustOK("vendors", s.do(tn.token, "GET", proj(Pb, "/vendors"), nil)), "data") {
				if int64(row["id"].(float64)) == E {
					return row["paidAmount"].(float64)
				}
			}
			return -1
		}
		if paid() != 5_000_000 {
			t.Fatalf("setup: vendor paid = %v", paid())
		}
		dv := s.do(tn.token, "DELETE", proj(Pb, fmt.Sprintf("/payments/%d", vp)), nil)
		s.mustOK("delete vendor payment", dv)
		routesCovered++
		if num(dv, "data", "ledgerSummary", "paid") != 0 || num(dv, "data", "ledgerSummary", "contractValue") != 20_000_000 || str(dv, "data", "restorableUntil") == "" {
			t.Errorf("vendor payment delete: %s", dv)
		}
		s.progressMatches(t, tn, Pb, "vendor payment delete", dv)
		if paid() != 0 {
			t.Error("the vendor's paid amount must follow the deletion")
		}
		if r := s.do(tn.token, "GET", proj(Pb, fmt.Sprintf("/evidence/%d/file", vProof)), nil); r.status != http.StatusNotFound {
			t.Errorf("vendor payment proof while deleted: want 404, got %s", r)
		}
		rv := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/payments/%d/restore", vp)), nil)
		s.mustOK("restore vendor payment", rv)
		routesCovered++
		if num(rv, "data", "ledgerSummary", "paid") != 5_000_000 || len(rows(rv, "data", "evidence")) != 1 || paid() != 5_000_000 {
			t.Errorf("vendor payment restore: %s", rv)
		}
		s.progressMatches(t, tn, Pb, "vendor payment restore", rv)

		venuePay := s.mustOK("venue payment", s.do(tn.token, "POST", proj(Pb, "/venue-payments"), map[string]any{
			"type": "DP", "amount": 1_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "VN-HAPUS",
		})).id("data", "id")
		dn := s.do(tn.token, "DELETE", proj(Pb, fmt.Sprintf("/venue-payments/%d", venuePay)), nil)
		s.mustOK("delete venue payment", dn)
		routesCovered++
		if dn.dig("data", "ledgerSummary") == nil || contains(ids(rows(s.mustOK("venue list", s.do(tn.token, "GET", proj(Pb, "/venue-payments"), nil)), "data")), venuePay) {
			t.Errorf("venue payment delete: %s", dn)
		}
		s.progressMatches(t, tn, Pb, "venue payment delete", dn)
		rn := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/venue-payments/%d/restore", venuePay)), nil)
		s.mustOK("restore venue payment", rn)
		routesCovered++
		if num(rn, "data", "id") != venuePay || rn.dig("data", "ledgerSummary") == nil {
			t.Errorf("venue payment restore: %s", rn)
		}
		s.progressMatches(t, tn, Pb, "venue payment restore", rn)

		// ---- delete a vendor engagement: milestones, payments, kendala go with it ----
		before := s.projectProgress(tn, Pb).(map[string]any)
		if before["vendorMilestoneStats"].(map[string]any)["total"] != float64(1) || before["openIssueCount"] != float64(1) {
			t.Fatalf("setup progress: %v", before)
		}
		if oc := ownerOverview(); num(oc, "data", "finance", "vendorCost") != 20_000_000 {
			t.Fatalf("setup vendorCost: %s", oc)
		}
		de := s.do(tn.token, "DELETE", proj(Pb, fmt.Sprintf("/vendors/%d", E)), nil)
		s.mustOK("delete engagement", de)
		routesCovered++
		if num(de, "data", "id") != E || str(de, "data", "restorableUntil") == "" {
			t.Errorf("engagement delete: %s", de)
		}
		after := de.dig("data", "projectProgress").(map[string]any)
		if after["vendorMilestoneStats"].(map[string]any)["total"] != float64(0) || after["openIssueCount"] != float64(0) {
			t.Errorf("a deleted engagement's milestones and kendala no longer count: %v", after)
		}
		s.progressMatches(t, tn, Pb, "engagement delete", de)
		if contains(ids(rows(s.mustOK("vendors", s.do(tn.token, "GET", proj(Pb, "/vendors"), nil)), "data")), E) {
			t.Error("a deleted engagement leaves the vendor list")
		}
		if contains(ids(rows(s.mustOK("payments", s.do(tn.token, "GET", proj(Pb, "/payments"), nil)), "data")), vp) {
			t.Error("its payments leave the payment list")
		}
		if contains(ids(rows(s.mustOK("issues", s.do(tn.token, "GET", proj(Pb, "/issues"), nil)), "data")), I) {
			t.Error("its kendala leave the issue list")
		}
		if num(ownerOverview(), "data", "finance", "vendorCost") != 0 {
			t.Error("overview: a deleted engagement costs nothing")
		}
		if r := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/payments/%d/restore", vp)), nil); r.status != http.StatusNotFound {
			t.Errorf("a payment hidden only because its engagement is deleted is not itself deleted: want 404, got %s", r)
		}
		if r := s.do(tn.token, "POST", proj(Pb, "/payments"), map[string]any{"projectVendorId": E, "type": "DP", "amount": 1, "paymentDate": day(-1), "method": "Transfer"}); r.status < 400 {
			t.Errorf("paying a deleted engagement must fail: %s", r)
		}
		re := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/vendors/%d/restore", E)), nil)
		s.mustOK("restore engagement", re)
		routesCovered++
		for _, k := range []string{"vendorName", "categoryName", "picName", "evidenceCount", "milestones", "paidAmount"} {
			if re.dig("data", k) == nil {
				t.Errorf("a restored engagement is the list item (%s missing): %s", k, re)
			}
		}
		if str(re, "data", "vendorName") != "Vendor T5 "+tn.marker || len(rows(re, "data", "milestones")) != 1 || num(re, "data", "paidAmount") != 5_000_000 || num(re, "data", "evidenceCount") != 1 {
			t.Errorf("restored engagement content: %s", re)
		}
		s.progressMatches(t, tn, Pb, "engagement restore", re)
		back := s.projectProgress(tn, Pb).(map[string]any)
		if back["vendorMilestoneStats"].(map[string]any)["total"] != float64(1) || back["openIssueCount"] != float64(1) {
			t.Errorf("everything that hung off the engagement counts again: %v", back)
		}
		if r := s.do(tn.token, "POST", proj(Pb, fmt.Sprintf("/vendors/%d/restore", E)), nil); r.status != http.StatusNotFound {
			t.Errorf("restoring a live engagement: want 404, got %s", r)
		}
	})

	t.Run("(c) purge: restore after purge is 404, rows, proof rows and stored files are removed", func(t *testing.T) {
		Pc := s.projectAt(tn, "Proyek T5 purge", 80_000_000, 100, 0, 0)
		s.setTerms(tn, Pc, term("DP", 10_000_000, 10))
		Z := s.clientPay(tn, Pc, "DP", 10_000_000, "PURGE-Z")
		proofZ := s.attach(tn, Pc, "clientPayment", Z, "Transfer Proof", "bukti-z", false)
		keyZ := s.storageKey(proofZ)
		W := s.clientPay(tn, Pc, "Termin", 1_000_000, "PURGE-W") // deleted but NOT aged: must survive the purge
		proofW := s.attach(tn, Pc, "clientPayment", W, "Transfer Proof", "bukti-w", false)

		EV := s.engage(tn, Pc, vendorID, category, "Dekor purge", 9_000_000, "Booked", nil)
		MS := s.mustOK("ms", s.do(tn.token, "POST", proj(Pc, fmt.Sprintf("/vendors/%d/milestones", EV)), map[string]any{"name": "m", "description": "d", "targetDate": day(30), "picStaffId": tn.staffID})).id("data", "id")
		msProof := s.attach(tn, Pc, "vendorMilestone", MS, "Photo", "foto-ms", false)
		IS := s.mustOK("issue", s.do(tn.token, "POST", proj(Pc, "/issues"), map[string]any{"projectVendorId": EV, "title": "k", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": tn.staffID})).id("data", "id")
		VP := s.mustOK("vp", s.do(tn.token, "POST", proj(Pc, "/payments"), map[string]any{"projectVendorId": EV, "type": "DP", "amount": 1_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "PURGE-VP"})).id("data", "id")
		vpProof := s.attach(tn, Pc, "payment", VP, "Transfer Proof", "bukti-vp-purge", false)
		keys := map[string]string{"ms": s.storageKey(msProof), "vp": s.storageKey(vpProof)}
		for name, k := range map[string]string{"Z": keyZ, "ms": keys["ms"], "vp": keys["vp"]} {
			if !objectExists(t, k) {
				t.Fatalf("setup: object of %s is not in storage", name)
			}
		}

		// A client: deleting it switches its login off, restoring brings it back.
		cu := fmt.Sprintf("purge.klien.%d", time.Now().UnixNano()%1_000_000_000)
		cTok := s.addClient(tn, Pc, "Bride", "Pengantin", cu)
		clients := s.mustOK("clients", s.do(tn.token, "GET", "/api/v1/clients?projectId="+fmt.Sprint(Pc), nil))
		clientID := int64(rows(clients, "data")[0]["id"].(float64))
		if r := s.do(cTok, "GET", "/api/v1/projects/me", nil); r.status != http.StatusOK {
			t.Fatalf("setup: the client signs in: %s", r)
		}
		dc := s.do(tn.token, "DELETE", fmt.Sprintf("/api/v1/clients/%d", clientID), nil)
		s.mustOK("delete client", dc)
		routesCovered++
		if num(dc, "data", "id") != clientID {
			t.Errorf("client delete: %s", dc)
		}
		until, err := time.Parse(time.RFC3339, str(dc, "data", "restorableUntil"))
		if err != nil {
			t.Fatalf("client restorableUntil: %v", err)
		}
		within(t, "client restorableUntil", until, time.Now().Add(24*time.Hour), time.Minute)
		if contains(ids(rows(s.mustOK("clients", s.do(tn.token, "GET", "/api/v1/clients?projectId="+fmt.Sprint(Pc), nil)), "data")), clientID) {
			t.Error("a deleted client leaves the project's clients")
		}
		if r := s.do(cTok, "GET", "/api/v1/projects/me", nil); r.status == http.StatusOK {
			t.Errorf("a deleted client must be locked out at once: %s", r)
		}
		if r := s.do(tn.token, "PATCH", fmt.Sprintf("/api/v1/clients/%d", clientID), map[string]any{"name": "x", "phone": "1", "email": ""}); r.status != http.StatusNotFound {
			t.Errorf("a deleted client cannot be edited: want 404, got %s", r)
		}
		rc := s.do(tn.token, "POST", fmt.Sprintf("/api/v1/clients/%d/restore", clientID), nil)
		s.mustOK("restore client", rc)
		routesCovered++
		if num(rc, "data", "id") != clientID || rc.dig("data", "loginStatus") != "active" || rc.dig("data", "isActive") != true {
			t.Errorf("a restored client is the list item with their login as it was: %s", rc)
		}
		if r := s.do(cTok, "GET", "/api/v1/projects/me", nil); r.status != http.StatusOK {
			t.Errorf("after restore the same client signs in again: %s", r)
		}
		if r := s.do(tn.token, "POST", fmt.Sprintf("/api/v1/clients/%d/restore", clientID), nil); r.status != http.StatusNotFound {
			t.Errorf("restoring a live client: want 404, got %s", r)
		}
		if r := s.do(other.token, "POST", fmt.Sprintf("/api/v1/clients/%d/restore", clientID), nil); r.status != http.StatusNotFound {
			t.Errorf("another workspace: want 404, got %s", r)
		}
		s.mustOK("delete client again", s.do(tn.token, "DELETE", fmt.Sprintf("/api/v1/clients/%d", clientID), nil))

		// A client who never set a password (still invited) is deletable and restorable too.
		inv := s.mustOK("invited client", s.do(tn.token, "POST", "/api/v1/clients", map[string]any{
			"projectId": Pc, "role": "Family Representative", "name": "Ibu Undangan", "phone": "081200000002", "email": "",
			"username": fmt.Sprintf("purge.undangan.%d", time.Now().UnixNano()%1_000_000_000),
		}))
		invID := inv.id("data", "client", "id")
		s.mustOK("delete invited client", s.do(tn.token, "DELETE", fmt.Sprintf("/api/v1/clients/%d", invID), nil))
		if r := s.do(tn.token, "POST", fmt.Sprintf("/api/v1/clients/%d/restore", invID), nil); r.status != http.StatusOK || r.dig("data", "loginStatus") != "invited" {
			t.Errorf("an invited client restores as invited: %s", r)
		}

		// Delete everything, then let the first batch grow old (past the 24 h window).
		s.mustOK("delete Z", s.do(tn.token, "DELETE", proj(Pc, fmt.Sprintf("/client-payments/%d", Z)), nil))
		s.mustOK("delete W", s.do(tn.token, "DELETE", proj(Pc, fmt.Sprintf("/client-payments/%d", W)), nil))
		s.mustOK("delete engagement", s.do(tn.token, "DELETE", proj(Pc, fmt.Sprintf("/vendors/%d", EV)), nil))
		s.ageDeleted("client_payments", Z, 49*time.Hour)
		s.ageDeleted("project_vendors", EV, 25*time.Hour)
		s.ageDeleted("clients", clientID, 30*time.Hour)

		// Past the window but not purged yet: already counts as purged.
		for name, path := range map[string]string{
			"client payment": proj(Pc, fmt.Sprintf("/client-payments/%d/restore", Z)),
			"engagement":     proj(Pc, fmt.Sprintf("/vendors/%d/restore", EV)),
			"client":         fmt.Sprintf("/api/v1/clients/%d/restore", clientID),
		} {
			if r := s.do(tn.token, "POST", path, nil); r.status != http.StatusNotFound {
				t.Errorf("restore %s past the window (purge has not run): want 404, got %s", name, r)
			}
		}
		if s.count("SELECT COUNT(*) FROM client_payments WHERE id = ?", Z) != 1 {
			t.Fatal("setup: the aged row is still in the table until the purge runs")
		}

		out := s.purgeNow()
		if !strings.Contains(out, "purge: done") {
			t.Errorf("purge output: %s", out)
		}
		gone := map[string]int{
			"client payment":      s.count("SELECT COUNT(*) FROM client_payments WHERE id = ?", Z),
			"its proof row":       s.count("SELECT COUNT(*) FROM evidence WHERE id = ?", proofZ),
			"engagement":          s.count("SELECT COUNT(*) FROM project_vendors WHERE id = ?", EV),
			"its milestone":       s.count("SELECT COUNT(*) FROM vendor_milestones WHERE id = ?", MS),
			"its milestone proof": s.count("SELECT COUNT(*) FROM evidence WHERE id = ?", msProof),
			"its kendala":         s.count("SELECT COUNT(*) FROM vendor_issues WHERE id = ?", IS),
			"its payment":         s.count("SELECT COUNT(*) FROM vendor_payments WHERE id = ?", VP),
			"its payment proof":   s.count("SELECT COUNT(*) FROM evidence WHERE id = ?", vpProof),
			"client":              s.count("SELECT COUNT(*) FROM clients WHERE id = ?", clientID),
			"client login":        s.count("SELECT COUNT(*) FROM credentials WHERE principal_type = 'client' AND principal_id = ?", fmt.Sprint(clientID)),
		}
		for what, n := range gone {
			if n != 0 {
				t.Errorf("after the purge %s must be gone, %d row(s) left", what, n)
			}
		}
		for name, k := range map[string]string{"Z": keyZ, "milestone": keys["ms"], "payment": keys["vp"]} {
			if objectExists(t, k) {
				t.Errorf("the stored file of %s must be deleted from object storage", name)
			}
		}
		// Not due yet: untouched, and still restorable.
		if s.count("SELECT COUNT(*) FROM client_payments WHERE id = ? AND deleted_at IS NOT NULL", W) != 1 || s.count("SELECT COUNT(*) FROM evidence WHERE id = ?", proofW) != 1 || !objectExists(t, s.storageKey(proofW)) {
			t.Error("a payment deleted a moment ago must survive the purge with its proof")
		}
		// Restore after the purge: 404, never a half-restored ghost.
		for name, path := range map[string]string{
			"client payment": proj(Pc, fmt.Sprintf("/client-payments/%d/restore", Z)),
			"engagement":     proj(Pc, fmt.Sprintf("/vendors/%d/restore", EV)),
			"client":         fmt.Sprintf("/api/v1/clients/%d/restore", clientID),
			"vendor payment": proj(Pc, fmt.Sprintf("/payments/%d/restore", VP)),
		} {
			if r := s.do(tn.token, "POST", path, nil); r.status != http.StatusNotFound {
				t.Errorf("restore %s after the purge: want 404, got %s", name, r)
			}
		}
		routesCovered += 7
		// Idempotent: a second run finds nothing more to do.
		rowsBefore := s.count("SELECT COUNT(*) FROM evidence")
		s.purgeNow()
		if s.count("SELECT COUNT(*) FROM evidence") != rowsBefore || s.count("SELECT COUNT(*) FROM client_payments WHERE id = ?", W) != 1 {
			t.Error("a second purge must change nothing")
		}
		s.mustOK("W restores", s.do(tn.token, "POST", proj(Pc, fmt.Sprintf("/client-payments/%d/restore", W)), nil))
		// The client's username and email are free again once the login is purged.
		if r := s.do(tn.token, "POST", "/api/v1/clients", map[string]any{"projectId": Pc, "role": "Bride", "name": "Pengganti", "phone": "0812", "username": cu, "email": ""}); r.status != http.StatusCreated {
			t.Errorf("the purged client's username must be free again: %s", r)
		}
	})

	t.Run("(d) kendala: no vendor, client visibility, and nothing hidden leaks to the portal", func(t *testing.T) {
		Pd := s.projectAt(tn, "Proyek T5 kendala", 40_000_000, 100, 0, 0)
		s.setTerms(tn, Pd, term("DP", 1_000_000, 30))
		EV := s.engage(tn, Pd, vendorID, category, "Dekor kendala", 3_000_000, "Booked", nil)
		clientTok := s.addClient(tn, Pd, "Bride", "Pengantin", fmt.Sprintf("kendala.klien.%d", time.Now().UnixNano()%1_000_000_000))
		mk := func(title, impact string, vendor int64, visible *bool) reply {
			body := map[string]any{"title": title, "description": "d", "impact": impact, "foundDate": day(-1), "picStaffId": tn.staffID}
			if vendor != 0 {
				body["projectVendorId"] = vendor
			}
			if visible != nil {
				body["isClientVisible"] = *visible
			}
			return s.do(tn.token, "POST", proj(Pd, "/issues"), body)
		}
		f, tr := false, true
		made := func(name string, r reply) reply {
			s.mustOK(name, r)
			s.progressMatches(t, tn, Pd, "issue create "+name, r)
			return r
		}
		shared := made("shared", mk("Dibagikan", "Low", EV, nil))   // default: shown
		hidden := made("hidden", mk("Rahasia", "Critical", EV, &f)) // internal
		generalShared := made("general shared", mk("Umum dibagikan", "Low", 0, &tr))
		generalHidden := made("general hidden", mk("Umum rahasia", "Low", 0, &f))
		sharedID, hiddenID, genSharedID, genHiddenID := shared.id("data", "id"), hidden.id("data", "id"), generalShared.id("data", "id"), generalHidden.id("data", "id")
		if shared.dig("data", "isClientVisible") != true || hidden.dig("data", "isClientVisible") != false {
			t.Errorf("isClientVisible: default true, false honoured: %s %s", shared, hidden)
		}
		if v, present := generalShared.json()["data"].(map[string]any)["projectVendorId"]; !present || v != nil {
			t.Errorf("an issue without a vendor has projectVendorId null: %s", generalShared)
		}
		if generalShared.dig("data", "vendorName") != "" || generalShared.dig("data", "picName") != ownerName || shared.dig("data", "vendorName") != "Vendor T5 "+tn.marker {
			t.Errorf("names on create: %s / %s", generalShared, shared)
		}
		hiddenProof := s.attach(tn, Pd, "issue", hiddenID, "Photo", "foto-rahasia", false)
		sharedProof := s.attach(tn, Pd, "issue", sharedID, "Photo", "foto-dibagikan", false)

		// Staff see all four, with the names.
		staffIssues := rows(s.mustOK("staff issues", s.do(tn.token, "GET", proj(Pd, "/issues"), nil)), "data")
		if len(staffIssues) != 4 {
			t.Fatalf("staff see every kendala: %v", staffIssues)
		}
		if row := rowByID(staffIssues, genSharedID); row == nil || row["vendorName"] != "" || row["vendorMilestoneName"] != "" || row["projectVendorId"] != nil || row["picName"] != ownerName {
			t.Errorf("general kendala row: %v", row)
		}
		// The client: only the two shared ones, however it asks.
		got := ids(rows(s.mustOK("client issues", s.do(clientTok, "GET", proj(Pd, "/issues"), nil)), "data"))
		if len(got) != 2 || !contains(got, sharedID) || !contains(got, genSharedID) {
			t.Errorf("the client sees only kendala shared with them, got %v", got)
		}
		cl := rows(s.mustOK("client issues", s.do(clientTok, "GET", proj(Pd, "/issues"), nil)), "data")
		if row := rowByID(cl, sharedID); row == nil || row["picName"] != "" {
			t.Errorf("a client never gets the staff PIC's name: %v", row)
		}
		// Proof of a hidden kendala never reaches the client; proof of a shared one does.
		evClient := ids(rows(s.mustOK("client evidence", s.do(clientTok, "GET", proj(Pd, "/evidence"), nil)), "data"))
		if contains(evClient, hiddenProof) || !contains(evClient, sharedProof) {
			t.Errorf("client evidence list: %v (hidden %d, shared %d)", evClient, hiddenProof, sharedProof)
		}
		for _, tail := range []string{"file", "link"} {
			if r := s.do(clientTok, "GET", proj(Pd, fmt.Sprintf("/evidence/%d/%s", hiddenProof, tail)), nil); r.status != http.StatusNotFound {
				t.Errorf("client GET hidden kendala proof /%s: want 404, got %s", tail, r)
			}
			if r := s.do(clientTok, "GET", proj(Pd, fmt.Sprintf("/evidence/%d/%s", sharedProof, tail)), nil); r.status != http.StatusOK {
				t.Errorf("client GET shared kendala proof /%s: want 200, got %s", tail, r)
			}
			if r := s.do(tn.token, "GET", proj(Pd, fmt.Sprintf("/evidence/%d/%s", hiddenProof, tail)), nil); r.status != http.StatusOK {
				t.Errorf("staff GET hidden kendala proof /%s: want 200, got %s", tail, r)
			}
		}
		// The portal home and the project: counts, actions, condition.
		home := s.mustOK("home", s.do(clientTok, "GET", "/api/v1/projects/me/home", nil))
		if num(home, "data", "openIssueCount") != 2 {
			t.Errorf("the client's openIssueCount counts only shared kendala: %s", home)
		}
		foundAction := false
		for _, a := range rows(home, "data", "actions") {
			if a["kind"] == "open_issues" {
				foundAction = true
				if a["title"] != "2 hal sedang kami tangani" {
					t.Errorf("the action counts only shared kendala: %v", a)
				}
			}
		}
		if !foundAction {
			t.Errorf("home has no open_issues action: %s", home)
		}
		staffProj := s.mustOK("staff project", s.do(tn.token, "GET", proj(Pd, ""), nil))
		clientProj := s.mustOK("client project", s.do(clientTok, "GET", proj(Pd, ""), nil))
		if staffProj.dig("data", "progress", "condition") != "At Risk" || num(staffProj, "data", "progress", "openIssueCount") != 4 {
			t.Errorf("staff progress counts everything: %v", staffProj.dig("data", "progress"))
		}
		if clientProj.dig("data", "progress", "condition") == "At Risk" || num(clientProj, "data", "progress", "openIssueCount") != 2 || num(clientProj, "data", "progress", "criticalOrHighOpenIssueCount") != 0 {
			t.Errorf("a hidden Critical kendala must not show in the client's progress: %v", clientProj.dig("data", "progress"))
		}
		if strings.Contains(string(home.body), "Rahasia") || strings.Contains(string(clientProj.body), "Rahasia") {
			t.Error("a hidden kendala's title leaked to the client")
		}
		// Resolving a hidden kendala is not news for the client.
		upd := func(id int64, body map[string]any) reply {
			b := map[string]any{"title": "Rahasia", "description": "d", "impact": "Critical", "foundDate": day(-1), "status": "Open", "picStaffId": tn.staffID, "projectVendorId": EV}
			for k, v := range body {
				b[k] = v
			}
			return s.do(tn.token, "PATCH", proj(Pd, fmt.Sprintf("/issues/%d", id)), b)
		}
		res := upd(hiddenID, map[string]any{"status": "Resolved"}) // isClientVisible omitted: stays hidden
		s.mustOK("resolve hidden", res)
		s.progressMatches(t, tn, Pd, "issue update", res)
		if res.dig("data", "isClientVisible") != false {
			t.Errorf("an update that does not send isClientVisible keeps it: %s", res)
		}
		home = s.mustOK("home", s.do(clientTok, "GET", "/api/v1/projects/me/home", nil))
		for _, u := range rows(home, "data", "recentUpdates") {
			if u["kind"] == "issue_resolved" {
				t.Errorf("resolving a hidden kendala must not be news: %v", u)
			}
		}
		// Edit: general -> vendor -> general again; sharing it later.
		if r := upd(genHiddenID, map[string]any{"title": "Umum rahasia", "impact": "Low", "projectVendorId": EV, "isClientVisible": true}); r.status != http.StatusOK || num(r, "data", "projectVendorId") != EV || r.dig("data", "isClientVisible") != true {
			t.Errorf("attach a vendor and share: %s", r)
		}
		if r := upd(genHiddenID, map[string]any{"title": "Umum rahasia", "impact": "Low", "projectVendorId": 0}); r.status != http.StatusOK || r.json()["data"].(map[string]any)["projectVendorId"] != nil || r.dig("data", "isClientVisible") != true {
			t.Errorf("detach the vendor: %s", r)
		}
		if got := ids(rows(s.mustOK("client issues", s.do(clientTok, "GET", proj(Pd, "/issues"), nil)), "data")); !contains(got, genHiddenID) {
			t.Errorf("a kendala that was shared later is visible to the client: %v", got)
		}
		// A deliverable needs a vendor.
		if r := mk("x", "Low", 0, nil); r.status != http.StatusCreated {
			t.Errorf("setup: %s", r)
		}
		bad := s.do(tn.token, "POST", proj(Pd, "/issues"), map[string]any{"title": "x", "description": "d", "impact": "Low", "foundDate": day(-1), "picStaffId": tn.staffID, "vendorMilestoneId": 12345})
		if bad.status != http.StatusUnprocessableEntity || !hasKey(bad, "vendorMilestoneId") {
			t.Errorf("a vendor milestone without a vendor: want 422 on vendorMilestoneId, got %s", bad)
		}
		// Dashboard: a vendor-less kendala is a row with no vendor, not a hole.
		d := s.mustOK("dashboard", s.do(tn.token, "GET", "/api/v1/dashboard", nil))
		var general map[string]any
		for _, row := range rows(d, "data", "openIssues") {
			if int64(row["id"].(float64)) == genSharedID {
				general = row
			}
		}
		if general == nil || general["vendorId"] != float64(0) || general["vendorName"] != "" || general["projectName"] != "Proyek T5 kendala" {
			t.Errorf("dashboard row of a kendala without a vendor: %v", general)
		}
	})

	t.Run("(e) activity: paginated, filterable, with actor names", func(t *testing.T) {
		page := s.mustOK("activity", s.do(tn.token, "GET", proj(Pb, "/activity?limit=3&page=1"), nil))
		routesCovered++
		entries := rows(page, "data")
		if len(entries) != 3 {
			t.Fatalf("limit=3: %d entries: %s", len(entries), page)
		}
		meta, _ := page.dig("meta").(map[string]any)
		if meta["page"] != float64(1) || meta["limit"] != float64(3) || meta["total"].(float64) < 6 || meta["total_pages"].(float64) < 2 {
			t.Errorf("meta: %v", meta)
		}
		for _, e := range entries {
			if e["actorName"] != ownerName {
				t.Errorf("actorName = %v, want %q", e["actorName"], ownerName)
			}
			for _, k := range []string{"id", "type", "actorStaffId", "projectId", "entityType", "entityId", "entityLabel", "description", "timestamp"} {
				if _, ok := e[k]; !ok {
					t.Errorf("the existing field %q is missing: %v", k, e)
				}
			}
		}
		for i := 1; i < len(entries); i++ {
			if entries[i-1]["id"].(float64) < entries[i]["id"].(float64) {
				t.Errorf("newest first: %v", ids(entries))
			}
		}
		page2 := rows(s.mustOK("page 2", s.do(tn.token, "GET", proj(Pb, "/activity?limit=3&page=2"), nil)), "data")
		if len(page2) == 0 || contains(ids(entries), ids(page2)[0]) {
			t.Errorf("page 2 continues where page 1 stopped: %v then %v", ids(entries), ids(page2))
		}
		def := s.mustOK("default", s.do(tn.token, "GET", proj(Pb, "/activity"), nil))
		if dm, _ := def.dig("meta").(map[string]any); dm["limit"] != float64(20) || len(rows(def, "data")) > 20 {
			t.Errorf("default page size is 20: %v", def.dig("meta"))
		}
		if hm, _ := s.mustOK("big", s.do(tn.token, "GET", proj(Pb, "/activity?limit=1000"), nil)).dig("meta").(map[string]any); hm["limit"] != float64(100) {
			t.Errorf("limit is capped at 100: %v", hm)
		}
		// Filter by entity: the deletion and the restore of payment X are in its history.
		fl := s.mustOK("filter", s.do(tn.token, "GET", proj(Pb, fmt.Sprintf("/activity?entityType=client_payment&entityId=%d&limit=50", X)), nil))
		types := map[string]int{}
		for _, e := range rows(fl, "data") {
			if e["entityType"] != "client_payment" || e["entityId"] != fmt.Sprint(X) {
				t.Errorf("the filter let another entity through: %v", e)
			}
			types[fmt.Sprint(e["type"])]++
		}
		if types["payment_deleted"] < 2 || types["item_restored"] < 2 || types["payment_recorded"] != 1 {
			t.Errorf("the log records delete and restore: %v", types)
		}
		if vf := rows(s.mustOK("vendor filter", s.do(tn.token, "GET", proj(Pb, "/activity?entityType=project_vendor&limit=50"), nil)), "data"); len(vf) < 3 {
			t.Errorf("project_vendor entries: %d", len(vf))
		}
		// An actor that no longer resolves has an empty name; nothing is invented.
		if _, err := s.db.Exec(`INSERT INTO activity_log (project_id, type, actor_staff_id, entity_type, entity_id, entity_label, description) VALUES (?, 'project_updated', 987654321, 'project', ?, 'x', 'oleh staf yang sudah tidak ada')`, Pb, fmt.Sprint(Pb)); err != nil {
			t.Fatal(err)
		}
		ghost := rows(s.mustOK("ghost", s.do(tn.token, "GET", proj(Pb, "/activity?limit=1"), nil)), "data")
		if len(ghost) != 1 || ghost[0]["actorName"] != "" || ghost[0]["actorStaffId"] != float64(987654321) {
			t.Errorf("unresolved actor: %v", ghost)
		}
		// A client and another workspace.
		if r := s.do(bride, "GET", proj(Pb, "/activity"), nil); r.status != http.StatusForbidden {
			t.Errorf("a client: want 403, got %s", r)
		}
		if r := s.do(other.token, "GET", proj(Pb, "/activity"), nil); r.status != http.StatusNotFound || vacuous(r) {
			t.Errorf("another workspace: want 404, got %s", r)
		}
		// The dashboard feed names its actors too.
		d := s.mustOK("dashboard", s.do(tn.token, "GET", "/api/v1/dashboard", nil))
		for _, a := range rows(d, "data", "recentActivity") {
			if a["actorStaffId"] == float64(tn.staffID) && a["actorName"] != ownerName {
				t.Errorf("dashboard activity actorName: %v", a)
			}
		}
		routesCovered += 4
	})

	t.Run("(e2) the vendor tab: picName and evidenceCount, per caller", func(t *testing.T) {
		Pv := s.projectAt(tn, "Proyek T5 vendor", 30_000_000, 100, 0, 0)
		EV := s.engage(tn, Pv, vendorID, category, "Dekor tab", 5_000_000, "Booked", nil)
		MS := s.mustOK("ms", s.do(tn.token, "POST", proj(Pv, fmt.Sprintf("/vendors/%d/milestones", EV)), map[string]any{"name": "m", "description": "d", "targetDate": day(30), "picStaffId": tn.staffID})).id("data", "id")
		s.attach(tn, Pv, "vendorMilestone", MS, "Photo", "foto-ms-tab", false)
		s.attach(tn, Pv, "projectVendor", EV, "Booking Proof", "booking", false)
		VP := s.mustOK("vp", s.do(tn.token, "POST", proj(Pv, "/payments"), map[string]any{"projectVendorId": EV, "type": "DP", "amount": 1_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "TAB"})).id("data", "id")
		s.attach(tn, Pv, "payment", VP, "Transfer Proof", "bukti-tab", false)
		clientTok := s.addClient(tn, Pv, "Bride", "Pengantin", fmt.Sprintf("tab.klien.%d", time.Now().UnixNano()%1_000_000_000))
		staffRow := rowByID(rows(s.mustOK("vendors", s.do(tn.token, "GET", proj(Pv, "/vendors"), nil)), "data"), EV)
		if staffRow["picName"] != ownerName || staffRow["evidenceCount"] != float64(3) || staffRow["vendorName"] != "Vendor T5 "+tn.marker {
			t.Errorf("staff row: picName/evidenceCount/vendorName = %v/%v/%v", staffRow["picName"], staffRow["evidenceCount"], staffRow["vendorName"])
		}
		clientRow := rowByID(rows(s.mustOK("client vendors", s.do(clientTok, "GET", proj(Pv, "/vendors"), nil)), "data"), EV)
		if clientRow["picName"] != "" || clientRow["evidenceCount"] != float64(1) {
			t.Errorf("a client counts only what ClientCanSeeEvidence allows (a deliverable's photo with portal.vendor.view; booking and payment proofs are internal) and never sees the PIC's name: %v/%v", clientRow["picName"], clientRow["evidenceCount"])
		}
		if cr := rows(s.mustOK("client vendors", s.do(clientTok, "GET", proj(Pv, "/vendors"), nil)), "data"); len(cr) != 1 {
			t.Errorf("client vendor rows: %v", cr)
		}
		routesCovered++
	})

	t.Run("(f) every mutation answers with a fresh projectProgress", func(t *testing.T) {
		tf := s.signUp("t5f") // a workspace of its own (the trial allows 5 active projects)
		tn := tf
		vendorID, category := s.newVendor(tn, "Vendor T5 "+tn.marker)
		noDelete := s.staffWith(tn, "tanpa hapus f", "project.edit", "finance.record")
		ownerName := "Owner " + tn.marker
		Pf := s.projectAt(tn, "Proyek T5 progres", 60_000_000, 100, 2_000_000, 0)
		bride := s.addClient(tn, Pf, "Bride", "Pengantin", fmt.Sprintf("f.klien.%d", time.Now().UnixNano()%1_000_000_000))
		EV := s.engage(tn, Pf, vendorID, category, "Dekor progres", 9_000_000, "Booked", nil)
		// project milestones
		cm := s.do(tn.token, "POST", proj(Pf, "/milestones"), map[string]any{"name": "Baru", "targetDate": day(-9)})
		s.progressMatches(t, tn, Pf, "milestone create", cm)
		mid := cm.id("data", "id")
		if mid == 0 || cm.dig("data", "name") != "Baru" {
			t.Errorf("a milestone answer keeps its own fields: %s", cm)
		}
		um := s.do(tn.token, "PATCH", proj(Pf, fmt.Sprintf("/milestones/%d", mid)), map[string]any{"status": "Completed", "targetDate": day(-9), "completedDate": day(-1)})
		s.progressMatches(t, tn, Pf, "milestone update", um)
		list := rows(s.mustOK("ms", s.do(tn.token, "GET", proj(Pf, "/milestones"), nil)), "data")
		var order []int64
		for i := len(list) - 1; i >= 0; i-- {
			order = append(order, int64(list[i]["id"].(float64)))
		}
		s.progressMatches(t, tn, Pf, "milestone reorder", s.do(tn.token, "PATCH", proj(Pf, "/milestones"), map[string]any{"orderedIds": order}))
		// vendor milestones
		cvm := s.do(tn.token, "POST", proj(Pf, fmt.Sprintf("/vendors/%d/milestones", EV)), map[string]any{"name": "vm", "description": "d", "targetDate": day(-5), "picStaffId": tn.staffID})
		s.progressMatches(t, tn, Pf, "vendor milestone create", cvm)
		s.progressMatches(t, tn, Pf, "vendor milestone update", s.do(tn.token, "PATCH", proj(Pf, fmt.Sprintf("/vendors/%d/milestones/%d", EV, cvm.id("data", "id"))), map[string]any{"status": "Blocked", "targetDate": day(-5), "picStaffId": tn.staffID, "description": "d"}))
		// issues
		ci := s.do(tn.token, "POST", proj(Pf, "/issues"), map[string]any{"projectVendorId": EV, "title": "k", "description": "d", "impact": "Critical", "foundDate": day(-1), "picStaffId": tn.staffID})
		s.progressMatches(t, tn, Pf, "issue create", ci)
		s.progressMatches(t, tn, Pf, "issue update", s.do(tn.token, "PATCH", proj(Pf, fmt.Sprintf("/issues/%d", ci.id("data", "id"))), map[string]any{"projectVendorId": EV, "title": "k", "description": "d", "impact": "Critical", "foundDate": day(-1), "status": "Resolved", "picStaffId": tn.staffID}))
		// engagements
		ce := s.do(tn.token, "POST", proj(Pf, "/vendors"), map[string]any{"vendorId": vendorID, "scope": "Foto progres", "contractValue": 1_000_000, "pricingTier": "Akad", "engagementStatus": "Planned", "picStaffId": tn.staffID})
		s.progressMatches(t, tn, Pf, "engagement create", ce)
		eid := ce.id("data", "id")
		s.progressMatches(t, tn, Pf, "engagement update", s.do(tn.token, "PATCH", proj(Pf, fmt.Sprintf("/vendors/%d", eid)), map[string]any{"vendorId": vendorID, "categoryId": category, "scope": "Foto progres", "contractValue": 1_000_000, "pricingTier": "Akad", "engagementStatus": "Booked", "eventDate": day(100), "picStaffId": tn.staffID}))
		s.progressMatches(t, tn, Pf, "engagement cancel", s.do(tn.token, "POST", proj(Pf, fmt.Sprintf("/vendors/%d/cancel", eid)), map[string]any{}))
		s.progressMatches(t, tn, Pf, "engagement delete", s.do(tn.token, "DELETE", proj(Pf, fmt.Sprintf("/vendors/%d", eid)), nil))
		s.progressMatches(t, tn, Pf, "engagement restore", s.do(tn.token, "POST", proj(Pf, fmt.Sprintf("/vendors/%d/restore", eid)), nil))
		// the three payment kinds: create / update / delete / restore, each with ledgerSummary where it applies
		pay := func(kind, path string, create map[string]any, update map[string]any) {
			c := s.do(tn.token, "POST", proj(Pf, path), create)
			s.progressMatches(t, tn, Pf, kind+" create", c)
			if c.dig("data", "ledgerSummary") == nil {
				t.Errorf("%s create keeps its ledgerSummary: %s", kind, c)
			}
			id := c.id("data", "id")
			u := s.do(tn.token, "PATCH", proj(Pf, fmt.Sprintf("%s/%d", path, id)), update)
			s.progressMatches(t, tn, Pf, kind+" update", u)
			if u.dig("data", "ledgerSummary") == nil || num(u, "data", "id") != id {
				t.Errorf("%s update carries ledgerSummary next to its own fields: %s", kind, u)
			}
			d := s.do(tn.token, "DELETE", proj(Pf, fmt.Sprintf("%s/%d", path, id)), nil)
			s.progressMatches(t, tn, Pf, kind+" delete", d)
			if d.dig("data", "ledgerSummary") == nil || num(d, "data", "id") != id {
				t.Errorf("%s delete carries ledgerSummary: %s", kind, d)
			}
			r := s.do(tn.token, "POST", proj(Pf, fmt.Sprintf("%s/%d/restore", path, id)), nil)
			s.progressMatches(t, tn, Pf, kind+" restore", r)
			if r.dig("data", "ledgerSummary") == nil {
				t.Errorf("%s restore carries ledgerSummary: %s", kind, r)
			}
		}
		pay("client payment", "/client-payments",
			map[string]any{"type": "DP", "amount": 1_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "F-C"},
			map[string]any{"type": "DP", "amount": 2_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "F-C"})
		pay("vendor payment", "/payments",
			map[string]any{"projectVendorId": EV, "type": "DP", "amount": 1_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "F-V"},
			map[string]any{"type": "DP", "amount": 2_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "F-V"})
		pay("venue payment", "/venue-payments",
			map[string]any{"type": "DP", "amount": 500_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "F-N"},
			map[string]any{"type": "DP", "amount": 600_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "F-N"})
		// evidence: upload, toggle, delete (incomplete proof changes the progress of a vendor payment)
		vpay := s.mustOK("vp", s.do(tn.token, "POST", proj(Pf, "/payments"), map[string]any{"projectVendorId": EV, "type": "DP", "amount": 100_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "F-V2"})).id("data", "id")
		up := s.do(tn.token, "POST", proj(Pf, "/evidence"), map[string]any{
			"name": "bukti", "type": "Transfer Proof", "fileName": "b.pdf", "mimeType": "application/pdf", "base64Data": pdf, "relatedKind": "payment", "relatedId": vpay,
		})
		s.progressMatches(t, tn, Pf, "evidence upload", up)
		if up.dig("data", "uploadedByName") != ownerName || up.id("data", "id") == 0 {
			t.Errorf("an evidence answer keeps its fields and names the uploader: %s", up)
		}
		msUp := s.do(tn.token, "POST", proj(Pf, "/evidence"), map[string]any{
			"name": "foto", "type": "Photo", "fileName": "f.pdf", "mimeType": "application/pdf", "base64Data": pdf, "relatedKind": "projectMilestone", "relatedId": mid,
		})
		s.mustOK("milestone photo", msUp)
		tg := s.do(tn.token, "POST", proj(Pf, fmt.Sprintf("/evidence/%d/toggle-client-visible", msUp.id("data", "id"))), nil)
		s.progressMatches(t, tn, Pf, "evidence toggle", tg)
		if tg.dig("data", "isClientVisible") != true {
			t.Errorf("toggle: %s", tg)
		}
		ed := s.do(tn.token, "DELETE", proj(Pf, fmt.Sprintf("/evidence/%d", up.id("data", "id"))), nil)
		s.progressMatches(t, tn, Pf, "evidence delete", ed)
		if num(ed, "data", "id") != up.id("data", "id") || s.count("SELECT COUNT(*) FROM evidence WHERE id = ?", up.id("data", "id")) != 0 {
			t.Errorf("evidence delete: %s", ed)
		}
		if r := s.do(noDelete, "DELETE", proj(Pf, fmt.Sprintf("/evidence/%d", s.attach(tn, Pf, "payment", vpay, "Invoice", "inv", false))), nil); r.status != http.StatusForbidden {
			t.Errorf("deleting a payment's proof needs finance.payment.delete: want 403, got %s", r)
		}
		if r := s.do(bride, "DELETE", proj(Pf, "/evidence/1"), nil); r.status != http.StatusForbidden {
			t.Errorf("a client cannot delete evidence: %s", r)
		}
	})

	t.Run("(g) dashboard: cashFlow, vendorPaymentsDue, unconfirmedVendors, gated by revenue permission", func(t *testing.T) {
		tg := s.signUp("t5g") // a workspace of its own, so every number is exact
		vid, cat := s.newVendor(tg, "Vendor T5G "+tg.marker)
		P1 := s.projectAt(tg, "Proyek T5G satu", 60_000_000, 10, 0, 0)
		P2 := s.projectAt(tg, "Proyek T5G jauh", 60_000_000, 90, 0, 0)
		s.setTerms(tg, P1, term("DP", 20_000_000, -1), term("Pelunasan", 40_000_000, 9))
		s.clientPay(tg, P1, "DP", 5_000_000, "G-1")
		five, twenty := 5, 20
		planned := s.engage(tg, P1, vid, cat, "Dekor planned", 10_000_000, "Planned", &five)
		booked := s.engage(tg, P1, vid, cat, "Foto booked", 8_000_000, "Booked", &twenty)
		s.mustOK("vp", s.do(tg.token, "POST", proj(P1, "/payments"), map[string]any{"projectVendorId": booked, "type": "DP", "amount": 3_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "G-V"}))
		gone := s.engage(tg, P1, vid, cat, "Batal", 99_000_000, "Booked", &five)
		s.mustOK("cancel", s.do(tg.token, "POST", proj(P1, fmt.Sprintf("/vendors/%d/cancel", gone)), map[string]any{}))
		s.engage(tg, P2, vid, cat, "Jauh planned", 4_000_000, "Planned", &twenty)

		r := s.mustOK("dashboard", s.do(tg.token, "GET", "/api/v1/dashboard", nil))
		routesCovered++
		weeks := rows(r, "data", "cashFlow")
		if len(weeks) != 8 {
			t.Fatalf("cashFlow has 8 weeks, got %d: %s", len(weeks), r)
		}
		var in, out float64
		var prev time.Time
		for i, w := range weeks {
			ws, err := time.Parse("2006-01-02", fmt.Sprint(w["weekStart"]))
			if err != nil || ws.Weekday() != time.Monday {
				t.Errorf("week %d weekStart %v must be a Monday (YYYY-MM-DD)", i+1, w["weekStart"])
			}
			if i > 0 && !ws.Equal(prev.AddDate(0, 0, 7)) {
				t.Errorf("weeks are consecutive: %v after %v", ws, prev)
			}
			prev = ws
			in += w["expectedIn"].(float64)
			out += w["expectedOut"].(float64)
		}
		if in != 15_000_000+40_000_000 || out != 10_000_000+5_000_000+4_000_000 {
			t.Errorf("cash flow totals: in %v (want 55M: 15M still owed on the overdue DP + 40M), out %v (want 19M: 10M + 5M left on the booked vendor + 4M of the other project; a cancelled one owes nothing)", in, out)
		}
		if weeks[0]["expectedIn"].(float64) < 15_000_000 {
			t.Errorf("an overdue installment counts in week 1: %v", weeks[0])
		}
		due := rows(r, "data", "vendorPaymentsDue")
		if len(due) != 3 || due[0]["vendorName"] != "Vendor T5G "+tg.marker || due[0]["remaining"] != float64(10_000_000) || due[0]["dueDate"] != day(5) ||
			due[1]["remaining"] != float64(5_000_000) || due[1]["dueDate"] != day(20) || due[2]["remaining"] != float64(4_000_000) || due[0]["projectName"] != "Proyek T5G satu" {
			t.Errorf("vendorPaymentsDue (nearest first, remaining = contract - paid): %v", due)
		}
		for _, k := range []string{"projectId", "projectName", "vendorName", "dueDate", "remaining"} {
			if due[0][k] == nil {
				t.Errorf("vendorPaymentsDue.%s missing", k)
			}
		}
		un := rows(r, "data", "unconfirmedVendors")
		if len(un) != 1 || un[0]["engagementStatus"] != "Planned" || int64(un[0]["projectId"].(float64)) != P1 || un[0]["eventDate"] != day(10) || un[0]["vendorName"] != "Vendor T5G "+tg.marker || un[0]["projectName"] != "Proyek T5G satu" {
			t.Errorf("unconfirmedVendors (event within 30 days, still Planned/Negotiation; the far project is out): %v", un)
		}
		// A deleted engagement leaves all three.
		s.mustOK("delete planned", s.do(tg.token, "DELETE", proj(P1, fmt.Sprintf("/vendors/%d", planned)), nil))
		r2 := s.mustOK("dashboard", s.do(tg.token, "GET", "/api/v1/dashboard", nil))
		if len(rows(r2, "data", "unconfirmedVendors")) != 0 || len(rows(r2, "data", "vendorPaymentsDue")) != 2 {
			t.Errorf("a deleted engagement is not owed, not unconfirmed: %s", r2)
		}
		var out2 float64
		for _, w := range rows(r2, "data", "cashFlow") {
			out2 += w["expectedOut"].(float64)
		}
		if out2 != 9_000_000 {
			t.Errorf("cash flow out after the deletion: %v", out2)
		}
		// Permission: money sections need dashboard.revenue.view; unconfirmed vendors do not.
		tgNoRev := s.staffWith(tg, "tanpa omzet g", "dashboard.view")
		tgRev := s.staffWith(tg, "dengan omzet g", "dashboard.view", "dashboard.revenue.view")
		nr := s.mustOK("dashboard without revenue", s.do(tgNoRev, "GET", "/api/v1/dashboard", nil))
		data, _ := nr.json()["data"].(map[string]any)
		for _, k := range []string{"cashFlow", "vendorPaymentsDue"} {
			if _, present := data[k]; present {
				t.Errorf("%s must be absent without dashboard.revenue.view", k)
			}
		}
		if _, present := data["unconfirmedVendors"]; !present || data["revenue"].(map[string]any)["total"] != float64(0) {
			t.Errorf("unconfirmedVendors stays, revenue is zeroed: %v", data)
		}
		wr := s.mustOK("dashboard with revenue", s.do(tgRev, "GET", "/api/v1/dashboard", nil))
		if len(rows(wr, "data", "cashFlow")) != 8 || wr.dig("data", "vendorPaymentsDue") == nil {
			t.Errorf("with dashboard.revenue.view both money sections are there: %s", wr)
		}
		routesCovered += 2
	})
}

func contains2(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
