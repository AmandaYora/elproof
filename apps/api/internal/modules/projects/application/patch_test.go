package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

func TestChangeSetRecordsOnlyRealChanges(t *testing.T) {
	var c changeSet
	name, n, note := "Ayu", int64(5), "lama"
	var venue *int64
	day := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	var done *time.Time

	setVal(&c, "name", "nama", &name, Opt[string]{})         // absent: untouched
	setVal(&c, "name", "nama", &name, Some("Ayu"))           // same value: no change
	setVal(&c, "notes", "catatan", &note, Some("baru"))      // changed
	setVal(&c, "n", "angka", &n, Null[int64]())              // null on a non-nullable field is ignored here (the handler refuses it)
	setNullable(&c, "venue", "venue", &venue, Null[int64]()) // clearing what is already empty
	setNullable(&c, "venue", "venue", &venue, Some(int64(3)))
	setDate(&c, "day", "tanggal", &day, Some(time.Date(2026, 5, 1, 7, 0, 0, 0, time.FixedZone("x", 7*3600)))) // same calendar day
	setDate(&c, "day", "tanggal", &day, Some(time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)))
	setNullableDate(&c, "done", "selesai", &done, Some(day))
	setNullableDate(&c, "done", "selesai", &done, Null[time.Time]())

	want := []string{"notes", "venue", "day", "done", "done"}
	if len(c.keys) != len(want) {
		t.Fatalf("keys = %v, want %v", c.keys, want)
	}
	for i, k := range want {
		if c.keys[i] != k {
			t.Errorf("keys = %v, want %v", c.keys, want)
		}
	}
	if name != "Ayu" || n != 5 || note != "baru" || venue == nil || *venue != 3 || done != nil {
		t.Errorf("values: %q %d %q %v %v", name, n, note, venue, done)
	}
	// Labels are named once even when a field is touched twice, in the order they changed.
	if got, want := c.summary(), "catatan, venue, tanggal, selesai"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if !c.has("venue") || c.has("name") {
		t.Error("has() must follow the keys")
	}
}

func TestFinishPatchTellsStaleFromGoneFromNothingToChange(t *testing.T) {
	v1 := time.Date(2026, 11, 10, 8, 0, 0, 1000, time.UTC)
	v2 := v1.Add(time.Second)
	row := func(v time.Time) func(context.Context) (*domain.Project, error) {
		return func(context.Context) (*domain.Project, error) { return &domain.Project{ID: 1, UpdatedAt: v}, nil }
	}
	gone := func(context.Context) (*domain.Project, error) { return nil, nil }
	version := func(p *domain.Project) time.Time { return p.UpdatedAt }
	missing := apperror.NotFound("tidak ada")

	// Written: the answer is the row as it is now.
	if p, err := finishPatch(context.Background(), &v1, true, row(v2), version, missing); err != nil || !p.UpdatedAt.Equal(v2) {
		t.Errorf("written: %v %v", p, err)
	}
	// Not matched and the row moved on: stale, carrying the current row.
	_, err := finishPatch(context.Background(), &v1, false, row(v2), version, missing)
	cur, ok := apperror.IsStale(err)
	if !ok || cur.(*domain.Project).UpdatedAt != v2 || apperror.HTTPStatus(err) != 409 {
		t.Errorf("stale: %v", err)
	}
	// Not matched, the row is still at the sent version: MySQL had nothing to change.
	if p, err := finishPatch(context.Background(), &v1, false, row(v1.Add(300)), version, missing); err != nil || p == nil {
		t.Errorf("nothing to change must not be a conflict: %v", err)
	}
	// Not matched and gone: the old 404, never a 409.
	if _, err := finishPatch(context.Background(), &v1, false, gone, version, missing); apperror.HTTPStatus(err) != 404 {
		t.Errorf("gone: %v", err)
	}
	// Without a version nothing is ever stale.
	if _, err := finishPatch(context.Background(), nil, false, row(v2), version, missing); err != nil {
		t.Errorf("no If-Match: %v", err)
	}
}

func TestPaymentPatchWritesNothingWhenNothingChanges(t *testing.T) {
	when := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	f := newVerifyFixture(domain.ClientPayment{ID: 1, ProjectID: 10, Type: domain.PaymentDP, Amount: 1000, PaymentDate: when, Method: "Transfer", ReferenceNumber: "R", Notes: "x"})
	f.attach(1)
	if _, err := f.svc.Verify(context.Background(), 10, 1, 5, time.Now()); err != nil {
		t.Fatal(err)
	}
	logged := len(f.activity.entries)
	p, err := f.svc.Patch(context.Background(), 1, 10, 1, 5, PaymentPatch{
		Type: Some(domain.PaymentDP), Amount: Some(int64(1000)), PaymentDate: Some(when), Method: Some("Transfer"), ReferenceNumber: Some("R"), Notes: Some("x"),
	}, nil)
	if err != nil || !p.IsVerified() {
		t.Fatalf("a no-op patch answers with the payment as it is: %+v %v", p, err)
	}
	if f.payments.patches != 0 || len(f.activity.entries) != logged {
		t.Errorf("a no-op PATCH wrote %d times and logged %d entries", f.payments.patches, len(f.activity.entries)-logged)
	}
	// One real change: one write, one entry, and it names only that field.
	if _, err := f.svc.Patch(context.Background(), 1, 10, 1, 5, PaymentPatch{Notes: Some("y"), Method: Some("Transfer")}, nil); err != nil {
		t.Fatal(err)
	}
	if f.payments.patches != 1 || len(f.activity.entries) != logged+1 {
		t.Errorf("one change: %d writes, %d entries", f.payments.patches, len(f.activity.entries)-logged)
	}
	if got := f.lastActivity(); got != "Pembayaran client diperbarui: catatan" {
		t.Errorf("activity = %q", got)
	}
}

func TestPaymentPatchRejectsWhatCreateRejects(t *testing.T) {
	f := newVerifyFixture(domain.ClientPayment{ID: 1, ProjectID: 10, Type: domain.PaymentDP, Amount: 1000})
	for _, patch := range []PaymentPatch{{Type: Some(domain.PaymentType("Hibah"))}, {Amount: Some(int64(0))}, {Amount: Some(int64(-5))}} {
		_, err := f.svc.Patch(context.Background(), 1, 10, 1, 5, patch, nil)
		if apperror.HTTPStatus(err) != 422 {
			t.Errorf("%+v: want 422, got %v", patch, err)
		}
	}
	if f.payments.patches != 0 {
		t.Error("a refused patch must not be written")
	}
	// A payment that does not exist is the old 404, with or without a version.
	v := time.Now()
	if _, err := f.svc.Patch(context.Background(), 1, 10, 99, 5, PaymentPatch{Notes: Some("x")}, &v); apperror.HTTPStatus(err) != 404 {
		t.Errorf("unknown payment: %v", err)
	}
}
