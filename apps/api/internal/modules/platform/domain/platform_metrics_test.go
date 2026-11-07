package domain

import (
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

func TestComputePlatformMetrics(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	days := func(n int) *time.Time { d := now.AddDate(0, 0, n); return &d }
	p := DefaultSubscriptionPolicy // grace 7, read-only 23, expiring soon 7

	tenants := []Tenant{
		{ID: 1, BusinessName: "Trial baru", TrialEndsAt: days(10)},
		{ID: 2, BusinessName: "Trial segera habis", TrialEndsAt: days(2)},
		{ID: 3, BusinessName: "Berbayar bulanan", PlanID: ptr(int64(1)), SubscriptionExpiresAt: days(20)},
		{ID: 4, BusinessName: "Berbayar tahunan", PlanID: ptr(int64(2)), SubscriptionExpiresAt: days(200)},
		{ID: 5, BusinessName: "Kompensasi gratis", PlanID: ptr(int64(1)), SubscriptionExpiresAt: days(20)},
		{ID: 6, BusinessName: "Tenggang", PlanID: ptr(int64(1)), SubscriptionExpiresAt: days(-3)},
		// Fell into read-only 10 days ago (ended 17 days ago, grace 7): churned in the window.
		{ID: 7, BusinessName: "Churn", PlanID: ptr(int64(1)), SubscriptionExpiresAt: days(-17)},
		// Trial that ended 5 days ago and converted.
		{ID: 8, BusinessName: "Konversi", PlanID: ptr(int64(1)), TrialEndsAt: days(-5), SubscriptionExpiresAt: days(25)},
		// Trial that ended 5 days ago and did not.
		{ID: 9, BusinessName: "Tidak konversi", TrialEndsAt: days(-5)},
		{ID: 10, BusinessName: "Dibekukan", PlanID: ptr(int64(1)), SubscriptionExpiresAt: days(20), IsSuspended: true},
	}
	m := ComputePlatformMetrics(MetricsInput{
		Now: now, Policy: p, Tenants: tenants,
		PlanMonthly: map[int64]int64{1: 100_000, 2: 90_000}, // plan 2 is yearly at 1.080.000
		PlanPrice:   map[int64]int64{1: 100_000, 2: 1_080_000},
		FreeTenants: map[int64]bool{5: true},
		Paid: []PaidPayment{
			{TenantID: 8, Amount: 100_000, PaidAt: now.AddDate(0, 0, -4), IsNew: true},
			{TenantID: 3, Amount: 100_000, PaidAt: now.AddDate(0, 0, -10)},
			{TenantID: 3, Amount: 100_000, PaidAt: now.AddDate(0, 0, -40)},
		},
	})

	if m.TotalTenants != 10 {
		t.Errorf("total %d", m.TotalTenants)
	}
	// Paying: 3, 4, 6 (grace, still counts), 8, 10 (frozen is still a paying tenant). Not 5 (free).
	if m.PaidTenants != 5 {
		t.Errorf("paid tenants %d, want 5", m.PaidTenants)
	}
	// 100k + 90k + 100k + 100k + 100k
	if m.MRR != 490_000 || m.ARPA != 98_000 {
		t.Errorf("MRR %d ARPA %d", m.MRR, m.ARPA)
	}
	if m.TrialsEnded30d != 2 || m.TrialsConverted30d != 1 {
		t.Errorf("trials ended %d converted %d", m.TrialsEnded30d, m.TrialsConverted30d)
	}
	if got := m.TrialConversionPct(); got == nil || *got != 50 {
		t.Errorf("conversion %v", got)
	}
	if m.Churned30d != 1 || m.ChurnedMRR != 100_000 {
		t.Errorf("churn %d / %d", m.Churned30d, m.ChurnedMRR)
	}
	if m.NewPaid30d != 1 || m.NewMRR30d != 100_000 {
		t.Errorf("new %d / %d", m.NewPaid30d, m.NewMRR30d)
	}
	if m.Revenue30d != 200_000 || m.RevenuePrev30d != 100_000 {
		t.Errorf("revenue %d prev %d", m.Revenue30d, m.RevenuePrev30d)
	}
	// Renewals owed: tenant 6 (grace) and 7 (read-only).
	if m.OverdueValue != 200_000 || len(m.OverdueTenants) != 2 {
		t.Errorf("overdue %d tenants %d", m.OverdueValue, len(m.OverdueTenants))
	}
	if len(m.TrialsEndingSoon) != 1 || m.TrialsEndingSoon[0].TenantID != 2 {
		t.Errorf("trials ending soon %+v", m.TrialsEndingSoon)
	}
	if m.Frozen != 1 || m.Trialing != 2 {
		t.Errorf("frozen %d trialing %d", m.Frozen, m.Trialing)
	}
	// Churn base: 5 paying now + 1 churned - 1 new = 5 -> 20%.
	if got := m.ChurnPct(); got == nil || *got != 20 {
		t.Errorf("churn %v", got)
	}
}

func TestMetricsEmpty(t *testing.T) {
	m := ComputePlatformMetrics(MetricsInput{Now: time.Now(), Policy: DefaultSubscriptionPolicy})
	if m.TrialConversionPct() != nil || m.ChurnPct() != nil || m.MRR != 0 || m.ARPA != 0 {
		t.Errorf("empty platform: %+v", m)
	}
}

func TestMetricsFreeTrialExtensionIsNotAConversion(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	days := func(n int) *time.Time { d := now.AddDate(0, 0, n); return &d }
	m := ComputePlatformMetrics(MetricsInput{
		Now: now, Policy: DefaultSubscriptionPolicy,
		Tenants: []Tenant{
			// Trial ended 5 days ago; the team then granted a free extension.
			{ID: 1, BusinessName: "Diperpanjang gratis", PlanID: ptr(int64(1)), TrialEndsAt: days(-5), SubscriptionExpiresAt: days(9)},
			// Trial ended 5 days ago; the customer paid.
			{ID: 2, BusinessName: "Bayar", PlanID: ptr(int64(1)), TrialEndsAt: days(-5), SubscriptionExpiresAt: days(25)},
		},
		PlanMonthly: map[int64]int64{1: 100_000}, PlanPrice: map[int64]int64{1: 100_000},
		FreeTenants: map[int64]bool{1: true},
	})
	if m.TrialsEnded30d != 2 || m.TrialsConverted30d != 1 {
		t.Fatalf("ended %d converted %d, want 2 and 1", m.TrialsEnded30d, m.TrialsConverted30d)
	}
}

func TestMetricsTenantWithoutPlanDoesNotDiluteARPA(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	days := func(n int) *time.Time { d := now.AddDate(0, 0, n); return &d }
	m := ComputePlatformMetrics(MetricsInput{
		Now: now, Policy: DefaultSubscriptionPolicy,
		Tenants: []Tenant{
			{ID: 1, BusinessName: "Berpaket", PlanID: ptr(int64(1)), SubscriptionExpiresAt: days(20)},
			{ID: 2, BusinessName: "Paket terhapus", PlanID: ptr(int64(99)), SubscriptionExpiresAt: days(20)},
		},
		PlanMonthly: map[int64]int64{1: 100_000}, PlanPrice: map[int64]int64{1: 100_000},
	})
	if m.PaidTenants != 2 || m.MRR != 100_000 || m.ARPA != 100_000 {
		t.Fatalf("paid %d mrr %d arpa %d, want 2, 100000, 100000", m.PaidTenants, m.MRR, m.ARPA)
	}
}
