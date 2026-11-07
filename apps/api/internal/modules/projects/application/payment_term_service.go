package application

import (
	"context"
	"strings"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type PaymentTermRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.PaymentTerm, error)
	ReplaceAll(ctx context.Context, projectID int64, terms []domain.PaymentTerm) error
}

const maxPaymentTerms = 20

// PaymentTermInput is one installment as submitted by the schedule editor.
type PaymentTermInput struct {
	Label   string
	Amount  int64
	DueDate time.Time
}

// PaymentTermService keeps the client's payment schedule and tells, against
// what was really paid, how each installment stands.
type PaymentTermService struct {
	repo           PaymentTermRepository
	clientPayments ClientPaymentRepository
	projects       *ProjectService
	activity       *ActivityService
}

func NewPaymentTermService(repo PaymentTermRepository, clientPayments ClientPaymentRepository, projects *ProjectService, activity *ActivityService) *PaymentTermService {
	return &PaymentTermService{repo: repo, clientPayments: clientPayments, projects: projects, activity: activity}
}

// List returns the schedule with each installment's progress as of `today`.
func (s *PaymentTermService) List(ctx context.Context, tenantID, projectID int64, today time.Time) ([]domain.TermProgress, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return s.progress(ctx, projectID, today)
}

func (s *PaymentTermService) progress(ctx context.Context, projectID int64, today time.Time) ([]domain.TermProgress, error) {
	terms, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	payments, err := s.clientPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	lines := make([]domain.PaymentLine, 0, len(payments))
	for _, p := range payments {
		lines = append(lines, domain.PaymentLine{Type: p.Type, Amount: p.Amount})
	}
	return domain.AllocateTerms(terms, domain.AppliedToContract(lines), today), nil
}

// Replace saves the whole schedule. The installments together may not exceed
// the contract value (when one is set): a schedule that bills more than the
// contract would make every later payment look overdue for no reason.
func (s *PaymentTermService) Replace(ctx context.Context, tenantID, projectID, actorStaffID int64, inputs []PaymentTermInput, today time.Time) ([]domain.TermProgress, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if len(inputs) > maxPaymentTerms {
		return nil, apperror.Validation("Jadwal pembayaran terlalu panjang", map[string][]string{
			"terms": {"Maksimal 20 termin per project"},
		})
	}
	var total int64
	terms := make([]domain.PaymentTerm, 0, len(inputs))
	for _, in := range inputs {
		label := strings.TrimSpace(in.Label)
		switch {
		case label == "" || len([]rune(label)) > 100:
			return nil, apperror.Validation("Nama termin wajib diisi (maksimal 100 karakter)", map[string][]string{"label": {"Isi nama termin"}})
		case in.Amount <= 0 || in.Amount > domain.MaxMoney:
			return nil, apperror.Validation("Nominal termin harus lebih dari 0 dan wajar", map[string][]string{"amount": {"Nominal harus lebih dari 0 dan wajar"}})
		case in.DueDate.IsZero():
			return nil, apperror.Validation("Tanggal jatuh tempo wajib diisi", map[string][]string{"dueDate": {"Isi tanggal jatuh tempo"}})
		}
		total += in.Amount
		terms = append(terms, domain.PaymentTerm{ProjectID: projectID, Label: label, Amount: in.Amount, DueDate: in.DueDate})
	}
	if p.ContractValue > 0 && total > p.ContractValue {
		msg := "Jumlah termin " + formatRupiah(total) + " melebihi nilai kontrak " + formatRupiah(p.ContractValue) + ". Kurangi nominal salah satu termin."
		return nil, apperror.Validation(msg, map[string][]string{"terms": {msg}})
	}
	if err := s.repo.ReplaceAll(ctx, projectID, terms); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityProjectUpdated, actorStaffID, "payment_terms", formatID(projectID), p.Name,
		"Jadwal pembayaran client diperbarui")
	return s.progress(ctx, projectID, today)
}
