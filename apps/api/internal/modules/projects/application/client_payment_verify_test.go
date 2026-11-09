package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// memEvidence is an in-memory EvidenceRepository (only what these tests touch).
type memEvidence struct {
	EvidenceRepository
	rows []domain.Evidence
	next int64
}

func (m *memEvidence) ListByRelated(_ context.Context, kind domain.EvidenceRelatedKind, id int64) ([]domain.Evidence, error) {
	var out []domain.Evidence
	for _, e := range m.rows {
		if e.RelatedKind == kind && e.RelatedID == id {
			out = append(out, e)
		}
	}
	return out, nil
}
func (m *memEvidence) FindByID(_ context.Context, projectID, id int64) (*domain.Evidence, error) {
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].ProjectID == projectID {
			e := m.rows[i]
			return &e, nil
		}
	}
	return nil, nil
}
func (m *memEvidence) DeleteByID(_ context.Context, projectID, id int64) error {
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].ProjectID == projectID {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			return nil
		}
	}
	return nil
}
func (m *memEvidence) RelatedBelongsToProject(context.Context, int64, domain.EvidenceRelatedKind, int64) (bool, error) {
	return true, nil
}
func (m *memEvidence) Create(_ context.Context, e *domain.Evidence) error {
	m.next++
	e.ID = 100 + m.next
	m.rows = append(m.rows, *e)
	return nil
}
func (m *memEvidence) SetClientVisible(_ context.Context, projectID, id int64, v bool) error {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].IsClientVisible = v
		}
	}
	return nil
}

type memActivity struct{ entries []domain.ActivityLogEntry }

func (m *memActivity) Create(_ context.Context, e *domain.ActivityLogEntry) error {
	m.entries = append(m.entries, *e)
	return nil
}

type verifyFixture struct {
	svc      *ClientPaymentService
	payments *fakeClientPayments
	evidence *EvidenceService
	files    *memEvidence
	activity *memActivity
}

func newVerifyFixture(rows ...domain.ClientPayment) *verifyFixture {
	files := &memEvidence{}
	act := &memActivity{}
	activity := NewActivityService(activityRepoAdapter{act})
	ev := NewEvidenceService(files, plainStorage{}, func(a, b, c, d string) string { return a + b + c + d }, activity)
	pay := &fakeClientPayments{rows: rows}
	return &verifyFixture{svc: NewClientPaymentService(pay, ev, activity, fakeContracts{project: 100_000_000}), payments: pay, evidence: ev, files: files, activity: act}
}

type activityRepoAdapter struct{ m *memActivity }

func (a activityRepoAdapter) Create(ctx context.Context, e *domain.ActivityLogEntry) error {
	return a.m.Create(ctx, e)
}
func (a activityRepoAdapter) ListByProject(context.Context, int64, int) ([]domain.ActivityLogEntry, error) {
	return nil, nil
}

func (f *verifyFixture) attach(paymentID int64) int64 {
	e := domain.Evidence{ProjectID: 10, Name: "Bukti", Type: domain.EvidenceTransferProof, FileName: "bukti.jpg", RelatedKind: domain.RelatedClientPayment, RelatedID: paymentID, UploadedAt: time.Now()}
	_ = f.files.Create(context.Background(), &e)
	return e.ID
}

func (f *verifyFixture) lastActivity() string {
	if len(f.activity.entries) == 0 {
		return ""
	}
	return f.activity.entries[len(f.activity.entries)-1].Description
}

func TestVerifyWithoutEvidenceIs422OnFieldEvidence(t *testing.T) {
	f := newVerifyFixture(domain.ClientPayment{ID: 1, ProjectID: 10, Type: domain.PaymentDP, Amount: 1000})
	_, err := f.svc.Verify(context.Background(), 10, 1, 5, time.Now())
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation {
		t.Fatalf("want a validation error, got %v", err)
	}
	if ae.Message != "Lampirkan bukti transfer sebelum memverifikasi pembayaran." {
		t.Errorf("message = %q", ae.Message)
	}
	if len(ae.Fields["evidence"]) != 1 {
		t.Errorf("want field key evidence, got %v", ae.Fields)
	}
	if f.payments.rows[0].IsVerified() {
		t.Error("nothing may be stored on a refused verification")
	}
}

