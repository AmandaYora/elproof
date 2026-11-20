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
	// ListPaymentCandidates is the WO's live payments to suppliers (the client's
	// own payments to a supplier are theirs to keep proof of).
	ListPaymentCandidates(ctx context.Context, tenantID int64) ([]domain.DashboardPaymentRow, error)
	ListRecentActivity(ctx context.Context, tenantID int64, limit int) ([]domain.ActivityLogEntry, error)
	ListOpenProjectTerms(ctx context.Context, tenantID int64) ([]domain.TermWithProject, error)
	ListAppliedToContract(ctx context.Context, tenantID int64) (map[int64]int64, error)
	ListProjectMilestonesDue(ctx context.Context, tenantID int64, from, to time.Time) ([]domain.DashboardTaskRow, error)
	// ListOpenSupplierBalances / ListUnconfirmedSuppliers back the cash flow,
	// "pembayaran vendor & venue jatuh tempo" and "belum terkonfirmasi" sections.
	// Balances are of the active suppliers the WO pays with an agreed cost and a
	// due date.
	ListOpenSupplierBalances(ctx context.Context, tenantID int64) ([]domain.SupplierBalanceRow, error)
	ListUnconfirmedSuppliers(ctx context.Context, tenantID int64, from, to time.Time, limit int) ([]domain.UnconfirmedSupplierRow, error)
	// ListClientBillTerms / ListClientSupplierPayments / ListPendingSupplierProofs
	// back the client-paid vendors' attention items (Paket WO Tahap 4): their
	// bills due, and the client's proofs waiting for the WO's check.
	ListClientBillTerms(ctx context.Context, tenantID int64) ([]domain.ClientBillTermRow, error)
	ListClientSupplierPayments(ctx context.Context, tenantID int64) ([]domain.SupplierPayment, error)
	ListPendingSupplierProofs(ctx context.Context, tenantID int64, limit int) ([]domain.SupplierProofPending, int, error)
}

type DashboardService struct {
	projects *ProjectService
	repo     DashboardRepository
	evidence *EvidenceService
}

// DashboardOptions chooses the optional parts of the dashboard.
type DashboardOptions struct {
	// IncludeMoney computes cashFlow and supplierPaymentsDue (the caller has
	// dashboard.revenue.view); without it they are not even read.
	IncludeMoney bool
	// IncludeClientBills computes the client-paid vendors' bills due and the
	// client's proofs waiting for a check (the caller has finance.record, PLAN
	// §4.3). Client money, so it does not need finance.margin.view (§9.15).
	IncludeClientBills bool
}

func NewDashboardService(projects *ProjectService, repo DashboardRepository, evidence *EvidenceService) *DashboardService {
	return &DashboardService{projects: projects, repo: repo, evidence: evidence}
}

const trendMonthsBack = 12

