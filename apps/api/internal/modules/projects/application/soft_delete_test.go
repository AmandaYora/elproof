package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

var delNow = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

func ids(list []domain.ClientPayment) []int64 {
	var out []int64
	for _, p := range list {
		out = append(out, p.ID)
	}
	return out
}

func TestDeletingAClientPaymentHidesItAndRestoreBringsItBack(t *testing.T) {
	verified := delNow.Add(-time.Hour)
	repo := &fakeClientPayments{rows: []domain.ClientPayment{
		{ID: 1, Type: domain.PaymentDP, Amount: 60, VerifiedAt: &verified, VerifiedByStaffID: 9},
		{ID: 2, Type: domain.PaymentTermin, Amount: 40},
	}}
	svc := NewClientPaymentService(repo, nil, nil, fakeContracts{project: 100})
	ctx := context.Background()

	until, err := svc.Delete(ctx, 1, 10, 1, 5, delNow)
	if err != nil {
		t.Fatalf("a verified payment is deletable like any other: %v", err)
	}
	if !until.Equal(delNow.Add(24 * time.Hour)) {
		t.Errorf("restorableUntil = %v, want now+24h", until)
	}
	live, _ := svc.List(ctx, 10)
	if got := ids(live); len(got) != 1 || got[0] != 2 {
		t.Fatalf("a deleted payment must leave the list, got %v", got)
	}
	if _, err := svc.Delete(ctx, 1, 10, 1, 5, delNow); apperror.HTTPStatus(err) != 404 {
		t.Errorf("deleting twice: want 404, got %v", err)
	}

	restored, err := svc.Restore(ctx, 1, 10, 1, 5, delNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.DeletedAt != nil || restored.VerifiedAt == nil || restored.VerifiedByStaffID != 9 {
		t.Errorf("a restored payment is the same row, verification included: %+v", restored)
	}
	live, _ = svc.List(ctx, 10)
	if len(live) != 2 {
		t.Errorf("restored payment must be listed again, got %v", ids(live))
	}
}

func TestRestoreIsA404WhenNotDeletedOrTooOld(t *testing.T) {
	old := delNow.Add(-25 * time.Hour)
	fresh := delNow.Add(-time.Hour)
	repo := &fakeClientPayments{rows: []domain.ClientPayment{
		{ID: 1, Type: domain.PaymentDP, Amount: 10},                    // live
		{ID: 2, Type: domain.PaymentDP, Amount: 10, DeletedAt: &old},   // deleted, past the retention (purge has not run yet)
		{ID: 3, Type: domain.PaymentDP, Amount: 10, DeletedAt: &fresh}, // deleted an hour ago
	}}
	svc := NewClientPaymentService(repo, nil, nil, fakeContracts{project: 1000})
	for _, id := range []int64{1, 2, 99} {
		if _, err := svc.Restore(context.Background(), 1, 10, id, 5, delNow); apperror.HTTPStatus(err) != 404 {
			t.Errorf("payment %d: want 404 (not deleted / purged / unknown), got %v", id, err)
		}
	}
	if _, err := svc.Restore(context.Background(), 1, 10, 3, 5, delNow); err != nil {
		t.Errorf("a payment deleted an hour ago restores: %v", err)
	}
}

// Restoring re-checks the overpayment rule: while a payment was deleted another
// one may have used up the room it needs.
func TestRestoreRechecksOverpayment(t *testing.T) {
	gone := delNow.Add(-time.Minute)
	repo := &fakeClientPayments{rows: []domain.ClientPayment{
		{ID: 1, Type: domain.PaymentDP, Amount: 60},
		{ID: 2, Type: domain.PaymentTermin, Amount: 40, DeletedAt: &gone},
		{ID: 3, Type: domain.PaymentPelunasan, Amount: 30}, // recorded after the deletion
	}}
	svc := NewClientPaymentService(repo, nil, nil, fakeContracts{project: 100})
	_, err := svc.Restore(context.Background(), 1, 10, 2, 5, delNow)
	if codeOf(err) != "payment_exceeds_contract" || apperror.HTTPStatus(err) != 422 {
		t.Fatalf("want 422 payment_exceeds_contract, got %v", err)
	}
	ae, _ := apperror.As(err)
	if len(ae.Fields["amount"]) == 0 {
		t.Errorf("the usual key is `amount`: %+v", ae.Fields)
	}
	if got, _ := repo.FindDeletedByID(context.Background(), 10, 2); got == nil {
		t.Error("a refused restore must leave the payment deleted")
	}
	// A Tambahan (outside the contract) never conflicts.
	repo.rows[1].Type = domain.PaymentTambahan
	if _, err := svc.Restore(context.Background(), 1, 10, 2, 5, delNow); err != nil {
		t.Errorf("Tambahan restores whatever the contract is: %v", err)
	}
}

// ---- vendor payments ----

type fakeVendorPayments struct {
	PaymentRepository
	rows []domain.VendorPayment
}

func (f *fakeVendorPayments) ListByProject(context.Context, int64) ([]domain.VendorPayment, error) {
	var out []domain.VendorPayment
	for _, r := range f.rows {
		if r.DeletedAt == nil {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeVendorPayments) FindByID(_ context.Context, _, id int64) (*domain.VendorPayment, error) {
	for _, r := range f.rows {
		if r.ID == id && r.DeletedAt == nil {
			p := r
			return &p, nil
		}
	}
	return nil, nil
}
func (f *fakeVendorPayments) FindDeletedByID(_ context.Context, _, id int64) (*domain.VendorPayment, error) {
	for _, r := range f.rows {
		if r.ID == id && r.DeletedAt != nil {
			p := r
			return &p, nil
		}
	}
	return nil, nil
}
func (f *fakeVendorPayments) SoftDelete(_ context.Context, _, id int64, at time.Time) (bool, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].DeletedAt == nil {
			f.rows[i].DeletedAt = &at
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeVendorPayments) Restore(_ context.Context, _, id int64, notBefore time.Time) (bool, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].DeletedAt != nil && f.rows[i].DeletedAt.After(notBefore) {
			f.rows[i].DeletedAt = nil
			return true, nil
		}
	}
	return false, nil
}

// goneEngagement is a contract reader for a vendor engagement that is deleted.
type goneEngagement struct{ fakeContracts }

func (goneEngagement) EngagementContractValue(context.Context, int64, int64) (int64, error) {
	return 0, apperror.NotFound("Kerja sama vendor tidak ditemukan")
}

func TestVendorPaymentDeleteReturnsItsEngagementAndLedgerExcludesIt(t *testing.T) {
	repo := &fakeVendorPayments{rows: []domain.VendorPayment{
		{ID: 1, ProjectVendorID: 7, Type: domain.PaymentDP, Amount: 30},
		{ID: 2, ProjectVendorID: 7, Type: domain.PaymentTermin, Amount: 20},
	}}
	svc := NewPaymentService(repo, nil, nil, fakeContracts{engagement: 100})
	p, until, err := svc.Delete(context.Background(), 10, 2, 5, delNow)
	if err != nil || p.ProjectVendorID != 7 || !until.Equal(delNow.Add(24*time.Hour)) {
		t.Fatalf("delete: %+v %v %v", p, until, err)
	}
	l, err := svc.LedgerSummary(context.Background(), 10, 7)
	if err != nil || l.Paid != 30 || l.Remaining != 70 || l.ContractValue != 100 {
		t.Errorf("the deleted payment must not count: %+v %v", l, err)
	}
	if _, err := svc.Restore(context.Background(), 10, 2, 5, delNow.Add(time.Minute)); err != nil {
		t.Fatalf("restore: %v", err)
	}
	l, _ = svc.LedgerSummary(context.Background(), 10, 7)
	if l.Paid != 50 || l.Remaining != 50 {
		t.Errorf("after restore the payment counts again: %+v", l)
	}
}

func TestVendorPaymentRestoreNeedsItsEngagement(t *testing.T) {
	gone := delNow.Add(-time.Minute)
	repo := &fakeVendorPayments{rows: []domain.VendorPayment{{ID: 2, ProjectVendorID: 7, Type: domain.PaymentDP, Amount: 20, DeletedAt: &gone}}}
	svc := NewPaymentService(repo, nil, nil, goneEngagement{})
	_, err := svc.Restore(context.Background(), 10, 2, 5, delNow)
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation || len(ae.Fields["projectVendorId"]) == 0 {
		t.Fatalf("want 422 on projectVendorId, got %v", err)
	}
}

// ---- vendor engagements ----

type fakeEngagements struct {
	VendorEngagementRepository
	rows []domain.ProjectVendor
}

func (f *fakeEngagements) FindByID(_ context.Context, _, id int64) (*domain.ProjectVendor, error) {
	for _, r := range f.rows {
		if r.ID == id && r.DeletedAt == nil {
			p := r
			return &p, nil
		}
	}
	return nil, nil
}
func (f *fakeEngagements) FindDeletedByID(_ context.Context, _, id int64) (*domain.ProjectVendor, error) {
	for _, r := range f.rows {
		if r.ID == id && r.DeletedAt != nil {
			p := r
			return &p, nil
		}
	}
	return nil, nil
}
func (f *fakeEngagements) SoftDelete(_ context.Context, _, id int64, at time.Time) (bool, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].DeletedAt == nil {
			f.rows[i].DeletedAt = &at
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeEngagements) Restore(_ context.Context, _, id int64, notBefore time.Time) (bool, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].DeletedAt != nil && f.rows[i].DeletedAt.After(notBefore) {
			f.rows[i].DeletedAt = nil
			return true, nil
		}
	}
	return false, nil
}

