package application

import (
	"context"

	"elproof/internal/modules/projects/domain"
)

// ClosingChecklistService answers "is this project settled?" before it is
// completed: client paid in full, vendors and venue paid, issues closed.
// Read-only; it informs the person closing the project, it does not block.
type ClosingChecklistService struct {
	projects       *ProjectService
	clientPayments ClientPaymentRepository
	vendorPayments PaymentRepository
	venuePayments  VenuePaymentRepository
	engagements    VendorEngagementRepository
	issues         IssueRepository
}

func NewClosingChecklistService(
	projects *ProjectService,
	clientPayments ClientPaymentRepository,
	vendorPayments PaymentRepository,
	venuePayments VenuePaymentRepository,
	engagements VendorEngagementRepository,
	issues IssueRepository,
) *ClosingChecklistService {
	return &ClosingChecklistService{
		projects: projects, clientPayments: clientPayments, vendorPayments: vendorPayments,
		venuePayments: venuePayments, engagements: engagements, issues: issues,
	}
}

func (s *ClosingChecklistService) Get(ctx context.Context, tenantID, projectID int64) ([]domain.ChecklistItem, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	clientRows, err := s.clientPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	vendorRows, err := s.vendorPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	venueRows, err := s.venuePayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	engagements, err := s.engagements.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	issueRows, err := s.issues.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	in := domain.ClosingInput{ContractValue: p.ContractValue}
	for _, r := range clientRows {
		in.ClientPayments = append(in.ClientPayments, domain.PaymentLine{Type: r.Type, Amount: r.Amount})
	}
	byEngagement := make(map[int64][]domain.PaymentLine)
	for _, r := range vendorRows {
		byEngagement[r.ProjectVendorID] = append(byEngagement[r.ProjectVendorID], domain.PaymentLine{Type: r.Type, Amount: r.Amount})
	}
	for _, e := range engagements {
		if e.EngagementStatus == domain.EngagementCancelled {
			continue
		}
		in.Vendors = append(in.Vendors, domain.VendorBalance{ContractValue: e.ContractValue, Payments: byEngagement[e.ID]})
	}
	if p.VenueRentalPrice != nil {
		in.VenueCost += *p.VenueRentalPrice
	}
	if p.VenueCharge != nil {
		in.VenueCost += *p.VenueCharge
	}
	for _, r := range venueRows {
		in.VenuePayments = append(in.VenuePayments, domain.PaymentLine{Type: r.Type, Amount: r.Amount})
	}
	for _, i := range issueRows {
		if i.Status.IsOpen() {
			in.OpenIssues++
		}
	}
	return domain.BuildClosingChecklist(in), nil
}
