package domain

import (
	"testing"
	"time"
)

func TestComputeLifecyclePaidPeriod(t *testing.T) {
	p := DefaultSubscriptionPolicy
	end := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	tenant := Tenant{SubscriptionExpiresAt: &end}

	cases := []struct {
		name   string
		now    time.Time
		status SubscriptionStatus
		access AccessLevel
	}{
		{"well before end", end.AddDate(0, 0, -30), StatusActive, AccessFull},
		{"inside warning window", end.AddDate(0, 0, -3), StatusExpiringSoon, AccessFull},
		{"just after end: grace", end.Add(time.Hour), StatusPastDue, AccessFull},
		{"last grace day", end.AddDate(0, 0, 7).Add(-time.Minute), StatusPastDue, AccessFull},
		{"after grace: read-only", end.AddDate(0, 0, 7), StatusReadOnly, AccessReadOnly},
		{"end of read-only", end.AddDate(0, 0, 30).Add(-time.Minute), StatusReadOnly, AccessReadOnly},
		{"after read-only: billing only", end.AddDate(0, 0, 30), StatusSuspended, AccessBillingOnly},
		{"long after: still kept, never deleted", end.AddDate(1, 0, 0), StatusSuspended, AccessBillingOnly},
	}
	for _, c := range cases {
		lc := ComputeLifecycle(tenant, c.now, p)
		if lc.Status != c.status || lc.Access != c.access {
			t.Errorf("%s: got %s/%s, want %s/%s", c.name, lc.Status, lc.Access, c.status, c.access)
		}
		if lc.IsTrial {
			t.Errorf("%s: a paid period must not be reported as trial", c.name)
		}
	}
}

func TestComputeLifecycleTrial(t *testing.T) {
	p := DefaultSubscriptionPolicy
	trialEnd := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	tenant := Tenant{TrialEndsAt: &trialEnd}

	lc := ComputeLifecycle(tenant, trialEnd.AddDate(0, 0, -10), p)
	if lc.Status != StatusTrialing || lc.Access != AccessFull || !lc.IsTrial || lc.DaysLeft != 10 {
		t.Errorf("early trial: %+v", lc)
	}
	lc = ComputeLifecycle(tenant, trialEnd.AddDate(0, 0, -2), p)
	if lc.Status != StatusTrialing || lc.DaysLeft != 2 {
		t.Errorf("trial in warning window keeps trialing status: %+v", lc)
	}
	lc = ComputeLifecycle(tenant, trialEnd.AddDate(0, 0, 1), p)
	if lc.Status != StatusPastDue || lc.Access != AccessFull {
		t.Errorf("trial grace: %+v", lc)
	}
	lc = ComputeLifecycle(tenant, trialEnd.AddDate(0, 0, 8), p)
	if lc.Status != StatusReadOnly || lc.Access != AccessReadOnly {
		t.Errorf("trial ended past grace: %+v", lc)
	}

	// Paying during the trial: the paid period wins.
	paidEnd := trialEnd.AddDate(1, 0, 0)
	tenant.SubscriptionExpiresAt = &paidEnd
	lc = ComputeLifecycle(tenant, trialEnd.AddDate(0, 0, 8), p)
	if lc.Status != StatusActive || lc.IsTrial {
		t.Errorf("paid after trial: %+v", lc)
	}
}

func TestComputeLifecycleFreezeAndLegacy(t *testing.T) {
	p := DefaultSubscriptionPolicy
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	end := now.AddDate(0, 6, 0)

	frozen := ComputeLifecycle(Tenant{SubscriptionExpiresAt: &end, IsSuspended: true}, now, p)
	if frozen.Access != AccessBlocked || !frozen.Frozen || frozen.Status != StatusActive {
		t.Errorf("platform freeze must block regardless of billing: %+v", frozen)
	}

	legacy := ComputeLifecycle(Tenant{}, now, p)
	if legacy.Status != StatusPendingPayment || legacy.Access != AccessReadOnly || legacy.PeriodEndsAt != nil {
		t.Errorf("legacy tenant without period: %+v", legacy)
	}
}

func TestDaysUntil(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	if got := daysUntil(now, now.Add(3*time.Hour)); got != 0 {
		t.Errorf("later today = %d, want 0", got)
	}
	if got := daysUntil(now, now.Add(14*24*time.Hour+500*time.Millisecond)); got != 14 {
		t.Errorf("14 days + rounding = %d, want 14", got)
	}
	if got := daysUntil(now, now.Add(13*time.Hour)); got != 1 {
		t.Errorf("tomorrow 01:00 = %d, want 1", got)
	}
	if got := daysUntil(now, now.Add(-3*time.Hour)); got != 0 {
		t.Errorf("3h ago = %d, want 0", got)
	}
	if got := daysUntil(now, now.AddDate(0, 0, -2)); got != -2 {
		t.Errorf("2 days ago = %d, want -2", got)
	}
}
