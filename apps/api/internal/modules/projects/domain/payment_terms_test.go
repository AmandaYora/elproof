package domain

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestAllocateTerms(t *testing.T) {
	today := d("2026-11-07")
	terms := []PaymentTerm{
		{ID: 3, Label: "Pelunasan", Amount: 30, DueDate: d("2027-02-01")},
		{ID: 1, Label: "DP", Amount: 30, DueDate: d("2026-10-01")},
		{ID: 2, Label: "Termin 1", Amount: 40, DueDate: d("2026-11-10")},
	}
	cases := []struct {
		name    string
		applied int64
		want    []TermStatus // in due-date order: DP, Termin 1, Pelunasan
		paid    []int64
	}{
		{"nothing paid", 0, []TermStatus{TermOverdue, TermDueSoon, TermUpcoming}, []int64{0, 0, 0}},
		{"DP paid", 30, []TermStatus{TermPaid, TermDueSoon, TermUpcoming}, []int64{30, 0, 0}},
		{"DP paid, term 1 half", 50, []TermStatus{TermPaid, TermPartial, TermUpcoming}, []int64{30, 20, 0}},
		{"DP only half: still overdue", 10, []TermStatus{TermOverdue, TermDueSoon, TermUpcoming}, []int64{10, 0, 0}},
		{"all paid", 100, []TermStatus{TermPaid, TermPaid, TermPaid}, []int64{30, 40, 30}},
		{"overpaid is not allocated beyond the schedule", 500, []TermStatus{TermPaid, TermPaid, TermPaid}, []int64{30, 40, 30}},
		{"negative applied counts as nothing", -5, []TermStatus{TermOverdue, TermDueSoon, TermUpcoming}, []int64{0, 0, 0}},
	}
	for _, c := range cases {
		got := AllocateTerms(terms, c.applied, today)
		if len(got) != 3 {
			t.Fatalf("%s: len %d", c.name, len(got))
		}
		for i := range got {
			if got[i].Status != c.want[i] || got[i].Paid != c.paid[i] {
				t.Errorf("%s: term %d = (%s, paid %d), want (%s, %d)", c.name, i, got[i].Status, got[i].Paid, c.want[i], c.paid[i])
			}
		}
	}
}

func TestAllocateTermsDueTodayIsNotOverdue(t *testing.T) {
	got := AllocateTerms([]PaymentTerm{{ID: 1, Amount: 10, DueDate: d("2026-11-07")}}, 0, d("2026-11-07"))
	if got[0].Status != TermDueSoon {
		t.Fatalf("due today: %s", got[0].Status)
	}
}

func TestBuildTermAlerts(t *testing.T) {
	today := d("2026-11-07")
	rows := []TermWithProject{
		{Term: PaymentTerm{ID: 1, ProjectID: 1, Label: "DP", Amount: 30, DueDate: d("2026-10-01")}, ProjectName: "A"},
		{Term: PaymentTerm{ID: 2, ProjectID: 1, Label: "Pelunasan", Amount: 70, DueDate: d("2026-11-20")}, ProjectName: "A"},
		{Term: PaymentTerm{ID: 3, ProjectID: 2, Label: "DP", Amount: 50, DueDate: d("2026-09-01")}, ProjectName: "B"},
		{Term: PaymentTerm{ID: 4, ProjectID: 2, Label: "Pelunasan", Amount: 50, DueDate: d("2027-06-01")}, ProjectName: "B"},
	}
	// A paid 10 of its 30 DP; B paid its DP in full.
	overdue, total, soon := BuildTermAlerts(rows, map[int64]int64{1: 10, 2: 50}, today)
	if len(overdue) != 1 || overdue[0].ProjectName != "A" || overdue[0].Remaining != 20 || total != 20 {
		t.Fatalf("overdue: %+v total %d", overdue, total)
	}
	if len(soon) != 1 || soon[0].Label != "Pelunasan" || soon[0].Remaining != 70 {
		t.Fatalf("due soon: %+v", soon)
	}
}
