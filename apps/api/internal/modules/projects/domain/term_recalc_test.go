package domain

import (
	"errors"
	"testing"
)

func amounts(terms []PaymentTerm) []int64 {
	out := make([]int64, len(terms))
	for i, t := range terms {
		out[i] = t.Amount
	}
	return out
}

func sameAmounts(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRecalculateTerms(t *testing.T) {
	// 30/40/30 of Rp 100 (the QA case: the contract later became Rp 100.000.000).
	schedule := []PaymentTerm{
		{ID: 3, Label: "Pelunasan", Amount: 30, DueDate: d("2027-06-05")},
		{ID: 1, Label: "DP", Amount: 30, DueDate: d("2026-11-15")},
		{ID: 2, Label: "Termin 1", Amount: 40, DueDate: d("2027-04-13")},
	}
	cases := []struct {
		name     string
		applied  int64
		contract int64
		want     []int64 // due-date order: DP, Termin 1, Pelunasan
		err      error
	}{
		{"nothing paid: whole schedule rescaled, shape kept", 0, 100_000_000, []int64{30_000_000, 40_000_000, 30_000_000}, nil},
		{"rounding remainder goes to the last installment", 0, 1_000, []int64{300, 400, 300}, nil},
		{"odd rupiah: whole rupiah, remainder last", 0, 101, []int64{30, 40, 31}, nil},
		{"DP paid: DP kept, rest spread 40:30", 30, 1_030, []int64{30, 571, 429}, nil},
		{"DP partly paid still counts as having money on it", 10, 230, []int64{30, 114, 86}, nil},
		{"lower contract: unpaid installments shrink", 30, 72, []int64{30, 24, 18}, nil},
		{"contract below what the paid installments hold", 30, 31, nil, ErrRecalcTooSmall},
		{"every installment has money on it", 75, 500, nil, ErrRecalcAllPaid},
		{"no contract value", 0, 0, nil, ErrRecalcNoContract},
	}
	for _, c := range cases {
		got, err := RecalculateTerms(schedule, c.applied, c.contract)
		if c.err != nil {
			if !errors.Is(err, c.err) {
				t.Errorf("%s: err %v, want %v", c.name, err, c.err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !sameAmounts(amounts(got), c.want) {
			t.Errorf("%s: amounts %v, want %v", c.name, amounts(got), c.want)
		}
		if TermsTotal(got) != c.contract {
			t.Errorf("%s: total %d, want the contract %d", c.name, TermsTotal(got), c.contract)
		}
		if got[0].Label != "DP" || got[2].Label != "Pelunasan" || !got[1].DueDate.Equal(d("2027-04-13")) {
			t.Errorf("%s: labels or dates changed: %+v", c.name, got)
		}
	}
}

func TestRecalculateTermsEmptyAndHuge(t *testing.T) {
	if _, err := RecalculateTerms(nil, 0, 100); !errors.Is(err, ErrRecalcNoTerms) {
		t.Errorf("empty schedule: %v", err)
	}
	// Near MaxMoney the product budget × amount passes int64; the split stays exact.
	big := []PaymentTerm{
		{ID: 1, Label: "A", Amount: MaxMoney / 2, DueDate: d("2027-01-01")},
		{ID: 2, Label: "B", Amount: MaxMoney / 2, DueDate: d("2027-02-01")},
	}
	got, err := RecalculateTerms(big, 0, MaxMoney)
	if err != nil {
		t.Fatal(err)
	}
	if TermsTotal(got) != MaxMoney || got[0].Amount != MaxMoney/2 {
		t.Errorf("huge: %v", amounts(got))
	}
	// Tiny weights never produce a zero installment.
	tiny := []PaymentTerm{
		{ID: 1, Label: "A", Amount: 1, DueDate: d("2027-01-01")},
		{ID: 2, Label: "B", Amount: 1_000_000, DueDate: d("2027-02-01")},
		{ID: 3, Label: "C", Amount: 1, DueDate: d("2027-03-01")},
	}
	got, err = RecalculateTerms(tiny, 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !sameAmounts(amounts(got), []int64{1, 1, 1}) {
		t.Errorf("tiny: %v", amounts(got))
	}
}