func TestEngagementSoftDeleteAndRestoreWithCheck(t *testing.T) {
	repo := &fakeEngagements{rows: []domain.ProjectVendor{{ID: 1, VendorID: 5, Scope: "Dekor"}}}
	svc := NewVendorEngagementService(repo, nil, nil)
	ctx := context.Background()
	if _, err := svc.Get(ctx, 10, 1); err != nil {
		t.Fatal(err)
	}
	_, until, err := svc.Delete(ctx, 10, 1, 5, delNow)
	if err != nil || !until.Equal(delNow.Add(24*time.Hour)) {
		t.Fatalf("delete: %v %v", until, err)
	}
	if _, err := svc.Get(ctx, 10, 1); apperror.HTTPStatus(err) != 404 {
		t.Errorf("a deleted engagement is not found: %v", err)
	}
	// The check can refuse (vendor hard-deleted meanwhile): 422 and it stays deleted.
	refuse := func(context.Context, *domain.ProjectVendor) error {
		return apperror.Validation("x", map[string][]string{"vendorId": {"Vendor tidak ditemukan"}})
	}
	if _, err := svc.Restore(ctx, 10, 1, 5, delNow, refuse); apperror.HTTPStatus(err) != 422 {
		t.Errorf("want 422, got %v", err)
	}
	if got, _ := repo.FindDeletedByID(ctx, 10, 1); got == nil {
		t.Error("a refused restore leaves it deleted")
	}
	if _, err := svc.Restore(ctx, 10, 1, 5, delNow.Add(25*time.Hour), nil); apperror.HTTPStatus(err) != 404 {
		t.Errorf("past the retention: want 404, got %v", err)
	}
	if _, err := svc.Restore(ctx, 10, 1, 5, delNow.Add(time.Hour), nil); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := svc.Restore(ctx, 10, 1, 5, delNow.Add(time.Hour), nil); apperror.HTTPStatus(err) != 404 {
		t.Errorf("restoring a live engagement: want 404, got %v", err)
	}
}

