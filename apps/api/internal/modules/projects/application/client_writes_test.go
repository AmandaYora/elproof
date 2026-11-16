package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// ---- ADR-0047 §B: the couple sends a transfer proof ----

func submission(amount int64) ClientSubmissionInput {
	return ClientSubmissionInput{Amount: amount, PaymentDate: payDay, Method: "Transfer Bank", ReferenceNumber: "TRF-9", Notes: "Termin dari Ayu"}
}

// nowOnPayDay is "today" in Jakarta on payDay.
var nowOnPayDay = time.Date(2026, 11, 2, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))

func TestClientSubmissionIsOneUnverifiedClientPayment(t *testing.T) {
	f := newAtomicFixture(fakeContracts{project: 10_000_000}, nil)
	tok := f.token(t, 10, "bukti")
	res, err := f.client.SubmitByClient(context.Background(), 1, 10, 5, submission(4_000_000),
		[]ProofItem{{UploadToken: tok, Type: domain.EvidenceTransferProof}}, nowOnPayDay)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Payment
	if p.SubmittedByClientID != 5 || p.RecordedByStaffID != 0 || p.IsVerified() || p.Source() != domain.SourceClient {
		t.Errorf("an unverified client payment, recordedBy empty: %+v", p)
	}
	if p.Type != domain.PaymentDP {
		t.Errorf("the first payment toward the contract is the DP: %s", p.Type)
	}
	if len(f.db.clientPays) != 1 || len(f.db.files.rows) != 1 {
		t.Fatalf("one payment and one proof: %d / %d", len(f.db.clientPays), len(f.db.files.rows))
	}
	e := f.db.files.rows[0]
	if e.RelatedKind != domain.RelatedClientPayment || e.RelatedID != p.ID || e.UploadedByStaffID != 0 || e.Type != domain.EvidenceTransferProof {
		t.Errorf("the proof hangs off the payment, uploaded by no staff member: %+v", e)
	}
	var found bool
	for _, a := range f.db.activity.entries {
		if a.Type == domain.ActivityPaymentRecorded {
			found = true
			if a.ActorStaffID != 0 || a.Description != "Pasangan mengirim bukti pembayaran Rp4.000.000" {
				t.Errorf("activity: %+v", a)
			}
		}
	}
	if !found {
		t.Error("the submission is in the activity log")
	}
	if len(f.tx.locked) != 1 || f.tx.locked[0] != 10 || f.tx.began != 1 {
		t.Errorf("one transaction holding the project lock (ADR-0033): %+v", f.tx)
	}
	if res.Ledger.Paid != 4_000_000 {
		t.Errorf("an unverified payment counts on the ledger like a staff one: %+v", res.Ledger)
	}
}

func TestClientSubmissionPicksItsType(t *testing.T) {
	f := newAtomicFixture(fakeContracts{project: 10_000_000}, nil)
	f.db.clientPays = []domain.ClientPayment{{ID: 90, ProjectID: 10, Type: domain.PaymentDP, Amount: 3_000_000}}
	res, err := f.client.SubmitByClient(context.Background(), 1, 10, 5, submission(2_000_000),
		[]ProofItem{{UploadToken: f.token(t, 10, "a"), Type: domain.EvidenceTransferProof}}, nowOnPayDay)
	if err != nil || res.Payment.Type != domain.PaymentTermin {
		t.Fatalf("a payment between the DP and the last one is a Termin: %v %+v", err, res)
	}
	res, err = f.client.SubmitByClient(context.Background(), 1, 10, 5, submission(5_000_000),
		[]ProofItem{{UploadToken: f.token(t, 10, "b"), Type: domain.EvidenceTransferProof}}, nowOnPayDay)
	if err != nil || res.Payment.Type != domain.PaymentPelunasan {
		t.Fatalf("the amount that settles the contract is the Pelunasan: %v %+v", err, res)
	}
}

