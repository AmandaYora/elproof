package presentation

import (
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/response"
)

type overviewFinanceResponse struct {
	ContractValue int64                 `json:"contractValue"`
	Paid          int64                 `json:"paid"`
	Remaining     int64                 `json:"remaining"`
	OverdueTotal  int64                 `json:"overdueTotal"`
	NextTerm      *homeNextTermResponse `json:"nextTerm"`
	Terms         []paymentTermResponse `json:"terms"`
	VendorCost    int64                 `json:"vendorCost"`
	VenueCost     int64                 `json:"venueCost"`
	// Margin is contract - vendor cost - venue cost. The key is absent unless the
	// caller has finance.margin.view.
	Margin *int64 `json:"margin,omitempty"`
}

type overviewActionResponse struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Tone   string `json:"tone"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Tab    string `json:"tab"`
}

type overviewMilestoneResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	TargetDate string `json:"targetDate"`
}

type overviewVendorResponse struct {
	ID               int64  `json:"id"`
	VendorName       string `json:"vendorName"`
	EngagementStatus string `json:"engagementStatus"`
}

type overviewTeamResponse struct {
	PICStaffID      int64  `json:"picStaffId"`
	PICName         string `json:"picName"`
	PICSalesStaffID int64  `json:"picSalesStaffId"`
	PICSalesName    string `json:"picSalesName"`
}

type overviewClientResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	HasLogin bool   `json:"hasLogin"`
}

// overviewResponse is everything a project's header, side panel and Ringkasan
// tab show, in one answer.
type overviewResponse struct {
	// Project is exactly what GET /projects/{id} answers (progress and picName included).
	Project            projectResponse             `json:"project"`
	Finance            overviewFinanceResponse     `json:"finance"`
	Actions            []overviewActionResponse    `json:"actions"`
	NextMilestones     []overviewMilestoneResponse `json:"nextMilestones"`
	UnconfirmedVendors []overviewVendorResponse    `json:"unconfirmedVendors"`
	OpenIssueCount     int                         `json:"openIssueCount"`
	Team               overviewTeamResponse        `json:"team"`
	Clients            []overviewClientResponse    `json:"clients"`
}

// toOverviewResponse maps the overview; canMargin says whether the caller may
// see the margin (finance.margin.view).
func toOverviewResponse(o *application.ProjectOverview, canMargin bool) overviewResponse {
	project := toProjectResponse(o.Project)
	progress := toProgressResponse(o.Progress)
	project.Progress = &progress
	project.PICName = o.PICName

	fin := overviewFinanceResponse{
		ContractValue: o.Finance.ContractValue, Paid: o.Finance.Paid, Remaining: o.Finance.Remaining, OverdueTotal: o.Finance.OverdueTotal,
		Terms: toPaymentTermResponses(o.Terms), VendorCost: o.Finance.VendorCost, VenueCost: o.Finance.VenueCost,
	}
	if t := o.NextTerm; t != nil {
		fin.NextTerm = &homeNextTermResponse{
			ID: t.Term.ID, Label: t.Term.Label, Amount: t.Term.Amount, DueDate: t.Term.DueDate.Format(dateLayout),
			Status: string(t.Status), Remaining: t.Remaining,
		}
	}
	if canMargin {
		m := o.Finance.Margin
		fin.Margin = &m
	}
	out := overviewResponse{
		Project: project, Finance: fin, OpenIssueCount: o.OpenIssueCount,
		Actions:            make([]overviewActionResponse, 0, len(o.Actions)),
		NextMilestones:     make([]overviewMilestoneResponse, 0, len(o.NextMilestones)),
		UnconfirmedVendors: make([]overviewVendorResponse, 0, len(o.Unconfirmed)),
		Clients:            make([]overviewClientResponse, 0, len(o.Clients)),
		Team: overviewTeamResponse{
			PICStaffID: o.Team.PICStaffID, PICName: o.Team.PICName, PICSalesStaffID: o.Team.PICSalesStaffID, PICSalesName: o.Team.PICSalesName,
		},
	}
	for _, a := range o.Actions {
		out.Actions = append(out.Actions, overviewActionResponse{Kind: string(a.Kind), ID: a.ID, Tone: a.Tone, Title: a.Title, Detail: a.Detail, Tab: a.Tab})
	}
	for _, m := range o.NextMilestones {
		out.NextMilestones = append(out.NextMilestones, overviewMilestoneResponse{ID: m.ID, Name: m.Name, Status: string(m.Status), TargetDate: m.TargetDate.Format(dateLayout)})
	}
	for _, v := range o.Unconfirmed {
		out.UnconfirmedVendors = append(out.UnconfirmedVendors, overviewVendorResponse{ID: v.ID, VendorName: v.VendorName, EngagementStatus: string(v.EngagementStatus)})
	}
	for _, c := range o.Clients {
		out.Clients = append(out.Clients, overviewClientResponse{ID: c.ID, Name: c.Name, Role: c.Role, HasLogin: c.HasLogin})
	}
	return out
}

// getOverview is GET /projects/{id}/overview (staff only: a client principal never
// reaches it, the portal allowlist does not list it). Access follows
// GET /projects/{id}: resolveProjectAccess already confirmed the project is the
// caller's workspace's and, for a role limited to its own projects, theirs.
func (h *Handler) getOverview(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	o, err := h.overview.Get(r.Context(), claims.tenantID, projectID, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toOverviewResponse(o, claims.can("finance.margin.view")))
}
