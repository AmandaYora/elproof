package domain

import (
	"testing"
	"time"
)

func TestMondayOf(t *testing.T) {
	for in, want := range map[string]string{
		"2026-11-09": "2026-11-09", // a Monday
		"2026-11-10": "2026-11-09",
		"2026-11-15": "2026-11-09", // a Sunday belongs to the week that began on Monday
		"2026-11-16": "2026-11-16",
		"2026-01-01": "2025-12-29", // across a year
	} {
		if got := MondayOf(d(in)).Format("2006-01-02"); got != want {
			t.Errorf("MondayOf(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestCashFlowBucketsByWeekWithOverdueInWeekOne(t *testing.T) {
	today := d("2026-11-11") // Wednesday; week 1 starts Monday 2026-11-09
	in := []DueAmount{
		{d("2026-10-01"), 100}, // long overdue -> week 1
		{d("2026-11-10"), 10},  // already past this week -> week 1
		{d("2026-11-15"), 1},   // Sunday of week 1
		{d("2026-11-16"), 20},  // Monday of week 2
		{d("2026-12-27"), 5},   // Sunday of week 7 (Dec 21 + 6)
		{d("2026-12-28"), 7},   // Monday of week 8
		{d("2027-01-03"), 3},   // Sunday of week 8
		{d("2027-01-04"), 999}, // first day of week 9: left out
		{d("2026-11-20"), 0},   // nothing left to pay: counts for nothing
		{d("2026-11-20"), -50}, // never negative
	}
	out := []DueAmount{{d("2026-09-01"), 40}, {d("2026-11-23"), 8}, {d("2027-03-01"), 1000}}
	weeks := BuildCashFlow(today, in, out)
	if len(weeks) != CashFlowWeeks || CashFlowWeeks != 8 {
		t.Fatalf("want 8 weeks, got %d", len(weeks))
	}
	for i, w := range weeks {
		want := d("2026-11-09").AddDate(0, 0, 7*i)
		if !w.WeekStart.Equal(want) || w.WeekStart.Weekday() != time.Monday {
			t.Errorf("week %d starts %v, want the Monday %v", i+1, w.WeekStart, want)
		}
	}
	checkIn := map[int]int64{0: 111, 1: 20, 6: 5, 7: 10}
	checkOut := map[int]int64{0: 40, 2: 8}
	for i, w := range weeks {
		if w.ExpectedIn != checkIn[i] {
			t.Errorf("week %d expectedIn = %d, want %d", i+1, w.ExpectedIn, checkIn[i])
		}
		if w.ExpectedOut != checkOut[i] {
			t.Errorf("week %d expectedOut = %d, want %d", i+1, w.ExpectedOut, checkOut[i])
		}
	}
}

func TestCashFlowWithNothingOwedIsEightZeroWeeks(t *testing.T) {
	weeks := BuildCashFlow(d("2026-11-11"), nil, nil)
	if len(weeks) != 8 {
		t.Fatal(len(weeks))
	}
	for _, w := range weeks {
		if w.ExpectedIn != 0 || w.ExpectedOut != 0 {
			t.Errorf("%+v", w)
		}
	}
}

func TestVendorPaymentsDueWindowOrderAndCap(t *testing.T) {
	today := d("2026-11-10")
	var rows []VendorDue
	rows = append(rows,
		VendorDue{ProjectID: 1, DueDate: d("2026-12-10"), Remaining: 5}, // exactly 30 days: in
		VendorDue{ProjectID: 2, DueDate: d("2026-12-11"), Remaining: 5}, // 31 days: out
		VendorDue{ProjectID: 3, DueDate: d("2026-10-01"), Remaining: 5}, // overdue: in, first
		VendorDue{ProjectID: 4, DueDate: d("2026-11-12"), Remaining: 0}, // settled: out
		VendorDue{ProjectID: 5, DueDate: d("2026-11-11"), Remaining: 9},
	)
	got := SelectVendorPaymentsDue(today, rows)
	var ids []int64
	for _, r := range got {
		ids = append(ids, r.ProjectID)
	}
	if len(ids) != 3 || ids[0] != 3 || ids[1] != 5 || ids[2] != 1 {
		t.Fatalf("got %v, want [3 5 1] (overdue first, then nearest)", ids)
	}
	var many []VendorDue
	for i := 0; i < 25; i++ {
		many = append(many, VendorDue{ProjectID: int64(i), DueDate: today.AddDate(0, 0, i), Remaining: 1})
	}
	if got := SelectVendorPaymentsDue(today, many); len(got) != MaxDashboardRows || got[0].ProjectID != 0 {
		t.Errorf("capped at %d nearest first, got %d rows", MaxDashboardRows, len(got))
	}
}

func TestEngagementBalanceRemaining(t *testing.T) {
	for _, c := range []struct {
		contract, applied, want int64
	}{{100, 30, 70}, {100, 100, 0}, {100, 150, 0}, {100, -20, 100}, {0, 0, 0}} {
		if got := (EngagementBalanceRow{ContractValue: c.contract, Applied: c.applied}).Remaining(); got != c.want {
			t.Errorf("contract %d applied %d: remaining %d, want %d", c.contract, c.applied, got, c.want)
		}
	}
}