func TestClientSubmissionRefusesWhatStaffWouldRefuse(t *testing.T) {
	ctx := context.Background()
	f := newAtomicFixture(fakeContracts{project: 10_000_000}, nil)
	f.db.clientPays = []domain.ClientPayment{{ID: 90, ProjectID: 10, Type: domain.PaymentDP, Amount: 8_000_000}}
	before := len(f.db.clientPays)

	// Overpayment: the same rule as staff, rolled back, a message without the
	// staff-only "pilih Tambahan" hint.
	_, err := f.client.SubmitByClient(ctx, 1, 10, 5, submission(3_000_000),
		[]ProofItem{{UploadToken: f.token(t, 10, "x"), Type: domain.EvidenceTransferProof}}, nowOnPayDay)
	ae, ok := apperror.As(err)
	if !ok || ae.Code != "payment_exceeds_contract" || len(ae.Fields["amount"]) == 0 || strings.Contains(ae.Message, "Tambahan") {
		t.Fatalf("overpayment: %v", err)
	}
	if len(f.db.clientPays) != before || len(f.db.files.rows) != 0 || f.tx.rolledBack != 1 {
		t.Errorf("nothing written: %d payments, %d files, %d rollbacks", len(f.db.clientPays), len(f.db.files.rows), f.tx.rolledBack)
	}

	cases := []struct {
		name  string
		in    ClientSubmissionInput
		items []ProofItem
		key   string
	}{
		{"no proof", submission(1000), nil, "evidence"},
		{"zero amount", submission(0), []ProofItem{{UploadToken: f.token(t, 10, "z"), Type: domain.EvidenceTransferProof}}, "amount"},
		{"no method", func() ClientSubmissionInput { s := submission(1000); s.Method = "  "; return s }(),
			[]ProofItem{{UploadToken: f.token(t, 10, "m"), Type: domain.EvidenceTransferProof}}, "method"},
		{"future date", func() ClientSubmissionInput { s := submission(1000); s.PaymentDate = payDay.AddDate(0, 0, 1); return s }(),
			[]ProofItem{{UploadToken: f.token(t, 10, "d"), Type: domain.EvidenceTransferProof}}, "paymentDate"},
		{"long reference", func() ClientSubmissionInput {
			s := submission(1000)
			s.ReferenceNumber = strings.Repeat("9", 101)
			return s
		}(),
			[]ProofItem{{UploadToken: f.token(t, 10, "r"), Type: domain.EvidenceTransferProof}}, "referenceNumber"},
		{"invoice is not a transfer proof", submission(1000), []ProofItem{{UploadToken: f.token(t, 10, "i"), Type: domain.EvidenceInvoice}}, "evidence[0].type"},
		{"tampered token", submission(1000), []ProofItem{{UploadToken: "nope", Type: domain.EvidenceTransferProof}}, "evidence[0].uploadToken"},
		{"another project's upload", submission(1000), []ProofItem{{UploadToken: f.token(t, 11, "o"), Type: domain.EvidenceTransferProof}}, "evidence[0].uploadToken"},
	}
	for _, c := range cases {
		_, err := f.client.SubmitByClient(ctx, 1, 10, 5, c.in, c.items, nowOnPayDay)
		if fields := fieldsOf(t, err); len(fields[c.key]) == 0 {
			t.Errorf("%s: want a 422 on %q, got %v", c.name, c.key, fields)
		}
	}
	if len(f.db.clientPays) != before {
		t.Errorf("no refused submission is written")
	}

	if _, err := f.client.SubmitByClient(ctx, 1, 10, 0, submission(1000), nil, nowOnPayDay); apperror.HTTPStatus(err) != 403 {
		t.Errorf("no client, no submission: %v", err)
	}
}

// ---- ADR-0047 §C: the couple marks a task done ----

func clientTaskFixture(t *testing.T) (*ProjectService, *followDB, *memActivity) {
	t.Helper()
	svc, db, mem := newFollow()
	db.project = domain.Project{ID: 1, TenantID: 1, Name: "Ayu & Bima", Status: domain.StatusPreparation}
	db.milestones = []domain.ProjectMilestone{
		{ID: 1, ProjectID: 1, Name: "Kirim daftar tamu", Status: domain.MilestoneNotStarted, ForClient: true},
		{ID: 2, ProjectID: 1, Name: "Survei venue", Status: domain.MilestoneInProgress},
		{ID: 3, ProjectID: 1, Name: "Pilih menu", Status: domain.MilestoneCancelled, ForClient: true},
	}
	return svc, db, mem
}