func (s *DashboardService) Get(ctx context.Context, tenantID int64, asOf time.Time, opts DashboardOptions) (*domain.DashboardStats, error) {
	totalProjects, err := s.projects.CountAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Bounded to the trend window (see ListForDashboard's doc comment)
	// instead of List's entire unbounded tenant history — PLAN.md
	// "Performance remediation". trendMonthsBack-1 months back from the
	// current month's start is the earliest point buildProjectTrend/
	// buildRevenueTrend can ever plot.
	trendWindowStart := time.Date(asOf.Year(), asOf.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(trendMonthsBack - 1), 0)
	allProjects, err := s.projects.ListForDashboard(ctx, tenantID, trendWindowStart)
	if err != nil {
		return nil, err
	}

	stats := &domain.DashboardStats{TotalProjects: int(totalProjects)}

	var upcoming []domain.Project
	var openProjectIDs []int64
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
			openProjectIDs = append(openProjectIDs, p.ID)
			if !p.IsArchived {
				switch p.Status {
				case domain.StatusDraft:
					stats.StageCounts.Draft++
				case domain.StatusPreparation:
					stats.StageCounts.Preparation++
				case domain.StatusReady:
					stats.StageCounts.Ready++
				}
			}
		}
	}
	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].EventDate.Before(upcoming[j].EventDate) })
	if len(upcoming) > 5 {
		upcoming = upcoming[:5]
	}
	stats.UpcomingProjects = upcoming

	// One batched progress computation over every open project instead of a
	// ComputeProgress call per project inside the loop above — PLAN.md
	// "Performance remediation".
	progressByID, err := s.projects.ComputeProgressBatch(ctx, tenantID, openProjectIDs, asOf)
	if err != nil {
		return nil, err
	}
	for _, p := range allProjects {
		isOpenProject := p.Status != domain.StatusCompleted && p.Status != domain.StatusCancelled
		if !isOpenProject {
			continue
		}
		progress := progressByID[p.ID]
		if domain.IsLagging(p.PrepStartDate, p.EventDate, asOf, progress.OverallPercent) {
			stats.LaggingProjects = append(stats.LaggingProjects, domain.LaggingProjectRow{Project: p, OverallPercent: progress.OverallPercent})
		}
	}

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
	hasInvoice, hasProof, err := s.evidenceLookup(ctx, paymentProjectIDs(paymentCandidates), domain.RelatedSupplierPayment)
	if err != nil {
		return nil, err
	}
	for _, row := range paymentCandidates {
		if !domain.IsPaymentEvidenceComplete(row.Payment.Type, row.Payment.ID, hasInvoice, hasProof) {
			stats.IncompletePayments = append(stats.IncompletePayments, row)
		}
	}

	terms, err := s.repo.ListOpenProjectTerms(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	applied, err := s.repo.ListAppliedToContract(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.OverdueTerms, stats.OverdueTermsTotal, stats.DueSoonTerms = domain.BuildTermAlerts(terms, applied, asOf)

	dueTasks, err := s.repo.ListProjectMilestonesDue(ctx, tenantID, asOf, asOf.AddDate(0, 0, 7))
	if err != nil {
		return nil, err
	}
	stats.TasksDueThisWeek = dueTasks

	recentActivity, err := s.repo.ListRecentActivity(ctx, tenantID, 8)
	if err != nil {
		return nil, err
	}
	stats.RecentActivity = recentActivity

	// Three sections added for the new Beranda (ADR-0034).
	if err := s.addTodaySections(ctx, tenantID, asOf, opts, stats, terms, applied); err != nil {
		return nil, err
	}

	stats.ProjectTrend = buildProjectTrend(allProjects, asOf, trendMonthsBack)
	stats.RevenueTrend = buildRevenueTrend(allProjects, asOf, trendMonthsBack)
	stats.Revenue = buildRevenueSummary(stats.RevenueTrend)

	return stats, nil
}

// evidenceLookup fetches evidence per distinct project id (not per payment)
// -- a small, bounded number of calls for what's meant to stay a curated
// dashboard widget, not an exhaustive report -- and cross-references it
// against a single relatedKind (see PaymentEvidenceStatus's own doc comment).
func (s *DashboardService) evidenceLookup(ctx context.Context, projectIDs []int64, kind domain.EvidenceRelatedKind) (hasInvoice, hasProof map[int64]bool, err error) {
	hasInvoice = make(map[int64]bool)
	hasProof = make(map[int64]bool)
	seen := make(map[int64]bool)
	for _, projectID := range projectIDs {
		if seen[projectID] {
			continue
		}
		seen[projectID] = true
		evidences, err := s.evidence.List(ctx, projectID)
		if err != nil {
			return nil, nil, err
		}
		inv, prf := domain.PaymentEvidenceStatus(evidences, kind)
		for id, v := range inv {
			hasInvoice[id] = v
		}
		for id, v := range prf {
			hasProof[id] = v
		}
	}
	return hasInvoice, hasProof, nil
}

func paymentProjectIDs(rows []domain.DashboardPaymentRow) []int64 {
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.Payment.ProjectID
	}
	return ids
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
			points[i].Total += p.ContractTotal
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

// addTodaySections fills the cash flow (next 8 weeks), the vendor & venue
// payments that are due and the suppliers still unconfirmed close to the event. The money
// sections are skipped unless the caller may see revenue figures.
func (s *DashboardService) addTodaySections(ctx context.Context, tenantID int64, asOf time.Time, opts DashboardOptions, stats *domain.DashboardStats, terms []domain.TermWithProject, applied map[int64]int64) error {
	if opts.IncludeMoney {
		balances, err := s.repo.ListOpenSupplierBalances(ctx, tenantID)
		if err != nil {
			return err
		}
		in := incomingDue(terms, applied, asOf)
		var out []domain.DueAmount
		var due []domain.SupplierDue
		for _, b := range balances {
			out = append(out, domain.DueAmount{DueDate: b.DueDate, Remaining: b.Remaining()})
			due = append(due, domain.SupplierDue{
				ProjectSupplierID: b.ProjectSupplierID, ProjectID: b.ProjectID, ProjectName: b.ProjectName,
				SupplierKind: b.SupplierKind, SupplierName: b.SupplierName, DueDate: b.DueDate, Remaining: b.Remaining(),
			})
		}
		stats.CashFlow = domain.BuildCashFlow(asOf, in, out)
		stats.SupplierPaymentsDue = domain.SelectSupplierPaymentsDue(asOf, due)
		stats.MoneySections = true
	}
	unconfirmed, err := s.repo.ListUnconfirmedSuppliers(ctx, tenantID,
		time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC),
		time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 30), domain.MaxDashboardRows)
	if err != nil {
		return err
	}
	stats.UnconfirmedSuppliers = unconfirmed
	if opts.IncludeClientBills {
		bills, err := s.repo.ListClientBillTerms(ctx, tenantID)
		if err != nil {
			return err
		}
		if len(bills) > 0 {
			payments, err := s.repo.ListClientSupplierPayments(ctx, tenantID)
			if err != nil {
				return err
			}
			stats.ClientBillsDue = domain.SelectClientBillsDue(asOf, bills, payments)
		}
		proofs, count, err := s.repo.ListPendingSupplierProofs(ctx, tenantID, domain.MaxDashboardRows)
		if err != nil {
			return err
		}
		stats.SupplierProofsPending, stats.SupplierProofsPendingCount = proofs, count
		stats.ClientBillSections = true
	}
	return nil
}

// incomingDue turns the open projects' payment schedules into what the clients
// still owe, by due date: each project's real payments are allocated over its
// schedule and every installment with something left counts.
func incomingDue(rows []domain.TermWithProject, applied map[int64]int64, today time.Time) []domain.DueAmount {
	byProject := make(map[int64][]domain.PaymentTerm)
	var order []int64
	for _, r := range rows {
		id := r.Term.ProjectID
		if _, seen := byProject[id]; !seen {
			order = append(order, id)
		}
		byProject[id] = append(byProject[id], r.Term)
	}
	var out []domain.DueAmount
	for _, id := range order {
		for _, tp := range domain.AllocateTerms(byProject[id], applied[id], today) {
			if tp.Remaining > 0 {
				out = append(out, domain.DueAmount{DueDate: tp.Term.DueDate, Remaining: tp.Remaining})
			}
		}
	}
	return out
}

// StaffNames names actors (best effort, one lookup).
func (s *DashboardService) StaffNames(ctx context.Context, tenantID int64, ids []int64) map[int64]string {
	return s.projects.StaffNames(ctx, tenantID, ids)
}
