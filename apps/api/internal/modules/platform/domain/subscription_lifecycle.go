package domain

import "time"

// AccessLevel is what a tenant's staff may do right now. The Client Portal
// is read-only by construction and stays readable at every billing level —
// a WO's unpaid invoice must not lock the couple out (product decision K4).
type AccessLevel string

const (
	// AccessFull: normal use (trial, paid, or within the grace period).
	AccessFull AccessLevel = "full"
	// AccessReadOnly: staff can view and export but not create or change.
	AccessReadOnly AccessLevel = "read_only"
	// AccessBillingOnly: staff can only reach the subscription/payment
	// pages (to pay and reactivate); data is kept, never auto-deleted (K10).
	AccessBillingOnly AccessLevel = "billing_only"
	// AccessBlocked: frozen by the platform team; nobody in the tenant may
	// log in or use the API.
	AccessBlocked AccessLevel = "blocked"
)

// SubscriptionPolicy holds the lifecycle durations in days.
type SubscriptionPolicy struct {
	TrialDays        int // length of a new tenant's free trial
	ExpiringSoonDays int // warn this many days before the period ends
	GraceDays        int // full access after the period ends, with banner
	ReadOnlyDays     int // read-only after grace, before billing-only
}

// DefaultSubscriptionPolicy follows product decisions K1/K4/K10 (PLAN.md §6).
var DefaultSubscriptionPolicy = SubscriptionPolicy{
	TrialDays:        14,
	ExpiringSoonDays: 7,
	GraceDays:        7,
	ReadOnlyDays:     23,
}

// Lifecycle is a tenant's subscription state at one moment.
type Lifecycle struct {
	Status SubscriptionStatus
	Access AccessLevel
	// PeriodEndsAt is when the current paid period or trial ends (nil for
	// a legacy tenant that has neither).
	PeriodEndsAt *time.Time
	// ReadOnlyAt / SuspendAt are when the next downgrades happen (nil once
	// passed or when there is no period).
	ReadOnlyAt *time.Time
	SuspendAt  *time.Time
	// DaysLeft is whole days until PeriodEndsAt (negative once passed).
	DaysLeft int
	// IsTrial is true while the period is a trial, not a paid period.
	IsTrial bool
	// Frozen is the platform-team freeze (Tenant.IsSuspended).
	Frozen bool
}

const day = 24 * time.Hour

// ComputeLifecycle derives the subscription state purely from the tenant's
// dates and the policy — no stored status is trusted, so a missed sweep can
// never leave a tenant with more (or less) access than it should have.
//
//	trialing/active ──(H-ExpiringSoonDays)──► expiring_soon (still full)
//	──(period end)──► past_due (full, grace) ──(+GraceDays)──► read_only
//	──(+ReadOnlyDays)──► suspended (billing only; data kept)
//
// A paid period (SubscriptionExpiresAt) always wins over a trial.
func ComputeLifecycle(t Tenant, now time.Time, p SubscriptionPolicy) Lifecycle {
	var end *time.Time
	isTrial := false
	switch {
	case t.SubscriptionExpiresAt != nil:
		end = t.SubscriptionExpiresAt
	case t.TrialEndsAt != nil:
		end = t.TrialEndsAt
		isTrial = true
	}

	lc := Lifecycle{IsTrial: isTrial, Frozen: t.IsSuspended}
	if end == nil {
		// Legacy tenant with neither a trial nor a paid period.
		lc.Status, lc.Access = StatusPendingPayment, AccessReadOnly
	} else {
		e := *end
		lc.PeriodEndsAt = &e
		lc.DaysLeft = daysUntil(now, e)
		readOnlyAt := e.Add(time.Duration(p.GraceDays) * day)
		suspendAt := readOnlyAt.Add(time.Duration(p.ReadOnlyDays) * day)

		switch {
		case now.Before(e.Add(-time.Duration(p.ExpiringSoonDays) * day)):
			lc.Status, lc.Access = StatusActive, AccessFull
			if isTrial {
				lc.Status = StatusTrialing
			}
		case now.Before(e):
			lc.Status, lc.Access = StatusExpiringSoon, AccessFull
			if isTrial {
				lc.Status = StatusTrialing
			}
		case now.Before(readOnlyAt):
			lc.Status, lc.Access = StatusPastDue, AccessFull
		case now.Before(suspendAt):
			lc.Status, lc.Access = StatusReadOnly, AccessReadOnly
		default:
			lc.Status, lc.Access = StatusSuspended, AccessBillingOnly
		}
		if now.Before(readOnlyAt) {
			lc.ReadOnlyAt = &readOnlyAt
		}
		if now.Before(suspendAt) {
			lc.SuspendAt = &suspendAt
		}
	}

	if t.IsSuspended {
		lc.Access = AccessBlocked
	}
	return lc
}

// daysUntil rounds up partial days so "ends in 3h" reads as 1 day left and
// "ended 3h ago" as 0, matching how people read a countdown.
func daysUntil(now, end time.Time) int {
	d := end.Sub(now)
	if d >= 0 {
		return int((d + day - 1) / day)
	}
	return -int(-d / day)
}
