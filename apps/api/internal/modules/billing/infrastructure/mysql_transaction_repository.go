package infrastructure

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

type MySQLTransactionRepository struct {
	db *sql.DB
}

func NewMySQLTransactionRepository(db *sql.DB) *MySQLTransactionRepository {
	return &MySQLTransactionRepository{db: db}
}

func (r *MySQLTransactionRepository) List(ctx context.Context, tenantID *int64) ([]domain.Transaction, error) {
	query := `SELECT id, tenant_id, type, amount, payment_method, payment_reference, status, created_at, paid_at
	          FROM subscription_transactions`
	args := []interface{}{}
	if tenantID != nil {
		query += ` WHERE tenant_id = ?`
		args = append(args, *tenantID)
	}
	query += ` ORDER BY created_at DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		var txType, status string
		var paidAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.TenantID, &txType, &t.Amount, &t.PaymentMethod, &t.PaymentReference, &status, &t.CreatedAt, &paidAt); err != nil {
			return nil, err
		}
		t.Type = domain.TransactionType(txType)
		t.Status = domain.TransactionStatus(status)
		if paidAt.Valid {
			t.PaidAt = &paidAt.Time
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

// FindByID returns one transaction, or nil. tenantID, when set, confines the
// lookup to that tenant (a staff member can only ever see their own).
func (r *MySQLTransactionRepository) FindByID(ctx context.Context, tenantID *int64, id int64) (*domain.Transaction, error) {
	query := `SELECT id, tenant_id, type, amount, payment_method, payment_reference, status, created_at, paid_at
	          FROM subscription_transactions WHERE id = ?`
	args := []interface{}{id}
	if tenantID != nil {
		query += ` AND tenant_id = ?`
		args = append(args, *tenantID)
	}
	var t domain.Transaction
	var txType, status string
	var paidAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query+` LIMIT 1`, args...).
		Scan(&t.ID, &t.TenantID, &txType, &t.Amount, &t.PaymentMethod, &t.PaymentReference, &status, &t.CreatedAt, &paidAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Type = domain.TransactionType(txType)
	t.Status = domain.TransactionStatus(status)
	if paidAt.Valid {
		t.PaidAt = &paidAt.Time
	}
	return &t, nil
}

// ListPaidSince returns the paid (not granted) transactions since a moment.
func (r *MySQLTransactionRepository) ListPaidSince(ctx context.Context, since time.Time) ([]domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, type, amount, payment_method, payment_reference, status, created_at, paid_at
		 FROM subscription_transactions WHERE status = 'paid' AND paid_at >= ? ORDER BY paid_at`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		var txType, status string
		var paidAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.TenantID, &txType, &t.Amount, &t.PaymentMethod, &t.PaymentReference, &status, &t.CreatedAt, &paidAt); err != nil {
			return nil, err
		}
		t.Type = domain.TransactionType(txType)
		t.Status = domain.TransactionStatus(status)
		if paidAt.Valid {
			t.PaidAt = &paidAt.Time
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// transactionSortColumns is the sort whitelist of GET /subscription-transactions
// (ADR-0035). tenantName is the tenant's name, which `platform` owns: the
// application layer reads the names and orders in memory (see ListAllFiltered).
var transactionSortColumns = map[string]pagination.SortColumn{
	"createdAt":  pagination.Col("created_at"),
	"amount":     pagination.Col("amount"),
	"status":     pagination.Col("status"),
	"tenantName": pagination.Col("tenant_id"),
}

// ListSortSpec is the whitelist and the default order of the list (today's:
// newest first).
func (r *MySQLTransactionRepository) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return transactionSortColumns, pagination.By("createdAt", true)
}

const transactionColumns = `id, tenant_id, type, amount, payment_method, payment_reference, status, created_at, paid_at`

// transactionWhere: the tenant scope (a staff member sees only their own tenant),
// the status, the date range on created_at and the search (payment reference, or
// the tenants whose name matched).
func transactionWhere(tenantID *int64, f domain.TransactionFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	if tenantID != nil {
		conditions = append(conditions, `tenant_id = ?`)
		args = append(args, *tenantID)
	}
	if f.Status != "" {
		conditions = append(conditions, `status = ?`)
		args = append(args, f.Status)
	}
	if !f.From.IsZero() {
		conditions = append(conditions, `created_at >= ?`)
		args = append(args, f.From)
	}
	if !f.To.IsZero() {
		conditions = append(conditions, `created_at < ?`)
		args = append(args, f.To)
	}
	if f.Search != "" {
		like := "%" + utils.EscapeLike(f.Search) + "%"
		if len(f.TenantIDs) > 0 {
			conditions = append(conditions, `(payment_reference LIKE ? OR tenant_id IN (`+utils.Placeholders(len(f.TenantIDs))+`))`)
			args = append(args, like)
			args = append(args, utils.Int64Args(f.TenantIDs)...)
		} else {
			conditions = append(conditions, `payment_reference LIKE ?`)
			args = append(args, like)
		}
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return ` WHERE ` + strings.Join(conditions, " AND "), args
}

func scanTransactions(rows *sql.Rows) ([]domain.Transaction, error) {
	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		var txType, status string
		var paidAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.TenantID, &txType, &t.Amount, &t.PaymentMethod, &t.PaymentReference, &status, &t.CreatedAt, &paidAt); err != nil {
			return nil, err
		}
		t.Type = domain.TransactionType(txType)
		t.Status = domain.TransactionStatus(status)
		if paidAt.Valid {
			t.PaidAt = &paidAt.Time
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

// ListPaginated backs the real `GET /subscription-transactions` list pages
// (Platform Console's transaction table and the WO Console's own history
// table) — List above stays as-is for dashboard revenue aggregation, which
// needs the full set to sum over.
func (r *MySQLTransactionRepository) ListPaginated(ctx context.Context, tenantID *int64, params pagination.Params, f domain.TransactionFilter, sort pagination.Sort) ([]domain.Transaction, int64, error) {
	where, args := transactionWhere(tenantID, f)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_transactions`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+transactionColumns+` FROM subscription_transactions`+where+sort.OrderBy("id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	transactions, err := scanTransactions(rows)
	return transactions, total, err
}

// ListAllFiltered returns every transaction matching the filters, newest first:
// the base of the in-memory ordering by tenant name.
func (r *MySQLTransactionRepository) ListAllFiltered(ctx context.Context, tenantID *int64, f domain.TransactionFilter) ([]domain.Transaction, error) {
	where, args := transactionWhere(tenantID, f)
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+transactionColumns+` FROM subscription_transactions`+where+` ORDER BY created_at DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

// UpdateStatusByReference flips an existing transaction's status (Fase 9's
// webhook-confirmed payment path) — `paidAt` is only stamped when non-nil,
// matching Create's own "only Paid/Granted get a PaidAt" convention.
func (r *MySQLTransactionRepository) UpdateStatusByReference(ctx context.Context, paymentReference string, status domain.TransactionStatus, paidAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscription_transactions SET status = ?, paid_at = ? WHERE payment_reference = ?`,
		string(status), paidAt, paymentReference,
	)
	return err
}

func (r *MySQLTransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO subscription_transactions (tenant_id, type, amount, payment_method, payment_reference, status, paid_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		tx.TenantID, string(tx.Type), tx.Amount, tx.PaymentMethod, tx.PaymentReference, string(tx.Status), tx.PaidAt,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	tx.ID = id
	return nil
}
