//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Tahap 3 (E4, E5, E6): the Client Portal's home, payment proof and the names
// inside already-filtered responses. These scenarios build their own rows in
// workspace B (a second payment, a completed step, attachments, a second
// vendor) so they do not depend on, or disturb, what the attack matrix uses.

// portal is what these scenarios add to the world.
type portal struct {
	brideToken, familyToken, aClientToken string
	Milestone                             int64 // completed step with two attachments
	EvUnflagged, EvFlagged                int64
	Payment, PaymentNoProof               int64
	PaymentProof                          int64
	OtherVendor                           int64 // exists in B's directory (engaged on another project), never on the client's
	VendorIssue                           int64 // issue tied to a vendor milestone
	ownerName                             string
}

func (s *stack) projectsPath(id int64, rest string) string {
	return fmt.Sprintf("/api/v1/projects/%d%s", id, rest)
}

// attach uploads a file to one item of a project and returns its id.
func (s *stack) attach(t tenant, projectID int64, kind string, relatedID int64, typ, name string, visible bool) int64 {
	s.t.Helper()
	r := s.mustOK("attach "+name, s.do(t.token, "POST", s.projectsPath(projectID, "/evidence"), map[string]any{
		"name": name, "type": typ, "fileName": name + ".pdf", "mimeType": "application/pdf", "base64Data": pdf,
		"documentDate": day(-1), "description": "Rahasia " + t.marker, "relatedKind": kind, "relatedId": relatedID, "isClientVisible": visible,
	}))
	return r.id("data", "id")
}

// addClient adds a client with the named portal role to a project and returns
// its signed-in token.
func (s *stack) addClient(t tenant, projectID int64, clientRole, portalRoleName, username string) string {
	s.t.Helper()
	var roleID int64
	roles := s.mustOK("client roles", s.do(t.token, "GET", "/api/v1/access/roles?kind=client", nil))
	if list, ok := roles.dig("data").([]any); ok {
		for _, item := range list {
			if m, ok := item.(map[string]any); ok && m["name"] == portalRoleName {
				roleID = int64(m["id"].(float64))
			}
		}
	}
	if roleID == 0 {
		s.t.Fatalf("no client role %q: %s", portalRoleName, roles)
	}
	cl := s.mustOK("client "+username, s.do(t.token, "POST", "/api/v1/clients", map[string]any{
		"projectId": projectID, "role": clientRole, "name": "Klien " + username + " " + t.marker, "phone": "0812" + fmt.Sprint(time.Now().UnixNano()%100000000),
		"username": username, "email": "", "accessRoleId": roleID,
	}))
	link, _ := cl.dig("data", "accessLink", "token").(string)
	sess := s.mustOK("client password", s.do("", "POST", "/api/v1/auth/links/"+link+"/password", map[string]any{"password": "Kata-sandi-klien-123"}))
	tok, _ := sess.dig("data", "accessToken").(string)
	if tok == "" {
		s.t.Fatalf("no token for client %s: %s", username, sess)
	}
	return tok
}

