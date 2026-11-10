package application

import (
	"context"
	"elproof/internal/shared/apperror"
	"sort"
	"strings"
	"time"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/pagination"
)

type TransactionRepository interface {
	List(ctx context.Context, tenantID *int64) ([]domain.Transaction, error)
	ListPaginated(ctx context.Context, tenantID *int64, params pagination.Params, f domain.TransactionFilter, sort pagination.Sort) ([]domain.Transaction, int64, error)
	// ListAllFiltered returns every match, newest first (ordering by tenant name happens in memory).
	ListAllFiltered(ctx context.Context, tenantID *int64, f domain.TransactionFilter) ([]domain.Transaction, error)
	// ListSortSpec is the ?sort= whitelist (public name -> fixed SQL) and the default order.
	ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// FindByID returns one transaction or nil; a non-nil tenantID confines it to that tenant.
	FindByID(ctx context.Context, tenantID *int64, id int64) (*domain.Transaction, error)
	// ListPaidSince returns transactions with status "paid" paid at or after `since` (granted ones are not revenue).
	ListPaidSince(ctx context.Context, since time.Time) ([]domain.Transaction, error)
	Create(ctx context.Context, tx *domain.Transaction) error
	UpdateStatusByReference(ctx context.Context, paymentReference string, status domain.TransactionStatus, paidAt *time.Time) error
}

// TenantDirectory is what the transaction list needs from `platform`: the
// tenants' names (the ledger stores only tenant ids) and the ids of the tenants
// whose name matches a search. Declared here and wired in cmd/server, so billing
// never imports platform.
type TenantDirectory interface {
	TenantNames(ctx context.Context, ids []int64) (map[int64]string, error)
	SearchTenantIDs(ctx context.Context, text string, limit int) ([]int64, error)
}

// maxSearchTenants bounds how many tenants a name search resolves: a search
// matching more than this many workspaces is too broad to be what was meant.
const maxSearchTenants = 1000

type TransactionService struct {
	repo     TransactionRepository
	tenants  TenantDirectory
	vouchers VoucherSettler
}

// VoucherSettler records how a charge that used a voucher ended (VoucherService).
type VoucherSettler interface {
	Settle(ctx context.Context, orderRef string, paid bool) error
}

// SetVouchers wires the voucher ledger (same module, built after this service).
func (s *TransactionService) SetVouchers(v VoucherSettler) { s.vouchers = v }

// SetTenants completes the two-phase wiring with `platform` (cmd/server).
func (s *TransactionService) SetTenants(d TenantDirectory) { s.tenants = d }

func NewTransactionService(repo TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) List(ctx context.Context, tenantID *int64) ([]domain.Transaction, error) {
	return s.repo.List(ctx, tenantID)
}

// TransactionSortSpec is the whitelist of ?sort= and its default order (today's:
// newest first). createdAt, amount and status are SQL; tenantName is ordered after
// the names are read from `platform`.
func (s *TransactionService) TransactionSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.ListSortSpec()
}

// TransactionRow is a transaction with its tenant's name ("" when it cannot be resolved).
type TransactionRow struct {
	domain.Transaction
	TenantName string
}

const sortTenantName = "tenantName"

// ListPage is the transaction list: filters and ordering run in SQL on this
// module's table; the tenant's name (and the tenants a name search matches) come
// from `platform` in one batch call. Ordering by tenantName needs every matching
// transaction's tenant name, so those rows are read, named, ordered and sliced here.
func (s *TransactionService) ListPage(ctx context.Context, tenantID *int64, params pagination.Params, f domain.TransactionFilter, sort pagination.Sort) ([]TransactionRow, int64, error) {
	if f.Search != "" && s.tenants != nil {
		ids, err := s.tenants.SearchTenantIDs(ctx, f.Search, maxSearchTenants)
		if err != nil {
			return nil, 0, err
		}
		f.TenantIDs = ids
	}

	var page []domain.Transaction
	var total int64
	var names map[int64]string
	if sort.Field == sortTenantName {
		all, err := s.repo.ListAllFiltered(ctx, tenantID, f)
		if err != nil {
			return nil, 0, err
		}
		if names, err = s.namesOf(ctx, all); err != nil {
			return nil, 0, err
		}
		sortByTenantName(all, names, sort.Desc)
		total = int64(len(all))
		start, end := params.Offset(), params.Offset()+params.Limit
		if start > len(all) {
			start = len(all)
		}
		if end > len(all) {
			end = len(all)
		}
		page = all[start:end]
	} else {
		var err error
		if page, total, err = s.repo.ListPaginated(ctx, tenantID, params, f, sort); err != nil {
			return nil, 0, err
		}
		if names, err = s.namesOf(ctx, page); err != nil {
			return nil, 0, err
		}
	}
	rows := make([]TransactionRow, len(page))
	for i, t := range page {
		rows[i] = TransactionRow{Transaction: t, TenantName: names[t.TenantID]}
	}
	return rows, total, nil
}

