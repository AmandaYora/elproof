package domain

import "testing"

func TestComputeLedger(t *testing.T) {
	line := func(t PaymentType, a int64) PaymentLine { return PaymentLine{Type: t, Amount: a} }
	cases := []struct {
		name  string
		lines []PaymentLine
		owed  int64
		want  Ledger
	}{
		{"nothing paid", nil, 1000, Ledger{1000, 0, 1000}},
		{"dp then termin", []PaymentLine{line(PaymentDP, 300), line(PaymentTermin, 200)}, 1000, Ledger{1000, 500, 500}},
		{"fully paid", []PaymentLine{line(PaymentDP, 400), line(PaymentPelunasan, 600)}, 1000, Ledger{1000, 1000, 0}},
		{"a refund gives room back", []PaymentLine{line(PaymentDP, 1000), line(PaymentRefund, 250)}, 1000, Ledger{1000, 750, 250}},
		{"tambahan is outside the contract", []PaymentLine{line(PaymentDP, 100), line(PaymentTambahan, 900)}, 1000, Ledger{1000, 100, 900}},
		{"overpaid by old data: remaining never negative", []PaymentLine{line(PaymentDP, 1500)}, 1000, Ledger{1000, 1500, 0}},
		{"refunds beyond payments: paid never negative", []PaymentLine{line(PaymentRefund, 50)}, 1000, Ledger{1000, 0, 1000}},
		{"no amount set yet", []PaymentLine{line(PaymentDP, 10)}, 0, Ledger{0, 10, 0}},
	}
	for _, c := range cases {
		if got := ComputeLedger(c.lines, c.owed); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestIsValidPaymentType(t *testing.T) {
	for _, ok := range []PaymentType{"DP", "Termin", "Pelunasan", "Tambahan", "Refund"} {
		if !IsValidPaymentType(ok) {
			t.Errorf("%q must be valid", ok)
		}
	}
	for _, bad := range []PaymentType{"", "dp", "Cicilan"} {
		if IsValidPaymentType(bad) {
			t.Errorf("%q must be invalid", bad)
		}
	}
}
