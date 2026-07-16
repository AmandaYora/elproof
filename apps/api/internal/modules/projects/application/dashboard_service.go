package application

import (
	"context"
	"sort"
	"time"

	"elproof/internal/modules/projects/domain"
)

// DashboardRepository holds the tenant-wide (cross-project) queries the
// dashboard needs — these join within the module's own tables (projects +
// its sub-entities), which is allowed; nothing here reaches into another
// module.
type DashboardRepository interface {
	CountActiveVendors(ctx context.Context, tenantID int64) (int, error)
	ListOpenIssues(ctx context.Context, tenantID int64) ([]domain.DashboardIssueRow, error)
	ListOverdueVendorMilestones(ctx context.Context, tenantID int64, asOf time.Time) ([]domain.DashboardMilestoneRow, error)
	ListPaymentCandidates(ctx context.Context, tenantID int64) ([]domain.DashboardPaymentRow, error)
	ListRecentActivity(ctx context.Context, tenantID int64, limit int) ([]domain.ActivityLogEntry, error)
}

type DashboardService struct {
	projects *ProjectService
	repo     DashboardRepository
}

func NewDashboardService(projects *ProjectService, repo DashboardRepository) *DashboardService {
	return &DashboardService{projects: projects, repo: repo}
}

const trendMonthsBack = 12

func (s *DashboardService) Get(ctx context.Context, tenantID int64, asOf time.Time) (*domain.DashboardStats, error) {
	allProjects, err := s.projects.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	stats := &domain.DashboardStats{TotalProjects: len(allProjects)}

	var upcoming []domain.Project
	for _, p := range allProjects {
		isOpenProject := p.Status != domain.StatusCompleted && p.Status != domain.StatusCancelled
		if p.Status == domain.StatusPreparation || p.Status == domain.StatusReady {
			stats.ActiveProjects++
		}
		if isOpenProject && domain.IsNearDDay(p.EventDate, asOf) {
			stats.NearDDayProjects = append(stats.NearDDayProjects, p)
		}
		if isOpenProject && daysBetweenPublic(asOf, p.EventDate) >= 0 {
			upcoming = append(upcoming, p)
		}
		if isOpenProject {
			progress, err := s.projects.ComputeProgress(ctx, tenantID, p.ID, asOf)
			if err != nil {
				return nil, err
			}
			if domain.IsLagging(p.PrepStartDate, p.EventDate, asOf, progress.OverallPercent) {
				stats.LaggingProjects = append(stats.LaggingProjects, domain.LaggingProjectRow{Project: p, OverallPercent: progress.OverallPercent})
			}
		}
	}
	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].EventDate.Before(upcoming[j].EventDate) })
	if len(upcoming) > 5 {
		upcoming = upcoming[:5]
	}
	stats.UpcomingProjects = upcoming

	activeVendorCount, err := s.repo.CountActiveVendors(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.ActiveVendorCount = activeVendorCount

	openIssues, err := s.repo.ListOpenIssues(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.OpenIssues = openIssues

	overdue, err := s.repo.ListOverdueVendorMilestones(ctx, tenantID, asOf)
	if err != nil {
		return nil, err
	}
	stats.OverdueVendorMilestones = overdue

	paymentCandidates, err := s.repo.ListPaymentCandidates(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, row := range paymentCandidates {
		if !row.Payment.IsEvidenceComplete() {
			stats.IncompletePayments = append(stats.IncompletePayments, row)
		}
	}

	recentActivity, err := s.repo.ListRecentActivity(ctx, tenantID, 8)
	if err != nil {
		return nil, err
	}
	stats.RecentActivity = recentActivity

	stats.ProjectTrend = buildProjectTrend(allProjects, asOf, trendMonthsBack)
	stats.RevenueTrend = buildRevenueTrend(allProjects, asOf, trendMonthsBack)
	stats.Revenue = buildRevenueSummary(stats.RevenueTrend)

	return stats, nil
}

var monthsID = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

func monthKeyLabel(year int, month time.Month) (string, string) {
	key := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	label := monthsID[month-1] + " " + time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("06")
	return key, label
}

func buildProjectTrend(projects []domain.Project, asOf time.Time, monthsBack int) []domain.ProjectTrendPoint {
	points := make([]domain.ProjectTrendPoint, monthsBack)
	index := make(map[string]int, monthsBack)
	cursor := time.Date(asOf.Year(), asOf.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := monthsBack - 1; i >= 0; i-- {
		m := cursor.AddDate(0, -i, 0)
		key, label := monthKeyLabel(m.Year(), m.Month())
		pointIndex := monthsBack - 1 - i
		points[pointIndex] = domain.ProjectTrendPoint{Key: key, Label: label}
		index[key] = pointIndex
	}
	for _, p := range projects {
		key := p.CreatedAt.Format("2006-01")
		if i, ok := index[key]; ok {
			points[i].Count++
		}
	}
	return points
}

func buildRevenueTrend(projects []domain.Project, asOf time.Time, monthsBack int) []domain.RevenueTrendPoint {
	points := make([]domain.RevenueTrendPoint, monthsBack)
	index := make(map[string]int, monthsBack)
	cursor := time.Date(asOf.Year(), asOf.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := monthsBack - 1; i >= 0; i-- {
		m := cursor.AddDate(0, -i, 0)
		key, label := monthKeyLabel(m.Year(), m.Month())
		pointIndex := monthsBack - 1 - i
		points[pointIndex] = domain.RevenueTrendPoint{Key: key, Label: label}
		index[key] = pointIndex
	}
	for _, p := range projects {
		if p.Status == domain.StatusCancelled {
			continue
		}
		key := p.EventDate.Format("2006-01")
		if i, ok := index[key]; ok {
			points[i].Total += p.ContractValue
		}
	}
	return points
}

func buildRevenueSummary(revenueTrend []domain.RevenueTrendPoint) domain.RevenueSummary {
	if len(revenueTrend) < 2 {
		return domain.RevenueSummary{}
	}
	previous := revenueTrend[len(revenueTrend)-2]
	current := revenueTrend[len(revenueTrend)-1]
	summary := domain.RevenueSummary{Total: current.Total, PreviousTotal: previous.Total}
	if previous.Total != 0 {
		delta := (float64(current.Total-previous.Total) / float64(previous.Total)) * 100
		summary.DeltaPercent = &delta
	}
	return summary
}

// daysBetweenPublic avoids exporting the domain package's private helper —
// this file needs the same day-diff, so it's re-derived from stdlib directly.
func daysBetweenPublic(from, to time.Time) int {
	f := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	t := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(t.Sub(f).Hours() / 24)
}