func rows(r reply, keys ...string) []map[string]any {
	var out []map[string]any
	if list, ok := r.dig(keys...).([]any); ok {
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func rowByID(list []map[string]any, id int64) map[string]any {
	for _, m := range list {
		if f, ok := m["id"].(float64); ok && int64(f) == id {
			return m
		}
	}
	return nil
}

func evidenceIDs(m map[string]any) []int64 {
	var ids []int64
	if list, ok := m["evidence"].([]any); ok {
		for _, e := range list {
			if em, ok := e.(map[string]any); ok {
				ids = append(ids, int64(em["id"].(float64)))
			}
		}
	}
	return ids
}

func (w *world) buildPortal(t *testing.T, s *stack) *portal {
	t.Helper()
	b := w.b
	p := &portal{ownerName: "Owner " + b.marker}
	P := w.Project

	// B's bank account, so the home can say where to pay.
	s.mustOK("business details", s.do(b.token, "PUT", "/api/v1/tenants/me/business-details", map[string]any{
		"address": "Jl. Mawar " + b.marker, "bankName": "BCA", "bankAccountNumber": "1234567890", "bankAccountHolder": "WO " + b.marker,
	}))

	// A schedule with a second installment, so there is an open one to show.
	s.mustOK("payment terms", s.do(b.token, "PUT", s.projectsPath(P, "/payment-terms"), map[string]any{
		"terms": []map[string]any{
			{"label": "Termin " + b.marker, "amount": 1000000, "dueDate": day(30)},
			{"label": "Pelunasan " + b.marker, "amount": 500000000, "dueDate": day(60)},
		},
	}))

	// A completed step with one internal and one shared attachment.
	p.Milestone = s.mustOK("milestone", s.do(b.token, "POST", s.projectsPath(P, "/milestones"), map[string]any{
		"name": "Dekorasi " + b.marker, "targetDate": day(20),
	})).id("data", "id")
	s.mustOK("complete milestone", s.do(b.token, "PATCH", s.projectsPath(P, fmt.Sprintf("/milestones/%d", p.Milestone)), map[string]any{
		"status": "Completed", "targetDate": day(20), "completedDate": day(-1),
	}))
	p.EvUnflagged = s.attach(b, P, "projectMilestone", p.Milestone, "Photo", "foto-internal", false)

	// A payment with a proof file, and one without.
	pay := func(ref string) int64 {
		return s.mustOK("payment "+ref, s.do(b.token, "POST", s.projectsPath(P, "/client-payments"), map[string]any{
			"type": "Termin", "amount": 100000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": ref + b.marker, "notes": "Catatan rahasia " + b.marker,
		})).id("data", "id")
	}
	p.Payment, p.PaymentNoProof = pay("PAY-"), pay("NOPROOF-")
	p.PaymentProof = s.attach(b, P, "clientPayment", p.Payment, "Transfer Proof", "bukti-transfer", false)

	// Vendors: B's engaged vendor already exists; another one sits in the
	// directory (engaged on another project) and must never be named to this client.
	p.OtherVendor = s.mustOK("other vendor", s.do(b.token, "POST", "/api/v1/vendors", vendorBody(s, b, "Vendor LAIN "+b.marker, w.Category))).id("data", "id")
	s.mustOK("engage other vendor on project 2", s.do(b.token, "POST", s.projectsPath(w.Project2, "/vendors"), map[string]any{
		"vendorId": p.OtherVendor, "categoryId": w.Category, "scope": "Katering LAIN", "contractValue": 1000, "pricingTier": "Akad",
		"engagementStatus": "Booked", "eventDate": day(120), "picStaffId": b.staffID,
	}))
	p.VendorIssue = s.mustOK("issue with milestone", s.do(b.token, "POST", s.projectsPath(P, "/issues"), map[string]any{
		"projectVendorId": w.Engagement, "vendorMilestoneId": w.VendorMilestone, "title": "Kendala tahapan " + b.marker, "description": "d",
		"impact": "Low", "foundDate": day(-1), "picStaffId": b.staffID, "targetResolutionDate": day(5),
	})).id("data", "id")

	// Clients: B's bride is w.clientToken; add a Keluarga (no finance) client of B
	// and a client of workspace A.
	p.brideToken = w.clientToken
	p.familyToken = s.addClient(b, P, "Family Representative", "Keluarga", fmt.Sprintf("keluarga.b.%d", time.Now().UnixNano()%1_000_000_000))
	p.aClientToken = s.addClient(w.a, w.aProject, "Bride", "Pengantin", fmt.Sprintf("klien.a.portal.%d", time.Now().UnixNano()%1_000_000_000))
	return p
}

func (w *world) portalScenarios(t *testing.T, s *stack) {
	p := w.buildPortal(t, s)
	P := w.Project
	home := "/api/v1/projects/me/home"

	t.Run("a: tenant A's client cannot reach tenant B, and the vendor directories are closed", func(t *testing.T) {
		for _, path := range []string{
			s.projectsPath(P, ""), s.projectsPath(P, "/vendors"), s.projectsPath(P, "/issues"), s.projectsPath(P, "/milestones"),
			s.projectsPath(P, "/client-payments"), s.projectsPath(w.Project2, "/vendors"),
		} {
			r := s.do(p.aClientToken, "GET", path, nil)
			if r.status != http.StatusForbidden && r.status != http.StatusNotFound {
				t.Errorf("A's client GET %s: want 403/404, got %s", path, r)
			}
			w.checkNoLeak(t, "A's client GET "+path, r)
		}
		// Its own home is its own project's, never B's.
		r := s.mustOK("A client home", s.do(p.aClientToken, "GET", home, nil))
		if r.id("data", "project", "id") != w.aProject {
			t.Errorf("A's client home is not A's project: %s", r)
		}
		w.checkNoLeak(t, "A's client home", r)
		// Closed for every client, including B's own.
		for name, tok := range map[string]string{"A's client": p.aClientToken, "B's bride": p.brideToken, "B's family": p.familyToken} {
			for _, path := range []string{"/api/v1/vendors/summary", "/api/v1/vendor-categories", "/api/v1/vendor-categories?all=true"} {
				if r := s.do(tok, "GET", path, nil); r.status != http.StatusForbidden {
					t.Errorf("%s GET %s: want 403, got %s", name, path, r)
				}
			}
		}
		// The home is for client principals only.
		for name, tok := range map[string]string{"A's owner": w.a.token, "B's owner": w.b.token} {
			if r := s.do(tok, "GET", home, nil); r.status != http.StatusForbidden {
				t.Errorf("%s GET projects/me/home: want 403, got %s", name, r)
			}
		}
		if r := s.do("", "GET", home, nil); r.status != http.StatusUnauthorized {
			t.Errorf("no token: want 401, got %s", r)
		}
	})

	t.Run("b: a client sees vendor names of its own project only", func(t *testing.T) {
		r := s.mustOK("vendors", s.do(p.brideToken, "GET", s.projectsPath(P, "/vendors"), nil))
		eng := rowByID(rows(r, "data"), w.Engagement)
		if eng == nil || eng["vendorName"] != "Vendor B "+w.b.marker || eng["categoryName"] != "Kategori "+w.b.marker {
			t.Fatalf("own engagement should carry vendorName and categoryName: %s", r)
		}
		if body := string(r.body); strings.Contains(body, "LAIN") || strings.Contains(body, w.a.marker) {
			t.Errorf("another vendor's name reached the client: %s", r)
		}
		if eng["contractValue"].(float64) != 0 || eng["dpAmount"].(float64) != 0 || eng["paidAmount"].(float64) != 0 {
			t.Errorf("vendor money must stay redacted: %v", eng)
		}
		// The issues list names the vendor and the step, from the same project.
		ir := s.mustOK("issues", s.do(p.brideToken, "GET", s.projectsPath(P, "/issues"), nil))
		issue := rowByID(rows(ir, "data"), p.VendorIssue)
		if issue == nil || issue["vendorName"] != "Vendor B "+w.b.marker || issue["vendorMilestoneName"] != "Milestone vendor "+w.b.marker {
			t.Errorf("issue should carry vendorName and vendorMilestoneName: %s", ir)
		}
		if body := string(ir.body); strings.Contains(body, "LAIN") {
			t.Errorf("another vendor's name reached the client through issues: %s", ir)
		}
		// An issue with no milestone has an empty step name.
		if general := rowByID(rows(ir, "data"), w.Issue); general == nil || general["vendorMilestoneName"] != "" {
			t.Errorf("a general issue has no milestone name: %v", general)
		}
		// Staff get the same names.
		sr := s.mustOK("staff vendors", s.do(w.b.token, "GET", s.projectsPath(P, "/vendors"), nil))
		if e := rowByID(rows(sr, "data"), w.Engagement); e == nil || e["vendorName"] != "Vendor B "+w.b.marker {
			t.Errorf("staff view lost vendorName: %s", sr)
		}
		// A's staff asking for B's project learns nothing.
		if r := s.do(w.a.token, "GET", s.projectsPath(P, "/vendors"), nil); r.status != http.StatusNotFound && r.status != http.StatusForbidden {
			t.Errorf("A's staff GET B's vendors: %s", r)
		}
	})

	t.Run("c: a step's attachment is invisible to the client until flagged", func(t *testing.T) {
		file := s.projectsPath(P, fmt.Sprintf("/evidence/%d/file", p.EvUnflagged))
		link := s.projectsPath(P, fmt.Sprintf("/evidence/%d/link", p.EvUnflagged))
		hidden := func(when string) {
			t.Helper()
			ms := s.mustOK("milestones", s.do(p.brideToken, "GET", s.projectsPath(P, "/milestones"), nil))
			if m := rowByID(rows(ms, "data"), p.Milestone); m == nil || len(evidenceIDs(m)) != 0 {
				t.Errorf("%s: client sees the attachment in the step: %s", when, ms)
			}
			if r := s.mustOK("evidence list", s.do(p.brideToken, "GET", s.projectsPath(P, "/evidence"), nil)); strings.Contains(string(r.body), "foto-internal") {
				t.Errorf("%s: client sees the attachment in the evidence list: %s", when, r)
			}
			for _, path := range []string{file, link} {
				if r := s.do(p.brideToken, "GET", path, nil); r.status != http.StatusNotFound && r.status != http.StatusForbidden {
					t.Errorf("%s: client GET %s: want 404/403, got %s", when, path, r)
				}
			}
			h := s.mustOK("home", s.do(p.brideToken, "GET", home, nil))
			for _, u := range rows(h, "data", "recentUpdates") {
				if u["kind"] == "milestone_done" && u["proof"] != nil {
					t.Errorf("%s: the home shows proof of an unflagged attachment: %v", when, u)
				}
			}
		}
		hidden("before flagging")

		// Staff see it in the step, and can flag it with the existing toggle.
		sm := s.mustOK("staff milestones", s.do(w.b.token, "GET", s.projectsPath(P, "/milestones"), nil))
		if m := rowByID(rows(sm, "data"), p.Milestone); m == nil || len(evidenceIDs(m)) != 1 {
			t.Fatalf("staff should see the step's attachment: %s", sm)
		}
		tg := s.mustOK("toggle", s.do(w.b.token, "POST", s.projectsPath(P, fmt.Sprintf("/evidence/%d/toggle-client-visible", p.EvUnflagged)), nil))
		if tg.dig("data", "isClientVisible") != true {
			t.Fatalf("toggle did not flag the attachment: %s", tg)
		}
		ms := s.mustOK("milestones after", s.do(p.brideToken, "GET", s.projectsPath(P, "/milestones"), nil))
		if m := rowByID(rows(ms, "data"), p.Milestone); m == nil || len(evidenceIDs(m)) != 1 || evidenceIDs(m)[0] != p.EvUnflagged {
			t.Errorf("after flagging the client should see exactly that attachment: %s", ms)
		}
		if r := s.do(p.brideToken, "GET", file, nil); r.status != http.StatusOK {
			t.Errorf("flagged attachment file: want 200, got %s", r)
		}
		if r := s.do(p.brideToken, "GET", link, nil); r.status != http.StatusOK && r.status != http.StatusNotImplemented {
			t.Errorf("flagged attachment link: want 200 or 501, got %s", r)
		}
		h := s.mustOK("home", s.do(p.brideToken, "GET", home, nil))
		found := false
		for _, u := range rows(h, "data", "recentUpdates") {
			if u["kind"] == "milestone_done" {
				proof, _ := u["proof"].(map[string]any)
				found = proof != nil && int64(proof["evidenceId"].(float64)) == p.EvUnflagged && proof["fileName"] == "foto-internal.pdf"
			}
		}
		if !found {
			t.Errorf("the home should show the flagged attachment as proof of the finished step: %s", h)
		}
		// Other tenants' staff cannot flag it.
		if r := s.do(w.a.token, "POST", s.projectsPath(P, fmt.Sprintf("/evidence/%d/toggle-client-visible", p.EvUnflagged)), nil); r.status != http.StatusNotFound && r.status != http.StatusForbidden {
			t.Errorf("A's staff toggling B's attachment: %s", r)
		}
		// Unflag again: invisible again, files closed again.
		s.mustOK("untoggle", s.do(w.b.token, "POST", s.projectsPath(P, fmt.Sprintf("/evidence/%d/toggle-client-visible", p.EvUnflagged)), nil))
		hidden("after unflagging")
	})

	t.Run("f: verifying a payment", func(t *testing.T) {
		verify := func(tok string, id int64) reply {
			return s.do(tok, "POST", s.projectsPath(P, fmt.Sprintf("/client-payments/%d/verify", id)), nil)
		}
		// Clients (any role) are read-only; another workspace's staff finds nothing.
		for name, tok := range map[string]string{"bride": p.brideToken, "family": p.familyToken, "A's client": p.aClientToken} {
			if r := verify(tok, p.Payment); r.status != http.StatusForbidden {
				t.Errorf("%s verifying: want 403, got %s", name, r)
			}
		}
		if r := verify(w.a.token, p.Payment); r.status != http.StatusNotFound && r.status != http.StatusForbidden {
			t.Errorf("A's staff verifying B's payment: want 404, got %s", r)
		}
		if r := verify("", p.Payment); r.status != http.StatusUnauthorized {
			t.Errorf("no token: want 401, got %s", r)
		}

		// No proof yet: 422 on field `evidence`, in Indonesian, nothing stored.
		r := verify(w.b.token, p.PaymentNoProof)
		if r.status != http.StatusUnprocessableEntity || r.dig("message") != "Lampirkan bukti transfer sebelum memverifikasi pembayaran." || r.dig("errors", "evidence") == nil {
			t.Errorf("verify without proof: want 422 with errors.evidence, got %s", r)
		}
		list := rows(s.mustOK("payments", s.do(w.b.token, "GET", s.projectsPath(P, "/client-payments"), nil)), "data")
		if row := rowByID(list, p.PaymentNoProof); row == nil || row["verifiedAt"] != nil {
			t.Errorf("a refused verification must store nothing: %v", row)
		}

		// With proof: verified by the owner; a second call changes nothing.
		first := s.mustOK("verify", verify(w.b.token, p.Payment))
		at, _ := first.dig("data", "verifiedAt").(string)
		if at == "" || first.dig("data", "verifiedBy", "name") != p.ownerName || first.dig("data", "recordedBy", "name") != p.ownerName {
			t.Fatalf("verify response: %s", first)
		}
		if ids := evidenceIDs(first.json()["data"].(map[string]any)); len(ids) != 1 || ids[0] != p.PaymentProof {
			t.Errorf("verify response should carry the proof: %s", first)
		}
		time.Sleep(1100 * time.Millisecond)
		again := s.mustOK("verify again", verify(w.b.token, p.Payment))
		if again.dig("data", "verifiedAt") != at || again.dig("data", "verifiedBy", "name") != p.ownerName {
			t.Errorf("second verification must not change who/when: %s vs %s", first, again)
		}

		// Editing what was paid clears it; a note-only edit does not.
		edit := func(amount int, note string) reply {
			return s.do(w.b.token, "PATCH", s.projectsPath(P, fmt.Sprintf("/client-payments/%d", p.Payment)), map[string]any{
				"type": "Termin", "amount": amount, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "PAY-" + w.b.marker, "notes": note,
			})
		}
		if r := s.mustOK("note edit", edit(100000, "catatan baru")); r.dig("data", "verifiedAt") != at {
			t.Errorf("a note-only edit must keep the verification: %s", r)
		}
		cleared := s.mustOK("amount edit", edit(110000, "catatan baru"))
		if cleared.dig("data", "verifiedAt") != nil || cleared.dig("data", "verifiedBy") != nil {
			t.Errorf("changing the amount must clear the verification: %s", cleared)
		}
		// Verify again for the checks below (proof is still attached).
		s.mustOK("re-verify", verify(w.b.token, p.Payment))
	})

	t.Run("e: recordedBy never reaches a client", func(t *testing.T) {
		staffList := rows(s.mustOK("staff payments", s.do(w.b.token, "GET", s.projectsPath(P, "/client-payments"), nil)), "data")
		if row := rowByID(staffList, p.Payment); row == nil || row["recordedBy"] == nil {
			t.Fatalf("staff should see who recorded the payment: %v", row)
		}
		for name, tok := range map[string]string{"bride": p.brideToken} {
			r := s.mustOK(name+" payments", s.do(tok, "GET", s.projectsPath(P, "/client-payments"), nil))
			list := rows(r, "data")
			if len(list) == 0 {
				t.Fatalf("%s sees no payments (the check would be vacuous): %s", name, r)
			}
			for _, row := range list {
				if row["recordedBy"] != nil {
					t.Errorf("%s sees recordedBy: %v", name, row)
				}
				if row["notes"] != "" {
					t.Errorf("%s sees staff notes: %v", name, row)
				}
			}
			row := rowByID(list, p.Payment)
			if row == nil || row["verifiedAt"] == nil || row["verifiedBy"] == nil || len(evidenceIDs(row)) != 1 {
				t.Errorf("%s should see the verification and the proof: %v", name, row)
			}
			if h := s.mustOK("home", s.do(tok, "GET", home, nil)); strings.Contains(string(h.body), "recordedBy") || strings.Contains(string(h.body), "Catatan rahasia") {
				t.Errorf("%s: the home leaks internal fields: %s", name, h)
			}
		}
	})

	t.Run("d: a role without finance gets no finance, payTo or payment proof", func(t *testing.T) {
		bride := s.mustOK("bride home", s.do(p.brideToken, "GET", home, nil))
		fin, _ := bride.dig("data", "finance").(map[string]any)
		if fin == nil || fin["contractValue"].(float64) != float64(w.bContract) {
			t.Fatalf("the bride's home should carry finance: %s", bride)
		}
		// 4,444,444 + 100,000 (the edited payment is now 110,000) + 100,000 (no-proof payment).
		if fin["paid"].(float64) != 4444444+110000+100000 || fin["remaining"].(float64) != float64(w.bContract)-(4444444+110000+100000) {
			t.Errorf("finance totals: %v", fin)
		}
		// The first installment (1,000,000) is covered; the payments left over eat into the second.
		applied := float64(4444444 + 110000 + 100000)
		nt, _ := fin["nextTerm"].(map[string]any)
		if nt == nil || nt["label"] != "Pelunasan "+w.b.marker || nt["status"] != "partial" || nt["dueDate"] != day(60) ||
			nt["amount"].(float64) != 500000000 || nt["remaining"].(float64) != 500000000-(applied-1000000) {
			t.Errorf("nextTerm should be the open second installment: %v", fin)
		}
		if fin["overdueCount"].(float64) != 0 {
			t.Errorf("nothing is overdue: %v", fin)
		}
		payTo, _ := bride.dig("data", "payTo").(map[string]any)
		if payTo == nil || payTo["bankName"] != "BCA" || payTo["accountNumber"] != "1234567890" {
			t.Errorf("the bride's home should say where to pay: %s", bride)
		}
		var verified map[string]any
		for _, u := range rows(bride, "data", "recentUpdates") {
			if u["kind"] == "payment_verified" {
				verified = u
			}
		}
		if verified == nil || verified["verifiedBy"] != p.ownerName || verified["title"] != "Pembayaran termin diterima" {
			t.Fatalf("the bride's news should include the verified payment: %s", bride)
		}
		if proof, _ := verified["proof"].(map[string]any); proof == nil || int64(proof["evidenceId"].(float64)) != p.PaymentProof {
			t.Errorf("verified payment news should carry its proof: %v", verified)
		}

		family := s.mustOK("family home", s.do(p.familyToken, "GET", home, nil))
		for _, key := range []string{"finance", "payTo"} {
			if _, present := family.json()["data"].(map[string]any)[key]; present {
				t.Errorf("Keluarga must get no %q: %s", key, family)
			}
		}
		body := string(family.body)
		for _, secret := range []string{"1234567890", "BCA", "payment_verified", "contractValue", "bukti-transfer", fmt.Sprint(w.bContract)} {
			if strings.Contains(body, secret) {
				t.Errorf("Keluarga's home leaks %q: %s", secret, family)
			}
		}
		// The other sections are there (and computed) for them.
		if family.dig("data", "progress") == nil || family.dig("data", "openIssueCount") == nil {
			t.Errorf("Keluarga keeps progress and issues: %s", family)
		}
		// Payments and their proof are closed.
		if r := s.do(p.familyToken, "GET", s.projectsPath(P, "/client-payments"), nil); r.status != http.StatusForbidden {
			t.Errorf("Keluarga GET client-payments: want 403, got %s", r)
		}
		proofFile := s.projectsPath(P, fmt.Sprintf("/evidence/%d/file", p.PaymentProof))
		if r := s.do(p.familyToken, "GET", proofFile, nil); r.status != http.StatusNotFound {
			t.Errorf("Keluarga GET payment proof: want 404, got %s", r)
		}
		if r := s.mustOK("family evidence", s.do(p.familyToken, "GET", s.projectsPath(P, "/evidence"), nil)); strings.Contains(string(r.body), "bukti-transfer") {
			t.Errorf("Keluarga's evidence list shows the payment proof: %s", r)
		}
		if r := s.do(p.brideToken, "GET", proofFile, nil); r.status != http.StatusOK {
			t.Errorf("the bride may open her own payment proof: %s", r)
		}
	})

	t.Run("home: shape, countdown, actions and the no-leak list", func(t *testing.T) {
		h := s.mustOK("home", s.do(p.brideToken, "GET", home, nil))
		d := h.json()["data"].(map[string]any)
		proj := d["project"].(map[string]any)
		if int64(proj["id"].(float64)) != P || proj["eventDate"] != day(120) {
			t.Errorf("project block: %v", proj)
		}
		if cd := d["countdown"].(map[string]any)["daysToEvent"].(float64); cd < 119 || cd > 120 {
			t.Errorf("daysToEvent should be about 120, got %v", cd)
		}
		if d["pic"] == nil || d["actions"] == nil || d["recentUpdates"] == nil {
			t.Errorf("pic/actions/recentUpdates are always present: %s", h)
		}
		if n := len(d["recentUpdates"].([]any)); n == 0 || n > 5 {
			t.Errorf("recentUpdates: 1..5 expected, got %d", n)
		}
		if n := len(d["actions"].([]any)); n > 3 {
			t.Errorf("at most 3 actions, got %d", n)
		}
		if oc, ok := d["openIssueCount"].(float64); !ok || oc < 2 {
			t.Errorf("openIssueCount should count B's open issues (>=2): %s", h)
		}
		body := string(h.body)
		// The home is a client view: none of the WO's internals.
		for _, leak := range []string{"picSalesStaffId", "prepStartDate", "venueRentalPrice", "venueCharge", "recordedBy", "staffId", "Vendor B"} {
			if strings.Contains(body, leak) {
				t.Errorf("the home leaks %q: %s", leak, h)
			}
		}
		if _, has := proj["contractValue"]; has {
			t.Errorf("contractValue must not be in the project block: %v", proj)
		}
		for _, note := range []string{"Rahasia ", "Catatan rahasia"} {
			if strings.Contains(body, note) {
				t.Errorf("internal notes leaked (%q): %s", note, h)
			}
		}
		// A's data is not in B's client's home.
		if strings.Contains(body, w.a.marker) {
			t.Errorf("A's data leaked into B's client home: %s", h)
		}
	})
}
