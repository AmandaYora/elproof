package presentation

import (
	"net/http"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

type dashboardIssueResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	VendorID    int64  `json:"vendorId"`
	Title       string `json:"title"`
	Impact      string `json:"impact"`
	FoundDate   string `json:"foundDate"`
	Status      string `json:"status"`
}

type dashboardMilestoneResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	VendorID    int64  `json:"vendorId"`
	Name        string `json:"name"`
	TargetDate  string `json:"targetDate"`
}

type dashboardPaymentResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	VendorID    int64  `json:"vendorId"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	PaymentDate string `json:"paymentDate"`
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

type dashboardResponse struct {
	TotalProjects           int                          `json:"totalProjects"`
	ActiveProjects          int                          `json:"activeProjects"`
	ActiveVendorCount       int                          `json:"activeVendorCount"`
	OpenIssues              []dashboardIssueResponse     `json:"openIssues"`
	OverdueVendorMilestones []dashboardMilestoneResponse `json:"overdueVendorMilestones"`
	IncompletePayments      []dashboardPaymentResponse   `json:"incompletePayments"`
	NearDDayProjects        []projectResponse            `json:"nearDDayProjects"`
	LaggingProjects         []laggingProjectResponse     `json:"laggingProjects"`
	UpcomingProjects        []projectResponse            `json:"upcomingProjects"`
	RecentActivity          []activityResponse           `json:"recentActivity"`
	Revenue                 revenueSummaryResponse       `json:"revenue"`
	ProjectTrend            []projectTrendPointResponse  `json:"projectTrend"`
	RevenueTrend            []revenueTrendPointResponse  `json:"revenueTrend"`
}

func toDashboardResponse(s domain.DashboardStats) dashboardResponse {
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
		NearDDayProjects:        make([]projectResponse, 0, len(s.NearDDayProjects)),
		LaggingProjects:         make([]laggingProjectResponse, 0, len(s.LaggingProjects)),
		UpcomingProjects:        make([]projectResponse, 0, len(s.UpcomingProjects)),
		RecentActivity:          make([]activityResponse, 0, len(s.RecentActivity)),
		ProjectTrend:            make([]projectTrendPointResponse, 0, len(s.ProjectTrend)),
		RevenueTrend:            make([]revenueTrendPointResponse, 0, len(s.RevenueTrend)),
	}
	for _, row := range s.OpenIssues {
		resp.OpenIssues = append(resp.OpenIssues, dashboardIssueResponse{
			ID: row.Issue.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, VendorID: row.VendorID,
			Title: row.Issue.Title, Impact: string(row.Issue.Impact), FoundDate: row.Issue.FoundDate.Format(dateLayout),
			Status: string(row.Issue.Status),
		})
	}
	for _, row := range s.OverdueVendorMilestones {
		resp.OverdueVendorMilestones = append(resp.OverdueVendorMilestones, dashboardMilestoneResponse{
			ID: row.Milestone.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, VendorID: row.VendorID,
			Name: row.Milestone.Name, TargetDate: row.Milestone.TargetDate.Format(dateLayout),
		})
	}
	for _, row := range s.IncompletePayments {
		resp.IncompletePayments = append(resp.IncompletePayments, dashboardPaymentResponse{
			ID: row.Payment.ID, ProjectID: row.Payment.ProjectID, ProjectName: row.ProjectName, VendorID: row.VendorID,
			Type: string(row.Payment.Type), Amount: row.Payment.Amount, PaymentDate: row.Payment.PaymentDate.Format(dateLayout),
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
		resp.RecentActivity = append(resp.RecentActivity, toActivityResponse(a))
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

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}
	stats, err := h.dashboard.Get(r.Context(), claims.tenantID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toDashboardResponse(*stats))
}
