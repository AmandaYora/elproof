package application

import (
	"context"
	"errors"
	"strconv"
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
	payments, err := s.clientPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.progressWith(ctx, projectID, payments, today)
}

// progressWith is progress for a caller that already loaded the project's
// client payments (one query instead of two).
func (s *PaymentTermService) progressWith(ctx context.Context, projectID int64, payments []domain.ClientPayment, today time.Time) ([]domain.TermProgress, error) {
	terms, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return domain.AllocateTerms(terms, appliedFromPayments(payments), today), nil
}

// appliedFromPayments is how much of the contract the payments cover (only the
// ones that count: a couple's unverified submission does not, ADR-0047).
func appliedFromPayments(payments []domain.ClientPayment) int64 {
	return domain.AppliedToContract(domain.ContractLines(payments))
}

// Replace saves the whole schedule. A schedule is made after the Rincian is
// saved (PLAN §9.1): with no contract value yet there is nothing to divide.
// The installments together may not exceed the contract value: a schedule that
// bills more than the contract would make every later payment look overdue for
// no reason. An empty schedule (removing it) is always allowed.
func (s *PaymentTermService) Replace(ctx context.Context, tenantID, projectID, actorStaffID int64, inputs []PaymentTermInput, today time.Time) ([]domain.TermProgress, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if len(inputs) > maxPaymentTerms {
		return nil, apperror.Validation("Jadwal pembayaran terlalu panjang", map[string][]string{
			"terms": {"Maksimal 20 termin per proyek"},
		})
	}
	terms, total, err := buildTerms(projectID, inputs)
	if err != nil {
		return nil, err
	}
	if len(terms) > 0 && p.ContractTotal <= 0 {
		msg := "Susun rincian dulu agar termin bisa dihitung."
		return nil, apperror.Validation(msg, map[string][]string{"terms": {msg}}).WithCode("no_contract")
	}
	if total > p.ContractTotal {
		msg := "Jumlah termin " + formatRupiah(total) + " melebihi nilai kontrak " + formatRupiah(p.ContractTotal) + ". Kurangi nominal salah satu termin."
		return nil, apperror.Validation(msg, map[string][]string{"terms": {msg}})
	}
	if err := s.repo.ReplaceAll(ctx, projectID, terms); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityProjectUpdated, actorStaffID, "payment_terms", formatID(projectID), p.Name,
		"Jadwal termin klien diperbarui")
	return s.progress(ctx, projectID, today)
}

// Recalculate fits the schedule to the project's current contract value after
// that value changed (EP-05), without touching money already received: the
// installments with any payment on them keep their amount, the ones with nothing
// paid share what is left in proportion to their current amounts, in whole
// rupiah with the remainder on the last (domain.RecalculateTerms). It is judged
// against the payments as they are now, which is why it is done here and not by
// the browser sending a rebuilt schedule. A schedule that cannot be fitted is a
// 422 on "terms" saying why and what to do instead.
func (s *PaymentTermService) Recalculate(ctx context.Context, tenantID, projectID, actorStaffID int64, today time.Time) ([]domain.TermProgress, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	terms, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	payments, err := s.clientPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	next, err := domain.RecalculateTerms(terms, appliedFromPayments(payments), p.ContractTotal)
	if err != nil {
		msg := recalcRefusal(err)
		return nil, apperror.Validation(msg, map[string][]string{"terms": {msg}})
	}
	for i := range next {
		next[i].ID = 0
		next[i].ProjectID = projectID
	}
	if err := s.repo.ReplaceAll(ctx, projectID, next); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityProjectUpdated, actorStaffID, "payment_terms", formatID(projectID), p.Name,
		"Jadwal termin klien dihitung ulang mengikuti nilai kontrak "+formatRupiah(p.ContractTotal)+
			" (sebelumnya "+formatRupiah(domain.TermsTotal(terms))+")")
	return s.progressWith(ctx, projectID, payments, today)
}

// recalcRefusal says, in the person's words, why the schedule cannot follow the
// contract value and what to do instead.
func recalcRefusal(err error) string {
	switch {
	case errors.Is(err, domain.ErrRecalcNoContract):
		return "Susun rincian dulu agar termin bisa dihitung."
	case errors.Is(err, domain.ErrRecalcNoTerms):
		return "Belum ada jadwal termin untuk dihitung ulang."
	case errors.Is(err, domain.ErrRecalcAllPaid):
		return "Semua termin sudah ada pembayarannya, jadi tidak ada yang bisa dihitung ulang. Ubah jadwal termin secara manual."
	default:
		return "Nilai kontrak lebih kecil dari termin yang sudah dibayar. Periksa rincian atau ubah jadwal termin secara manual."
	}
}

// buildTerms validates every row and reports every problem at once, keyed by
// row ("terms[2].amount"), so the form can mark each one instead of stopping at
// the first.
func buildTerms(projectID int64, inputs []PaymentTermInput) ([]domain.PaymentTerm, int64, error) {
	var total int64
	fields := map[string][]string{}
	var first string
	problem := func(key, msg string) {
		fields[key] = append(fields[key], msg)
		if first == "" {
			first = msg
		}
	}
	terms := make([]domain.PaymentTerm, 0, len(inputs))
	for i, in := range inputs {
		row := "terms[" + strconv.Itoa(i) + "]"
		label := strings.TrimSpace(in.Label)
		if label == "" || len([]rune(label)) > 100 {
			problem(row+".label", "Isi nama termin (maksimal 100 karakter)")
		}
		if in.Amount <= 0 || in.Amount > domain.MaxMoney {
			problem(row+".amount", "Nominal harus lebih dari 0 dan wajar")
		}
		if in.DueDate.IsZero() {
			problem(row+".dueDate", "Isi tanggal jatuh tempo")
		}
		total += in.Amount
		terms = append(terms, domain.PaymentTerm{ProjectID: projectID, Label: label, Amount: in.Amount, DueDate: in.DueDate})
	}
	if len(fields) > 0 {
		msg := first
		if n := len(fields); n > 1 {
			msg += " Ada " + strconv.Itoa(n-1) + " isian lain yang perlu diperbaiki."
		}
		return nil, 0, apperror.Validation(msg, fields)
	}
	return terms, total, nil
}
