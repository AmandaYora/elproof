package domain

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestComputeListMoney(t *testing.T) {
	terms := []PaymentTerm{
		// Deliberately out of order: the rule sorts by due date, then id.
		{ID: 3, Label: "Pelunasan", Amount: 40_000_000, DueDate: day("2027-03-01")},
		{ID: 1, Label: "DP", Amount: 30_000_000, DueDate: day("2027-01-10")},
		{ID: 2, Label: "Termin 1", Amount: 30_000_000, DueDate: day("2027-02-10")},
	}
	pay := func(t PaymentType, amount int64) PaymentLine { return PaymentLine{Type: t, Amount: amount} }

	cases := []struct {
		name      string
		contract  int64
		payments  []PaymentLine
		terms     []PaymentTerm
		remaining int64
		nextDue   string // "" = nil
	}{
		{"nothing paid: first installment is next", 100_000_000, nil, terms, 100_000_000, "2027-01-10"},
		{"DP paid: next is Termin 1", 100_000_000, []PaymentLine{pay(PaymentDP, 30_000_000)}, terms, 70_000_000, "2027-02-10"},
		{"part of Termin 1 paid: it is still the next one", 100_000_000, []PaymentLine{pay(PaymentDP, 30_000_000), pay(PaymentTermin, 10_000_000)}, terms, 60_000_000, "2027-02-10"},
		{"all paid: nothing owed, no next date", 100_000_000, []PaymentLine{pay(PaymentDP, 30_000_000), pay(PaymentTermin, 30_000_000), pay(PaymentPelunasan, 40_000_000)}, terms, 0, ""},
		{"a Tambahan does not reduce what is owed", 100_000_000, []PaymentLine{pay(PaymentTambahan, 50_000_000)}, terms, 100_000_000, "2027-01-10"},
		{"a refund gives the money back", 100_000_000, []PaymentLine{pay(PaymentDP, 30_000_000), pay(PaymentRefund, 10_000_000)}, terms, 80_000_000, "2027-01-10"},
		{"refund larger than the payments never makes it negative either way", 100_000_000, []PaymentLine{pay(PaymentRefund, 5_000_000)}, terms, 100_000_000, "2027-01-10"},
		{"overpaid: remaining is 0, not negative", 50_000_000, []PaymentLine{pay(PaymentDP, 80_000_000)}, nil, 0, ""},
		{"no schedule: no next date", 100_000_000, []PaymentLine{pay(PaymentDP, 10_000_000)}, nil, 90_000_000, ""},
		{"no contract value", 0, nil, nil, 0, ""},
		{"same due date: lower id first", 10, nil, []PaymentTerm{{ID: 9, Amount: 5, DueDate: day("2027-05-05")}, {ID: 8, Amount: 5, DueDate: day("2027-05-05")}}, 10, "2027-05-05"},
	}
	for _, c := range cases {
		got := ComputeListMoney(c.contract, c.payments, c.terms)
		if got.Remaining != c.remaining {
			t.Errorf("%s: remaining %d, want %d", c.name, got.Remaining, c.remaining)
		}
		switch {
		case c.nextDue == "" && got.NextDueDate != nil:
			t.Errorf("%s: next due %v, want nil", c.name, got.NextDueDate)
		case c.nextDue != "" && (got.NextDueDate == nil || !got.NextDueDate.Equal(day(c.nextDue))):
			t.Errorf("%s: next due %v, want %s", c.name, got.NextDueDate, c.nextDue)
		}
	}
}

// The derived columns must agree with the overview's finance numbers (they are
// the same rule), whatever the mix of payments.
func TestComputeListMoneyAgreesWithTheOverview(t *testing.T) {
	lines := []PaymentLine{{Type: PaymentDP, Amount: 12}, {Type: PaymentRefund, Amount: 2}, {Type: PaymentTambahan, Amount: 99}}
	terms := []PaymentTerm{{ID: 1, Amount: 10, DueDate: day("2027-01-01")}, {ID: 2, Amount: 20, DueDate: day("2027-02-01")}}
	list := ComputeListMoney(30, lines, terms)
	overview := ComputeFinanceOverview(30, lines, AllocateTerms(terms, AppliedToContract(lines), day("2026-12-01")), nil, nil, nil)
	if list.Remaining != overview.Remaining {
		t.Errorf("list %d vs overview %d", list.Remaining, overview.Remaining)
	}
}
