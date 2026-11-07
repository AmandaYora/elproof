package domain

import "testing"

func TestOverpaymentExcess(t *testing.T) {
	contract := int64(100_000_000)
	existing := []PaymentLine{
		{PaymentDP, 30_000_000},
		{PaymentTermin, 40_000_000},
		{PaymentTambahan, 15_000_000}, // outside the contract: must not count
	}
	cases := []struct {
		name     string
		existing []PaymentLine
		next     PaymentLine
		contract int64
		want     int64
	}{
		{"exact remainder fits", existing, PaymentLine{PaymentPelunasan, 30_000_000}, contract, 0},
		{"one rupiah over", existing, PaymentLine{PaymentPelunasan, 30_000_001}, contract, 1},
		{"Tambahan is never limited", existing, PaymentLine{PaymentTambahan, 500_000_000}, contract, 0},
		{"Refund is never limited", existing, PaymentLine{PaymentRefund, 500_000_000}, contract, 0},
		{"refund gives room back", append(append([]PaymentLine{}, existing...), PaymentLine{PaymentRefund, 10_000_000}), PaymentLine{PaymentPelunasan, 40_000_000}, contract, 0},
		{"no contract amount set: not limited", existing, PaymentLine{PaymentPelunasan, 999}, 0, 0},
		{"first payment above contract", nil, PaymentLine{PaymentDP, 100_000_001}, contract, 1},
	}
	for _, c := range cases {
		if got := OverpaymentExcess(c.existing, c.next, c.contract); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestAppliedToContractIgnoresTambahan(t *testing.T) {
	got := AppliedToContract([]PaymentLine{{PaymentDP, 10}, {PaymentTambahan, 99}, {PaymentRefund, 4}})
	if got != 6 {
		t.Fatalf("got %d, want 6", got)
	}
}

func TestExcessAfterChange(t *testing.T) {
	contract := int64(100)
	dp := PaymentLine{PaymentDP, 100}
	refund := PaymentLine{PaymentRefund, 50}
	termin := PaymentLine{PaymentTermin, 50}

	// The bypass found in review: refund makes room, a Termin uses it, then
	// removing the refund must not leave the client at 150 of 100.
	before := []PaymentLine{dp, refund, termin}
	if got := ExcessAfterChange(before, []PaymentLine{dp, termin}, contract); got != 50 {
		t.Errorf("deleting the refund: excess %d, want 50", got)
	}
	// Shrinking the refund, or turning it into Tambahan, is the same hole.
	if got := ExcessAfterChange(before, []PaymentLine{dp, {PaymentRefund, 1}, termin}, contract); got != 49 {
		t.Errorf("shrinking the refund: excess %d, want 49", got)
	}
	if got := ExcessAfterChange(before, []PaymentLine{dp, {PaymentTambahan, 50}, termin}, contract); got != 50 {
		t.Errorf("refund to Tambahan: excess %d, want 50", got)
	}
	// Lowering or deleting a limited payment is always allowed.
	over := []PaymentLine{dp, {PaymentTermin, 30}} // 130 of 100: overpaid already
	if got := ExcessAfterChange(over, []PaymentLine{dp}, contract); got != 0 {
		t.Errorf("deleting a payment: excess %d, want 0", got)
	}
	// An edit that leaves the total unchanged (reference number, date) is allowed on an overpaid project.
	if got := ExcessAfterChange(over, append([]PaymentLine(nil), over...), contract); got != 0 {
		t.Errorf("unchanged total: excess %d, want 0", got)
	}
	// Raising it further is not.
	if got := ExcessAfterChange(over, []PaymentLine{dp, {PaymentTermin, 31}}, contract); got != 31 {
		t.Errorf("raising an overpaid total: excess %d, want 31", got)
	}
	// No contract amount set: nothing limited.
	if got := ExcessAfterChange(nil, []PaymentLine{dp}, 0); got != 0 {
		t.Errorf("no contract: excess %d, want 0", got)
	}
}
