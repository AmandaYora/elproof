package domain

import (
	"testing"
	"time"
)

func fd(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// TermStatusOf must be exactly the status AllocateTerms gives (the finance list
// reproduces the rule in SQL; the isolation suite compares the two on a real DB).
func TestTermStatusOfMatchesAllocateTerms(t *testing.T) {
	today := fd("2026-11-09")
	terms := []PaymentTerm{
		{ID: 1, ProjectID: 7, Label: "DP", Amount: 30, DueDate: fd("2026-10-01")},
		{ID: 2, ProjectID: 7, Label: "Termin 1", Amount: 40, DueDate: fd("2026-11-02")},
		{ID: 3, ProjectID: 7, Label: "Termin 2", Amount: 20, DueDate: fd("2026-11-12")},
		{ID: 4, ProjectID: 7, Label: "Termin 3", Amount: 20, DueDate: fd("2026-11-16")},
		{ID: 5, ProjectID: 7, Label: "Pelunasan", Amount: 50, DueDate: fd("2027-01-20")},
	}
	for _, applied := range []int64{0, 10, 30, 50, 70, 80, 95, 130, 999} {
		for _, tp := range AllocateTerms(terms, applied, today) {
			if got := TermStatusOf(tp.Paid, tp.Remaining, tp.Term.DueDate, today); got != tp.Status {
				t.Errorf("applied %d, term %d: TermStatusOf = %s, AllocateTerms = %s", applied, tp.Term.ID, got, tp.Status)
			}
		}
	}
	// Spot checks of each branch.
	cases := []struct {
		paid, remaining int64
		due             string
		want            TermStatus
	}{
		{30, 0, "2026-10-01", TermPaid},
		{10, 20, "2026-11-08", TermOverdue}, // partly paid but past due: overdue wins
		{10, 20, "2026-11-09", TermPartial}, // due today is not late
		{0, 20, "2026-11-16", TermDueSoon},  // 7 days ahead
		{0, 20, "2026-11-17", TermUpcoming}, // 8 days ahead
	}
	for _, c := range cases {
		if got := TermStatusOf(c.paid, c.remaining, fd(c.due), today); got != c.want {
			t.Errorf("paid %d remaining %d due %s: got %s, want %s", c.paid, c.remaining, c.due, got, c.want)
		}
	}
}

func TestDaysLate(t *testing.T) {
	today := fd("2026-11-09")
	if got := DaysLate(fd("2026-11-02"), today, 5); got != 7 {
		t.Errorf("7 days late: got %d", got)
	}
	if got := DaysLate(fd("2026-11-09"), today, 5); got != 0 {
		t.Errorf("due today: got %d", got)
	}
	if got := DaysLate(fd("2026-11-02"), today, 0); got != 0 {
		t.Errorf("paid: got %d", got)
	}
}

func TestSummarizeReceivables(t *testing.T) {
	today := fd("2026-11-09")
	row := func(project int64, due string, paid, remaining int64) ReceivableRow {
		return ReceivableRow{Term: PaymentTerm{ProjectID: project, DueDate: fd(due), Amount: paid + remaining}, Paid: paid, Remaining: remaining}
	}
	rows := []ReceivableRow{
		row(1, "2026-11-02", 10, 20), // overdue, partly paid
		row(1, "2026-10-01", 0, 5),   // overdue, same project
		row(2, "2026-11-08", 0, 40),  // overdue, another project
		row(2, "2026-11-09", 0, 7),   // due today: due soon, not overdue
		row(3, "2026-12-09", 0, 11),  // day 30: inside the window
		row(3, "2026-12-10", 0, 13),  // day 31: outside
		row(4, "2026-11-20", 100, 0), // paid: counts for nothing
		row(4, "2026-11-01", 50, -1), // nonsense remaining: ignored
	}
	overdue, dueSoon, due := SummarizeReceivables(rows, today)
	if overdue != (MoneyCount{Amount: 65, Count: 3, Projects: 2}) {
		t.Errorf("overdue = %+v", overdue)
	}
	if dueSoon != (MoneyCount{Amount: 18, Count: 2, Projects: 2}) {
		t.Errorf("dueSoon = %+v", dueSoon)
	}
	if len(due) != 6 {
		t.Errorf("due amounts for the cash flow: got %d, want 6 (every unpaid installment)", len(due))
	}
	// The cash flow is the dashboard's own function over these amounts: overdue
	// lands in week 1, beyond 8 weeks is left out.
	weeks := BuildCashFlow(today, due, nil)
	var in int64
	for _, w := range weeks {
		in += w.ExpectedIn
	}
	if weeks[0].ExpectedIn != 20+5+40+7 || in != 20+5+40+7+11+13 {
		t.Errorf("cash flow in: week1 %d total %d", weeks[0].ExpectedIn, in)
	}
}

func TestPayables(t *testing.T) {
	today := fd("2026-11-09")
	past, future := fd("2026-11-01"), fd("2026-11-20")
	rows := []PayableRow{
		{Kind: SupplierVendor, AgreedCost: 100, Paid: 30, DueDate: &past},  // 70 overdue
		{Kind: SupplierVendor, AgreedCost: 50, Paid: 0, DueDate: &future},  // 50 unpaid
		{Kind: SupplierVendor, AgreedCost: 40, Paid: 0},                    // 40 unpaid, no due date
		{Kind: SupplierVenue, AgreedCost: 200, Paid: 50},                   // 150 unpaid, venue
		{Kind: SupplierVendor, AgreedCost: 20, Paid: 25, DueDate: &past},   // overpaid: nothing left
		{Kind: SupplierVendor, Paid: 10, DueDate: &past, Incomplete: true}, // no agreed cost: listed, counted, no amount
	}
	wantStatus := []PayableStatus{PayableOverdue, PayableUnpaid, PayableUnpaid, PayableUnpaid, PayablePaid, PayableIncomplete}
	for i, r := range rows {
		if got := r.Status(today); got != wantStatus[i] {
			t.Errorf("row %d: status %s, want %s", i, got, wantStatus[i])
		}
	}
	if rows[4].Remaining() != 0 {
		t.Errorf("overpaid remaining = %d, want 0", rows[4].Remaining())
	}
	sum, due := SummarizePayables(rows, today)
	if rows[5].Remaining() != 0 {
		t.Errorf("an incomplete row owes no known amount: %d", rows[5].Remaining())
	}
	if sum != (PayablesSummary{Amount: 310, Count: 4, OverdueAmount: 70, OverdueCount: 1, IncompleteCount: 1}) {
		t.Errorf("summary = %+v", sum)
	}
	// Only balances with a due date go into the cash flow (as /dashboard).
	if len(due) != 2 || due[0].Remaining != 70 || due[1].Remaining != 50 {
		t.Errorf("due = %+v", due)
	}
}

func TestReceivedAmount(t *testing.T) {
	got := ReceivedAmount([]PaymentLine{
		{Type: PaymentDP, Amount: 100}, {Type: PaymentTermin, Amount: 50}, {Type: PaymentTambahan, Amount: 20}, {Type: PaymentRefund, Amount: 30},
	})
	if got != 140 {
		t.Errorf("got %d, want 140 (a Tambahan is money received, a Refund goes back)", got)
	}
}
