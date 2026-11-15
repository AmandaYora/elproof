package domain

import (
	"testing"
	"time"
)

func TestMilestoneTarget(t *testing.T) {
	cases := []struct {
		name    string
		event   string
		booking string
		days    int
		want    string
	}{
		{"H-30 from the event date", "2027-06-12", "2026-11-15", 30, "2027-05-13"},
		{"Hari-H is the event date", "2027-06-12", "2026-11-15", 0, "2027-06-12"},
		{"H+ is after the event", "2027-06-12", "2026-11-15", -3, "2027-06-15"},
		{"short notice: a step before booking is due on the booking date", "2026-12-20", "2026-11-15", 90, "2026-11-15"},
		{"short notice never moves Hari-H", "2026-11-20", "2026-11-15", 0, "2026-11-20"},
		{"historical project (booked after the event): real offsets, no pull", "2026-10-01", "2026-11-15", 14, "2026-09-17"},
		{"historical project: Hari-H is still the event day", "2026-10-01", "2026-11-15", 0, "2026-10-01"},
	}
	for _, c := range cases {
		got := MilestoneTarget(d(c.event), d(c.booking), c.days)
		if got.Format("2006-01-02") != c.want {
			t.Errorf("%s: %s, want %s", c.name, got.Format("2006-01-02"), c.want)
		}
	}
}

func TestShiftMilestones(t *testing.T) {
	ptr := func(n int) *int { return &n }
	ms := []ProjectMilestone{
		{ID: 1, Name: "Technical Meeting", Status: MilestoneNotStarted, TargetDate: d("2026-11-15"), DaysBeforeEvent: ptr(14)},
		{ID: 2, Name: "Hari-H Pernikahan", Status: MilestoneInProgress, TargetDate: d("2026-11-15"), DaysBeforeEvent: ptr(0)},
		{ID: 3, Name: "Survei", Status: MilestoneCompleted, TargetDate: d("2026-11-15"), DaysBeforeEvent: ptr(90)},
		{ID: 4, Name: "Dibuat manual", Status: MilestoneNotStarted, TargetDate: d("2026-12-01")},
		{ID: 5, Name: "Batal", Status: MilestoneCancelled, TargetDate: d("2026-11-15"), DaysBeforeEvent: ptr(7)},
		{ID: 6, Name: "Sudah pas", Status: MilestoneBlocked, TargetDate: d("2027-06-11"), DaysBeforeEvent: ptr(1)},
	}
	moved := ShiftMilestones(ms, d("2027-06-12"), d("2026-11-15"))
	if len(moved) != 2 {
		t.Fatalf("moved %d steps, want 2: %+v", len(moved), moved)
	}
	want := map[int64]string{1: "2027-05-29", 2: "2027-06-12"}
	for _, m := range moved {
		if m.TargetDate.Format("2006-01-02") != want[m.ID] {
			t.Errorf("step %d: %s, want %s", m.ID, m.TargetDate.Format("2006-01-02"), want[m.ID])
		}
	}
	if ms[0].TargetDate.Format("2006-01-02") != "2026-11-15" {
		t.Error("ShiftMilestones must not change its input")
	}
}

func TestIsMilestoneOverdueComparesWIBDays(t *testing.T) {
	due := d("2026-11-15") // DATE column: midnight of that day
	cases := []struct {
		name string
		asOf time.Time
		want bool
	}{
		{"due today, early morning WIB", time.Date(2026, 11, 15, 0, 30, 0, 0, WIB), false},
		{"due today, late evening WIB", time.Date(2026, 11, 15, 23, 59, 0, 0, WIB), false},
		{"due today, server clock in UTC (17:00 UTC = 00:00 WIB next day)", time.Date(2026, 11, 15, 16, 59, 0, 0, time.UTC), false},
		{"the day after (WIB), server clock in UTC", time.Date(2026, 11, 15, 17, 0, 0, 0, time.UTC), true},
		{"the day after", time.Date(2026, 11, 16, 8, 0, 0, 0, WIB), true},
		{"the day before", time.Date(2026, 11, 14, 8, 0, 0, 0, WIB), false},
	}
	for _, c := range cases {
		if got := IsMilestoneOverdue(MilestoneInProgress, due, c.asOf); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	if IsMilestoneOverdue(MilestoneCompleted, due, time.Date(2027, 1, 1, 0, 0, 0, 0, WIB)) {
		t.Error("a completed step is never late")
	}
	stats := ComputeMilestoneStats([]MilestoneLike{
		{Status: MilestoneNotStarted, TargetDate: due},
		{Status: MilestoneNotStarted, TargetDate: d("2026-11-14")},
	}, time.Date(2026, 11, 15, 10, 0, 0, 0, WIB))
	if stats.Overdue != 1 {
		t.Errorf("overdue %d, want 1 (only the step due yesterday)", stats.Overdue)
	}
}

func TestStampCompletion(t *testing.T) {
	today := time.Date(2026, 11, 15, 10, 0, 0, 0, WIB)
	var completed *time.Time
	if !StampCompletion(MilestoneInProgress, MilestoneCompleted, &completed, today) || completed == nil || completed.Format("2006-01-02") != "2026-11-15" {
		t.Fatalf("becoming Completed must stamp today, got %v", completed)
	}
	earlier := d("2026-11-01")
	completed = &earlier
	if StampCompletion(MilestoneInProgress, MilestoneCompleted, &completed, today) || completed.Format("2006-01-02") != "2026-11-01" {
		t.Error("an existing completed date is kept")
	}
	if !StampCompletion(MilestoneCompleted, MilestoneInProgress, &completed, today) || completed != nil {
		t.Error("leaving Completed clears the date")
	}
	if StampCompletion(MilestoneBlocked, MilestoneBlocked, &completed, today) {
		t.Error("no status change, no stamp")
	}
}