// NameOf labels one transaction (the receipt page); "" when unresolved.
func (s *TransactionService) NameOf(ctx context.Context, t domain.Transaction) string {
	names, err := s.namesOf(ctx, []domain.Transaction{t})
	if err != nil {
		return ""
	}
	return names[t.TenantID]
}

func (s *TransactionService) namesOf(ctx context.Context, list []domain.Transaction) (map[int64]string, error) {
	if s.tenants == nil || len(list) == 0 {
		return map[int64]string{}, nil
	}
	seen := map[int64]bool{}
	var ids []int64
	for _, t := range list {
		if !seen[t.TenantID] {
			seen[t.TenantID] = true
			ids = append(ids, t.TenantID)
		}
	}
	return s.tenants.TenantNames(ctx, ids)
}

// sortByTenantName orders transactions by their tenant's name, ignoring case;
// a tenant whose name does not resolve sorts last whatever the direction, and
// ties keep the transaction id order in the same direction.
func sortByTenantName(list []domain.Transaction, names map[int64]string, desc bool) {
	sort.SliceStable(list, func(i, j int) bool {
		a, aok := names[list[i].TenantID]
		b, bok := names[list[j].TenantID]
		if aok != bok {
			return aok // unresolved last, both directions
		}
		a, b = strings.ToLower(a), strings.ToLower(b)
		if a == b {
			if desc {
				return list[i].ID > list[j].ID
			}
			return list[i].ID < list[j].ID
		}
		if desc {
			return a > b
		}
		return a < b
	})
}

// PaidSince backs the platform revenue metrics.
// Get returns one transaction, or a not-found error when it does not exist or
// belongs to another tenant than the (non-nil) tenantID: both look the same.
func (s *TransactionService) Get(ctx context.Context, tenantID *int64, id int64) (*domain.Transaction, error) {
	t, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperror.NotFound("Transaksi tidak ditemukan")
	}
	return t, nil
}

func (s *TransactionService) PaidSince(ctx context.Context, since time.Time) ([]domain.Transaction, error) {
	return s.repo.ListPaidSince(ctx, since)
}

type RecordTransactionInput struct {
	TenantID int64
	Type     domain.TransactionType
	Amount   int64
	// OriginalAmount and DiscountAmount describe a voucher discount; without
	// one OriginalAmount is left 0 and recorded as Amount.
	OriginalAmount   int64
	DiscountAmount   int64
	VoucherCode      string
	PaymentMethod    string
	PaymentReference string
	Status           domain.TransactionStatus
}

// RecordTransaction is called, via contracts, by `platform`'s orchestration
// for tenant registration (status "unpaid"), self-service payment ("paid"),
// and manual activation ("granted") — only "paid"/"granted" get a PaidAt.
func (s *TransactionService) RecordTransaction(ctx context.Context, input RecordTransactionInput) (*domain.Transaction, error) {
	tx := &domain.Transaction{
		TenantID:         input.TenantID,
		Type:             input.Type,
		Amount:           input.Amount,
		OriginalAmount:   input.OriginalAmount,
		DiscountAmount:   input.DiscountAmount,
		VoucherCode:      input.VoucherCode,
		PaymentMethod:    input.PaymentMethod,
		PaymentReference: input.PaymentReference,
		Status:           input.Status,
	}
	if tx.OriginalAmount == 0 {
		tx.OriginalAmount = tx.Amount + tx.DiscountAmount
	}
	if input.Status == domain.StatusPaid || input.Status == domain.StatusGranted {
		now := time.Now()
		tx.PaidAt = &now
	}
	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// UpdateStatus is called, via contracts, by `platform`'s webhook consumer
// (Fase 9) once a gateway charge is confirmed paid/expired — stamps PaidAt
// itself when the new status is Paid, same convention as RecordTransaction.
func (s *TransactionService) UpdateStatus(ctx context.Context, paymentReference string, status domain.TransactionStatus) error {
	var paidAt *time.Time
	if status == domain.StatusPaid {
		now := time.Now()
		paidAt = &now
	}
	if err := s.repo.UpdateStatusByReference(ctx, paymentReference, status, paidAt); err != nil {
		return err
	}
	// A voucher use counts while its charge is pending: paid makes it final,
	// expired or cancelled gives it back to the quota and to the customer.
	if s.vouchers != nil {
		switch status {
		case domain.StatusPaid:
			return s.vouchers.Settle(ctx, paymentReference, true)
		case domain.StatusExpired, domain.StatusCancelled:
			return s.vouchers.Settle(ctx, paymentReference, false)
		}
	}
	return nil
}
