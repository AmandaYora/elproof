//go:build integration

package isolation

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Tahap 4 (E3, E7): signed uploads, a payment recorded together with its proof,
// Idempotency-Key, and a project created together with its venue and payment
// schedule. These scenarios sign up their own workspaces (a trial workspace may
// have at most 5 active projects, and the attack matrix leaves A near that), and
// need real object storage: they skip without TEST_S3_ENDPOINT.

func needStorage(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_S3_ENDPOINT") == "" {
		t.Skip("TEST_S3_ENDPOINT not set: signed uploads need a real bucket")
	}
}

func (s *stack) count(query string, args ...any) int {
	s.t.Helper()
	var n int
	if err := s.db.QueryRow(query, args...).Scan(&n); err != nil {
		s.t.Fatalf("count %q: %v", query, err)
	}
	return n
}

func num(r reply, keys ...string) int64 {
	f, _ := r.dig(keys...).(float64)
	return int64(f)
}

func str(r reply, keys ...string) string {
	v, _ := r.dig(keys...).(string)
	return v
}

func fieldKeys(r reply) []string {
	var out []string
	if m, ok := r.dig("errors").(map[string]any); ok {
		for k := range m {
			out = append(out, k)
		}
	}
	return out
}

func hasKey(r reply, key string) bool {
	m, ok := r.dig("errors").(map[string]any)
	if !ok {
		return false
	}
	_, ok = m[key]
	return ok
}

