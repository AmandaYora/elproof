package domain

import (
	"errors"
	"testing"
	"time"
)

var day1 = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

func TestVerifyNeedsEvidenceAndIsIdempotent(t *testing.T) {
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	p := ClientPayment{ID: 1, Type: PaymentDP, Amount: 1000, PaymentDate: day1}

	changed, err := p.Verify(now, 7, 0)
	if !errors.Is(err, ErrVerifyNeedsEvidence) || changed || p.IsVerified() {
		t.Fatalf("no proof: want ErrVerifyNeedsEvidence and nothing set, got changed=%v err=%v verified=%v", changed, err, p.IsVerified())
	}

	changed, err = p.Verify(now, 7, 1)
	if err != nil || !changed || !p.IsVerified() || p.VerifiedByStaffID != 7 || !p.VerifiedAt.Equal(now) {
		t.Fatalf("with proof: got changed=%v err=%v %+v", changed, err, p)
	}

	// A second verification (even by someone else, even later) changes nothing.
	changed, err = p.Verify(now.Add(time.Hour), 9, 1)
	if err != nil || changed || p.VerifiedByStaffID != 7 || !p.VerifiedAt.Equal(now) {
		t.Fatalf("idempotent verify changed the payment: changed=%v err=%v %+v", changed, err, p)
	}
}

func TestVerifyStoresWholeSeconds(t *testing.T) {
	p := ClientPayment{}
	p.Verify(time.Date(2026, 3, 2, 10, 0, 5, 900_000_000, time.UTC), 7, 1)
	if p.VerifiedAt.Nanosecond() != 0 || p.VerifiedAt.Second() != 5 {
		t.Errorf("verifiedAt must be truncated to whole seconds, got %v", p.VerifiedAt)
	}
}

func TestEditClearsVerificationOnlyWhenWhatWasPaidChanges(t *testing.T) {
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	base := func() ClientPayment {
		p := ClientPayment{Type: PaymentDP, Amount: 1000, PaymentDate: day1, Method: "Transfer", ReferenceNumber: "R1", Notes: "a"}
		p.Verify(now, 7, 1)
		return p
	}
	cases := []struct {
		name    string
		edit    func(p *ClientPayment) bool
		cleared bool
	}{
		{"amount", func(p *ClientPayment) bool { return p.ApplyEdit(PaymentDP, 2000, day1, "Transfer", "R1", "a") }, true},
		{"type", func(p *ClientPayment) bool { return p.ApplyEdit(PaymentTermin, 1000, day1, "Transfer", "R1", "a") }, true},
		{"date", func(p *ClientPayment) bool {
			return p.ApplyEdit(PaymentDP, 1000, day1.AddDate(0, 0, 1), "Transfer", "R1", "a")
		}, true},
		{"method", func(p *ClientPayment) bool { return p.ApplyEdit(PaymentDP, 1000, day1, "Tunai", "R1", "a") }, true},
		{"reference", func(p *ClientPayment) bool { return p.ApplyEdit(PaymentDP, 1000, day1, "Transfer", "R2", "a") }, true},
		{"note only", func(p *ClientPayment) bool {
			return p.ApplyEdit(PaymentDP, 1000, day1, "Transfer", "R1", "catatan baru")
		}, false},
		{"identical", func(p *ClientPayment) bool { return p.ApplyEdit(PaymentDP, 1000, day1, "Transfer", "R1", "a") }, false},
	}
	for _, c := range cases {
		p := base()
		got := c.edit(&p)
		if got != c.cleared || p.IsVerified() == c.cleared || (c.cleared && p.VerifiedByStaffID != 0) {
			t.Errorf("%s: cleared=%v verified=%v, want cleared=%v", c.name, got, p.IsVerified(), c.cleared)
		}
	}

	// Editing an unverified payment never reports a clearing.
	p := ClientPayment{Type: PaymentDP, Amount: 1, PaymentDate: day1}
	if p.ApplyEdit(PaymentDP, 2, day1, "", "", "") {
		t.Error("an unverified payment has nothing to clear")
	}
}