func TestVerifySetsWhoAndWhenOnceAndLogsIt(t *testing.T) {
	f := newVerifyFixture(domain.ClientPayment{ID: 1, ProjectID: 10, Type: domain.PaymentDP, Amount: 1000})
	f.attach(1)
	at := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	p, err := f.svc.Verify(context.Background(), 10, 1, 5, at)
	if err != nil || !p.IsVerified() || p.VerifiedByStaffID != 5 || !p.VerifiedAt.Equal(at) {
		t.Fatalf("verify: %+v %v", p, err)
	}
	if f.lastActivity() != "Pembayaran client diverifikasi" {
		t.Errorf("activity = %q", f.lastActivity())
	}
	logged := len(f.activity.entries)

	// Idempotent: a second call by someone else, later, changes neither who nor when, and logs nothing.
	p2, err := f.svc.Verify(context.Background(), 10, 1, 9, at.Add(48*time.Hour))
	if err != nil || p2.VerifiedByStaffID != 5 || !p2.VerifiedAt.Equal(at) {
		t.Fatalf("second verify changed the state: %+v %v", p2, err)
	}
	if len(f.activity.entries) != logged {
		t.Error("an idempotent no-op must not add an activity entry")
	}
}

func TestVerifyUnknownPaymentIs404(t *testing.T) {
	f := newVerifyFixture()
	_, err := f.svc.Verify(context.Background(), 10, 99, 5, time.Now())
	if ae, ok := apperror.As(err); !ok || ae.Kind != apperror.KindNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestEditClearsVerificationAndLogsIt(t *testing.T) {
	f := newVerifyFixture(domain.ClientPayment{ID: 1, ProjectID: 10, Type: domain.PaymentDP, Amount: 1000, PaymentDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Method: "Transfer", ReferenceNumber: "R"})
	f.attach(1)
	if _, err := f.svc.Verify(context.Background(), 10, 1, 5, time.Now()); err != nil {
		t.Fatal(err)
	}
	in := ClientPaymentInput{Type: domain.PaymentDP, Amount: 1000, PaymentDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Method: "Transfer", ReferenceNumber: "R", Notes: "catatan"}

	// A note-only edit keeps the verification.
	p, err := f.svc.Update(context.Background(), 1, 10, 1, 5, in)
	if err != nil || !p.IsVerified() {
		t.Fatalf("note-only edit must keep the verification: %+v %v", p, err)
	}

	in.Amount = 2000
	p, err = f.svc.Update(context.Background(), 1, 10, 1, 5, in)
	if err != nil || p.IsVerified() || p.VerifiedByStaffID != 0 {
		t.Fatalf("changing the amount must clear the verification: %+v %v", p, err)
	}
	if f.payments.rows[0].IsVerified() {
		t.Error("the stored row must be cleared too")
	}
	if f.lastActivity() != "Verifikasi pembayaran client dibatalkan karena data pembayaran diubah" {
		t.Errorf("activity = %q", f.lastActivity())
	}
}

func TestRemovingTheLastProofClearsVerification(t *testing.T) {
	f := newVerifyFixture(domain.ClientPayment{ID: 1, ProjectID: 10, Type: domain.PaymentDP, Amount: 1000})
	first, second := f.attach(1), f.attach(1)
	if _, err := f.svc.Verify(context.Background(), 10, 1, 5, time.Now()); err != nil {
		t.Fatal(err)
	}

	// One of two files goes: still has proof, still verified.
	if err := f.evidence.Delete(context.Background(), 10, first, 5); err != nil {
		t.Fatal(err)
	}
	if !f.payments.rows[0].IsVerified() {
		t.Fatal("a payment with proof left must stay verified")
	}
	// The last one goes: the verification goes with it.
	if err := f.evidence.Delete(context.Background(), 10, second, 5); err != nil {
		t.Fatal(err)
	}
	if f.payments.rows[0].IsVerified() {
		t.Fatal("a verified payment must never be left without proof")
	}
	if f.lastActivity() != "Verifikasi pembayaran client dibatalkan karena bukti terakhir dihapus" {
		t.Errorf("activity = %q", f.lastActivity())
	}
	// And it cannot be re-verified until proof is attached again.
	if _, err := f.svc.Verify(context.Background(), 10, 1, 5, time.Now()); err == nil {
		t.Error("re-verifying with no proof must be refused")
	}
}

func TestDeletingEvidenceOfAnotherKindLeavesPaymentsAlone(t *testing.T) {
	f := newVerifyFixture(domain.ClientPayment{ID: 1, ProjectID: 10, Type: domain.PaymentDP, Amount: 1000})
	f.attach(1)
	if _, err := f.svc.Verify(context.Background(), 10, 1, 5, time.Now()); err != nil {
		t.Fatal(err)
	}
	other := domain.Evidence{ProjectID: 10, RelatedKind: domain.RelatedProjectMilestone, RelatedID: 1}
	_ = f.files.Create(context.Background(), &other)
	if err := f.evidence.Delete(context.Background(), 10, other.ID, 5); err != nil {
		t.Fatal(err)
	}
	if !f.payments.rows[0].IsVerified() {
		t.Error("removing a timeline file must not touch a payment with the same numeric id")
	}
	if err := f.evidence.Delete(context.Background(), 10, 9999, 5); err == nil {
		t.Error("deleting an unknown file is a 404")
	}
}

func TestMilestoneEvidenceCanBeFlaggedClientVisible(t *testing.T) {
	f := newVerifyFixture()
	ctx := context.Background()
	milestone := domain.Evidence{ProjectID: 10, RelatedKind: domain.RelatedProjectMilestone, RelatedID: 3}
	payment := domain.Evidence{ProjectID: 10, RelatedKind: domain.RelatedClientPayment, RelatedID: 3}
	_ = f.files.Create(ctx, &milestone)
	_ = f.files.Create(ctx, &payment)

	e, err := f.evidence.ToggleClientVisible(ctx, 10, milestone.ID)
	if err != nil || !e.IsClientVisible {
		t.Fatalf("staff must be able to flag a timeline attachment: %+v %v", e, err)
	}
	e, err = f.evidence.ToggleClientVisible(ctx, 10, milestone.ID)
	if err != nil || e.IsClientVisible {
		t.Fatalf("the flag toggles back: %+v %v", e, err)
	}
	if _, err := f.evidence.ToggleClientVisible(ctx, 10, payment.ID); err == nil {
		t.Error("a payment proof has no visibility flag")
	}
}

func TestUploadHonoursVisibilityFlagOnlyForKindsThatCarryIt(t *testing.T) {
	f := newVerifyFixture()
	ctx := context.Background()
	pdf := "JVBERi0xLjQK" // "%PDF-1.4\n"
	up := func(kind domain.EvidenceRelatedKind) bool {
		e, err := f.evidence.Upload(ctx, 1, 10, 5, UploadEvidenceInput{
			Name: "x", Type: domain.EvidenceDocument, FileName: "x.pdf", MimeType: "application/pdf", Base64Data: pdf,
			RelatedKind: kind, RelatedID: 3, IsClientVisible: true,
		})
		if err != nil {
			t.Fatalf("upload %s: %v", kind, err)
		}
		return e.IsClientVisible
	}
	if !up(domain.RelatedGeneral) || !up(domain.RelatedProjectMilestone) {
		t.Error("general documents and timeline attachments may be uploaded as client-visible")
	}
	if up(domain.RelatedClientPayment) || up(domain.RelatedIssue) || up(domain.RelatedPayment) {
		t.Error("the flag must be dropped for every other kind")
	}
}
