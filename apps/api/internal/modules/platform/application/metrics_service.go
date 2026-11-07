package application

import (
	"context"
	"time"

	"elproof/internal/modules/platform/domain"
)

// PlanNames maps plan ids to their names for the given tenants; a plan that
// no longer resolves is simply absent.
func (s *TenantService) PlanNames(ctx context.Context, tenants []domain.Tenant) map[int64]string {
	names := map[int64]string{}
	for _, t := range tenants {
		if t.PlanID == nil {
			continue
		}
		if _, seen := names[*t.PlanID]; seen {
			continue
		}
		if plan, err := s.billing.GetPlan(ctx, *t.PlanID); err == nil && plan != nil {
			names[plan.ID] = plan.Name
		}
	}
	return names
}

// PlatformMetrics computes the SaaS numbers for the Platform Console's
// Ringkasan (FLOW-IDEAL 1.3) on the server, so the browser no longer downloads
// every tenant and transaction to add them up.
func (s *TenantService) PlatformMetrics(ctx context.Context, now time.Time) (domain.PlatformMetrics, error) {
	tenants, err := s.repo.List(ctx)
	if err != nil {
		return domain.PlatformMetrics{}, err
	}
	free, err := s.repo.FreeGrantTenantIDs(ctx)
	if err != nil {
		return domain.PlatformMetrics{}, err
	}

	monthly := map[int64]int64{}
	price := map[int64]int64{}
	for _, t := range tenants {
		if t.PlanID == nil {
			continue
		}
		if _, seen := price[*t.PlanID]; seen {
			continue
		}
		plan, err := s.billing.GetPlan(ctx, *t.PlanID)
		if err != nil || plan == nil {
			// A deleted plan must not break the whole summary; that tenant is
			// simply left out of the money figures.
			continue
		}
		price[plan.ID] = plan.Price
		months := plan.DurationMonths
		if months < 1 {
			months = 1
		}
		// Yearly plans count a twelfth per month, rounded to the rupiah.
		monthly[plan.ID] = (plan.Price + int64(months)/2) / int64(months)
	}

	// The whole paid ledger: the money windows only need 60 days, but "has
	// this tenant ever paid" needs all of it.
	paidTx, err := s.billing.PaidSince(ctx, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return domain.PlatformMetrics{}, err
	}
	recent := now.AddDate(0, 0, -60)
	paid := make([]domain.PaidPayment, 0, len(paidTx))
	for _, p := range paidTx {
		// A free grant stacked on top of a tenant that really paid is a bonus
		// period: that tenant is a customer, not a free account.
		delete(free, p.TenantID)
		if p.PaidAt.Before(recent) {
			continue
		}
		paid = append(paid, domain.PaidPayment{TenantID: p.TenantID, Amount: p.Amount, PaidAt: p.PaidAt, IsNew: string(p.Type) == "new"})
	}

	return domain.ComputePlatformMetrics(domain.MetricsInput{
		Now: now, Policy: domain.CurrentSubscriptionPolicy(), Tenants: tenants,
		PlanMonthly: monthly, PlanPrice: price, FreeTenants: free, Paid: paid,
	}), nil
}

// PlatformTrends is the data behind the dashboard's two charts, bucketed on
// the server so the console never needs every tenant and every transaction.
func (s *TenantService) PlatformTrends(ctx context.Context, period domain.TrendPeriod, now time.Time) (domain.Trends, error) {
	wib := time.FixedZone("WIB", 7*60*60)
	local := now.In(wib)
	yearStart := time.Date(local.Year(), 1, 1, 0, 0, 0, 0, wib)

	paidTx, err := s.billing.PaidSince(ctx, yearStart)
	if err != nil {
		return domain.Trends{}, err
	}
	paid := make([]domain.PaidPayment, 0, len(paidTx))
	for _, p := range paidTx {
		paid = append(paid, domain.PaidPayment{TenantID: p.TenantID, Amount: p.Amount, PaidAt: p.PaidAt})
	}
	joined, err := s.repo.JoinedSince(ctx, yearStart)
	if err != nil {
		return domain.Trends{}, err
	}
	return domain.ComputeTrends(period, now, paid, joined), nil
}

// maxTenantNames bounds one lookup; a table page never needs more.
const maxTenantNames = 100

// TenantNames resolves business names for the tenants shown on one page of a
// list (invoices, requests). Unknown ids are simply absent.
func (s *TenantService) TenantNames(ctx context.Context, ids []int64) (map[int64]string, error) {
	if len(ids) > maxTenantNames {
		ids = ids[:maxTenantNames]
	}
	return s.repo.NamesByIDs(ctx, ids)
}
