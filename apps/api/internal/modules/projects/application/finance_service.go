package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/pagination"
)

// FinanceRepository reads money across the projects of one scope (the Keuangan
// page). Every query joins only this module's own tables (projects and their
// installments, payments and engagements); vendor and staff names are resolved
// afterwards through contracts. Projects that are cancelled or archived are left
// out; a completed project still counts (money owed after the event is real).
type FinanceRepository interface {
	// UnpaidReceivables returns every installment with something left to pay.
	UnpaidReceivables(ctx context.Context, scope domain.FinanceScope) ([]domain.ReceivableRow, error)
	// ListReceivables is the paginated installment list, filtered and sorted in SQL.
	ListReceivables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.ReceivableFilter, params pagination.Params, sort pagination.Sort) ([]domain.ReceivableRow, int64, error)
	ReceivableSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// OpenPayables returns every vendor engagement and venue with something left to
	// pay, and every WO-paid one without an agreed cost yet (Incomplete).
	OpenPayables(ctx context.Context, scope domain.FinanceScope) ([]domain.PayableRow, error)
	ListPayables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.PayableFilter, params pagination.Params, sort pagination.Sort) ([]domain.PayableRow, int64, error)
	PayableSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// ReceivedPayments returns the live client payments dated from `from` to `to` (inclusive).
	ReceivedPayments(ctx context.Context, scope domain.FinanceScope, from, to time.Time) ([]domain.PaymentLine, error)
	// PendingVerification counts the live client payments that have proof but no verification, and their total.
	PendingVerification(ctx context.Context, scope domain.FinanceScope) (domain.MoneyCount, error)
	ListVerifications(ctx context.Context, scope domain.FinanceScope, params pagination.Params, sort pagination.Sort) ([]domain.VerificationRow, int64, error)
	VerificationSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
}

// FinanceService assembles the Keuangan page.
type FinanceService struct {
	repo FinanceRepository
}

func NewFinanceService(repo FinanceRepository) *FinanceService {
	return &FinanceService{repo: repo}
}

// Summary computes the headline figures and the 8-week cash flow (BuildCashFlow,
// the dashboard's own function, over the same "unpaid installment" and "vendor
// balance with a due date" amounts, restricted to the scope).
func (s *FinanceService) Summary(ctx context.Context, scope domain.FinanceScope, today time.Time) (*domain.FinanceSummary, error) {
	receivables, err := s.repo.UnpaidReceivables(ctx, scope)
	if err != nil {
		return nil, err
	}
	payables, err := s.repo.OpenPayables(ctx, scope)
	if err != nil {
		return nil, err
	}
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	received, err := s.repo.ReceivedPayments(ctx, scope, day.AddDate(0, 0, -(domain.ReceivedWindowDays-1)), day)
	if err != nil {
		return nil, err
	}
	pending, err := s.repo.PendingVerification(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := &domain.FinanceSummary{PendingVerification: pending}
	var in, outgoing []domain.DueAmount
	out.Overdue, out.DueSoon, in = domain.SummarizeReceivables(receivables, today)
	out.Payables, outgoing = domain.SummarizePayables(payables, today)
	out.Received = domain.MoneyCount{Amount: domain.ReceivedAmount(received), Count: len(received)}
	out.CashFlow = domain.BuildCashFlow(today, in, outgoing)
	return out, nil
}

func (s *FinanceService) Receivables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.ReceivableFilter, params pagination.Params, sort pagination.Sort) ([]domain.ReceivableRow, int64, error) {
	return s.repo.ListReceivables(ctx, scope, today, f, params, sort)
}

func (s *FinanceService) ReceivableSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.ReceivableSortSpec()
}

func (s *FinanceService) Payables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.PayableFilter, params pagination.Params, sort pagination.Sort) ([]domain.PayableRow, int64, error) {
	return s.repo.ListPayables(ctx, scope, today, f, params, sort)
}

func (s *FinanceService) PayableSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.PayableSortSpec()
}

func (s *FinanceService) Verifications(ctx context.Context, scope domain.FinanceScope, params pagination.Params, sort pagination.Sort) ([]domain.VerificationRow, int64, error) {
	return s.repo.ListVerifications(ctx, scope, params, sort)
}

func (s *FinanceService) VerificationSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.VerificationSortSpec()
}
