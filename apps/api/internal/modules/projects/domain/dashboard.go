package domain

import "time"

// DashboardIssueRow/DashboardMilestoneRow/DashboardPaymentRow denormalize a
// tenant-wide join (issue/milestone/payment + its project + vendor) purely
// for the dashboard aggregation — nothing else in the module needs this
// shape, so it lives next to the aggregate it serves rather than polluting
// the entities themselves.
type DashboardIssueRow struct {
	Issue       VendorIssue
	ProjectID   int64
	ProjectName string
	VendorID    int64
}

type DashboardMilestoneRow struct {
	Milestone   VendorMilestone
	ProjectID   int64
	ProjectName string
	VendorID    int64
}

type DashboardPaymentRow struct {
	Payment     VendorPayment
	ProjectName string
	VendorID    int64
}

// DashboardVenuePaymentRow deliberately stays a separate slice/type from
// DashboardPaymentRow above rather than a merged, source-discriminated one --
// there's no VendorID-equivalent for a venue payment, and keeping the
// already-working vendor path untouched was judged safer than reshaping it
// just to accommodate a second source. The two lists are merged into one
// labeled feed at the presentation layer instead (dashboard/lib/attention.ts
// on the frontend). See PLAN.md "Venue Payments + Pembayaran tab
// restructuring".
type DashboardVenuePaymentRow struct {
	Payment     VenuePayment
	ProjectName string
}

type LaggingProjectRow struct {
	Project        Project
	OverallPercent int
}

type ProjectTrendPoint struct {
	Key   string
	Label string
	Count int
}

type RevenueTrendPoint struct {
	Key   string
	Label string
	Total int64
}

type RevenueSummary struct {
	Total         int64
	PreviousTotal int64
	// nil when PreviousTotal is 0 (division by zero — mirrors the frontend's
	// pre-integration mock, which shows a neutral message instead of a percent).
	DeltaPercent *float64
}

type DashboardStats struct {
	TotalProjects           int
	ActiveProjects          int
	ActiveVendorCount       int
	OpenIssues              []DashboardIssueRow
	OverdueVendorMilestones []DashboardMilestoneRow
	IncompletePayments      []DashboardPaymentRow
	IncompleteVenuePayments []DashboardVenuePaymentRow
	NearDDayProjects        []Project
	LaggingProjects         []LaggingProjectRow
	UpcomingProjects        []Project
	OverdueTerms            []DashboardTermRow
	OverdueTermsTotal       int64
	DueSoonTerms            []DashboardTermRow
	TasksDueThisWeek        []DashboardTaskRow
	StageCounts             ProjectStageCounts
	RecentActivity          []ActivityLogEntry
	Revenue                 RevenueSummary
	ProjectTrend            []ProjectTrendPoint
	RevenueTrend            []RevenueTrendPoint
}

const nearDDayThresholdDays = 30

func daysBetween(from, to time.Time) int {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(to.Sub(from).Hours() / 24)
}

// IsNearDDay mirrors mock/selectors.ts's NEAR_D_DAY_THRESHOLD check.
func IsNearDDay(eventDate, asOf time.Time) bool {
	d := daysBetween(asOf, eventDate)
	return d >= 0 && d <= nearDDayThresholdDays
}

// IsLagging mirrors the mock's getDashboardStats laggingProjects rule:
// elapsed-vs-total prep time outpacing actual milestone completion by >15pp.
// prepStartDate displays in the UI as "Tanggal Booking" (PLAN.md,
// display-text-only rename) -- the field and this math are unchanged.
func IsLagging(prepStartDate, eventDate, asOf time.Time, overallPercent int) bool {
	totalDays := daysBetween(prepStartDate, eventDate)
	elapsedDays := daysBetween(prepStartDate, asOf)
	var expectedRatio float64
	if totalDays <= 0 {
		expectedRatio = 1
	} else {
		expectedRatio = float64(elapsedDays) / float64(totalDays)
		if expectedRatio > 1 {
			expectedRatio = 1
		}
		if expectedRatio < 0 {
			expectedRatio = 0
		}
	}
	return float64(overallPercent)/100 < expectedRatio-0.15
}

// DashboardTermRow is one unpaid client installment that needs a look: it is
// overdue, or falls due within the next 30 days.
type DashboardTermRow struct {
	ProjectID   int64
	ProjectName string
	Label       string
	DueDate     time.Time
	Remaining   int64
}

// DashboardTaskRow is a project Timeline item due soon.
type DashboardTaskRow struct {
	ProjectID   int64
	ProjectName string
	Name        string
	TargetDate  time.Time
}

// TermWithProject is a stored installment together with its project's name,
// as the dashboard reads them across the whole workspace.
type TermWithProject struct {
	Term        PaymentTerm
	ProjectName string
}

// ProjectStageCounts is how many open projects sit at each stage.
type ProjectStageCounts struct {
	Draft       int
	Preparation int
	Ready       int
}

const termAlertHorizonDays = 30

// BuildTermAlerts allocates each project's real payments over its schedule
// (see AllocateTerms) and returns the installments still unpaid that are
// overdue (with their total) or due within the next 30 days. `applied` maps a
// project id to what its client has paid against the contract.
func BuildTermAlerts(rows []TermWithProject, applied map[int64]int64, today time.Time) (overdue []DashboardTermRow, overdueTotal int64, dueSoon []DashboardTermRow) {
	byProject := make(map[int64][]PaymentTerm)
	names := make(map[int64]string)
	var order []int64
	for _, r := range rows {
		id := r.Term.ProjectID
		if _, seen := names[id]; !seen {
			order = append(order, id)
		}
		names[id] = r.ProjectName
		byProject[id] = append(byProject[id], r.Term)
	}
	for _, id := range order {
		for _, tp := range AllocateTerms(byProject[id], applied[id], today) {
			if tp.Status == TermPaid {
				continue
			}
			row := DashboardTermRow{ProjectID: id, ProjectName: names[id], Label: tp.Term.Label, DueDate: tp.Term.DueDate, Remaining: tp.Remaining}
			if tp.Status == TermOverdue {
				overdue = append(overdue, row)
				overdueTotal += tp.Remaining
			} else if daysBetween(today, tp.Term.DueDate) <= termAlertHorizonDays {
				dueSoon = append(dueSoon, row)
			}
		}
	}
	return overdue, overdueTotal, dueSoon
}