func TestClientCompletesOnlyTheirOwnTasks(t *testing.T) {
	ctx := context.Background()
	svc, db, mem := clientTaskFixture(t)

	m, err := svc.CompleteMilestoneByClient(ctx, 1, 1, 1, 5, nowOnPayDay)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != domain.MilestoneCompleted || m.CompletedByClientID != 5 || m.CompletedDate == nil || m.CompletedDate.Format("2006-01-02") != "2026-11-02" {
		t.Errorf("completed today (WIB) by the client: %+v", m)
	}
	if len(db.patched) != 1 || db.patched[0] != "1:status,completedDate,completedByClientId" {
		t.Errorf("one write of exactly those columns: %v", db.patched)
	}
	if n := len(mem.entries); n != 1 || mem.entries[0].Description != "Diselesaikan oleh pasangan: Kirim daftar tamu" || mem.entries[0].ActorStaffID != 0 {
		t.Errorf("activity: %+v", mem.entries)
	}

	// Idempotent: again is 200 unchanged, nothing written, nothing logged.
	again, err := svc.CompleteMilestoneByClient(ctx, 1, 1, 1, 6, nowOnPayDay.AddDate(0, 0, 3))
	if err != nil || again.CompletedByClientID != 5 || again.CompletedDate.Format("2006-01-02") != "2026-11-02" {
		t.Errorf("already completed: unchanged: %v %+v", err, again)
	}
	if len(db.patched) != 1 || len(mem.entries) != 1 {
		t.Errorf("an idempotent repeat writes nothing: %v %d", db.patched, len(mem.entries))
	}

	// A staff step is refused with 403 not_client_task.
	_, err = svc.CompleteMilestoneByClient(ctx, 1, 1, 2, 5, nowOnPayDay)
	if ae, ok := apperror.As(err); !ok || apperror.HTTPStatus(err) != 403 || ae.Code != "not_client_task" {
		t.Errorf("not for the client: %v", err)
	}
	// A cancelled task is refused with 422 task_cancelled.
	_, err = svc.CompleteMilestoneByClient(ctx, 1, 1, 3, 5, nowOnPayDay)
	if ae, ok := apperror.As(err); !ok || apperror.HTTPStatus(err) != 422 || ae.Code != "task_cancelled" {
		t.Errorf("cancelled: %v", err)
	}
	// Another project, another tenant, an unknown step: not found.
	for _, c := range [][3]int64{{2, 1, 1}, {1, 9, 1}, {1, 1, 99}} {
		if _, err := svc.CompleteMilestoneByClient(ctx, c[0], c[1], c[2], 5, nowOnPayDay); apperror.HTTPStatus(err) != 404 {
			t.Errorf("tenant %d project %d step %d: want 404, got %v", c[0], c[1], c[2], err)
		}
	}
	if len(db.patched) != 1 {
		t.Errorf("no refused completion is written: %v", db.patched)
	}
}

func TestStaffSetsForClientAndTheirStatusClearsTheClientMark(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := clientTaskFixture(t)

	m, err := svc.PatchMilestone(ctx, 1, 1, 2, 10, MilestonePatch{ForClient: Some(true)}, nil, nowOnPayDay)
	if err != nil || !m.ForClient {
		t.Fatalf("forClient is patchable: %v %+v", err, m)
	}
	if db.patched[len(db.patched)-1] != "2:forClient" {
		t.Errorf("only forClient written: %v", db.patched)
	}

	if _, err := svc.CompleteMilestoneByClient(ctx, 1, 1, 1, 5, nowOnPayDay); err != nil {
		t.Fatal(err)
	}
	// Staff reopening the task: it is no longer "done by the couple".
	m, err = svc.PatchMilestone(ctx, 1, 1, 1, 10, MilestonePatch{Status: Some(domain.MilestoneInProgress)}, nil, nowOnPayDay)
	if err != nil || m.CompletedByClientID != 0 || m.CompletedDate != nil {
		t.Fatalf("reopened by staff: %v %+v", err, m)
	}
	if got := db.patched[len(db.patched)-1]; got != "1:status,completedDate,completedByClientId" {
		t.Errorf("the client mark is cleared in the same write: %s", got)
	}
}

// ---- ADR-0047 §A: the cover photo ----

type memCovers struct {
	path  *string
	at    time.Time
	found bool
	sets  int
}

func (m *memCovers) LockCover(context.Context, int64, int64) (*string, bool, error) {
	return m.path, m.found, nil
}
func (m *memCovers) SetCover(_ context.Context, _, _ int64, path *string, at time.Time) error {
	m.path, m.at = path, at
	m.sets++
	return nil
}

