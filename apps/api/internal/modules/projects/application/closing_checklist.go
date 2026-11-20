package application

import (
	"context"

	"elproof/internal/modules/projects/domain"
)

// ClosingChecklistService answers "is this project settled?" before it is
// completed: client paid in full, the vendors and venues the WO pays paid,
// kendala closed. Read-only; it informs the person closing the project, it does
// not block.
type ClosingChecklistService struct {
	projects         *ProjectService
	clientPayments   ClientPaymentRepository
	supplierPayments SupplierPaymentRepository
	suppliers        SupplierRepository
	issues           IssueRepository
}

func NewClosingChecklistService(
	projects *ProjectService,
	clientPayments ClientPaymentRepository,
	supplierPayments SupplierPaymentRepository,
	suppliers SupplierRepository,
	issues IssueRepository,
) *ClosingChecklistService {
	return &ClosingChecklistService{
		projects: projects, clientPayments: clientPayments, supplierPayments: supplierPayments,
		suppliers: suppliers, issues: issues,
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
	paymentRows, err := s.supplierPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	suppliers, err := s.suppliers.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	issueRows, err := s.issues.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	in := domain.ClosingInput{ContractValue: p.ContractTotal}
	in.ClientPayments = domain.ContractLines(clientRows)
	for _, sup := range suppliers {
		if !sup.IsActive() || sup.PaidBy != domain.PayerWO || sup.AgreedCost == nil {
			continue
		}
		in.Suppliers = append(in.Suppliers, domain.SupplierBalance{
			AgreedCost: *sup.AgreedCost, Payments: supplierLines(paymentRows, sup.ID, domain.PayerWO, 0),
		})
	}
	for _, i := range issueRows {
		if i.Status.IsOpen() {
			in.OpenIssues++
		}
	}
	return domain.BuildClosingChecklist(in), nil
}