// uploadFile asks for a signed upload and puts the bytes where the grant says
// (straight to object storage, or through the API), then returns the token.
func (s *stack) uploadFile(t tenant, projectID int64, fileName, mime string, data []byte) (token string, grant reply) {
	s.t.Helper()
	grant = s.mustOK("upload grant", s.do(t.token, "POST", s.projectsPath(projectID, "/uploads"),
		map[string]any{"fileName": fileName, "mimeType": mime, "size": len(data)}))
	token = str(grant, "data", "uploadToken")
	url := str(grant, "data", "url")
	if token == "" || url == "" {
		s.t.Fatalf("grant has no token/url: %s", grant)
	}
	if grant.dig("data", "direct") == true {
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
		if err != nil {
			s.t.Fatal(err)
		}
		req.Header.Set("Content-Type", mime)
		resp, err := s.client.Do(req)
		if err != nil {
			s.t.Fatalf("direct upload: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			s.t.Fatalf("direct upload to object storage answered %d", resp.StatusCode)
		}
	} else {
		s.mustOK("upload through the API", s.putRaw(t.token, url, mime, data))
	}
	return token, grant
}

func proofOf(token, typ string) map[string]any {
	return map[string]any{"uploadToken": token, "type": typ}
}

func clientPayBody(amount int64, ref string, evidence ...map[string]any) map[string]any {
	b := map[string]any{"type": "DP", "amount": amount, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": ref}
	if len(evidence) > 0 {
		b["evidence"] = evidence
	}
	return b
}

func (w *world) tahap4Scenarios(t *testing.T, s *stack) {
	needStorage(t)
	tn, other := s.signUp("t4a"), s.signUp("t4b")
	venue := s.mustOK("venue", s.do(tn.token, "POST", "/api/v1/venues", venueBody(tn, "Venue T4 "+tn.marker))).id("data", "id")
	otherVenue := s.mustOK("other venue", s.do(other.token, "POST", "/api/v1/venues", venueBody(other, "Venue T4 lain "+other.marker))).id("data", "id")
	vendor := s.mustOK("vendor", s.do(tn.token, "POST", "/api/v1/vendors", vendorBody(s, tn, "Vendor T4 "+tn.marker, 0)))
	vendorID, vendorCategory := vendor.id("data", "id"), vendor.id("data", "categoryId")
	pX := s.createProject(tn, "Proyek T4 X "+tn.marker, 60_000_000)
	pY := s.createProject(tn, "Proyek T4 Y "+tn.marker, 60_000_000)
	oP := s.createProject(other, "Proyek T4 lain "+other.marker, 60_000_000)
	proj := func(id int64, rest string) string { return s.projectsPath(id, rest) }
	file := []byte("%PDF-1.4 bukti transfer tahap 4\n%%EOF")
	cnt := func(table string, project int64) int {
		return s.count("SELECT COUNT(*) FROM "+table+" WHERE project_id = ?", project)
	}

	t.Run("(a) a client payment and its proof are one call", func(t *testing.T) {
		s.mustOK("schedule", s.do(tn.token, "PUT", proj(pX, "/payment-terms"), map[string]any{"terms": []map[string]any{
			{"label": "DP", "amount": 30_000_000, "dueDate": day(5)}, {"label": "Pelunasan", "amount": 30_000_000, "dueDate": day(40)},
		}}))
		tok, grant := s.uploadFile(tn, pX, "bukti dp.pdf", "application/pdf", file)
		routesCovered += 3
		if grant.dig("data", "method") != "PUT" || grant.dig("data", "headers", "Content-Type") != "application/pdf" || str(grant, "data", "expiresAt") == "" {
			t.Errorf("grant shape: %s", grant)
		}
		if exp, err := time.Parse(time.RFC3339, str(grant, "data", "expiresAt")); err != nil || time.Until(exp) < 14*time.Minute || time.Until(exp) > 16*time.Minute {
			t.Errorf("expiry must be ~15 minutes away: %v %v", exp, err)
		}
		if grant.dig("data", "direct") != true {
			t.Errorf("with presign on, uploads go straight to storage: %s", grant)
		}

		r := s.doWith(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(15_000_000, "T4-REF-1", proofOf(tok, "Transfer Proof")), nil)
		if r.status != http.StatusCreated {
			t.Fatalf("payment with proof: %s", r)
		}
		ev := rows(r, "data", "evidence")
		if len(ev) != 1 || ev[0]["type"] != "Transfer Proof" || ev[0]["fileName"] != "bukti dp.pdf" || ev[0]["uploadedAt"] == nil || ev[0]["id"] == nil {
			t.Errorf("evidence refs on the payment: %s", r)
		}
		if r.dig("data", "evidenceComplete") != true {
			t.Errorf("evidenceComplete must be true at once: %s", r)
		}
		if num(r, "data", "ledgerSummary", "contractValue") != 60_000_000 || num(r, "data", "ledgerSummary", "paid") != 15_000_000 || num(r, "data", "ledgerSummary", "remaining") != 45_000_000 {
			t.Errorf("ledger summary: %s", r)
		}
		terms := s.mustOK("terms", s.do(tn.token, "GET", proj(pX, "/payment-terms"), nil)).dig("data")
		if !reflect.DeepEqual(terms, r.dig("data", "ledgerSummary", "terms")) {
			t.Errorf("ledgerSummary.terms must be exactly what GET /payment-terms returns:\n%v\n%v", r.dig("data", "ledgerSummary", "terms"), terms)
		}
		if cnt("client_payments", pX) != 1 || s.count("SELECT COUNT(*) FROM evidence WHERE project_id = ? AND related_kind = 'clientPayment' AND related_id = ?", pX, r.id("data", "id")) != 1 {
			t.Error("both rows must exist")
		}
		// The file really is stored and readable through the evidence route.
		got := s.do(tn.token, "GET", proj(pX, fmt.Sprintf("/evidence/%d/file", int64(ev[0]["id"].(float64)))), nil)
		if got.status != http.StatusOK || !bytes.Equal(got.body, file) {
			t.Errorf("stored file: %s", got)
		}
		// The list shows the same payment with its proof.
		list := s.mustOK("list", s.do(tn.token, "GET", proj(pX, "/client-payments"), nil))
		if l := rows(list, "data"); len(l) != 1 || len(rows(list, "data")[0]["evidence"].([]any)) != 1 {
			t.Errorf("list: %s", list)
		}
	})

	t.Run("(a) a bad token creates nothing, and the good one is not used up", func(t *testing.T) {
		good, _ := s.uploadFile(tn, pX, "bukti dua.pdf", "application/pdf", file)
		payments, proofs := cnt("client_payments", pX), cnt("evidence", pX)
		r := s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(1_000_000, "T4-REF-BAD", proofOf(good, "Transfer Proof"), proofOf(good[:len(good)-3]+"AAA", "Transfer Proof")))
		routesCovered++
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[1]") || !hasKey(r, "evidence[1].uploadToken") || hasKey(r, "evidence[0]") {
			t.Errorf("422 on the item that carried the bad token: %s", r)
		}
		if cnt("client_payments", pX) != payments || cnt("evidence", pX) != proofs {
			t.Errorf("nothing may be created: payments %d->%d, proofs %d->%d", payments, cnt("client_payments", pX), proofs, cnt("evidence", pX))
		}
		// A token that names a file that was never uploaded.
		g := s.mustOK("grant", s.do(tn.token, "POST", proj(pX, "/uploads"), map[string]any{"fileName": "x.pdf", "mimeType": "application/pdf", "size": 10}))
		r = s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(1_000_000, "T4-REF-NOFILE", proofOf(str(g, "data", "uploadToken"), "Transfer Proof")))
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[0]") {
			t.Errorf("a file that was never uploaded: %s", r)
		}
		// Too many files, and a type a client payment does not take.
		r = s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(1_000_000, "x", proofOf(good, "Invoice")))
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[0].type") {
			t.Errorf("client payments take a transfer proof only: %s", r)
		}
		// An overpayment with a valid proof: refused, nothing written, the upload is still usable.
		r = s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(500_000_000, "T4-REF-OVER", proofOf(good, "Transfer Proof")))
		if r.status != http.StatusUnprocessableEntity || r.dig("errors", "code") != "payment_exceeds_contract" || !hasKey(r, "amount") {
			t.Errorf("overpayment keeps its code and field: %s", r)
		}
		if cnt("client_payments", pX) != payments || cnt("evidence", pX) != proofs {
			t.Error("an overpayment must not leave a proof behind")
		}
		ok := s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(1_000_000, "T4-REF-OK", proofOf(good, "Transfer Proof")))
		if ok.status != http.StatusCreated {
			t.Errorf("the corrected request may reuse the same upload: %s", ok)
		}
		// Single use: the same upload cannot prove a second payment, and nothing is created.
		before := cnt("client_payments", pX)
		r = s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(1_000_000, "T4-REF-TWICE", proofOf(good, "Transfer Proof")))
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[0]") || cnt("client_payments", pX) != before {
			t.Errorf("a second use of one token: %s (payments %d -> %d)", r, before, cnt("client_payments", pX))
		}
	})

	t.Run("(a) many requests racing with one token create exactly one payment", func(t *testing.T) {
		tok, _ := s.uploadFile(tn, pY, "race.pdf", "application/pdf", file)
		payments, proofs := cnt("client_payments", pY), cnt("evidence", pY)
		const racers = 8
		results := make(chan reply, racers)
		for i := 0; i < racers; i++ {
			go func() {
				results <- s.do(tn.token, "POST", proj(pY, "/client-payments"), clientPayBody(1_000, "T4-RACE", proofOf(tok, "Transfer Proof")))
			}()
		}
		created, refused := 0, 0
		for i := 0; i < racers; i++ {
			switch r := <-results; r.status {
			case http.StatusCreated:
				created++
			case http.StatusUnprocessableEntity:
				refused++
				if !hasKey(r, "evidence[0]") {
					t.Errorf("a refused racer is told about its evidence: %s", r)
				}
			default:
				t.Errorf("unexpected answer: %s", r)
			}
		}
		routesCovered += racers
		if created != 1 || refused != racers-1 {
			t.Errorf("one token proves one payment: %d created, %d refused", created, refused)
		}
		// The losers inserted a payment inside their transaction before the unique
		// path refused their proof; none of that may be visible.
		if cnt("client_payments", pY) != payments+1 || cnt("evidence", pY) != proofs+1 {
			t.Errorf("rows: payments %d -> %d, proofs %d -> %d", payments, cnt("client_payments", pY), proofs, cnt("evidence", pY))
		}
	})

	t.Run("(a) vendor and venue payments, and the vendor derives category and date", func(t *testing.T) {
		eng := map[string]any{"vendorId": vendorID, "scope": "Dekorasi T4", "contractValue": 5_000_000, "pricingTier": "Akad", "engagementStatus": "Booked", "picStaffId": tn.staffID}
		routesCovered += 4
		bad := map[string]any{}
		for k, v := range eng {
			bad[k] = v
		}
		bad["eventDate"], bad["categoryId"] = day(121), vendorCategory+1000
		r := s.do(tn.token, "POST", proj(pX, "/vendors"), bad)
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "eventDate") || !hasKey(r, "categoryId") {
			t.Errorf("a date or category that disagrees with the server is refused, both at once: %s", r)
		}
		r = s.mustOK("engagement without category and date", s.do(tn.token, "POST", proj(pX, "/vendors"), eng))
		if num(r, "data", "categoryId") != vendorCategory || str(r, "data", "eventDate") != day(120) {
			t.Errorf("category from the vendor, date from the project: %s", r)
		}
		same := map[string]any{}
		for k, v := range eng {
			same[k] = v
		}
		same["eventDate"], same["categoryId"] = day(120), vendorCategory
		s.mustOK("engagement with matching values", s.do(tn.token, "POST", proj(pX, "/vendors"), same))
		engID := r.id("data", "id")

		inv, _ := s.uploadFile(tn, pX, "invoice.pdf", "application/pdf", file)
		prf, _ := s.uploadFile(tn, pX, "transfer.pdf", "application/pdf", file)
		vp := s.doWith(tn.token, "POST", proj(pX, "/payments"), map[string]any{
			"projectVendorId": engID, "type": "DP", "amount": 2_000_000, "paymentDate": day(-1), "method": "Transfer", "referenceNumber": "T4-V1",
			"evidence": []map[string]any{proofOf(inv, "Invoice"), proofOf(prf, "Transfer Proof")},
		}, nil)
		if vp.status != http.StatusCreated || len(rows(vp, "data", "evidence")) != 2 || vp.dig("data", "evidenceComplete") != true {
			t.Fatalf("vendor payment with invoice and proof: %s", vp)
		}
		if num(vp, "data", "ledgerSummary", "contractValue") != 5_000_000 || num(vp, "data", "ledgerSummary", "paid") != 2_000_000 || num(vp, "data", "ledgerSummary", "remaining") != 3_000_000 {
			t.Errorf("vendor ledger: %s", vp)
		}
		if _, has := vp.json()["data"].(map[string]any)["ledgerSummary"].(map[string]any)["terms"]; has {
			t.Error("only the client ledger carries terms")
		}
		// The list has the proof refs too.
		if l := rows(s.mustOK("vendor payments", s.do(tn.token, "GET", proj(pX, "/payments"), nil)), "data"); len(l) != 1 || len(l[0]["evidence"].([]any)) != 2 {
			t.Errorf("vendor payments list carries evidence: %v", l)
		}
		// Overpaying the engagement leaves nothing behind.
		extra, _ := s.uploadFile(tn, pX, "x.pdf", "application/pdf", file)
		proofs := cnt("evidence", pX)
		r = s.do(tn.token, "POST", proj(pX, "/payments"), map[string]any{"projectVendorId": engID, "type": "Pelunasan", "amount": 9_000_000, "paymentDate": day(-1), "method": "Transfer", "evidence": []map[string]any{proofOf(extra, "Transfer Proof")}})
		if r.status != http.StatusUnprocessableEntity || r.dig("errors", "code") != "payment_exceeds_contract" || cnt("evidence", pX) != proofs {
			t.Errorf("vendor overpayment: %s", r)
		}
	})

	var e7Project int64
	t.Run("(e) a project with its venue and schedule is one call", func(t *testing.T) {
		body := func(name string, extra map[string]any) map[string]any {
			b := map[string]any{
				"name": name, "brideName": "A", "groomName": "B", "eventDate": day(150), "venue": "-", "prepStartDate": day(1),
				"packageName": "Paket", "contractValue": 100_000_000, "picStaffId": tn.staffID,
			}
			for k, v := range extra {
				b[k] = v
			}
			return b
		}
		routesCovered += 6
		r := s.do(tn.token, "POST", "/api/v1/projects", body("Proyek T4 E7 "+tn.marker, map[string]any{
			"venueId": venue,
			"terms":   []map[string]any{{"label": "DP", "amount": 30_000_000, "dueDate": day(10)}, {"label": "Pelunasan", "amount": 70_000_000, "dueDate": day(100)}},
		}))
		if r.status != http.StatusCreated {
			t.Fatalf("atomic create: %s", r)
		}
		e7Project = r.id("data", "id")
		if num(r, "data", "venueId") != venue || num(r, "data", "venueRentalPrice") != 10_000_000 || num(r, "data", "venueCharge") != 500_000 ||
			str(r, "data", "venue") != "Venue T4 "+tn.marker || str(r, "data", "venueSummary", "name") != "Venue T4 "+tn.marker {
			t.Errorf("venue attached with the venue's current prices: %s", r)
		}
		terms := rows(r, "data", "terms")
		if len(terms) != 2 || terms[0]["label"] != "DP" || terms[0]["status"] == nil || terms[1]["remaining"] != float64(70_000_000) {
			t.Errorf("terms progress in the response: %s", r)
		}
		if got := rows(s.mustOK("terms", s.do(tn.token, "GET", proj(e7Project, "/payment-terms"), nil)), "data"); len(got) != 2 {
			t.Errorf("schedule saved: %v", got)
		}
		if ms := rows(s.mustOK("milestones", s.do(tn.token, "GET", proj(e7Project, "/milestones"), nil)), "data"); len(ms) == 0 {
			t.Error("default milestones are created as before")
		}
		if v := s.mustOK("venue", s.do(tn.token, "GET", proj(e7Project, "/venue"), nil)); str(v, "data", "name") != "Venue T4 "+tn.marker {
			t.Errorf("GET /venue: %s", v)
		}
		// Explicit prices win; without venue/terms the response is the old shape plus empty extras.
		r2 := s.do(tn.token, "POST", "/api/v1/projects", body("Proyek T4 E7b "+tn.marker, map[string]any{"venueId": venue, "venueRentalPrice": 9_000_000, "venueCharge": 0}))
		if r2.status != http.StatusCreated || num(r2, "data", "venueRentalPrice") != 9_000_000 || r2.dig("data", "venueCharge") != float64(0) {
			t.Errorf("explicit prices: %s", r2)
		}
		// Cancel it again to stay under the trial cap of active projects.
		s.mustOK("cancel", s.do(tn.token, "POST", proj(r2.id("data", "id"), "/cancel"), map[string]any{}))
		plain := s.do(tn.token, "POST", "/api/v1/projects", body("Proyek T4 polos "+tn.marker, nil))
		if plain.status != http.StatusCreated || plain.dig("data", "venueSummary") != nil || len(rows(plain, "data", "terms")) != 0 || plain.dig("data", "terms") == nil {
			t.Errorf("a plain create still works: %s", plain)
		}
		s.mustOK("cancel", s.do(tn.token, "POST", proj(plain.id("data", "id"), "/cancel"), map[string]any{}))

		// Problems in several parts at once: all reported, nothing created.
		total := s.count("SELECT COUNT(*) FROM projects WHERE tenant_id = ?", tn.tenant)
		termsBefore := s.count("SELECT COUNT(*) FROM client_payment_terms")
		bad := s.do(tn.token, "POST", "/api/v1/projects", body("Proyek T4 GAGAL "+tn.marker, map[string]any{
			"venueId": otherVenue, "eventDate": "31-12-2026",
			"terms": []map[string]any{{"label": "", "amount": 10, "dueDate": day(10)}, {"label": "B", "amount": 0, "dueDate": day(11)}, {"label": "C", "amount": 5, "dueDate": "besok"}},
		}))
		for _, k := range []string{"venueId", "eventDate", "terms[0].label", "terms[1].amount", "terms[2].dueDate"} {
			if !hasKey(bad, k) {
				t.Errorf("422 must include %q: %s", k, bad)
			}
		}
		if bad.status != http.StatusUnprocessableEntity {
			t.Errorf("status: %s", bad)
		}
		if s.count("SELECT COUNT(*) FROM projects WHERE tenant_id = ?", tn.tenant) != total || s.count("SELECT COUNT(*) FROM projects WHERE name LIKE 'Proyek T4 GAGAL%'") != 0 ||
			s.count("SELECT COUNT(*) FROM client_payment_terms") != termsBefore {
			t.Error("a refused create must leave no project, milestone or term behind")
		}
		// A venue of another workspace alone; a schedule above the contract alone; a venue that does not exist.
		for name, extra := range map[string]map[string]any{
			"another workspace's venue": {"venueId": otherVenue},
			"no such venue":             {"venueId": 987654321},
			"schedule above contract":   {"terms": []map[string]any{{"label": "A", "amount": 99_000_000, "dueDate": day(10)}, {"label": "B", "amount": 2_000_000, "dueDate": day(20)}}},
			"negative venue price":      {"venueId": venue, "venueRentalPrice": -1},
		} {
			r := s.do(tn.token, "POST", "/api/v1/projects", body("Proyek T4 GAGAL2 "+tn.marker, extra))
			if r.status != http.StatusUnprocessableEntity {
				t.Errorf("%s: want 422, got %s", name, r)
			}
			if strings.Contains(string(r.body), other.marker) {
				t.Errorf("%s leaks the other workspace: %s", name, r)
			}
		}
		if s.count("SELECT COUNT(*) FROM projects WHERE name LIKE 'Proyek T4 GAGAL2%'") != 0 {
			t.Error("none of those may create a project")
		}
	})

	t.Run("(a) a venue payment with proof", func(t *testing.T) {
		tok, _ := s.uploadFile(tn, e7Project, "venue.pdf", "application/pdf", file)
		r := s.do(tn.token, "POST", proj(e7Project, "/venue-payments"), clientPayBody(4_000_000, "T4-VN", proofOf(tok, "Transfer Proof")))
		routesCovered++
		if r.status != http.StatusCreated || len(rows(r, "data", "evidence")) != 1 {
			t.Fatalf("venue payment: %s", r)
		}
		if num(r, "data", "ledgerSummary", "contractValue") != 10_500_000 || num(r, "data", "ledgerSummary", "paid") != 4_000_000 || num(r, "data", "ledgerSummary", "remaining") != 6_500_000 {
			t.Errorf("venue ledger (rental + charge): %s", r)
		}
	})

	t.Run("(b) Idempotency-Key replays instead of recording twice", func(t *testing.T) {
		key := map[string]string{"Idempotency-Key": "t4-key-" + fmt.Sprint(time.Now().UnixNano())}
		tok, _ := s.uploadFile(tn, pY, "idem.pdf", "application/pdf", file)
		body := clientPayBody(2_000_000, "T4-IDEM", proofOf(tok, "Transfer Proof"))
		payments, proofs := cnt("client_payments", pY), cnt("evidence", pY)
		first := s.doWith(tn.token, "POST", proj(pY, "/client-payments"), body, key)
		second := s.doWith(tn.token, "POST", proj(pY, "/client-payments"), body, key)
		routesCovered += 5
		if first.status != http.StatusCreated || second.status != http.StatusCreated {
			t.Fatalf("both answer 201: %s / %s", first, second)
		}
		if first.id("data", "id") == 0 || first.id("data", "id") != second.id("data", "id") || !bytes.Equal(first.body, second.body) {
			t.Errorf("the replay is the same answer:\n%s\n%s", first, second)
		}
		if second.header.Get("Idempotent-Replay") != "true" || first.header.Get("Idempotent-Replay") != "" {
			t.Errorf("replay header: first=%q second=%q", first.header.Get("Idempotent-Replay"), second.header.Get("Idempotent-Replay"))
		}
		if cnt("client_payments", pY) != payments+1 || cnt("evidence", pY) != proofs+1 {
			t.Errorf("a replay creates nothing more: payments %d -> %d, proofs %d -> %d", payments, cnt("client_payments", pY), proofs, cnt("evidence", pY))
		}
		changed := s.doWith(tn.token, "POST", proj(pY, "/client-payments"), clientPayBody(3_000_000, "T4-IDEM", proofOf(tok, "Transfer Proof")), key)
		if changed.status != http.StatusUnprocessableEntity || changed.dig("errors", "code") != "idempotency_key_reuse" || !hasKey(changed, "Idempotency-Key") {
			t.Errorf("same key, different body: %s", changed)
		}
		if bad := s.doWith(tn.token, "POST", proj(pY, "/client-payments"), body, map[string]string{"Idempotency-Key": "short"}); bad.status != http.StatusUnprocessableEntity || !hasKey(bad, "Idempotency-Key") {
			t.Errorf("malformed key: %s", bad)
		}
		// A refused request frees its key: the corrected request is a new attempt.
		k2 := map[string]string{"Idempotency-Key": "t4-fix-" + fmt.Sprint(time.Now().UnixNano())}
		if r := s.doWith(tn.token, "POST", proj(pY, "/client-payments"), clientPayBody(0, "x"), k2); r.status != http.StatusUnprocessableEntity {
			t.Errorf("amount 0: %s", r)
		}
		if r := s.doWith(tn.token, "POST", proj(pY, "/client-payments"), clientPayBody(500_000, "T4-FIX"), k2); r.status != http.StatusCreated {
			t.Errorf("corrected retry with the same key: %s", r)
		}
		// The same key on a project create: one project.
		pk := map[string]string{"Idempotency-Key": "t4-proj-" + fmt.Sprint(time.Now().UnixNano())}
		pb := map[string]any{"name": "Proyek T4 idem " + tn.marker, "brideName": "A", "groomName": "B", "eventDate": day(160), "venue": "-", "prepStartDate": day(1), "packageName": "P", "contractValue": 1_000_000, "picStaffId": tn.staffID}
		p1, p2 := s.doWith(tn.token, "POST", "/api/v1/projects", pb, pk), s.doWith(tn.token, "POST", "/api/v1/projects", pb, pk)
		if p1.status != http.StatusCreated || p1.id("data", "id") != p2.id("data", "id") || p2.header.Get("Idempotent-Replay") != "true" ||
			s.count("SELECT COUNT(*) FROM projects WHERE tenant_id = ? AND name = ?", tn.tenant, "Proyek T4 idem "+tn.marker) != 1 {
			t.Errorf("project create replay: %s / %s", p1, p2)
		}
		s.mustOK("cancel", s.do(tn.token, "POST", proj(p1.id("data", "id"), "/cancel"), map[string]any{}))
		// The same key on a vendor payment route is a different request (route is part of the key).
		vk := map[string]string{"Idempotency-Key": "t4-vend-" + fmt.Sprint(time.Now().UnixNano())}
		vb := map[string]any{"type": "DP", "amount": 100_000, "paymentDate": day(-1), "method": "Transfer"}
		v1 := s.doWith(tn.token, "POST", proj(e7Project, "/venue-payments"), vb, vk)
		v2 := s.doWith(tn.token, "POST", proj(e7Project, "/venue-payments"), vb, vk)
		if v1.status != http.StatusCreated || v1.id("data", "id") != v2.id("data", "id") {
			t.Errorf("venue payment replay: %s / %s", v1, v2)
		}
		// Another workspace using the very same key text is unaffected.
		if r := s.doWith(other.token, "POST", proj(oP, "/client-payments"), clientPayBody(100_000, "T4-O"), key); r.status != http.StatusCreated {
			t.Errorf("keys are per workspace: %s", r)
		}
	})

	t.Run("(c) an upload token cannot cross workspace or project", func(t *testing.T) {
		tokX, _ := s.uploadFile(tn, pX, "x.pdf", "application/pdf", file)
		tokO, _ := s.uploadFile(other, oP, "o.pdf", "application/pdf", file)
		routesCovered += 7
		payY, payO, payX := cnt("client_payments", pY), cnt("client_payments", oP), cnt("client_payments", pX)
		// Same workspace, another project.
		r := s.do(tn.token, "POST", proj(pY, "/client-payments"), clientPayBody(1000, "c1", proofOf(tokX, "Transfer Proof")))
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[0].uploadToken") {
			t.Errorf("A's token from project X in project Y: %s", r)
		}
		// Another workspace uses it in its own project.
		r = s.do(other.token, "POST", proj(oP, "/client-payments"), clientPayBody(1000, "c2", proofOf(tokX, "Transfer Proof")))
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[0].uploadToken") {
			t.Errorf("A's token in B's project: %s", r)
		}
		// ...and the reverse.
		r = s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(1000, "c3", proofOf(tokO, "Transfer Proof")))
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[0].uploadToken") {
			t.Errorf("B's token in A's project: %s", r)
		}
		// On the vendor and venue routes too.
		r = s.do(other.token, "POST", proj(oP, "/venue-payments"), clientPayBody(1000, "c4", proofOf(tokX, "Transfer Proof")))
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence[0].uploadToken") {
			t.Errorf("venue route: %s", r)
		}
		// The PUT route refuses a token that is not for that project.
		if r := s.putRaw(tn.token, proj(pY, "/uploads/"+tokX), "application/pdf", file); r.status != http.StatusUnprocessableEntity || !hasKey(r, "uploadToken") {
			t.Errorf("PUT with another project's token: %s", r)
		}
		if r := s.putRaw(other.token, proj(oP, "/uploads/"+tokX), "application/pdf", file); r.status != http.StatusUnprocessableEntity || !hasKey(r, "uploadToken") {
			t.Errorf("PUT with another workspace's token: %s", r)
		}
		if r := s.putRaw(other.token, proj(pX, "/uploads/"+tokX), "application/pdf", file); r.status != http.StatusForbidden && r.status != http.StatusNotFound {
			t.Errorf("another workspace cannot even reach the project's upload route: %s", r)
		}
		if cnt("client_payments", pY) != payY || cnt("client_payments", oP) != payO || cnt("client_payments", pX) != payX {
			t.Error("none of those may create a payment")
		}
		// And the token still works where it belongs.
		if r := s.do(tn.token, "POST", proj(pX, "/client-payments"), clientPayBody(1000, "c5", proofOf(tokX, "Transfer Proof"))); r.status != http.StatusCreated {
			t.Errorf("the token in its own project: %s", r)
		}
	})

	t.Run("(f) a client principal is refused on every new endpoint", func(t *testing.T) {
		c, P := w.clientToken, w.Project
		tok := "abc.def"
		for _, d := range []attack{
			{name: "upload create", method: "POST", path: s.projectsPath(P, "/uploads"), body: map[string]any{"fileName": "a.pdf", "mimeType": "application/pdf", "size": 10}},
			{name: "upload put", method: "PUT", path: s.projectsPath(P, "/uploads/"+tok)},
			{name: "client payment with proof", method: "POST", path: s.projectsPath(P, "/client-payments"), body: clientPayBody(1, "x", proofOf(tok, "Transfer Proof"))},
			{name: "vendor payment with proof", method: "POST", path: s.projectsPath(P, "/payments"), body: map[string]any{"projectVendorId": w.Engagement, "type": "DP", "amount": 1, "evidence": []map[string]any{proofOf(tok, "Invoice")}}},
			{name: "venue payment with proof", method: "POST", path: s.projectsPath(P, "/venue-payments"), body: clientPayBody(1, "x", proofOf(tok, "Invoice"))},
			{name: "project with venue and terms", method: "POST", path: "/api/v1/projects", body: map[string]any{"name": "x", "venueId": w.Venue, "terms": []map[string]any{{"label": "a", "amount": 1, "dueDate": day(3)}}}},
		} {
			routesCovered++
			r := s.doWith(c, d.method, d.path, d.body, map[string]string{"Idempotency-Key": "t4-client-0001"})
			if r.status != http.StatusForbidden {
				t.Errorf("client -> %s %s: want 403, got %s", d.method, d.path, r)
			}
		}
		if n := s.count("SELECT COUNT(*) FROM idempotency_keys WHERE idempotency_key = 't4-client-0001'"); n != 0 {
			t.Errorf("a refused client must not even reserve a key: %d", n)
		}
	})
}

