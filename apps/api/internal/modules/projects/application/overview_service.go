package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/logger"
)

// ProjectClientInfo is one client of a project as the overview lists them.
// Primitive-typed so `projects` never imports `clients`; main.go adapts the
// clients contract to ClientDirectory (the same bridge idiom as ClientCleaner).
type ProjectClientInfo struct {
	ID   int64
	Name string
	Role string
	// HasLogin: the client set a password and can sign in (an invitation not yet
	// accepted does not count).
	HasLogin bool
}

// ClientDirectory is what the overview needs from `clients`.
type ClientDirectory interface {
	ListProjectClients(ctx context.Context, tenantID, projectID int64) ([]ProjectClientInfo, error)
}

// OverviewService assembles GET /projects/{id}/overview: the header, side panel
// and Ringkasan tab of a project in one pass.
type OverviewService struct {
	projects       *ProjectService
	clientPayments ClientPaymentRepository
	terms          *PaymentTermService
	refs           *References
	clients        ClientDirectory
}

func NewOverviewService(projects *ProjectService, clientPayments ClientPaymentRepository, terms *PaymentTermService, refs *References) *OverviewService {
	return &OverviewService{projects: projects, clientPayments: clientPayments, terms: terms, refs: refs}
}

// SetClients completes the two-phase wiring with `clients`; without it the
// overview lists no clients.
func (s *OverviewService) SetClients(c ClientDirectory) { s.clients = c }

// UnconfirmedVendor is an engagement still Planned or in Negotiation.
type UnconfirmedVendor struct {
	ID               int64
	VendorName       string
	EngagementStatus domain.EngagementStatus
}

// OverviewTeam names the project's two PIC slots ("" when unresolved).
type OverviewTeam struct {
	PICStaffID      int64
	PICName         string
	PICSalesStaffID int64
	PICSalesName    string
}

// ProjectOverview is the whole answer.
type ProjectOverview struct {
	Project        domain.Project
	Progress       domain.ProjectProgress
	PICName        string
	Finance        domain.FinanceOverview
	NextTerm       *domain.TermProgress
	Terms          []domain.TermProgress
	Actions        []domain.ProjectAction
	NextMilestones []domain.ProjectMilestone
	Unconfirmed    []UnconfirmedVendor
	OpenIssueCount int
	Team           OverviewTeam
	Clients        []ProjectClientInfo
}

// Get builds the overview of a project of the tenant. `now` carries the Jakarta
// zone ("today" is the Indonesian calendar day, ADR-0007). Every source is read
// once, whatever the number of vendors or terms: the project, seven progress
// sources (milestones, engagements, vendor milestones, kendala, vendor payments,
// venue payments, evidence), the client's payments, the schedule, the vendors'
// names, the two PICs' names and the clients (plus one login lookup per client,
// a project has at most a handful).
func (s *OverviewService) Get(ctx context.Context, tenantID, projectID int64, now time.Time) (*ProjectOverview, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	data, err := s.projects.loadProgressData(ctx, []int64{projectID})
	if err != nil {
		return nil, err
	}
	payments, err := s.clientPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	terms, err := s.terms.progressWith(ctx, projectID, payments, now)
	if err != nil {
		return nil, err
	}

	engagements, issues, milestones := data.engagements[projectID], data.issues[projectID], data.milestones[projectID]
	vendorIDs := make([]int64, 0, len(engagements))
	for _, e := range engagements {
		vendorIDs = append(vendorIDs, e.VendorID)
	}
	vendorNames := s.refs.VendorNames(ctx, tenantID, vendorIDs)
	nameOf := make(map[int64]string, len(engagements))
	for _, e := range engagements {
		nameOf[e.ID] = vendorNames[e.VendorID]
	}
	staff := s.projects.StaffNames(ctx, tenantID, []int64{p.PICStaffID, p.PICSalesStaffID})

	lines := make([]domain.PaymentLine, 0, len(payments))
	for _, pay := range payments {
		lines = append(lines, lineOf(pay.Type, pay.Amount))
	}
	out := &ProjectOverview{
		Project:  *p,
		Progress: data.progressFor(projectID, now, false),
		PICName:  staff[p.PICStaffID],
		Finance:  domain.ComputeFinanceOverview(p.ContractValue, lines, terms, engagements, p.VenueRentalPrice, p.VenueCharge),
		NextTerm: domain.NextOpenTerm(terms),
		Terms:    terms,
		Actions: domain.BuildProjectActions(domain.ActionInput{
			Today: now, EventDate: p.EventDate, Terms: terms, Issues: issues,
			Engagement: engagements, VendorName: nameOf, Milestones: milestones,
		}),
		NextMilestones: domain.NextMilestones(milestones),
		OpenIssueCount: domain.OpenIssueCount(issues),
		Team: OverviewTeam{
			PICStaffID: p.PICStaffID, PICName: staff[p.PICStaffID],
			PICSalesStaffID: p.PICSalesStaffID, PICSalesName: staff[p.PICSalesStaffID],
		},
		Clients: []ProjectClientInfo{},
	}
	for _, e := range domain.UnconfirmedEngagements(engagements) {
		out.Unconfirmed = append(out.Unconfirmed, UnconfirmedVendor{ID: e.ID, VendorName: nameOf[e.ID], EngagementStatus: e.EngagementStatus})
	}
	if s.clients != nil {
		list, err := s.clients.ListProjectClients(ctx, tenantID, projectID)
		if err != nil {
			// The clients are context, not the point of the page: a failure here
			// must not take the whole overview down.
			logger.Error("overview: listing the clients of project %d: %v", projectID, err)
		} else if list != nil {
			out.Clients = list
		}
	}
	return out, nil
}