func coverFixture(t *testing.T) (*CoverService, *followDB, *memCovers, *objStorage, *memActivity) {
	t.Helper()
	svc, db, mem := newFollow()
	db.project = domain.Project{ID: 1, TenantID: 1, Name: "Ayu & Bima"}
	covers := &memCovers{found: true}
	store := newObjStorage(false)
	cs := NewCoverService(svc, covers, nil, store, buildKeyForTests, NewActivityService(activityRepoAdapter{mem}))
	cs.now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	return cs, db, covers, store, mem
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func coverInput(data []byte, mime string) CoverInput {
	return CoverInput{FileName: "sampul.jpg", MimeType: mime, Base64Data: base64.StdEncoding.EncodeToString(data)}
}

func TestCoverReplacesAndDeletesThePreviousObject(t *testing.T) {
	ctx := context.Background()
	cs, _, covers, store, mem := coverFixture(t)

	if _, err := cs.Set(ctx, 1, 1, 10, coverInput(testJPEG(t, 2400, 1200, 90), "image/jpeg")); err != nil {
		t.Fatal(err)
	}
	if covers.path == nil || !strings.HasPrefix(*covers.path, "up/1/1/cover/1800000000-") || !strings.HasSuffix(*covers.path, ".jpg") {
		t.Fatalf("stored under the project's cover key: %v", covers.path)
	}
	first := *covers.path
	stored, _, err := image.DecodeConfig(bytes.NewReader(store.objs[first]))
	if err != nil || stored.Width != 2000 || stored.Height != 1000 || store.types[first] != "image/jpeg" {
		t.Errorf("re-encoded to at most 2000 px on the longest side: %+v %v %s", stored, err, store.types[first])
	}

	if _, err := cs.Set(ctx, 1, 1, 10, coverInput(testPNG(t, 64, 32), "image/png")); err != nil {
		t.Fatal(err)
	}
	if *covers.path == first || !strings.HasSuffix(*covers.path, ".png") || store.types[*covers.path] != "image/png" {
		t.Errorf("a PNG stays PNG under a new key: %s", *covers.path)
	}
	if _, still := store.objs[first]; still || len(store.deleted) != 1 || store.deleted[0] != first {
		t.Errorf("the replaced object is deleted: %v", store.deleted)
	}

	if err := cs.Remove(ctx, 1, 1, 10); err != nil {
		t.Fatal(err)
	}
	if covers.path != nil || len(store.deleted) != 2 {
		t.Errorf("removed: path %v, deleted %v", covers.path, store.deleted)
	}
	descs := []string{}
	for _, a := range mem.entries {
		descs = append(descs, a.Description)
	}
	if strings.Join(descs, "|") != "Foto sampul portal diunggah|Foto sampul portal diganti|Foto sampul portal dihapus" {
		t.Errorf("activity: %v", descs)
	}
	// Removing again is a no-op.
	if err := cs.Remove(ctx, 1, 1, 10); err != nil || len(store.deleted) != 2 || len(mem.entries) != 3 {
		t.Errorf("remove without a cover: %v %v", err, store.deleted)
	}
}

func TestCoverRefusesWhatIsNotAPhoto(t *testing.T) {
	ctx := context.Background()
	cs, _, covers, store, _ := coverFixture(t)
	jpg := testJPEG(t, 8, 8, 80)
	big := append(append([]byte(nil), jpg...), make([]byte, maxCoverDecodedSize)...)
	cases := []struct {
		name string
		in   CoverInput
		key  string
	}{
		{"a PDF", coverInput([]byte("%PDF-1.4 x"), "application/pdf"), "mimeType"},
		{"a HEIC", coverInput(jpg, "image/heic"), "mimeType"},
		{"PNG bytes declared JPEG", coverInput(testPNG(t, 4, 4), "image/jpeg"), "base64Data"},
		{"a renamed binary", coverInput([]byte("MZ\x90\x00not a photo at all"), "image/png"), "base64Data"},
		{"empty", CoverInput{MimeType: "image/jpeg"}, "base64Data"},
		{"not base64", CoverInput{MimeType: "image/jpeg", Base64Data: "%%%"}, "base64Data"},
		{"past 6 MB", coverInput(big, "image/jpeg"), "base64Data"},
	}
	for _, c := range cases {
		_, err := cs.Set(ctx, 1, 1, 10, c.in)
		if fields := fieldsOf(t, err); len(fields[c.key]) == 0 {
			t.Errorf("%s: want a 422 on %q, got %v", c.name, c.key, fields)
		}
	}
	if covers.sets != 0 || len(store.objs) != 0 {
		t.Errorf("nothing stored: %d sets, %d objects", covers.sets, len(store.objs))
	}

	// Another tenant's project: 404 before anything is stored.
	if _, err := cs.Set(ctx, 2, 1, 10, coverInput(jpg, "image/jpeg")); apperror.HTTPStatus(err) != 404 || len(store.objs) != 0 {
		t.Errorf("other tenant: %v", err)
	}
	// The row vanished between the check and the write: the new object is cleaned up.
	covers.found = false
	if _, err := cs.Set(ctx, 1, 1, 10, coverInput(jpg, "image/jpeg")); apperror.HTTPStatus(err) != 404 || len(store.objs) != 0 {
		t.Errorf("vanished project: %v, objects %d", err, len(store.objs))
	}
}

func TestCoverOpen(t *testing.T) {
	ctx := context.Background()
	cs, db, _, store, _ := coverFixture(t)
	if _, _, err := cs.Open(ctx, 1, 1); apperror.HTTPStatus(err) != 404 {
		t.Errorf("no cover: 404, got %v", err)
	}
	key := "up/1/1/cover/1-a.png"
	store.objs[key] = []byte("png bytes")
	db.project.CoverPhotoPath = &key
	r, ct, err := cs.Open(ctx, 1, 1)
	if err != nil || ct != "image/png" {
		t.Fatalf("open: %v %s", err, ct)
	}
	r.Close()
	if _, _, err := cs.Open(ctx, 2, 1); apperror.HTTPStatus(err) != 404 {
		t.Errorf("another tenant's cover: 404, got %v", err)
	}
}
