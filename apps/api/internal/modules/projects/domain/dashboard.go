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
	NearDDayProjects        []Project
	LaggingProjects         []LaggingProjectRow
	UpcomingProjects        []Project
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