// TestUploadThroughTheAPI: when object storage cannot be reached by the browser
// (S3_PRESIGN=false) the grant points at the API's own route, with the same token.
func TestUploadThroughTheAPI(t *testing.T) {
	needStorage(t)
	s := startStackWith(t, "S3_PRESIGN=false")
	tn := s.signUp("px")
	p := s.createProject(tn, "Proyek proxy "+tn.marker, 20_000_000)
	pay := func(rest string) string { return s.projectsPath(p, rest) }
	data := []byte("%PDF-1.4 lewat API\n%%EOF")

	// The grant is validated.
	r := s.do(tn.token, "POST", pay("/uploads"), map[string]any{"fileName": "../x", "mimeType": "text/html", "size": 16 * 1024 * 1024})
	routesCovered++
	if r.status != http.StatusUnprocessableEntity || !hasKey(r, "fileName") || !hasKey(r, "mimeType") || !hasKey(r, "size") {
		t.Errorf("a bad description is refused field by field: %s", r)
	}

	g := s.mustOK("grant", s.do(tn.token, "POST", pay("/uploads"), map[string]any{"fileName": "bukti.pdf", "mimeType": "application/pdf", "size": len(data)}))
	token, url := str(g, "data", "uploadToken"), str(g, "data", "url")
	if g.dig("data", "direct") != false || url != "/api/v1/projects/"+fmt.Sprint(p)+"/uploads/"+token || g.dig("data", "method") != "PUT" {
		t.Fatalf("storage that cannot presign gives the API route: %s", g)
	}
	// Refusals leave nothing stored.
	if r := s.putRaw(tn.token, url, "image/png", data); r.status != http.StatusUnprocessableEntity || !hasKey(r, "contentType") {
		t.Errorf("another content type: %s", r)
	}
	if r := s.putRaw(tn.token, url, "application/pdf", append(data, make([]byte, 100)...)); r.status != http.StatusRequestEntityTooLarge {
		t.Errorf("more than declared: %s", r)
	}
	if r := s.putRaw("", url, "application/pdf", data); r.status != http.StatusUnauthorized {
		t.Errorf("no login: %s", r)
	}
	if r := s.putRaw(tn.token, url, "application/pdf", nil); r.status != http.StatusUnprocessableEntity {
		t.Errorf("empty body: %s", r)
	}
	// Nothing was stored yet, so a payment naming the token is refused.
	noFile := s.do(tn.token, "POST", pay("/client-payments"), clientPayBody(1_000_000, "P0", proofOf(token, "Transfer Proof")))
	if noFile.status != http.StatusUnprocessableEntity || !hasKey(noFile, "evidence[0]") {
		t.Errorf("nothing stored yet: %s", noFile)
	}
	routesCovered += 5
	// The PUT stores the object.
	if r := s.putRaw(tn.token, url, "application/pdf", data); r.status != http.StatusOK {
		t.Fatalf("PUT: %s", r)
	}
	if r := s.putRaw(tn.token, url, "application/pdf", data); r.status != http.StatusOK {
		t.Errorf("a retried PUT before use is fine: %s", r)
	}
	first := s.do(tn.token, "POST", pay("/client-payments"), clientPayBody(1_000_000, "P1", proofOf(token, "Transfer Proof")))
	if first.status != http.StatusCreated {
		t.Fatalf("payment with the uploaded proof: %s", first)
	}
	evID := int64(rows(first, "data", "evidence")[0]["id"].(float64))
	if got := s.do(tn.token, "GET", pay(fmt.Sprintf("/evidence/%d/file", evID)), nil); got.status != http.StatusOK || !bytes.Equal(got.body, data) {
		t.Errorf("stored bytes: %s", got)
	}
	// A second use of the same token fails, and so does writing over the recorded proof.
	second := s.do(tn.token, "POST", pay("/client-payments"), clientPayBody(1_000_000, "P2", proofOf(token, "Transfer Proof")))
	if second.status != http.StatusUnprocessableEntity || !hasKey(second, "evidence[0]") {
		t.Errorf("second use: %s", second)
	}
	if n := s.count("SELECT COUNT(*) FROM client_payments WHERE project_id = ?", p); n != 1 {
		t.Errorf("the refused second use created a payment: %d", n)
	}
	if r := s.putRaw(tn.token, url, "application/pdf", []byte("%PDF tampered")); r.status != http.StatusConflict {
		t.Errorf("PUT after use: %s", r)
	}
	if got := s.do(tn.token, "GET", pay(fmt.Sprintf("/evidence/%d/file", evID)), nil); !bytes.Equal(got.body, data) {
		t.Error("the recorded proof changed")
	}
	routesCovered += 4
}
