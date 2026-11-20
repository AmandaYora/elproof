//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Paket WO, Tahap 4: vendors and venues the client pays directly. Their bills
// (/projects/{id}/supplier-bills), the couple's proof of paying one
// (supplier-bills/submissions) and the WO's check of it
// (/supplier-payments/{pid}/verify|reject). Another workspace can neither list,
// read, write nor verify them; a client of one workspace cannot see or send to
// the other's; a submission cannot point at another project's bill; Keluarga
// sees no bill; and the WO's cash and revenue never move.
func TestPaketTahap4(t *testing.T) {
	s := startStack(t)
	a, b := s.signUp("T4A"), s.signUp("T4B")
	proj := func(id int64, rest string) string { return s.projectsPath(id, rest) }
	stamp := time.Now().UnixNano() % 1_000_000_000

	type side struct {
		project, photographer, decor, bill int64
	}
	build := func(tn tenant, name string) side {
		category := s.mustOK("category", s.do(tn.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": "Foto " + name + " " + tn.marker})).id("data", "id")
		photo := s.mustOK("photographer", s.do(tn.token, "POST", "/api/v1/vendors", vendorBody(s, tn, "Fotografer Klien "+tn.marker, category))).id("data", "id")
		decor := s.mustOK("decor", s.do(tn.token, "POST", "/api/v1/vendors", vendorBody(s, tn, "Dekor Rahasia "+tn.marker, category))).id("data", "id")
		P := s.createProject(tn, name+" "+tn.marker, 50_000_000)
		x := side{project: P}
		x.photographer = s.engageBy(tn, P, photo, "Foto klien "+tn.marker, "Client", false)
		x.decor = s.engage(tn, P, decor, "Dekor WO "+tn.marker, 5_000_000, "Booked", nil)
		x.bill = s.mustOK("bill", s.do(tn.token, "POST", proj(P, "/supplier-bills/terms"), map[string]any{
			"projectSupplierId": x.photographer, "label": "DP fotografer " + tn.marker, "amount": 3_000_000, "dueDate": day(5),
		})).id("data", "id")
		return x
	}
	A, B := build(a, "Proyek T4"), build(b, "Proyek T4")
	A2 := build(a, "Proyek T4 kedua") // another project of the same workspace

	bride := s.addClient(a, A.project, "Bride", "Pengantin", fmt.Sprintf("pengantin.t4.%d", stamp))
	family := s.addClient(a, A.project, "Family Representative", "Keluarga", fmt.Sprintf("keluarga.t4.%d", stamp))
	bBride := s.addClient(b, B.project, "Bride", "Pengantin", fmt.Sprintf("pengantin.t4b.%d", stamp))
	brideAsTenant := tenant{token: bride}

	noLeakOfA := func(t *testing.T, what string, r reply) {
		t.Helper()
		if strings.Contains(string(r.body), a.marker) {
			t.Errorf("%s leaks workspace A: %s", what, r)
		}
	}

	t.Run("bills only for a vendor the client pays", func(t *testing.T) {
		routesCovered++
		r := s.do(a.token, "POST", proj(A.project, "/supplier-bills/terms"), map[string]any{
			"projectSupplierId": A.decor, "label": "DP dekor", "amount": 1_000_000, "dueDate": day(5),
		})
		if r.status != http.StatusUnprocessableEntity || r.dig("errors", "code") != "supplier_paid_by_wo" {
			t.Errorf("a bill on a WO-paid vendor: want 422 supplier_paid_by_wo, got %s", r)
		}
		// Another project's supplier is not this project's.
		r = s.do(a.token, "POST", proj(A.project, "/supplier-bills/terms"), map[string]any{
			"projectSupplierId": A2.photographer, "label": "Salah proyek", "amount": 1_000_000, "dueDate": day(5),
		})
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "projectSupplierId") {
			t.Errorf("a bill on another project's supplier: want 422 on projectSupplierId, got %s", r)
		}
		list := rows(s.mustOK("A bills", s.do(a.token, "GET", proj(A.project, "/supplier-bills"), nil)), "data")
		if len(list) != 1 || list[0]["projectSupplierId"] != float64(A.photographer) {
			t.Errorf("only the client-paid photographer has bills: %v", list)
		}
	})

	t.Run("the portal: Pengantin sees the bill, Keluarga none", func(t *testing.T) {
		routesCovered += 4
		r := s.mustOK("Pengantin bills", s.do(bride, "GET", proj(A.project, "/supplier-bills"), nil))
		list := rows(r, "data")
		if len(list) != 1 || list[0]["name"] != "Fotografer Klien "+a.marker {
			t.Errorf("Pengantin sees the client-paid photographer by its name: %s", r)
		}
		for _, leak := range []string{"Dekor Rahasia", "agreedCost", "5000000"} {
			if strings.Contains(string(r.body), leak) {
				t.Errorf("the portal bills carry %q: %s", leak, r)
			}
		}
		wantStatus(t, "Keluarga GET supplier-bills", s.do(family, "GET", proj(A.project, "/supplier-bills"), nil), http.StatusForbidden)
		home := s.mustOK("Pengantin home", s.do(bride, "GET", "/api/v1/projects/me/home", nil))
		if str(home, "data", "finance", "nextSupplierBill", "supplierName") != "Fotografer Klien "+a.marker {
			t.Errorf("Pengantin's home shows the next vendor bill: %s", home)
		}
		famHome := s.mustOK("Keluarga home", s.do(family, "GET", "/api/v1/projects/me/home", nil))
		if strings.Contains(string(famHome.body), "nextSupplierBill") || strings.Contains(string(famHome.body), "DP fotografer") {
			t.Errorf("Keluarga's home has no vendor bill: %s", famHome)
		}
		// The WO's bill writes are not the client's.
		for _, at := range []attack{
			{name: "client creates a bill", method: "POST", path: proj(A.project, "/supplier-bills/terms"), body: map[string]any{"projectSupplierId": A.photographer, "label": "x", "amount": 1, "dueDate": day(5)}},
			{name: "client edits a bill", method: "PATCH", path: proj(A.project, fmt.Sprintf("/supplier-bills/terms/%d", A.bill)), body: map[string]any{"amount": 1}},
			{name: "client deletes a bill", method: "DELETE", path: proj(A.project, fmt.Sprintf("/supplier-bills/terms/%d", A.bill))},
		} {
			wantStatus(t, at.name, s.do(bride, at.method, at.path, at.body), http.StatusForbidden)
		}
	})

	t.Run("WO B cannot list, read, write or verify WO A's bills", func(t *testing.T) {
		for _, at := range []attack{
			{name: "bills list", method: "GET", path: proj(A.project, "/supplier-bills")},
			{name: "bill create", method: "POST", path: proj(A.project, "/supplier-bills/terms"), body: map[string]any{"projectSupplierId": A.photographer, "label": "B " + b.marker, "amount": 1, "dueDate": day(5)}},
			{name: "bill update", method: "PATCH", path: proj(A.project, fmt.Sprintf("/supplier-bills/terms/%d", A.bill)), body: map[string]any{"amount": 1}},
			{name: "bill delete", method: "DELETE", path: proj(A.project, fmt.Sprintf("/supplier-bills/terms/%d", A.bill))},
			{name: "bill restore", method: "POST", path: proj(A.project, fmt.Sprintf("/supplier-bills/terms/%d/restore", A.bill))},
			{name: "submission as staff", method: "POST", path: proj(A.project, "/supplier-bills/submissions"), body: map[string]any{"billTermId": A.bill, "amount": 1}},
		} {
			routesCovered++
			r := s.do(b.token, at.method, at.path, at.body)
			if r.status != http.StatusNotFound && r.status != http.StatusForbidden {
				t.Errorf("B %s: want 403 or 404, got %s", at.name, r)
			}
			if vacuous(r) {
				t.Errorf("B %s: router miss, not an ownership answer: %s", at.name, r)
			}
			noLeakOfA(t, "B "+at.name, r)
		}
		// A's bill id in B's own project: not found.
		for _, at := range []attack{
			{name: "B project + A bill update", method: "PATCH", path: proj(B.project, fmt.Sprintf("/supplier-bills/terms/%d", A.bill)), body: map[string]any{"amount": 1}},
			{name: "B project + A bill delete", method: "DELETE", path: proj(B.project, fmt.Sprintf("/supplier-bills/terms/%d", A.bill))},
		} {
			routesCovered++
			r := s.doWith(b.token, at.method, at.path, at.body, map[string]string{"If-Match": staleVersion})
			if r.status != http.StatusNotFound {
				t.Errorf("%s: want 404, got %s", at.name, r)
			}
			noLeakOfA(t, at.name, r)
		}
		// B's bill on A's supplier: that supplier is not B's project's.
		r := s.do(b.token, "POST", proj(B.project, "/supplier-bills/terms"), map[string]any{"projectSupplierId": A.photographer, "label": "B " + b.marker, "amount": 1, "dueDate": day(5)})
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "projectSupplierId") {
			t.Errorf("B's bill on A's supplier: want 422, got %s", r)
		}
		if n := s.count("SELECT COUNT(*) FROM supplier_bill_terms WHERE id = ? AND deleted_at IS NULL AND amount = 3000000", A.bill); n != 1 {
			t.Errorf("B's attempts changed A's bill")
		}
	})

	t.Run("a client cannot reach another workspace's bills", func(t *testing.T) {
		for _, c := range []struct{ who, tok string }{{"A Pengantin", bride}, {"A Keluarga", family}} {
			routesCovered += 2
			for _, at := range []attack{
				{name: "B bills", method: "GET", path: proj(B.project, "/supplier-bills")},
				{name: "B submission", method: "POST", path: proj(B.project, "/supplier-bills/submissions"), body: map[string]any{"billTermId": B.bill, "amount": 1}},
			} {
				if r := s.do(c.tok, at.method, at.path, at.body); r.status != http.StatusForbidden && r.status != http.StatusNotFound {
					t.Errorf("%s %s: want 403/404, got %s", c.who, at.name, r)
				}
			}
		}
		if r := s.do(bBride, "GET", proj(A.project, "/supplier-bills"), nil); r.status != http.StatusForbidden && r.status != http.StatusNotFound {
			t.Errorf("B's client reads A's bills: %s", r)
		}
	})

	t.Run("submit, verify, reject: the WO's cash and revenue stay", func(t *testing.T) {
		needStorage(t)
		routesCovered += 6
		summary := func() reply {
			return s.mustOK("A finance summary", s.do(a.token, "GET", "/api/v1/finance/summary", nil))
		}
		before := summary()
		proof, _ := s.uploadFile(brideAsTenant, A.project, "bukti-foto.jpg", "image/jpeg", coverJPEG(t, 32, 32))
		body := map[string]any{"billTermId": A.bill, "amount": 3_000_000, "paymentDate": day(-1), "method": "Transfer Bank",
			"referenceNumber": "FOTO-" + a.marker, "evidence": []map[string]any{proofOf(proof, "Transfer Proof")}}

		// Keluarga and another workspace's client cannot send it.
		wantStatus(t, "Keluarga submits", s.do(family, "POST", proj(A.project, "/supplier-bills/submissions"), body), http.StatusForbidden)
		if r := s.do(bBride, "POST", proj(A.project, "/supplier-bills/submissions"), body); r.status != http.StatusForbidden && r.status != http.StatusNotFound {
			t.Errorf("B's client submits to A: %s", r)
		}
		// Neither another project's bill (same workspace) nor another workspace's.
		for name, bill := range map[string]int64{"another project's bill": A2.bill, "another workspace's bill": B.bill} {
			other, _ := s.uploadFile(brideAsTenant, A.project, "bukti-lain.jpg", "image/jpeg", coverJPEG(t, 32, 32))
			wrong := map[string]any{"billTermId": bill, "amount": 3_000_000, "paymentDate": day(-1), "method": "Transfer Bank",
				"evidence": []map[string]any{proofOf(other, "Transfer Proof")}}
			if r := s.do(bride, "POST", proj(A.project, "/supplier-bills/submissions"), wrong); r.status != http.StatusUnprocessableEntity || !hasKey(r, "billTermId") {
				t.Errorf("%s: want 422 on billTermId, got %s", name, r)
			}
		}

		sent := s.doWith(bride, "POST", proj(A.project, "/supplier-bills/submissions"), body, map[string]string{"Idempotency-Key": fmt.Sprintf("t4-%d", stamp)})
		if sent.status != http.StatusCreated || str(sent, "data", "status") != "pending" || str(sent, "data", "source") == "" {
			t.Fatalf("Pengantin sends the proof: %s", sent)
		}
		pid := sent.id("data", "id")
		if num(sent, "data", "supplierBill", "pending") != 3_000_000 || num(sent, "data", "supplierBill", "paid") != 0 {
			t.Errorf("an unchecked proof is pending, not paid: %s", sent)
		}

		// The WO's queue lists it; B's does not.
		found := false
		for _, v := range rows(s.mustOK("A queue", s.do(a.token, "GET", "/api/v1/finance/verifications?limit=100", nil)), "data") {
			if v["kind"] == "supplierPayment" && v["id"] == float64(pid) {
				found = true
				if v["supplierName"] != "Fotografer Klien "+a.marker || v["billTermId"] != float64(A.bill) {
					t.Errorf("queue row: %v", v)
				}
			}
		}
		if !found {
			t.Error("A's verification queue lists the client's proof to the photographer")
		}
		bQueue := s.mustOK("B queue", s.do(b.token, "GET", "/api/v1/finance/verifications?limit=100", nil))
		noLeakOfA(t, "B queue", bQueue)

		// B cannot verify or reject it, in A's project or its own.
		for _, path := range []string{
			proj(A.project, fmt.Sprintf("/supplier-payments/%d/verify", pid)),
			proj(A.project, fmt.Sprintf("/supplier-payments/%d/reject", pid)),
			proj(B.project, fmt.Sprintf("/supplier-payments/%d/verify", pid)),
			proj(B.project, fmt.Sprintf("/supplier-payments/%d/reject", pid)),
		} {
			r := s.do(b.token, "POST", path, map[string]any{"reason": "B " + b.marker})
			if r.status != http.StatusNotFound && r.status != http.StatusForbidden {
				t.Errorf("B POST %s: want 403/404, got %s", path, r)
			}
			noLeakOfA(t, "B "+path, r)
		}
		// Nor the couple.
		wantStatus(t, "client verifies", s.do(bride, "POST", proj(A.project, fmt.Sprintf("/supplier-payments/%d/verify", pid)), map[string]any{}), http.StatusForbidden)
		if s.count("SELECT COUNT(*) FROM supplier_payments WHERE id = ? AND verified_at IS NULL AND rejected_at IS NULL", pid) != 1 {
			t.Fatal("the attempts checked A's proof")
		}

		v := s.mustOK("A verifies", s.do(a.token, "POST", proj(A.project, fmt.Sprintf("/supplier-payments/%d/verify", pid)), map[string]any{}))
		if str(v, "data", "status") != "verified" || str(v, "data", "verifiedAt") == "" {
			t.Errorf("verified: %s", v)
		}
		bills := rows(s.mustOK("bills after", s.do(bride, "GET", proj(A.project, "/supplier-bills"), nil)), "data")
		if len(bills) != 1 || bills[0]["paid"] != float64(3_000_000) || bills[0]["pending"] != float64(0) {
			t.Errorf("once verified the bill is paid: %v", bills)
		}

		// A second proof, rejected with a reason: never counts.
		again, _ := s.uploadFile(brideAsTenant, A.project, "bukti-2.jpg", "image/jpeg", coverJPEG(t, 32, 32))
		second := s.mustOK("second proof", s.do(bride, "POST", proj(A.project, "/supplier-bills/submissions"), map[string]any{
			"billTermId": A.bill, "amount": 1_000_000, "paymentDate": day(-1), "method": "Transfer Bank",
			"evidence": []map[string]any{proofOf(again, "Transfer Proof")}})).id("data", "id")
		wantStatus(t, "reject without a reason", s.do(a.token, "POST", proj(A.project, fmt.Sprintf("/supplier-payments/%d/reject", second)), map[string]any{"reason": ""}), http.StatusUnprocessableEntity)
		rj := s.mustOK("A rejects", s.do(a.token, "POST", proj(A.project, fmt.Sprintf("/supplier-payments/%d/reject", second)), map[string]any{"reason": "Nominal tidak sesuai"}))
		if str(rj, "data", "status") != "rejected" || str(rj, "data", "rejectionReason") != "Nominal tidak sesuai" {
			t.Errorf("rejected: %s", rj)
		}

		after := summary()
		for _, key := range []string{"received", "pendingVerification"} {
			if num(before, "data", key, "amount") != num(after, "data", key, "amount") {
				t.Errorf("the client's payment to its vendor moved the WO's %s: %s -> %s", key, before, after)
			}
		}
		if fmt.Sprint(before.dig("data", "cashFlow")) != fmt.Sprint(after.dig("data", "cashFlow")) {
			t.Errorf("the WO's cash flow moved: %v -> %v", before.dig("data", "cashFlow"), after.dig("data", "cashFlow"))
		}
		if fmt.Sprint(before.dig("data", "payables")) != fmt.Sprint(after.dig("data", "payables")) {
			t.Errorf("the WO's payables moved: %v -> %v", before.dig("data", "payables"), after.dig("data", "payables"))
		}
	})
}