// ---- purge ----

type fakePurgeRepo struct {
	locked   bool
	batches  []domain.PurgeReport
	keys     [][]string
	calls    int
	cutoffs  []time.Time
	failWith error
}

func (f *fakePurgeRepo) TryLock(context.Context) (func(), bool, error) {
	if f.locked {
		return nil, false, nil
	}
	f.locked = true
	return func() { f.locked = false }, true, nil
}
func (f *fakePurgeRepo) PurgeBatch(_ context.Context, cutoff time.Time, _ int) (domain.PurgeReport, []string, error) {
	f.cutoffs = append(f.cutoffs, cutoff)
	if f.failWith != nil {
		return domain.PurgeReport{}, nil, f.failWith
	}
	if f.calls >= len(f.batches) {
		f.calls++
		return domain.PurgeReport{}, nil, nil
	}
	i := f.calls
	f.calls++
	return f.batches[i], f.keys[i], nil
}

func TestPurgeRunsUntilNothingIsLeftAndDeletesObjects(t *testing.T) {
	store := newObjStorage(false)
	for _, k := range []string{"a", "b", "c"} {
		store.objs[k] = []byte("x")
	}
	repo := &fakePurgeRepo{
		batches: []domain.PurgeReport{{ClientPayments: 2, Evidence: 2}, {Engagements: 1, Evidence: 1}},
		keys:    [][]string{{"a", "b"}, {"c"}},
	}
	svc := NewPurgeService(repo, NewEvidenceService(&memEvidence{}, store, buildKeyForTests, nil))
	report, ran, err := svc.Run(context.Background(), delNow)
	if err != nil || !ran {
		t.Fatalf("run: %v %v", ran, err)
	}
	if report.ClientPayments != 2 || report.Engagements != 1 || report.Evidence != 3 || report.Objects != 3 {
		t.Errorf("report = %+v", report)
	}
	if len(store.objs) != 0 {
		t.Errorf("every stored file of the purged rows must be deleted, left: %v", store.objs)
	}
	if repo.calls != 3 {
		t.Errorf("two batches with work and one empty batch to stop, got %d calls", repo.calls)
	}
	want := delNow.Add(-24 * time.Hour)
	for _, c := range repo.cutoffs {
		if !c.Equal(want) {
			t.Errorf("cutoff = %v, want now-24h", c)
		}
	}
	// Idempotent: the next run finds nothing.
	report, ran, err = svc.Run(context.Background(), delNow)
	if err != nil || !ran || report.Total() != 0 {
		t.Errorf("second run: %+v %v %v", report, ran, err)
	}
}

