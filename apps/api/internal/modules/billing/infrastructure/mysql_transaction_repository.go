package infrastructure

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/pagination"
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

// ListPaginated backs the real `GET /subscription-transactions` list pages
// (Platform Console's transaction table and the WO Console's own history
// table) — List above stays as-is for dashboard revenue aggregation, which
// needs the full set to sum over.
func (r *MySQLTransactionRepository) ListPaginated(ctx context.Context, tenantID *int64, params pagination.Params, status string) ([]domain.Transaction, int64, error) {
	countQuery := `SELECT COUNT(*) FROM subscription_transactions`
	listQuery := `SELECT id, tenant_id, type, amount, payment_method, payment_reference, status, created_at, paid_at
	          FROM subscription_transactions`
	var args []interface{}
	var conditions []string
	if tenantID != nil {
		conditions = append(conditions, `tenant_id = ?`)
		args = append(args, *tenantID)
	}
	if status != "" {
		conditions = append(conditions, `status = ?`)
		args = append(args, status)
	}
	if len(conditions) > 0 {
		where := ` WHERE ` + strings.Join(conditions, " AND ")
		countQuery += where
		listQuery += where
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, listQuery, append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		var txType, status string
		var paidAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.TenantID, &txType, &t.Amount, &t.PaymentMethod, &t.PaymentReference, &status, &t.CreatedAt, &paidAt); err != nil {
			return nil, 0, err
		}
		t.Type = domain.TransactionType(txType)
		t.Status = domain.TransactionStatus(status)
		if paidAt.Valid {
			t.PaidAt = &paidAt.Time
		}
		transactions = append(transactions, t)
	}
	return transactions, total, rows.Err()
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
