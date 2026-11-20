package presentation

import (
	"context"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

type dashboardIssueResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	// SupplierName is "" for a kendala about the event in general.
	SupplierName string `json:"supplierName"`
	Title        string `json:"title"`
	Impact       string `json:"impact"`
	FoundDate    string `json:"foundDate"`
	Status       string `json:"status"`
}

type dashboardMilestoneResponse struct {
	ID           int64  `json:"id"`
	ProjectID    int64  `json:"projectId"`
	ProjectName  string `json:"projectName"`
	SupplierName string `json:"supplierName"`
	Name         string `json:"name"`
	TargetDate   string `json:"targetDate"`
}

// dashboardPaymentResponse is a payment to a vendor or venue that still lacks
// its invoice or transfer proof.
type dashboardPaymentResponse struct {
	ID           int64  `json:"id"`
	ProjectID    int64  `json:"projectId"`
	ProjectName  string `json:"projectName"`
	SupplierKind string `json:"supplierKind"`
	SupplierName string `json:"supplierName"`
	Payer        string `json:"payer"`
	Type         string `json:"type"`
	Amount       int64  `json:"amount"`
	PaymentDate  string `json:"paymentDate"`
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

type supplierPaymentDueResponse struct {
	ProjectID    int64  `json:"projectId"`
	ProjectName  string `json:"projectName"`
	SupplierKind string `json:"supplierKind"`
	SupplierName string `json:"supplierName"`
	DueDate      string `json:"dueDate"`
	Remaining    int64  `json:"remaining"`
}

type unconfirmedSupplierResponse struct {
	ProjectID        int64  `json:"projectId"`
	ProjectName      string `json:"projectName"`
	SupplierKind     string `json:"supplierKind"`
	SupplierName     string `json:"supplierName"`
	EngagementStatus string `json:"engagementStatus"`
	EventDate        string `json:"eventDate"`
}

// clientBillDueResponse is one bill of a vendor or venue the client pays
// directly that is overdue or due within 7 days (Paket WO Tahap 4). The
// supplier is always named: the client pays it.
type clientBillDueResponse struct {
	BillTermID        int64  `json:"billTermId"`
	ProjectSupplierID int64  `json:"projectSupplierId"`
	ProjectID         int64  `json:"projectId"`
	ProjectName       string `json:"projectName"`
	SupplierKind      string `json:"supplierKind"`
	SupplierName      string `json:"supplierName"`
	Label             string `json:"label"`
	DueDate           string `json:"dueDate"`
	Remaining         int64  `json:"remaining"`
	// Pending is what the client sent for this bill that the WO has not checked.
	Pending int64  `json:"pending"`
	Status  string `json:"status"`
}

// supplierProofPendingResponse is one proof the client sent to a vendor it
// pays directly that waits for the WO's check.
type supplierProofPendingResponse struct {
	SupplierPaymentID int64  `json:"supplierPaymentId"`
	ProjectID         int64  `json:"projectId"`
	ProjectName       string `json:"projectName"`
	ProjectSupplierID int64  `json:"projectSupplierId"`
	SupplierKind      string `json:"supplierKind"`
	SupplierName      string `json:"supplierName"`
	BillTermID        *int64 `json:"billTermId"`
	BillTermLabel     string `json:"billTermLabel"`
	Type              string `json:"type"`
	Amount            int64  `json:"amount"`
	PaymentDate       string `json:"paymentDate"`
	SubmittedAt       string `json:"submittedAt"`
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
	OverdueTerms            []dashboardTermResponse      `json:"overdueTerms"`
	OverdueTermsTotal       int64                        `json:"overdueTermsTotal"`
	DueSoonTerms            []dashboardTermResponse      `json:"dueSoonTerms"`
	TasksDueThisWeek        []dashboardTaskResponse      `json:"tasksDueThisWeek"`
	StageCounts             stageCountsResponse          `json:"stageCounts"`
	RecentActivity          []activityResponse           `json:"recentActivity"`
	Revenue                 revenueSummaryResponse       `json:"revenue"`
	ProjectTrend            []projectTrendPointResponse  `json:"projectTrend"`
	RevenueTrend            []revenueTrendPointResponse  `json:"revenueTrend"`
	// CashFlow and SupplierPaymentsDue are money: absent (the keys are not even
	// there) without dashboard.revenue.view; without finance.margin.view the
	// cash flow has no money out and SupplierPaymentsDue is absent. UnconfirmedSuppliers is always present.
	CashFlow             *[]cashFlowWeekResponse       `json:"cashFlow,omitempty"`
	SupplierPaymentsDue  *[]supplierPaymentDueResponse `json:"supplierPaymentsDue,omitempty"`
	UnconfirmedSuppliers []unconfirmedSupplierResponse `json:"unconfirmedSuppliers"`
	// ClientBillsDue, SupplierProofsPending and SupplierProofsPendingCount are
	// the client-paid vendors' attention items: absent (the keys are not there)
	// without finance.record. The client's own money, so they stay without
	// finance.margin.view (PLAN §9.15).
	ClientBillsDue             *[]clientBillDueResponse        `json:"clientBillsDue,omitempty"`
	SupplierProofsPending      *[]supplierProofPendingResponse `json:"supplierProofsPending,omitempty"`
	SupplierProofsPendingCount *int                            `json:"supplierProofsPendingCount,omitempty"`
}

// actorNames names the actors of the activity rows (ids that do not resolve
// show an empty name).
func toDashboardResponse(s domain.DashboardStats, actorNames map[int64]string) dashboardResponse {
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
		OverdueTerms:            make([]dashboardTermResponse, 0, len(s.OverdueTerms)),
		OverdueTermsTotal:       s.OverdueTermsTotal,
		DueSoonTerms:            make([]dashboardTermResponse, 0, len(s.DueSoonTerms)),
		TasksDueThisWeek:        make([]dashboardTaskResponse, 0, len(s.TasksDueThisWeek)),
		StageCounts:             stageCountsResponse{Draft: s.StageCounts.Draft, Preparation: s.StageCounts.Preparation, Ready: s.StageCounts.Ready},
		RecentActivity:          make([]activityResponse, 0, len(s.RecentActivity)),
		ProjectTrend:            make([]projectTrendPointResponse, 0, len(s.ProjectTrend)),
		RevenueTrend:            make([]revenueTrendPointResponse, 0, len(s.RevenueTrend)),
		UnconfirmedSuppliers:    make([]unconfirmedSupplierResponse, 0, len(s.UnconfirmedSuppliers)),
	}
	for _, row := range s.OpenIssues {
		resp.OpenIssues = append(resp.OpenIssues, dashboardIssueResponse{
			ID: row.Issue.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, SupplierName: row.SupplierName,
			Title: row.Issue.Title, Impact: string(row.Issue.Impact), FoundDate: row.Issue.FoundDate.Format(dateLayout),
			Status: string(row.Issue.Status),
		})
	}
	for _, row := range s.OverdueVendorMilestones {
		resp.OverdueVendorMilestones = append(resp.OverdueVendorMilestones, dashboardMilestoneResponse{
			ID: row.Milestone.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, SupplierName: row.SupplierName,
			Name: row.Milestone.Name, TargetDate: row.Milestone.TargetDate.Format(dateLayout),
		})
	}
	for _, row := range s.IncompletePayments {
		resp.IncompletePayments = append(resp.IncompletePayments, dashboardPaymentResponse{
			ID: row.Payment.ID, ProjectID: row.Payment.ProjectID, ProjectName: row.ProjectName, SupplierKind: string(row.SupplierKind), SupplierName: row.SupplierName,
			Payer: string(row.Payment.Payer), Type: string(row.Payment.Type), Amount: row.Payment.Amount, PaymentDate: row.Payment.PaymentDate.Format(dateLayout),
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
		due := make([]supplierPaymentDueResponse, 0, len(s.SupplierPaymentsDue))
		for _, d := range s.SupplierPaymentsDue {
			due = append(due, supplierPaymentDueResponse{
				ProjectID: d.ProjectID, ProjectName: d.ProjectName, SupplierKind: string(d.SupplierKind), SupplierName: d.SupplierName,
				DueDate: d.DueDate.Format(dateLayout), Remaining: d.Remaining,
			})
		}
		resp.CashFlow, resp.SupplierPaymentsDue = &weeks, &due
	}
	for _, u := range s.UnconfirmedSuppliers {
		resp.UnconfirmedSuppliers = append(resp.UnconfirmedSuppliers, unconfirmedSupplierResponse{
			ProjectID: u.ProjectID, ProjectName: u.ProjectName, SupplierKind: string(u.SupplierKind), SupplierName: u.SupplierName,
			EngagementStatus: string(u.EngagementStatus), EventDate: u.EventDate.Format(dateLayout),
		})
	}
	if s.ClientBillSections {
		bills := make([]clientBillDueResponse, 0, len(s.ClientBillsDue))
		for _, b := range s.ClientBillsDue {
			bills = append(bills, clientBillDueResponse{
				BillTermID: b.BillTermID, ProjectSupplierID: b.ProjectSupplierID, ProjectID: b.ProjectID, ProjectName: b.ProjectName,
				SupplierKind: string(b.SupplierKind), SupplierName: b.SupplierName, Label: b.Label, DueDate: b.DueDate.Format(dateLayout),
				Remaining: b.Remaining, Pending: b.Pending, Status: string(b.Status),
			})
		}
		proofs := make([]supplierProofPendingResponse, 0, len(s.SupplierProofsPending))
		for _, p := range s.SupplierProofsPending {
			proofs = append(proofs, supplierProofPendingResponse{
				SupplierPaymentID: p.SupplierPaymentID, ProjectID: p.ProjectID, ProjectName: p.ProjectName, ProjectSupplierID: p.ProjectSupplierID,
				SupplierKind: string(p.SupplierKind), SupplierName: p.SupplierName, BillTermID: optionalID(p.BillTermID), BillTermLabel: p.BillTermLabel,
				Type: string(p.Type), Amount: p.Amount, PaymentDate: p.PaymentDate.Format(dateLayout), SubmittedAt: p.SubmittedAt.In(wib).Format(time.RFC3339),
			})
		}
		count := s.SupplierProofsPendingCount
		resp.ClientBillsDue, resp.SupplierProofsPending, resp.SupplierProofsPendingCount = &bills, &proofs, &count
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
	stats, err := h.dashboard.Get(r.Context(), claims.tenantID, nowWIB(), application.DashboardOptions{
		IncludeMoney: canRevenue, IncludeClientBills: claims.can("finance.record"),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	actorIDs := make([]int64, 0, len(stats.RecentActivity))
	for _, a := range stats.RecentActivity {
		actorIDs = append(actorIDs, a.ActorStaffID)
	}
	resp := toDashboardResponse(*stats, h.dashboard.StaffNames(r.Context(), claims.tenantID, actorIDs))
	h.describeDashboardProjects(r.Context(), claims.tenantID, &resp)
	if !claims.can("finance.margin.view") {
		hideSupplierCosts(&resp)
	}
	if !canRevenue {
		resp.Revenue = revenueSummaryResponse{}
		resp.RevenueTrend = []revenueTrendPointResponse{}
	}
	response.OK(w, "ok", resp)
}

// describeDashboardProjects fills each listed project's location and events
// line ("2 acara: Akad 13 Mar · Resepsi 14 Mar"), as the project list does.
func (h *Handler) describeDashboardProjects(ctx context.Context, tenantID int64, resp *dashboardResponse) {
	list := make([]*projectResponse, 0, len(resp.NearDDayProjects)+len(resp.LaggingProjects)+len(resp.UpcomingProjects))
	for i := range resp.NearDDayProjects {
		list = append(list, &resp.NearDDayProjects[i])
	}
	for i := range resp.LaggingProjects {
		list = append(list, &resp.LaggingProjects[i].Project)
	}
	for i := range resp.UpcomingProjects {
		list = append(list, &resp.UpcomingProjects[i])
	}
	h.describeProjects(ctx, tenantID, list)
}

// hideSupplierCosts removes what the WO pays its vendors and venues from a
// dashboard read without finance.margin.view: the money out of the cash flow,
// the supplier payments due, and WO-paid payments still missing their proof.
func hideSupplierCosts(resp *dashboardResponse) {
	if resp.CashFlow != nil {
		for i := range *resp.CashFlow {
			(*resp.CashFlow)[i].ExpectedOut = 0
		}
	}
	resp.SupplierPaymentsDue = nil
	kept := resp.IncompletePayments[:0]
	for _, p := range resp.IncompletePayments {
		if p.Payer != string(domain.PayerWO) {
			kept = append(kept, p)
		}
	}
	resp.IncompletePayments = kept
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
