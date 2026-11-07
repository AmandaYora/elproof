package domain

import (
	"testing"
	"time"
)

func TestComputeTrendsMonth(t *testing.T) {
	// 7 Nov 2026, 12:00 WIB.
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, wibZone)
	paid := []PaidPayment{
		{Amount: 100, PaidAt: time.Date(2026, 11, 7, 9, 0, 0, 0, wibZone)},
		{Amount: 50, PaidAt: time.Date(2026, 11, 7, 11, 0, 0, 0, wibZone)},
		{Amount: 70, PaidAt: time.Date(2026, 11, 1, 0, 30, 0, 0, wibZone)},
		{Amount: 999, PaidAt: time.Date(2026, 10, 31, 23, 0, 0, 0, wibZone)}, // last month
		{Amount: 999, PaidAt: time.Date(2025, 11, 7, 9, 0, 0, 0, wibZone)},   // last year
		{Amount: 999, PaidAt: now.Add(48 * time.Hour)},                       // future
	}
	// 6 Nov 17:30 UTC is already 7 Nov 00:30 in WIB.
	joined := []time.Time{time.Date(2026, 11, 6, 17, 30, 0, 0, time.UTC), time.Date(2026, 11, 2, 8, 0, 0, 0, wibZone)}
	got := ComputeTrends(TrendMonth, now, paid, joined)
	if len(got.Revenue) != 7 || got.Revenue[0].Label != "1" || got.Revenue[6].Label != "7" {
		t.Fatalf("month view must run 1..7, got %d points", len(got.Revenue))
	}
	if got.Revenue[6].Value != 150 || got.Revenue[0].Value != 70 {
		t.Errorf("revenue by day: day7=%d day1=%d", got.Revenue[6].Value, got.Revenue[0].Value)
	}
	if got.NewTenants[6].Value != 1 || got.NewTenants[1].Value != 1 {
		t.Errorf("new tenants by day: day7=%d day2=%d (UTC evening belongs to the next WIB day)", got.NewTenants[6].Value, got.NewTenants[1].Value)
	}
	var total int64
	for _, p := range got.Revenue {
		total += p.Value
	}
	if total != 220 {
		t.Errorf("revenue total %d, want 220 (other months, years and the future are ignored)", total)
	}
}

func TestComputeTrendsYear(t *testing.T) {
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, wibZone)
	got := ComputeTrends(TrendYear, now, []PaidPayment{
		{Amount: 10, PaidAt: time.Date(2026, 1, 5, 0, 0, 0, 0, wibZone)},
		{Amount: 20, PaidAt: time.Date(2026, 3, 1, 0, 0, 0, 0, wibZone)},
		{Amount: 5, PaidAt: time.Date(2025, 12, 31, 20, 0, 0, 0, wibZone)},
	}, nil)
	if len(got.Revenue) != 3 || got.Revenue[0].Label != "Jan" || got.Revenue[2].Label != "Mar" {
		t.Fatalf("year view must run Jan..Mar, got %v", got.Revenue)
	}
	if got.Revenue[0].Value != 10 || got.Revenue[2].Value != 20 {
		t.Errorf("revenue by month: %v", got.Revenue)
	}
	if got := ComputeTrends("nonsense", now, nil, nil); got.Period != TrendMonth {
		t.Errorf("an unknown period falls back to month, got %q", got.Period)
	}
}
