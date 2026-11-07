package domain

import (
	"sort"
	"time"
)

// PaidPayment is one successfully paid subscription transaction (granted
// activations are never included: they are not revenue).
type PaidPayment struct {
	TenantID int64
	Amount   int64
	PaidAt   time.Time
	// IsNew is a first subscription (type "new"), not a renewal.
	IsNew bool
}

// MetricsInput is everything the platform metrics are computed from.
type MetricsInput struct {
	Now    time.Time
	Policy SubscriptionPolicy
	// Tenants is every workspace.
	Tenants []Tenant
	// PlanMonthly maps plan id -> price per month (yearly plans / 12, rounded).
	PlanMonthly map[int64]int64
	// PlanPrice maps plan id -> full price of one period.
	PlanPrice map[int64]int64
	// FreeTenants are tenants whose current paid period was granted for free
	// (compensation, trial extension, internal): active, but not revenue.
	FreeTenants map[int64]bool
	// Paid are the paid transactions of the last 60 days.
	Paid []PaidPayment
}

// AttentionTenant is a tenant that needs the platform team's attention.
type AttentionTenant struct {
	TenantID int64
	Name     string
	Status   SubscriptionStatus
	// DaysLeft: days until the period ends (negative = days past).
	DaysLeft int
}

// PlatformMetrics follows the definitions in FLOW-IDEAL.md §1.3.
type PlatformMetrics struct {
	TotalTenants int
	// Counts by lifecycle state (a tenant is in exactly one; Frozen is separate).
	Trialing    int
	Active      int // paid: active + expiring_soon
	PastDue     int // past the end, within grace (still full access)
	ReadOnly    int
	Suspended   int // billing only
	NoPeriod    int
	Frozen      int
	PaidTenants int // paid period, not free, status active/expiring_soon/past_due

	// MRR: monthly value of paid (non-free) subscriptions; trial and free grants excluded.
	MRR  int64
	ARPA int64

	TrialsEnded30d     int
	TrialsConverted30d int

	Churned30d int
	ChurnedMRR int64
	NewPaid30d int
	NewMRR30d  int64

	Revenue30d     int64
	RevenuePrev30d int64
	// OverdueValue: what renewals owed by tenants past their period would bring.
	OverdueValue int64

	TrialsEndingSoon []AttentionTenant
	OverdueTenants   []AttentionTenant
}

// TrialConversionPct is converted/ended as a percentage; nil when no trial ended.
func (m PlatformMetrics) TrialConversionPct() *float64 {
	if m.TrialsEnded30d == 0 {
		return nil
	}
	v := float64(m.TrialsConverted30d) * 100 / float64(m.TrialsEnded30d)
	return &v
}

// ChurnPct is churned / (paying at the start of the window); nil when there was nobody.
func (m PlatformMetrics) ChurnPct() *float64 {
	base := m.PaidTenants + m.Churned30d - m.NewPaid30d
	if base <= 0 {
		return nil
	}
	v := float64(m.Churned30d) * 100 / float64(base)
	return &v
}

const attentionListMax = 10

// ComputePlatformMetrics derives the platform's SaaS numbers from tenants,
// plans and the paid ledger. Pure: no I/O, so every definition is testable.
func ComputePlatformMetrics(in MetricsInput) PlatformMetrics {
	var m PlatformMetrics
	m.TotalTenants = len(in.Tenants)
	window := 30 * 24 * time.Hour
	since := in.Now.Add(-window)

	// priced counts paying tenants whose plan is known: one with a deleted plan
	// has no monthly value and must not dilute the average.
	priced := 0
	newlyPaid := map[int64]bool{}
	for _, p := range in.Paid {
		switch {
		case !p.PaidAt.Before(since):
			m.Revenue30d += p.Amount
			if p.IsNew {
				newlyPaid[p.TenantID] = true
			}
		case !p.PaidAt.Before(in.Now.Add(-2 * window)):
			m.RevenuePrev30d += p.Amount
		}
	}

	for _, t := range in.Tenants {
		lc := ComputeLifecycle(t, in.Now, in.Policy)
		if t.IsSuspended {
			m.Frozen++
		}
		paidPeriod := t.SubscriptionExpiresAt != nil
		free := in.FreeTenants[t.ID]
		monthly := in.PlanMonthly[planOf(t)]

		switch lc.Status {
		case StatusTrialing:
			m.Trialing++
		case StatusActive, StatusExpiringSoon:
			m.Active++
		case StatusPastDue:
			m.PastDue++
		case StatusReadOnly:
			m.ReadOnly++
		case StatusSuspended:
			m.Suspended++
		default:
			m.NoPeriod++
		}

		paying := paidPeriod && !free && (lc.Status == StatusActive || lc.Status == StatusExpiringSoon || lc.Status == StatusPastDue)
		if paying {
			m.PaidTenants++
			m.MRR += monthly
			if monthly > 0 {
				priced++
			}
		}
		if newlyPaid[t.ID] && !free {
			m.NewPaid30d++
			m.NewMRR30d += monthly
		}

		// Trials that ended in the last 30 days, and which of them paid.
		if t.TrialEndsAt != nil && !t.TrialEndsAt.After(in.Now) && !t.TrialEndsAt.Before(since) {
			m.TrialsEnded30d++
			// A free extension or compensation is not a customer deciding to pay.
			if paidPeriod && !free {
				m.TrialsConverted30d++
			}
		}

		// Churn: a paid tenant that fell into read-only during the last 30 days
		// and has not come back.
		if paidPeriod && !free && (lc.Status == StatusReadOnly || lc.Status == StatusSuspended) {
			enteredReadOnly := t.SubscriptionExpiresAt.Add(time.Duration(in.Policy.GraceDays) * day)
			if !enteredReadOnly.After(in.Now) && !enteredReadOnly.Before(since) {
				m.Churned30d++
				m.ChurnedMRR += monthly
			}
		}

		// Renewals owed by tenants past their paid period.
		if paidPeriod && !free && (lc.Status == StatusPastDue || lc.Status == StatusReadOnly || lc.Status == StatusSuspended) {
			m.OverdueValue += in.PlanPrice[planOf(t)]
			m.OverdueTenants = append(m.OverdueTenants, AttentionTenant{TenantID: t.ID, Name: t.BusinessName, Status: lc.Status, DaysLeft: lc.DaysLeft})
		}
		if lc.Status == StatusTrialing && lc.DaysLeft >= 0 && lc.DaysLeft <= 3 {
			m.TrialsEndingSoon = append(m.TrialsEndingSoon, AttentionTenant{TenantID: t.ID, Name: t.BusinessName, Status: lc.Status, DaysLeft: lc.DaysLeft})
		}
	}

	if priced > 0 {
		m.ARPA = m.MRR / int64(priced)
	}
	sort.Slice(m.TrialsEndingSoon, func(i, j int) bool { return m.TrialsEndingSoon[i].DaysLeft < m.TrialsEndingSoon[j].DaysLeft })
	sort.Slice(m.OverdueTenants, func(i, j int) bool { return m.OverdueTenants[i].DaysLeft < m.OverdueTenants[j].DaysLeft })
	if len(m.TrialsEndingSoon) > attentionListMax {
		m.TrialsEndingSoon = m.TrialsEndingSoon[:attentionListMax]
	}
	if len(m.OverdueTenants) > attentionListMax {
		m.OverdueTenants = m.OverdueTenants[:attentionListMax]
	}
	return m
}

func planOf(t Tenant) int64 {
	if t.PlanID == nil {
		return 0
	}
	return *t.PlanID
}
