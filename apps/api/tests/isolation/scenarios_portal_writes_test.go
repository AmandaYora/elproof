//go:build integration

package isolation

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ADR-0047: the Client Portal gains a cover photo (staff upload, both read) and
// two write paths for the couple — a transfer proof that becomes an unverified
// client payment, and marking a task done. These scenarios sign up their own
// workspaces and need real object storage.

func coverJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y += 3 {
		for x := 0; x < w; x += 3 {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 140, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (w *world) portalWriteScenarios(t *testing.T, s *stack) {
	needStorage(t)
	tn, other := s.signUp("pw"), s.signUp("pwo")
	P := s.createProject(tn, "Proyek Portal "+tn.marker, 10_000_000)
	oP := s.createProject(other, "Proyek Portal lain "+other.marker, 10_000_000)
	proj := func(id int64, rest string) string { return s.projectsPath(id, rest) }
	stamp := time.Now().UnixNano() % 1_000_000_000
	bride := s.addClient(tn, P, "Bride", "Pengantin", fmt.Sprintf("pengantin.pw.%d", stamp))
	family := s.addClient(tn, P, "Family Representative", "Keluarga", fmt.Sprintf("keluarga.pw.%d", stamp))
	otherBride := s.addClient(other, oP, "Bride", "Pengantin", fmt.Sprintf("pengantin.pwo.%d", stamp))
	brideAsTenant := tenant{token: bride}

	t.Run("cover: staff upload, both read, replace and remove", func(t *testing.T) {
		routesCovered += 6
		photo := base64.StdEncoding.EncodeToString(coverJPEG(t, 2400, 1200))
		r := s.do(tn.token, "PUT", proj(P, "/cover"), map[string]any{"fileName": "sampul.jpg", "mimeType": "image/jpeg", "base64Data": photo})
		url := str(r, "data", "coverUrl")
		if r.status != http.StatusOK || !strings.HasPrefix(url, fmt.Sprintf("/api/v1/projects/%d/cover?v=", P)) {
			t.Fatalf("PUT cover: %s", r)
		}
		if got := str(s.do(tn.token, "GET", proj(P, ""), nil), "data", "coverUrl"); got != url {
			t.Errorf("GET /projects/{id} coverUrl %q, want %q", got, url)
		}
		if got := str(s.do(tn.token, "GET", proj(P, "/overview"), nil), "data", "project", "coverUrl"); got != url {
			t.Errorf("overview coverUrl %q", got)
		}
		if got := str(s.do(bride, "GET", "/api/v1/projects/me/home", nil), "data", "project", "coverUrl"); got != url {
			t.Errorf("portal home coverUrl %q", got)
		}
		for who, tok := range map[string]string{"staff": tn.token, "client": bride, "family": family} {
			img := s.do(tok, "GET", url, nil)
			if img.status != http.StatusOK || img.header.Get("Content-Type") != "image/jpeg" || img.header.Get("Cache-Control") != "private, max-age=300" {
				t.Errorf("%s GET cover: %d %v", who, img.status, img.header)
				continue
			}
			cfg, _, err := image.DecodeConfig(bytes.NewReader(img.body))
			if err != nil || cfg.Width != 2000 || cfg.Height != 1000 {
				t.Errorf("%s: the stored cover is downscaled to 2000 px: %+v %v", who, cfg, err)
			}
		}
		// Another workspace's staff and client, and the couple writing it: refused.
		for _, d := range []struct {
			name, tok, method string
			body              any
		}{
			{"other staff reads", other.token, "GET", nil},
			{"other client reads", otherBride, "GET", nil},
			{"other staff writes", other.token, "PUT", map[string]any{"fileName": "x.jpg", "mimeType": "image/jpeg", "base64Data": photo}},
			{"client writes", bride, "PUT", map[string]any{"fileName": "x.jpg", "mimeType": "image/jpeg", "base64Data": photo}},
			{"client removes", bride, "DELETE", nil},
		} {
			if r := s.do(d.tok, d.method, proj(P, "/cover"), d.body); r.status != http.StatusForbidden && r.status != http.StatusNotFound {
				t.Errorf("%s: want 403/404, got %s", d.name, r)
			}
		}
		// A renamed file is refused.
		if r := s.do(tn.token, "PUT", proj(P, "/cover"), map[string]any{"fileName": "x.jpg", "mimeType": "image/jpeg", "base64Data": pdf}); r.status != http.StatusUnprocessableEntity || !hasKey(r, "base64Data") {
			t.Errorf("a PDF declared JPEG: %s", r)
		}
		// Replace: a new key, the old object gone from storage.
		var oldKey string
		_ = s.db.QueryRow("SELECT cover_photo_path FROM projects WHERE id = ?", P).Scan(&oldKey)
		time.Sleep(1100 * time.Millisecond) // a new ?v= second
		r = s.do(tn.token, "PUT", proj(P, "/cover"), map[string]any{"fileName": "b.jpg", "mimeType": "image/jpeg", "base64Data": base64.StdEncoding.EncodeToString(coverJPEG(t, 64, 64))})
		if r.status != http.StatusOK || str(r, "data", "coverUrl") == url {
			t.Errorf("replace: %s", r)
		}
		var newKey string
		_ = s.db.QueryRow("SELECT cover_photo_path FROM projects WHERE id = ?", P).Scan(&newKey)
		if newKey == oldKey || newKey == "" {
			t.Errorf("a replaced cover gets a new key: %q -> %q", oldKey, newKey)
		}
		// Remove.
		r = s.do(tn.token, "DELETE", proj(P, "/cover"), nil)
		if r.status != http.StatusOK || r.dig("data", "coverUrl") != nil {
			t.Errorf("DELETE cover: %s", r)
		}
		if r := s.do(bride, "GET", proj(P, "/cover"), nil); r.status != http.StatusNotFound {
			t.Errorf("no cover: 404, got %s", r)
		}
		if r := s.do(bride, "GET", "/api/v1/projects/me/home", nil); r.dig("data", "project", "coverUrl") != nil {
			t.Errorf("home coverUrl after removal: %s", r)
		}
	})

	t.Run("proof: the couple submits, staff see it and verify it", func(t *testing.T) {
		routesCovered += 5
		proofJPEG := coverJPEG(t, 32, 32)
		token, _ := s.uploadFile(brideAsTenant, P, "bukti.jpg", "image/jpeg", proofJPEG)
		body := map[string]any{"amount": 2_500_000, "paymentDate": day(0), "method": "Transfer Bank", "referenceNumber": "TRF-" + tn.marker,
			"notes": "Dari Ayu", "evidence": []map[string]any{proofOf(token, "Transfer Proof")}}
		key := map[string]string{"Idempotency-Key": fmt.Sprintf("pw-submit-%d", stamp)}
		r := s.doWith(bride, "POST", proj(P, "/client-payments/submissions"), body, key)
		if r.status != http.StatusCreated || str(r, "data", "source") != "client" || r.dig("data", "verifiedAt") != nil ||
			r.dig("data", "recordedBy") != nil || str(r, "data", "type") != "DP" {
			t.Fatalf("submission: %s", r)
		}
		payID := r.id("data", "id")
		// A retry with the same key replays; nothing is recorded twice.
		again := s.doWith(bride, "POST", proj(P, "/client-payments/submissions"), body, key)
		if again.status != http.StatusCreated || again.id("data", "id") != payID || again.header.Get("Idempotent-Replay") != "true" {
			t.Errorf("replay: %s", again)
		}
		if n := s.count("SELECT COUNT(*) FROM client_payments WHERE project_id = ? AND deleted_at IS NULL", P); n != 1 {
			t.Errorf("one payment, got %d", n)
		}
		var recorded, submitted any
		_ = s.db.QueryRow("SELECT recorded_by_staff_id, submitted_by_client_id FROM client_payments WHERE id = ?", payID).Scan(&recorded, &submitted)
		if recorded != nil || submitted == nil {
			t.Errorf("recorded_by stays NULL and submitted_by_client_id is set: %v %v", recorded, submitted)
		}
		if n := s.count("SELECT COUNT(*) FROM evidence WHERE related_kind = 'clientPayment' AND related_id = ?", payID); n != 1 {
			t.Errorf("the proof is recorded with the payment: %d", n)
		}
		if n := s.count("SELECT COUNT(*) FROM activity_log WHERE project_id = ? AND description = 'Pasangan mengirim bukti pembayaran Rp2.500.000'", P); n != 1 {
			t.Errorf("activity: %d", n)
		}

		// The couple sees their own unverified submission.
		list := rows(s.do(bride, "GET", proj(P, "/client-payments"), nil), "data")
		if row := rowByID(list, payID); row == nil || row["source"] != "client" || row["verifiedAt"] != nil {
			t.Errorf("client list: %v", list)
		}
		// Staff see it in the ledger and in Keuangan "Verifikasi bukti".
		if row := rowByID(rows(s.do(tn.token, "GET", proj(P, "/client-payments"), nil), "data"), payID); row == nil || row["source"] != "client" {
			t.Errorf("staff ledger: %v", row)
		}
		if row := rowByID(rows(s.do(tn.token, "GET", "/api/v1/finance/verifications", nil), "data"), payID); row == nil || row["source"] != "client" || row["recordedByName"] != "" {
			t.Errorf("finance verifications: %v", row)
		}
		// Staff verify with the existing endpoint.
		v := s.do(tn.token, "POST", proj(P, fmt.Sprintf("/client-payments/%d/verify", payID)), map[string]any{})
		if v.status != http.StatusOK || v.dig("data", "verifiedAt") == nil || str(v, "data", "source") != "client" {
			t.Errorf("verify: %s", v)
		}

		// Refusals: a Keluarga client (no portal.finance.submit), another
		// workspace's client, staff on the client route, an overpayment, no proof.
		for _, d := range []struct {
			name, tok, method, path string
			body                    any
			want                    int
		}{
			{"family asks for an upload", family, "POST", proj(P, "/uploads"), map[string]any{"fileName": "a.jpg", "mimeType": "image/jpeg", "size": 10}, http.StatusForbidden},
			{"family submits", family, "POST", proj(P, "/client-payments/submissions"), body, http.StatusForbidden},
			{"other client submits here", otherBride, "POST", proj(P, "/client-payments/submissions"), body, http.StatusForbidden},
			{"other client uploads here", otherBride, "POST", proj(P, "/uploads"), map[string]any{"fileName": "a.jpg", "mimeType": "image/jpeg", "size": 10}, http.StatusForbidden},
			{"staff on the client route", tn.token, "POST", proj(P, "/client-payments/submissions"), body, http.StatusForbidden},
			{"client records a staff payment", bride, "POST", proj(P, "/client-payments"), clientPayBody(1000, "x"), http.StatusForbidden},
			{"client verifies", bride, "POST", proj(P, fmt.Sprintf("/client-payments/%d/verify", payID)), map[string]any{}, http.StatusForbidden},
			{"client deletes", bride, "DELETE", proj(P, fmt.Sprintf("/client-payments/%d", payID)), nil, http.StatusForbidden},
		} {
			if r := s.do(d.tok, d.method, d.path, d.body); r.status != d.want {
				t.Errorf("%s: want %d, got %s", d.name, d.want, r)
			}
		}
		over, _ := s.uploadFile(brideAsTenant, P, "bukti2.jpg", "image/jpeg", proofJPEG)
		r = s.do(bride, "POST", proj(P, "/client-payments/submissions"), map[string]any{"amount": 9_000_000, "paymentDate": day(0), "method": "Transfer Bank",
			"evidence": []map[string]any{proofOf(over, "Transfer Proof")}})
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "amount") {
			t.Errorf("overpayment: %s", r)
		}
		r = s.do(bride, "POST", proj(P, "/client-payments/submissions"), map[string]any{"amount": 1000, "paymentDate": day(0), "method": "Transfer Bank"})
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "evidence") {
			t.Errorf("no proof: %s", r)
		}
		if n := s.count("SELECT COUNT(*) FROM client_payments WHERE project_id = ? AND deleted_at IS NULL", P); n != 1 {
			t.Errorf("no refused submission is written: %d", n)
		}

		// Staff may delete an unverified submission (finance.payment.delete).
		token3, _ := s.uploadFile(brideAsTenant, P, "bukti3.jpg", "image/jpeg", proofJPEG)
		r = s.do(bride, "POST", proj(P, "/client-payments/submissions"), map[string]any{"amount": 1000, "paymentDate": day(-1), "method": "QRIS",
			"evidence": []map[string]any{proofOf(token3, "Transfer Proof")}})
		if r.status != http.StatusCreated || str(r, "data", "type") != "Termin" {
			t.Fatalf("second submission: %s", r)
		}
		if d := s.do(tn.token, "DELETE", proj(P, fmt.Sprintf("/client-payments/%d", r.id("data", "id"))), nil); d.status != http.StatusOK {
			t.Errorf("staff delete an unverified submission: %s", d)
		}
	})

	t.Run("tasks: staff flag a step, the couple completes it", func(t *testing.T) {
		routesCovered += 3
		task := s.mustOK("task", s.do(tn.token, "POST", proj(P, "/milestones"), map[string]any{"name": "Kirim daftar tamu " + tn.marker, "targetDate": day(10)})).id("data", "id")
		staffStep := s.mustOK("staff step", s.do(tn.token, "POST", proj(P, "/milestones"), map[string]any{"name": "Survei venue " + tn.marker, "targetDate": day(5)})).id("data", "id")
		r := s.doFresh(tn.token, "PATCH", proj(P, fmt.Sprintf("/milestones/%d", task)), map[string]any{"forClient": true})
		if r.status != http.StatusOK || r.dig("data", "forClient") != true || r.dig("data", "completedByClient") != false {
			t.Fatalf("PATCH forClient: %s", r)
		}
		home := s.do(bride, "GET", "/api/v1/projects/me/home", nil)
		tasks := rows(home, "data", "clientTasks")
		if len(tasks) != 1 || int64(tasks[0]["id"].(float64)) != task || tasks[0]["status"] != "Not Started" || tasks[0]["completedDate"] != nil {
			t.Errorf("home clientTasks: %v", tasks)
		}
		// Refusals first: a staff step, a Keluarga client, another workspace's client.
		if r := s.do(bride, "POST", proj(P, fmt.Sprintf("/milestones/%d/complete", staffStep)), nil); r.status != http.StatusForbidden || !strings.Contains(string(r.body), "not_client_task") {
			t.Errorf("a staff step: %s", r)
		}
		if r := s.do(family, "POST", proj(P, fmt.Sprintf("/milestones/%d/complete", task)), nil); r.status != http.StatusForbidden {
			t.Errorf("family: %s", r)
		}
		if r := s.do(otherBride, "POST", proj(P, fmt.Sprintf("/milestones/%d/complete", task)), nil); r.status != http.StatusForbidden {
			t.Errorf("other workspace's client: %s", r)
		}
		if r := s.do(tn.token, "POST", proj(P, fmt.Sprintf("/milestones/%d/complete", task)), nil); r.status != http.StatusForbidden {
			t.Errorf("staff on the client route: %s", r)
		}
		r = s.do(bride, "POST", proj(P, fmt.Sprintf("/milestones/%d/complete", task)), nil)
		if r.status != http.StatusOK || str(r, "data", "status") != "Completed" || r.dig("data", "completedByClient") != true || str(r, "data", "completedDate") != time.Now().In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02") {
			t.Fatalf("complete: %s", r)
		}
		version := str(r, "data", "updatedAt")
		again := s.do(bride, "POST", proj(P, fmt.Sprintf("/milestones/%d/complete", task)), nil)
		if again.status != http.StatusOK || str(again, "data", "updatedAt") != version {
			t.Errorf("idempotent: %s", again)
		}
		if n := s.count("SELECT COUNT(*) FROM activity_log WHERE project_id = ? AND description = ?", P, "Diselesaikan oleh pasangan: Kirim daftar tamu "+tn.marker); n != 1 {
			t.Errorf("one activity entry: %d", n)
		}
		staffView := rowByID(rows(s.do(tn.token, "GET", proj(P, "/milestones"), nil), "data"), task)
		if staffView == nil || staffView["completedByClient"] != true {
			t.Errorf("staff see who completed it: %v", staffView)
		}
		// Staff reopen it: no longer "done by the couple".
		r = s.doFresh(tn.token, "PATCH", proj(P, fmt.Sprintf("/milestones/%d", task)), map[string]any{"status": "In Progress"})
		if r.status != http.StatusOK || r.dig("data", "completedByClient") != false || r.dig("data", "completedDate") != nil {
			t.Errorf("reopened: %s", r)
		}
	})

	t.Run("contact.whatsapp falls back from the PIC to the business number", func(t *testing.T) {
		home := s.do(bride, "GET", "/api/v1/projects/me/home", nil)
		if pic := str(home, "data", "pic", "phone"); pic != "" && str(home, "data", "contact", "whatsapp") != pic {
			t.Errorf("the PIC's phone comes first: %s", home)
		}
	})
}
