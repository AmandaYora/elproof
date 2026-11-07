package domain

import (
	"testing"
	"time"
)

func TestValidatePolicy(t *testing.T) {
	if f := ValidatePolicy(DefaultSubscriptionPolicy); len(f) != 0 {
		t.Fatalf("defaults must be valid: %v", f)
	}
	bad := SubscriptionPolicy{TrialDays: 0, ExpiringSoonDays: 31, GraceDays: -1, ReadOnlyDays: 181}
	if f := ValidatePolicy(bad); len(f) != 4 {
		t.Fatalf("every field out of range must be reported: %v", f)
	}
}

func TestCurrentPolicyCanBeReplaced(t *testing.T) {
	defer SetSubscriptionPolicy(DefaultSubscriptionPolicy)
	SetSubscriptionPolicy(SubscriptionPolicy{TrialDays: 30, ExpiringSoonDays: 5, GraceDays: 3, ReadOnlyDays: 10})
	if CurrentSubscriptionPolicy().TrialDays != 30 {
		t.Fatal("policy not replaced")
	}
	// The lifecycle follows the policy in force.
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	end := now.AddDate(0, 0, -4) // ended 4 days ago: past the 3-day grace
	lc := ComputeLifecycle(Tenant{SubscriptionExpiresAt: &end}, now, CurrentSubscriptionPolicy())
	if lc.Status != StatusReadOnly {
		t.Fatalf("status %s, want read_only under the shorter grace", lc.Status)
	}
}

func TestAnnouncementWindowAndTargeting(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	end := now.Add(time.Hour)
	a := Announcement{StartsAt: now.Add(-time.Hour), EndsAt: &end}
	if !a.ActiveAt(now) || a.ActiveAt(now.Add(2*time.Hour)) || a.ActiveAt(now.Add(-2*time.Hour)) {
		t.Fatal("window")
	}
	open := Announcement{StartsAt: now.Add(-time.Hour)}
	if !open.ActiveAt(now.AddDate(1, 0, 0)) {
		t.Fatal("no end date means until withdrawn")
	}
	plan := int64(7)
	tenant := Tenant{PlanID: &plan}
	lc := Lifecycle{Status: StatusPastDue}
	cases := []struct {
		a    Announcement
		want bool
	}{
		{Announcement{Audience: AudienceAll}, true},
		{Announcement{Audience: AudiencePlan, AudienceValue: "7"}, true},
		{Announcement{Audience: AudiencePlan, AudienceValue: "8"}, false},
		{Announcement{Audience: AudienceStatus, AudienceValue: "past_due"}, true},
		{Announcement{Audience: AudienceStatus, AudienceValue: "trialing"}, false},
	}
	for _, c := range cases {
		if got := c.a.Targets(tenant, lc); got != c.want {
			t.Errorf("%+v: got %v", c.a, got)
		}
	}
	if (Announcement{Audience: AudiencePlan, AudienceValue: "7"}).Targets(Tenant{}, lc) {
		t.Error("a tenant without a plan is not on plan 7")
	}
}

func TestSupportAccessGrant(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	g := SupportAccessGrant{ExpiresAt: now.Add(time.Hour)}
	if !g.ActiveAt(now) || g.ActiveAt(now.Add(2*time.Hour)) {
		t.Fatal("expiry")
	}
	revoked := now
	g.RevokedAt = &revoked
	if g.ActiveAt(now) {
		t.Fatal("a revoked grant is not active")
	}
}