func TestPurgeDoesNothingWhenAnotherInstanceHoldsTheLock(t *testing.T) {
	repo := &fakePurgeRepo{locked: true}
	svc := NewPurgeService(repo, NewEvidenceService(&memEvidence{}, newObjStorage(false), buildKeyForTests, nil))
	report, ran, err := svc.Run(context.Background(), delNow)
	if err != nil || ran || report.Total() != 0 || repo.calls != 0 {
		t.Errorf("ran=%v calls=%d report=%+v err=%v: must stand aside", ran, repo.calls, report, err)
	}
	// The lock is released after a run (also after a failed one).
	repo = &fakePurgeRepo{failWith: context.DeadlineExceeded}
	svc = NewPurgeService(repo, NewEvidenceService(&memEvidence{}, newObjStorage(false), buildKeyForTests, nil))
	if _, ran, err := svc.Run(context.Background(), delNow); err == nil || !ran {
		t.Errorf("a failed batch is reported: %v %v", ran, err)
	}
	if repo.locked {
		t.Error("the lock must be released even after a failure")
	}
}

// The dashboard's cash flow counts what clients still owe: each project's real
// payments are allocated over its schedule, oldest installment first.
func TestIncomingDueAllocatesPaymentsOverTheSchedule(t *testing.T) {
	day := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	rows := []domain.TermWithProject{
		{Term: domain.PaymentTerm{ID: 1, ProjectID: 1, Amount: 30, DueDate: day("2026-11-01")}},
		{Term: domain.PaymentTerm{ID: 2, ProjectID: 1, Amount: 30, DueDate: day("2026-12-01")}},
		{Term: domain.PaymentTerm{ID: 3, ProjectID: 2, Amount: 50, DueDate: day("2026-11-20")}},
	}
	got := incomingDue(rows, map[int64]int64{1: 40}, day("2026-11-10"))
	// Project 1: 40 paid covers installment 1 (30) and 10 of installment 2 -> 20 left.
	// Project 2: nothing paid -> 50 left.
	if len(got) != 2 || got[0].Remaining != 20 || !got[0].DueDate.Equal(day("2026-12-01")) || got[1].Remaining != 50 {
		t.Errorf("got %+v", got)
	}
}
