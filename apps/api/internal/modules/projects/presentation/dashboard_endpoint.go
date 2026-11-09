package presentation

import (
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

type dashboardIssueResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	// VendorID is 0 and VendorName "" for a kendala about the event in general.
	VendorID   int64  `json:"vendorId"`
	VendorName string `json:"vendorName"`
	Title      string `json:"title"`
	Impact     string `json:"impact"`
	FoundDate  string `json:"foundDate"`
	Status     string `json:"status"`
}

type dashboardMilestoneResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	VendorID    int64  `json:"vendorId"`
	VendorName  string `json:"vendorName"`
	Name        string `json:"name"`
	TargetDate  string `json:"targetDate"`
}

type dashboardPaymentResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	VendorID    int64  `json:"vendorId"`
	VendorName  string `json:"vendorName"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	PaymentDate string `json:"paymentDate"`
}

// dashboardVenuePaymentResponse has no VendorID -- a venue payment isn't
// tied to any vendor. Kept as its own response type/field (see
// domain.DashboardVenuePaymentRow's doc comment) rather than reshaping
// dashboardPaymentResponse into a source-discriminated union.
type dashboardVenuePaymentResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	PaymentDate string `json:"paymentDate"`
}

type dashboardTermResponse struct {
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	Label       string `json:"label"`
	DueDate     string `json:"dueDate"`
	Remaining   int64  `json:"remaining"`
}

type dashboardTaskResponse struct {
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	Name        string `json:"name"`
	TargetDate  string `json:"targetDate"`
}

type stageCountsResponse struct {
	Draft       int `json:"draft"`
	Preparation int `json:"preparation"`
	Ready       int `json:"ready"`
}

type laggingProjectResponse struct {
	Project        projectResponse `json:"project"`
	OverallPercent int             `json:"overallPercent"`
}

type projectTrendPointResponse struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type revenueTrendPointResponse struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Total int64  `json:"total"`
}

type revenueSummaryResponse struct {
	Total         int64    `json:"total"`
	PreviousTotal int64    `json:"previousTotal"`
	DeltaPercent  *float64 `json:"deltaPercent"`
}

// cashFlowWeekResponse is one week of the 8-week cash flow: what is still
// expected to come in (client installments) and go out (vendor balances).
type cashFlowWeekResponse struct {
	WeekStart   string `json:"weekStart"` // the Monday, YYYY-MM-DD
	ExpectedIn  int64  `json:"expectedIn"`
	ExpectedOut int64  `json:"expectedOut"`
}

type vendorPaymentDueResponse struct {
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	VendorName  string `json:"vendorName"`
	DueDate     string `json:"dueDate"`
	Remaining   int64  `json:"remaining"`
}

type unconfirmedVendorResponse struct {
	ProjectID        int64  `json:"projectId"`
	ProjectName      string `json:"projectName"`
	VendorName       string `json:"vendorName"`
	EngagementStatus string `json:"engagementStatus"`
	EventDate        string `json:"eventDate"`
}

type dashboardResponse struct {
	TotalProjects           int                             `json:"totalProjects"`
	ActiveProjects          int                             `json:"activeProjects"`
	ActiveVendorCount       int                             `json:"activeVendorCount"`
	OpenIssues              []dashboardIssueResponse        `json:"openIssues"`
	OverdueVendorMilestones []dashboardMilestoneResponse    `json:"overdueVendorMilestones"`
	IncompletePayments      []dashboardPaymentResponse      `json:"incompletePayments"`
	IncompleteVenuePayments []dashboardVenuePaymentResponse `json:"incompleteVenuePayments"`
	NearDDayProjects        []projectResponse               `json:"nearDDayProjects"`
	LaggingProjects         []laggingProjectResponse        `json:"laggingProjects"`
	UpcomingProjects        []projectResponse               `json:"upcomingProjects"`
	OverdueTerms            []dashboardTermResponse         `json:"overdueTerms"`
	OverdueTermsTotal       int64                           `json:"overdueTermsTotal"`
	DueSoonTerms            []dashboardTermResponse         `json:"dueSoonTerms"`
	TasksDueThisWeek        []dashboardTaskResponse         `json:"tasksDueThisWeek"`
	StageCounts             stageCountsResponse             `json:"stageCounts"`
	RecentActivity          []activityResponse              `json:"recentActivity"`
	Revenue                 revenueSummaryResponse          `json:"revenue"`
	ProjectTrend            []projectTrendPointResponse     `json:"projectTrend"`
	RevenueTrend            []revenueTrendPointResponse     `json:"revenueTrend"`
	// CashFlow and VendorPaymentsDue are money: absent (the keys are not even
	// there) without dashboard.revenue.view. UnconfirmedVendors is always present.
	CashFlow           *[]cashFlowWeekResponse     `json:"cashFlow,omitempty"`
	VendorPaymentsDue  *[]vendorPaymentDueResponse `json:"vendorPaymentsDue,omitempty"`
	UnconfirmedVendors []unconfirmedVendorResponse `json:"unconfirmedVendors"`
}

// vendorNames/actorNames name the vendors and actors of the rows (ids that do not
// resolve show an empty name).
func toDashboardResponse(s domain.DashboardStats, vendorNames, actorNames map[int64]string) dashboardResponse {
	resp := dashboardResponse{
		TotalProjects:     s.TotalProjects,
		ActiveProjects:    s.ActiveProjects,
		ActiveVendorCount: s.ActiveVendorCount,
		// Every slice is pre-allocated (not left nil) so the JSON response
		// always has `[]`, never `null`, for an empty list — matches the
		// convention used everywhere else in this module (see listProjects
		// etc.) and avoids every frontend consumer needing a null-guard.
		OpenIssues:              make([]dashboardIssueResponse, 0, len(s.OpenIssues)),
		OverdueVendorMilestones: make([]dashboardMilestoneResponse, 0, len(s.OverdueVendorMilestones)),
		IncompletePayments:      make([]dashboardPaymentResponse, 0, len(s.IncompletePayments)),
		IncompleteVenuePayments: make([]dashboardVenuePaymentResponse, 0, len(s.IncompleteVenuePayments)),
		NearDDayProjects:        make([]projectResponse, 0, len(s.NearDDayProjects)),
		LaggingProjects:         make([]laggingProjectResponse, 0, len(s.LaggingProjects)),
		UpcomingProjects:        make([]projectResponse, 0, len(s.UpcomingProjects)),
		OverdueTerms:            make([]dashboardTermResponse, 0, len(s.OverdueTerms)),
		OverdueTermsTotal:       s.OverdueTermsTotal,
		DueSoonTerms:            make([]dashboardTermResponse, 0, len(s.DueSoonTerms)),
		TasksDueThisWeek:        make([]dashboardTaskResponse, 0, len(s.TasksDueThisWeek)),
		StageCounts:             stageCountsResponse{Draft: s.StageCounts.Draft, Preparation: s.StageCounts.Preparation, Ready: s.StageCounts.Ready},
		RecentActivity:          make([]activityResponse, 0, len(s.RecentActivity)),
		ProjectTrend:            make([]projectTrendPointResponse, 0, len(s.ProjectTrend)),
		RevenueTrend:            make([]revenueTrendPointResponse, 0, len(s.RevenueTrend)),
		UnconfirmedVendors:      make([]unconfirmedVendorResponse, 0, len(s.UnconfirmedVendors)),
	}
	for _, row := range s.OpenIssues {
		resp.OpenIssues = append(resp.OpenIssues, dashboardIssueResponse{
			ID: row.Issue.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, VendorID: row.VendorID, VendorName: vendorNames[row.VendorID],
			Title: row.Issue.Title, Impact: string(row.Issue.Impact), FoundDate: row.Issue.FoundDate.Format(dateLayout),
			Status: string(row.Issue.Status),
		})
	}
	for _, row := range s.OverdueVendorMilestones {
		resp.OverdueVendorMilestones = append(resp.OverdueVendorMilestones, dashboardMilestoneResponse{
			ID: row.Milestone.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, VendorID: row.VendorID, VendorName: vendorNames[row.VendorID],
			Name: row.Milestone.Name, TargetDate: row.Milestone.TargetDate.Format(dateLayout),
		})
	}
	for _, row := range s.IncompletePayments {
		resp.IncompletePayments = append(resp.IncompletePayments, dashboardPaymentResponse{
			ID: row.Payment.ID, ProjectID: row.Payment.ProjectID, ProjectName: row.ProjectName, VendorID: row.VendorID, VendorName: vendorNames[row.VendorID],
			Type: string(row.Payment.Type), Amount: row.Payment.Amount, PaymentDate: row.Payment.PaymentDate.Format(dateLayout),
		})
	}
	for _, row := range s.IncompleteVenuePayments {
		resp.IncompleteVenuePayments = append(resp.IncompleteVenuePayments, dashboardVenuePaymentResponse{
			ID: row.Payment.ID, ProjectID: row.Payment.ProjectID, ProjectName: row.ProjectName,
			Type: string(row.Payment.Type), Amount: row.Payment.Amount, PaymentDate: row.Payment.PaymentDate.Format(dateLayout),
		})
	}
	for _, row := range s.OverdueTerms {
		resp.OverdueTerms = append(resp.OverdueTerms, dashboardTermResponse{
			ProjectID: row.ProjectID, ProjectName: row.ProjectName, Label: row.Label, DueDate: row.DueDate.Format(dateLayout), Remaining: row.Remaining,
		})
	}
	for _, row := range s.DueSoonTerms {
		resp.DueSoonTerms = append(resp.DueSoonTerms, dashboardTermResponse{
			ProjectID: row.ProjectID, ProjectName: row.ProjectName, Label: row.Label, DueDate: row.DueDate.Format(dateLayout), Remaining: row.Remaining,
		})
	}
	for _, row := range s.TasksDueThisWeek {
		resp.TasksDueThisWeek = append(resp.TasksDueThisWeek, dashboardTaskResponse{
			ProjectID: row.ProjectID, ProjectName: row.ProjectName, Name: row.Name, TargetDate: row.TargetDate.Format(dateLayout),
		})
	}
	for _, p := range s.NearDDayProjects {
		resp.NearDDayProjects = append(resp.NearDDayProjects, toProjectResponse(p))
	}
	for _, row := range s.LaggingProjects {
		resp.LaggingProjects = append(resp.LaggingProjects, laggingProjectResponse{Project: toProjectResponse(row.Project), OverallPercent: row.OverallPercent})
	}
	for _, p := range s.UpcomingProjects {
		resp.UpcomingProjects = append(resp.UpcomingProjects, toProjectResponse(p))
	}
	for _, a := range s.RecentActivity {
		ar := toActivityResponse(a)
		ar.ActorName = actorNames[a.ActorStaffID]
		resp.RecentActivity = append(resp.RecentActivity, ar)
	}
	if s.MoneySections {
		weeks := make([]cashFlowWeekResponse, 0, len(s.CashFlow))
		for _, w := range s.CashFlow {
			weeks = append(weeks, cashFlowWeekResponse{WeekStart: w.WeekStart.Format(dateLayout), ExpectedIn: w.ExpectedIn, ExpectedOut: w.ExpectedOut})
		}
		due := make([]vendorPaymentDueResponse, 0, len(s.VendorPaymentsDue))
		for _, d := range s.VendorPaymentsDue {
			due = append(due, vendorPaymentDueResponse{
				ProjectID: d.ProjectID, ProjectName: d.ProjectName, VendorName: vendorNames[d.VendorID],
				DueDate: d.DueDate.Format(dateLayout), Remaining: d.Remaining,
			})
		}
		resp.CashFlow, resp.VendorPaymentsDue = &weeks, &due
	}
	for _, u := range s.UnconfirmedVendors {
		resp.UnconfirmedVendors = append(resp.UnconfirmedVendors, unconfirmedVendorResponse{
			ProjectID: u.ProjectID, ProjectName: u.ProjectName, VendorName: vendorNames[u.VendorID],
			EngagementStatus: string(u.EngagementStatus), EventDate: u.EventDate.Format(dateLayout),
		})
	}
	for _, pt := range s.ProjectTrend {
		resp.ProjectTrend = append(resp.ProjectTrend, projectTrendPointResponse{Key: pt.Key, Label: pt.Label, Count: pt.Count})
	}
	for _, pt := range s.RevenueTrend {
		resp.RevenueTrend = append(resp.RevenueTrend, revenueTrendPointResponse{Key: pt.Key, Label: pt.Label, Total: pt.Total})
	}
	resp.Revenue = revenueSummaryResponse{Total: s.Revenue.Total, PreviousTotal: s.Revenue.PreviousTotal, DeltaPercent: s.Revenue.DeltaPercent}
	return resp
}

// Dashboard is Owner/Admin only (confirmed role rule) — it aggregates stats
// across every project/vendor/issue in the whole tenant, which neither a
// Wedding Planner nor Sales (both scoped to only their own PIC'd projects
// everywhere else in this module) must ever see in bulk, even though
// requireStaff alone would let any staff role through.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return
	}
	// The dashboard aggregates every project, so it needs dashboard.view
	// and a role that reaches every project.
	if !claims.can("dashboard.view") || claims.assignedOnly() {
		forbiddenPermission(w, "Peran Anda tidak dapat membuka dasbor")
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}
	canRevenue := claims.can("dashboard.revenue.view")
	stats, err := h.dashboard.Get(r.Context(), claims.tenantID, nowWIB(), application.DashboardOptions{IncludeMoney: canRevenue})
	if err != nil {
		writeAppError(w, err)
		return
	}
	// One lookup each for the vendors and the actors named in the rows.
	var vendorIDs, actorIDs []int64
	for _, r := range stats.OpenIssues {
		vendorIDs = append(vendorIDs, r.VendorID)
	}
	for _, r := range stats.OverdueVendorMilestones {
		vendorIDs = append(vendorIDs, r.VendorID)
	}
	for _, r := range stats.IncompletePayments {
		vendorIDs = append(vendorIDs, r.VendorID)
	}
	for _, r := range stats.VendorPaymentsDue {
		vendorIDs = append(vendorIDs, r.VendorID)
	}
	for _, r := range stats.UnconfirmedVendors {
		vendorIDs = append(vendorIDs, r.VendorID)
	}
	for _, a := range stats.RecentActivity {
		actorIDs = append(actorIDs, a.ActorStaffID)
	}
	resp := toDashboardResponse(*stats,
		h.dashboard.VendorNames(r.Context(), claims.tenantID, vendorIDs), h.dashboard.StaffNames(r.Context(), claims.tenantID, actorIDs))
	if !canRevenue {
		resp.Revenue = revenueSummaryResponse{}
		resp.RevenueTrend = []revenueTrendPointResponse{}
	}
	response.OK(w, "ok", resp)
}

type myProjectTaskResponse struct {
	Name       string `json:"name"`
	TargetDate string `json:"targetDate"`
}

type myProjectResponse struct {
	ID                int64                   `json:"id"`
	Name              string                  `json:"name"`
	BrideName         string                  `json:"brideName"`
	GroomName         string                  `json:"groomName"`
	EventDate         string                  `json:"eventDate"`
	Status            string                  `json:"status"`
	ProgressPercent   int                     `json:"progressPercent"`
	OpenIssues        int                     `json:"openIssues"`
	OverdueTerms      int                     `json:"overdueTerms"`
	OverdueTotal      int64                   `json:"overdueTotal"`
	NextTermLabel     string                  `json:"nextTermLabel,omitempty"`
	NextTermDueDate   string                  `json:"nextTermDueDate,omitempty"`
	NextTermRemaining int64                   `json:"nextTermRemaining,omitempty"`
	TasksDueThisWeek  []myProjectTaskResponse `json:"tasksDueThisWeek"`
}

// MyDashboard is GET /dashboard/mine: the projects the signed-in staff member
// holds, with what needs attention. Open to every staff role (it is scoped to
// their own projects), unlike the tenant-wide Dashboard.
func (h *Handler) MyDashboard(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}
	list, err := h.myProjects.Get(r.Context(), claims.tenantID, claims.staffID, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	out := make([]myProjectResponse, 0, len(list))
	for _, mp := range list {
		item := myProjectResponse{
			ID: mp.Project.ID, Name: mp.Project.Name, BrideName: mp.Project.BrideName, GroomName: mp.Project.GroomName,
			EventDate: mp.Project.EventDate.Format(dateLayout), Status: string(mp.Project.Status),
			ProgressPercent: mp.ProgressPercent, OpenIssues: mp.OpenIssues, OverdueTerms: mp.OverdueTerms, OverdueTotal: mp.OverdueTotal,
			TasksDueThisWeek: make([]myProjectTaskResponse, 0, len(mp.TasksDueThisWeek)),
		}
		if mp.NextTerm != nil {
			item.NextTermLabel = mp.NextTerm.Term.Label
			item.NextTermDueDate = mp.NextTerm.Term.DueDate.Format(dateLayout)
			item.NextTermRemaining = mp.NextTerm.Remaining
		}
		for _, m := range mp.TasksDueThisWeek {
			item.TasksDueThisWeek = append(item.TasksDueThisWeek, myProjectTaskResponse{Name: m.Name, TargetDate: m.TargetDate.Format(dateLayout)})
		}
		out = append(out, item)
	}
	response.OK(w, "ok", out)
}
