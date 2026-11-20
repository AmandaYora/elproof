//go:build integration

package isolation

// Fixes for the full QA report (FQ-01..FQ-08, RT-07, RT-08): each scenario is
// the report's reproduction, run against the real server.

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestQAFindings(t *testing.T) {
	s := startStack(t)
	tn := s.signUp("qa")
	proj := s.projectsPath

	// FQ-01: a project that ever had payments to its vendor and its venue (live
	// and soft-deleted, with files), a deleted supplier and a client payment is
	// hard-deleted with everything it owns: events and sessions, suppliers and
	// their steps, supplier and client payments, kendala, files.
	t.Run("FQ-01 hard delete removes every payment and file", func(t *testing.T) {
		needStorage(t)
		venueID := s.mustOK("venue", s.do(tn.token, "POST", "/api/v1/venues", venueBody(tn, "Gedung Hapus "+tn.marker))).id("data", "id")
		P := s.projectAt(tn, "Proyek Hapus Total "+tn.marker, 50_000_000, 90, venueID)
		VS := s.venueSupplier(tn, P, venueID)
		vendorID, _ := s.newVendor(tn, "Vendor Hapus "+tn.marker)
		E := s.engage(tn, P, vendorID, "Dekor", 20_000_000, "Booked", nil)
		E2 := s.engage(tn, P, vendorID, "Tenda", 5_000_000, "Booked", nil)
		M := s.mustOK("vendor milestone", s.do(tn.token, "POST", proj(P, fmt.Sprintf("/suppliers/%d/milestones", E)), map[string]any{
			"name": "Pasang", "description": "d", "targetDate": day(30), "picStaffId": tn.staffID,
		})).id("data", "id")
		s.mustOK("issue", s.do(tn.token, "POST", proj(P, "/issues"), map[string]any{
			"projectSupplierId": E, "vendorMilestoneId": M, "title": "Kendala", "description": "d", "impact": "Medium", "foundDate": day(-1), "picStaffId": tn.staffID,
		}))
		vpLive, vpGone, vpOfE2 := s.supplierPay(tn, P, E, "DP", 1_000_000, "VP-1"), s.supplierPay(tn, P, E, "DP", 1_000_000, "VP-2"), s.supplierPay(tn, P, E2, "DP", 1_000_000, "VP-3")
		vnLive, vnGone := s.supplierPay(tn, P, VS, "DP", 1_000_000, "VN-1"), s.supplierPay(tn, P, VS, "DP", 1_000_000, "VN-2")
		cp := s.clientPay(tn, P, "DP", 2_000_000, "CP-1")
		eventID := int64(s.count("SELECT IFNULL(MAX(id), 0) FROM project_events WHERE project_id = ?", P))
		if eventID == 0 || VS == 0 {
			t.Fatalf("project without event (%d) or venue supplier (%d)", eventID, VS)
		}
		var files []int64
		files = append(files,
			s.attach(tn, P, "supplierPayment", vpLive, "Invoice", "inv-live", false),
			s.attach(tn, P, "supplierPayment", vpGone, "Transfer Proof", "bukti-gone", false),
			s.attach(tn, P, "supplierPayment", vpOfE2, "Transfer Proof", "bukti-e2", false),
			s.attach(tn, P, "supplierPayment", vnLive, "Transfer Proof", "bukti-venue", false),
			s.attach(tn, P, "supplierPayment", vnGone, "Transfer Proof", "bukti-venue-gone", false),
			s.attach(tn, P, "clientPayment", cp, "Transfer Proof", "bukti-klien", true),
			s.upload(tn, P, "Dokumen umum", true),
		)
		keys := make([]string, 0, len(files))
		for _, f := range files {
			keys = append(keys, s.storageKey(f))
		}
		s.mustOK("delete vendor payment", s.do(tn.token, "DELETE", proj(P, fmt.Sprintf("/supplier-payments/%d", vpGone)), nil))
		s.mustOK("delete venue payment", s.do(tn.token, "DELETE", proj(P, fmt.Sprintf("/supplier-payments/%d", vnGone)), nil))
		s.mustOK("delete supplier", s.do(tn.token, "DELETE", proj(P, fmt.Sprintf("/suppliers/%d", E2)), nil))
		s.setTerms(tn, P, term("DP", 10_000_000, 10))

		// Not cancelled or archived yet: a clear 422, not a 500.
		if r := s.do(tn.token, "DELETE", proj(P, ""), nil); r.status != http.StatusUnprocessableEntity {
			t.Fatalf("delete of a running project: want 422, got %s", r)
		}
		s.mustOK("cancel", s.do(tn.token, "POST", proj(P, "/cancel"), map[string]any{}))
		s.mustOK("archive", s.do(tn.token, "PATCH", proj(P, ""), map[string]any{"isArchived": true}))

		r := s.do(tn.token, "DELETE", proj(P, ""), nil)
		if r.status != http.StatusOK {
			t.Fatalf("hard delete: want 200, got %s", r)
		}
		if got := str(r, "message"); got != "Proyek berhasil dihapus permanen" {
			t.Errorf("hard delete message: %q", got)
		}
		for _, table := range []string{"projects WHERE id = ?", "project_suppliers WHERE project_id = ?", "supplier_payments WHERE project_id = ?",
			"project_events WHERE project_id = ?", "client_payments WHERE project_id = ?", "client_payment_terms WHERE project_id = ?",
			"vendor_issues WHERE project_id = ?", "evidence WHERE project_id = ?", "project_milestones WHERE project_id = ?", "activity_log WHERE project_id = ?"} {
			if n := s.count("SELECT COUNT(*) FROM "+table, P); n != 0 {
				t.Errorf("%s: %d rows left", table, n)
			}
		}
		if n := s.count("SELECT COUNT(*) FROM vendor_milestones WHERE project_supplier_id IN (?, ?)", E, E2); n != 0 {
			t.Errorf("vendor_milestones: %d rows left", n)
		}
		if n := s.count("SELECT COUNT(*) FROM project_event_sessions WHERE event_id = ?", eventID); n != 0 {
			t.Errorf("project_event_sessions: %d rows left", n)
		}
		for _, k := range keys {
			if objectExists(t, k) {
				t.Errorf("stored object %s survived the delete", k)
			}
		}
		if r := s.do(tn.token, "GET", proj(P, ""), nil); r.status != http.StatusNotFound {
			t.Errorf("deleted project: want 404, got %s", r)
		}
	})

	// FQ-02: a vendor or venue a project still uses cannot be deleted; once the
	// project is gone (or the supplier removed from it) it can.
	t.Run("FQ-02 directory delete refused while in use", func(t *testing.T) {
		vendorID, _ := s.newVendor(tn, "Vendor Dipakai "+tn.marker)
		venueID := s.mustOK("venue", s.do(tn.token, "POST", "/api/v1/venues", venueBody(tn, "Gedung Dipakai "+tn.marker))).id("data", "id")
		PA := s.projectAt(tn, "Proyek Alfa "+tn.marker, 10_000_000, 60, venueID)
		PB := s.projectAt(tn, "Proyek Beta "+tn.marker, 10_000_000, 90, 0)
		ea := s.engage(tn, PA, vendorID, "Dekor", 1_000_000, "Booked", nil)
		s.engage(tn, PB, vendorID, "Dekor", 1_000_000, "Planned", nil)
		s.mustOK("cancel B", s.do(tn.token, "POST", proj(PB, "/cancel"), map[string]any{}))
		s.mustOK("archive B", s.do(tn.token, "PATCH", proj(PB, ""), map[string]any{"isArchived": true}))

		r := s.do(tn.token, "DELETE", fmt.Sprintf("/api/v1/vendors/%d", vendorID), nil)
		msg := str(r, "message")
		if r.status != http.StatusConflict || str(r, "errors", "code") != "vendor_in_use" ||
			!strings.HasPrefix(msg, "Vendor ini masih dipakai di 2 proyek: ") || !strings.Contains(msg, "Proyek Alfa "+tn.marker) ||
			!strings.Contains(msg, "Proyek Beta "+tn.marker) || !strings.HasSuffix(msg, "Nonaktifkan saja bila tidak dipakai lagi.") ||
			len(rows(r, "data", "affectedProjects")) != 2 {
			t.Fatalf("vendor in use (an archived, cancelled project counts too): %s", r)
		}
		if n := s.count("SELECT COUNT(*) FROM vendors WHERE id = ?", vendorID); n != 1 {
			t.Fatal("the refused vendor was deleted")
		}
		// The project's supplier row still shows the vendor's name.
		for _, row := range rows(s.mustOK("suppliers", s.do(tn.token, "GET", proj(PA, "/suppliers"), nil)), "data") {
			if int64(row["id"].(float64)) == ea && row["name"] != "Vendor Dipakai "+tn.marker {
				t.Errorf("supplier lost its vendor name: %v", row)
			}
		}

		r = s.do(tn.token, "DELETE", fmt.Sprintf("/api/v1/venues/%d", venueID), nil)
		if r.status != http.StatusConflict || str(r, "errors", "code") != "venue_in_use" ||
			str(r, "message") != "Venue ini masih dipakai di 1 proyek: Proyek Alfa "+tn.marker+". Nonaktifkan saja bila tidak dipakai lagi." {
			t.Fatalf("venue in use: %s", r)
		}
		if r := s.do(tn.token, "GET", fmt.Sprintf("/api/v1/venues/%d", venueID), nil); r.status != http.StatusOK {
			t.Errorf("refused venue must still be readable: %s", r)
		}

		// Remove the uses: delete the live supplier of A, hard-delete B, then A.
		s.mustOK("delete supplier", s.do(tn.token, "DELETE", proj(PA, fmt.Sprintf("/suppliers/%d", ea)), nil))
		s.mustOK("delete B", s.do(tn.token, "DELETE", proj(PB, ""), nil))
		if r := s.do(tn.token, "DELETE", fmt.Sprintf("/api/v1/vendors/%d", vendorID), nil); r.status != http.StatusOK {
			t.Errorf("vendor no longer used: want 200, got %s", r)
		}
		s.mustOK("cancel A", s.do(tn.token, "POST", proj(PA, "/cancel"), map[string]any{}))
		s.mustOK("delete A", s.do(tn.token, "DELETE", proj(PA, ""), nil))
		if r := s.do(tn.token, "DELETE", fmt.Sprintf("/api/v1/venues/%d", venueID), nil); r.status != http.StatusOK {
			t.Errorf("venue no longer used: want 200, got %s", r)
		}
	})

	// FQ-03: category names are unique per workspace, case-insensitive after trim.
	t.Run("FQ-03 duplicate category name", func(t *testing.T) {
		first := s.mustOK("category", s.do(tn.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": "Bunga Segar", "description": "d"})).id("data", "id")
		for _, name := range []string{"Bunga Segar", "  bunga segar ", "BUNGA SEGAR"} {
			r := s.do(tn.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": name, "description": "d"})
			if r.status != http.StatusUnprocessableEntity || fmt.Sprint(r.dig("errors", "name")) != "[Kategori dengan nama ini sudah ada.]" {
				t.Errorf("create %q: %s", name, r)
			}
		}
		second := s.mustOK("category 2", s.do(tn.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": "Kue Pengantin", "description": "d"})).id("data", "id")
		r := s.do(tn.token, "PATCH", fmt.Sprintf("/api/v1/vendor-categories/%d", second), map[string]any{"name": "bunga segar", "description": "d"})
		if r.status != http.StatusUnprocessableEntity || !hasKey(r, "name") {
			t.Errorf("rename onto another: %s", r)
		}
		s.mustOK("own name, other case", s.do(tn.token, "PATCH", fmt.Sprintf("/api/v1/vendor-categories/%d", first), map[string]any{"name": "BUNGA SEGAR", "description": "baru"}))
		// Another workspace may use the same name.
		other := s.signUp("qa2")
		s.mustOK("same name elsewhere", s.do(other.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": "Bunga Segar", "description": "d"}))
	})

	// FQ-04: Jadwal bawaan offsets are at most two years either side.
	t.Run("FQ-04 template days bounded", func(t *testing.T) {
		for _, days := range []int{99999, 731, -731} {
			r := s.do(tn.token, "POST", "/api/v1/milestone-templates", map[string]any{"name": "Survey lokasi", "daysBeforeEvent": days})
			if r.status != http.StatusUnprocessableEntity || !hasKey(r, "daysBeforeEvent") || !strings.HasPrefix(str(r, "message"), "Jumlah hari maksimal 730 (2 tahun)") {
				t.Errorf("%d days: %s", days, r)
			}
		}
		id := s.mustOK("730 days", s.do(tn.token, "POST", "/api/v1/milestone-templates", map[string]any{"name": "Survey lokasi", "daysBeforeEvent": 730})).id("data", "id")
		if r := s.do(tn.token, "PATCH", fmt.Sprintf("/api/v1/milestone-templates/%d", id), map[string]any{"name": "Survey lokasi", "daysBeforeEvent": 5000}); r.status != http.StatusUnprocessableEntity {
			t.Errorf("update past the bound: %s", r)
		}
		s.mustOK("H+730", s.do(tn.token, "PATCH", fmt.Sprintf("/api/v1/milestone-templates/%d", id), map[string]any{"name": "Kirim album", "daysBeforeEvent": -730}))
	})

	// FQ-06: the venue PIC phone is required by the API, as by the form.
	t.Run("FQ-06 venue PIC phone required", func(t *testing.T) {
		body := venueBody(tn, "Gedung Tanpa Telepon "+tn.marker)
		for _, phone := range []string{"", "   ", "12345"} {
			body["phonePic"] = phone
			r := s.do(tn.token, "POST", "/api/v1/venues", body)
			if r.status != http.StatusUnprocessableEntity || fmt.Sprint(r.dig("errors", "phonePic")) != "[Isi nomor telepon PIC, minimal 6 angka]" {
				t.Errorf("create with phonePic %q: %s", phone, r)
			}
		}
		body["phonePic"] = "081233334444"
		id := s.mustOK("venue with phone", s.do(tn.token, "POST", "/api/v1/venues", body)).id("data", "id")
		body["phonePic"] = ""
		if r := s.do(tn.token, "PATCH", fmt.Sprintf("/api/v1/venues/%d", id), body); r.status != http.StatusUnprocessableEntity || !hasKey(r, "phonePic") {
			t.Errorf("update clearing the phone: %s", r)
		}
	})

	// FQ-07: a refused import row names each column and its problem.
	t.Run("FQ-07 import errors per column", func(t *testing.T) {
		catName := "Katering QA " + tn.marker
		s.mustOK("category", s.do(tn.token, "POST", "/api/v1/vendor-categories", map[string]any{"name": catName, "description": "d"}))
		file := importWorkbook(t, [][]any{
			{"Nama Vendor", "Kategori", "Nama PIC", "No Tlp Vendor", "Email", "Sosial Media", "Kota", "Alamat", "Harga Akad", "Harga Akad+Resepsi", "Catatan"},
			{"Vendor Baik " + tn.marker, catName, "Ayu", "0812", "", "", "Kota Bandung", "", 1000000, 2000000, ""},
			{"Vendor Salah " + tn.marker, "Tidak Ada", "Ayu", "0812", "", "", "Kota Bandung", "", -5, "dua juta", ""},
			{"", catName, "", "0812", "", "", "Gotham", "", 1000000, 2000000, ""},
		})
		r := s.mustOK("import", s.do(tn.token, "POST", "/api/v1/vendors/import", map[string]any{"base64Data": file}))
		if num(r, "data", "insertedCount") != 1 {
			t.Fatalf("one good row: %s", r)
		}
		errs := rows(r, "data", "errors")
		if len(errs) != 2 {
			t.Fatalf("two refused rows: %s", r)
		}
		cols := func(row map[string]any) []string {
			var out []string
			for _, c := range row["errors"].([]any) {
				m := c.(map[string]any)
				out = append(out, m["column"].(string)+": "+m["message"].(string))
			}
			return out
		}
		want3 := []string{`Kategori: Kategori "Tidak Ada" tidak ditemukan. Pilih dari daftar kategori di template.`,
			"Harga Akad: Harga Akad tidak boleh negatif", "Harga Akad+Resepsi: Harga Akad+Resepsi harus berupa angka bulat"}
		if errs[0]["row"] != float64(3) || strings.Join(cols(errs[0]), "|") != strings.Join(want3, "|") ||
			!strings.HasPrefix(errs[0]["message"].(string), `Kategori "Tidak Ada" tidak ditemukan`) {
			t.Errorf("row 3: %v", errs[0])
		}
		want4 := []string{"Nama Vendor: Nama Vendor wajib diisi", "Nama PIC: Nama PIC wajib diisi", `Kota: Kota "Gotham" tidak ada di daftar kota. Pilih dari daftar di template.`}
		if errs[1]["row"] != float64(4) || strings.Join(cols(errs[1]), "|") != strings.Join(want4, "|") {
			t.Errorf("row 4: %v", errs[1])
		}

		venues := importWorkbook(t, [][]any{
			{"Nama Venue", "Nama PIC", "No Tlp PIC", "No Tlp Venue", "Email", "Alamat", "Kota", "Kapasitas Berdiri", "Kapasitas Duduk", "Harga Akad", "Harga Resepsi", "Harga Akad+Resepsi", "Charge", "Fasilitas", "Sosial Media", "Catatan"},
			{"Gedung QA " + tn.marker, "Budi", "123", "", "", "", "Kota Bandung", 300, "", "", "", "sepuluh", -1, "", "", ""},
		})
		v := s.mustOK("venue import", s.do(tn.token, "POST", "/api/v1/venues/import", map[string]any{"base64Data": venues}))
		if verrs := rows(v, "data", "errors"); len(verrs) != 1 || len(verrs[0]["errors"].([]any)) != 3 {
			t.Errorf("venue row errors per column: %s", v)
		}
	})

	// RT-07: a proof the couple sent from the portal names them in the Dokumen tab.
	t.Run("RT-07 client upload names the client", func(t *testing.T) {
		needStorage(t)
		P := s.createProject(tn, "Proyek Bukti "+tn.marker, 10_000_000)
		bride := s.addClient(tn, P, "Bride", "Pengantin", fmt.Sprintf("pengantin.qa.%d", time.Now().UnixNano()%1_000_000_000))
		token, _ := s.uploadFile(tenant{token: bride}, P, "bukti.jpg", "image/jpeg", coverJPEG(t, 32, 32))
		sub := s.do(bride, "POST", proj(P, "/client-payments/submissions"), map[string]any{
			"amount": 1_000_000, "paymentDate": day(0), "method": "Transfer Bank", "evidence": []map[string]any{proofOf(token, "Transfer Proof")}})
		if sub.status != http.StatusCreated {
			t.Fatalf("submission: %s", sub)
		}
		staffFile := s.upload(tn, P, "Rundown", false)
		list := rows(s.mustOK("evidence", s.do(tn.token, "GET", proj(P, "/evidence"), nil)), "data")
		var fromClient map[string]any
		for _, e := range list {
			if e["relatedKind"] == "clientPayment" {
				fromClient = e
			}
		}
		if fromClient == nil || fromClient["uploadedBySource"] != "client" || !strings.HasPrefix(fromClient["uploadedByName"].(string), "Klien pengantin.qa.") ||
			fromClient["uploadedByStaffId"] != float64(0) {
			t.Errorf("client upload: %v", fromClient)
		}
		if row := rowByID(list, staffFile); row == nil || row["uploadedBySource"] != "staff" || row["uploadedByName"] == "" {
			t.Errorf("staff upload: %v", row)
		}
		// A client reading the list never sees names.
		for _, e := range rows(s.mustOK("client evidence", s.do(bride, "GET", proj(P, "/evidence"), nil)), "data") {
			if e["uploadedByName"] != "" || e["uploadedByStaffId"] != float64(0) {
				t.Errorf("client view leaks the uploader: %v", e)
			}
		}
	})
}

// importWorkbook builds an .xlsx (first sheet = rows) and returns it base64-encoded.
func importWorkbook(t *testing.T, rows [][]any) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// RT-08: a toggle in the Platform Console reads as its result and its object.
// Its own stack: platformAdmin() reseeds the database.
func TestQAAuditToggles(t *testing.T) {
	s := startStack(t)
	admin := s.platformAdmin()
	tn := s.signUp("audit")
	plan := s.createPlan(admin, elproofApp, "QA-TEST Paket", 1_000_000)
	s.mustOK("plan off", s.do(admin, "POST", fmt.Sprintf("/api/v1/plans/%d/toggle-active", plan), map[string]any{}))
	s.mustOK("plan on", s.do(admin, "POST", fmt.Sprintf("/api/v1/plans/%d/toggle-active", plan), map[string]any{}))
	s.mustOK("tenant frozen", s.do(admin, "POST", fmt.Sprintf("/api/v1/tenants/%d/toggle-suspension", tn.tenant), map[string]any{}))
	s.mustOK("tenant back", s.do(admin, "POST", fmt.Sprintf("/api/v1/tenants/%d/toggle-suspension", tn.tenant), map[string]any{}))

	descriptions := func(action string) []string {
		r := s.mustOK("audit "+action, s.do(admin, "GET", withQuery("/api/v1/audit-log", "action", action, "sort", "createdAt:asc"), nil))
		var out []string
		for _, row := range rows(r, "data") {
			out = append(out, row["description"].(string))
		}
		return out
	}
	if got, want := descriptions("POST /plans/{id}/toggle-active"), []string{"Menonaktifkan paket QA-TEST Paket", "Mengaktifkan paket QA-TEST Paket"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("plan toggles: %q", got)
	}
	business := "WO " + tn.marker
	if got, want := descriptions("POST /tenants/{id}/toggle-suspension"), []string{"Membekukan tenant " + business, "Mengaktifkan kembali tenant " + business}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("tenant toggles: %q", got)
	}
	// An older row without a stored sentence still reads from its route.
	if _, err := s.db.Exec(`UPDATE audit_log SET description = '' WHERE action = 'POST /plans/{id}/toggle-active'`); err != nil {
		t.Fatal(err)
	}
	if got := descriptions("POST /plans/{id}/toggle-active"); len(got) != 2 || got[0] != fmt.Sprintf("Menonaktifkan atau mengaktifkan paket #%d", plan) {
		t.Errorf("old rows: %q", got)
	}
}

// TestQAMigrations walks 000083..000080 down and up again over rows shaped like
// production's: English plan features and role descriptions are rewritten only
// where they still hold the exact seeded text, and a portal proof gets its client.
func TestQAMigrations(t *testing.T) {
	s := startStack(t)
	tn := s.signUp("mig")
	run := func(arg string) {
		cmd := exec.Command(s.bin, "migrate", arg)
		cmd.Dir, cmd.Env = s.dir, s.env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("migrate %s: %v\n%s", arg, err, out)
		}
	}
	exec1 := func(q string, args ...any) {
		if _, err := s.db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	version := func() int {
		var v int
		if err := s.db.QueryRow(`SELECT version FROM schema_migrations`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	// A portal proof recorded before 000080 (no uploaded_by_client_id yet).
	P := s.createProject(tn, "Proyek Migrasi "+tn.marker, 10_000_000)
	for version() > 79 {
		run("down")
	}
	exec1(`INSERT INTO client_payments (project_id, type, amount, payment_date, method, reference_number, notes, submitted_by_client_id) VALUES (?, 'DP', 1000, CURDATE(), 'Transfer', '', '', 77)`, P)
	var payID int64
	if err := s.db.QueryRow(`SELECT MAX(id) FROM client_payments WHERE project_id = ?`, P).Scan(&payID); err != nil {
		t.Fatal(err)
	}
	exec1(`INSERT INTO evidence (project_id, name, type, storage_path, file_name, uploaded_at, uploaded_by_staff_id, related_kind, related_id)
		VALUES (?, 'Bukti', 'Transfer Proof', ?, 'b.jpg', NOW(), 0, 'clientPayment', ?)`, P, fmt.Sprintf("mig/%d.jpg", time.Now().UnixNano()), payID)
	exec1(`UPDATE tenant_roles SET description = 'Mengelola semua project, client, vendor, dan keuangan. Tanpa pengaturan workspace.' WHERE tenant_id = ? AND template_key = 'admin'`, tn.tenant)
	exec1(`UPDATE tenant_roles SET description = 'Deskripsi buatan WO tentang project' WHERE tenant_id = ? AND template_key = 'sales'`, tn.tenant)
	exec1(`INSERT INTO subscription_plans (app_id, name, duration_months, price) VALUES ('platform-billing', 'Mig Paket', 12, 1)`)
	exec1(`INSERT INTO plan_features (plan_id, label, sort_order) SELECT id, 'Full Features', 1 FROM subscription_plans WHERE name = 'Mig Paket'`)
	exec1(`INSERT INTO plan_features (plan_id, label, sort_order) SELECT id, ' Full Seat ', 2 FROM subscription_plans WHERE name = 'Mig Paket'`)
	exec1(`INSERT INTO plan_features (plan_id, label, sort_order) SELECT id, 'Unlimited Project', 3 FROM subscription_plans WHERE name = 'Mig Paket'`)
	exec1(`INSERT INTO plan_features (plan_id, label, sort_order) SELECT id, 'unlimited project untuk tim', 4 FROM subscription_plans WHERE name = 'Mig Paket'`)

	run("up")
	labels := func() string {
		var out []string
		r, err := s.db.Query(`SELECT f.label FROM plan_features f JOIN subscription_plans p ON p.id = f.plan_id WHERE p.name = 'Mig Paket' ORDER BY f.sort_order`)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		for r.Next() {
			var l string
			_ = r.Scan(&l)
			out = append(out, l)
		}
		return strings.Join(out, "|")
	}
	if got := labels(); got != "Semua fitur|Anggota tim tanpa batas|Proyek tanpa batas|unlimited project untuk tim" {
		t.Errorf("plan features: %q", got)
	}
	role := func(key string) string {
		var d string
		_ = s.db.QueryRow(`SELECT description FROM tenant_roles WHERE tenant_id = ? AND template_key = ?`, tn.tenant, key).Scan(&d)
		return d
	}
	if got := role("admin"); got != "Mengelola semua proyek, klien, vendor, dan keuangan. Tanpa pengaturan workspace." {
		t.Errorf("admin description: %q", got)
	}
	if got := role("sales"); got != "Deskripsi buatan WO tentang project" {
		t.Errorf("an edited description must stay: %q", got)
	}
	if n := s.count(`SELECT COUNT(*) FROM evidence WHERE related_kind = 'clientPayment' AND related_id = ? AND uploaded_by_client_id = 77`, payID); n != 1 {
		t.Errorf("portal proof backfill: %d", n)
	}
	if n := s.count(`SELECT COUNT(*) FROM audit_log WHERE description <> ''`); n != 0 {
		t.Errorf("old audit rows keep an empty sentence: %d", n)
	}
	// Down and up once more: both directions run cleanly.
	for version() > 79 {
		run("down")
	}
	if got := labels(); got != "Full Features|Full Seat|Unlimited Project|unlimited project untuk tim" {
		t.Errorf("after down: %q", got)
	}
	run("up")
}
