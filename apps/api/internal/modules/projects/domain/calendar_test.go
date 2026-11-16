package domain

import "testing"

// day() is the shared test helper of project_list_test.go.

func TestParseCalendarKinds(t *testing.T) {
	all, ok := ParseCalendarKinds("")
	if !ok || !all.Has(CalendarEvent) || !all.Has(CalendarMilestone) || !all.Has(CalendarPayment) {
		t.Fatalf("empty kinds must mean every kind, got %v %v", all, ok)
	}
	k, ok := ParseCalendarKinds(" event , payment ")
	if !ok || !k.Has(CalendarEvent) || !k.Has(CalendarPayment) || k.Has(CalendarMilestone) {
		t.Fatalf("event,payment: got %v %v", k, ok)
	}
	if _, ok := ParseCalendarKinds("event,meeting"); ok {
		t.Fatal("an unknown kind must be refused, not ignored")
	}
	if k, ok := ParseCalendarKinds(","); !ok || len(k) != 3 {
		t.Fatalf("only separators is the same as empty, got %v %v", k, ok)
	}
}

func TestCalendarRangeProblem(t *testing.T) {
	cases := []struct {
		from, to string
		bad      bool
	}{
		{"2026-11-01", "2026-11-30", false},
		{"2026-10-27", "2026-12-07", false}, // a six-week month grid
		{"2026-11-01", "2026-11-01", false},
		{"2026-11-01", "2027-02-01", false}, // 93 days inclusive
		{"2026-11-01", "2027-02-02", true},  // 94 days
		{"2026-11-02", "2026-11-01", true},
	}
	for _, c := range cases {
		if got := CalendarRangeProblem(day(c.from), day(c.to)) != ""; got != c.bad {
			t.Errorf("%s..%s: problem=%v, want %v", c.from, c.to, got, c.bad)
		}
	}
}

// The payments of a project are allocated over its whole schedule, so an
// installment in the range is only "paid" when the earlier ones (outside the
// range) are covered too; installments outside the range are not returned.
func TestCalendarPaymentItemsAllocatesWholeSchedule(t *testing.T) {
	rows := []TermWithProject{
		{Term: PaymentTerm{ID: 1, ProjectID: 7, Label: "DP", Amount: 30, DueDate: day("2026-10-01")}, ProjectName: "Citra & Bima"},
		{Term: PaymentTerm{ID: 2, ProjectID: 7, Label: "Termin 2", Amount: 40, DueDate: day("2026-11-10")}, ProjectName: "Citra & Bima"},
		{Term: PaymentTerm{ID: 3, ProjectID: 7, Label: "Pelunasan", Amount: 30, DueDate: day("2026-11-28")}, ProjectName: "Citra & Bima"},
		{Term: PaymentTerm{ID: 9, ProjectID: 8, Label: "DP", Amount: 50, DueDate: day("2026-11-20")}, ProjectName: "Ayu & Dimas"},
	}
	applied := map[int64]int64{7: 50, 8: 50}
	items := CalendarPaymentItems(rows, applied, day("2026-11-01"), day("2026-11-30"), day("2026-11-16"))
	if len(items) != 3 {
		t.Fatalf("want the three installments due in November, got %d: %+v", len(items), items)
	}
	byRef := map[int64]CalendarItem{}
	for _, it := range items {
		if it.Kind != CalendarPayment || it.Amount == nil || it.Remaining == nil {
			t.Fatalf("payment items carry kind, amount and remaining: %+v", it)
		}
		byRef[it.RefID] = it
	}
	// 50 paid: DP (30) covered, Termin 2 gets 20 of 40 and its due date passed.
	if it := byRef[2]; it.Status != string(TermOverdue) || *it.Remaining != 20 || *it.Amount != 40 || it.ProjectName != "Citra & Bima" {
		t.Errorf("Termin 2: %+v (remaining %d)", it, *it.Remaining)
	}
	if it := byRef[3]; it.Status != string(TermUpcoming) || *it.Remaining != 30 {
		t.Errorf("Pelunasan: %+v", it)
	}
	if it := byRef[9]; it.Status != string(TermPaid) || *it.Remaining != 0 {
		t.Errorf("fully paid DP: %+v", it)
	}
}

// A due date of today is not overdue yet (WIB day comparison, ADR-0042).
func TestCalendarPaymentDueTodayIsNotOverdue(t *testing.T) {
	rows := []TermWithProject{{Term: PaymentTerm{ID: 1, ProjectID: 1, Label: "DP", Amount: 10, DueDate: day("2026-11-16")}, ProjectName: "X"}}
	items := CalendarPaymentItems(rows, nil, day("2026-11-01"), day("2026-11-30"), day("2026-11-16"))
	if len(items) != 1 || items[0].Status != string(TermDueSoon) {
		t.Fatalf("due today: %+v", items)
	}
}

func TestSortCalendarItems(t *testing.T) {
	items := []CalendarItem{
		{Date: day("2026-11-20"), Kind: CalendarMilestone, ProjectName: "A", RefID: 5},
		{Date: day("2026-11-20"), Kind: CalendarPayment, ProjectName: "B", RefID: 1},
		{Date: day("2026-11-20"), Kind: CalendarEvent, ProjectName: "C", ProjectID: 3},
		{Date: day("2026-11-02"), Kind: CalendarMilestone, ProjectName: "Z", RefID: 9},
	}
	SortCalendarItems(items)
	want := []CalendarKind{CalendarMilestone, CalendarEvent, CalendarPayment, CalendarMilestone}
	for i, it := range items {
		if it.Kind != want[i] {
			t.Fatalf("order %d: got %s, want %s (%+v)", i, it.Kind, want[i], items)
		}
	}
	if !items[0].Date.Equal(day("2026-11-02")) {
		t.Fatal("earliest day first")
	}
}
