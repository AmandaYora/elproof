package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/payment/domain"
)

// MySQLDispatchRepository backs the Charge Dispatch Index — a thin
// order_ref -> app_id lookup, never a ledger (see MODULE_PAYMENT.md §4).
// ExpiresAt/ResolvedAt (added for reconciliation — see the same doc) are the
// only exception to "never a ledger": completion markers, not amounts.
type MySQLDispatchRepository struct {
	db *sql.DB
}

func NewMySQLDispatchRepository(db *sql.DB) *MySQLDispatchRepository {
	return &MySQLDispatchRepository{db: db}
}

func (r *MySQLDispatchRepository) Create(ctx context.Context, d *domain.ChargeDispatch) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO payment_charge_dispatch (order_ref, app_id, provider_ref, expires_at) VALUES (?, ?, ?, ?)`,
		d.OrderRef, d.AppID, nullableString(d.ProviderRef), nullableTime(d.ExpiresAt),
	)
	return err
}

const dispatchColumns = `order_ref, app_id, provider_ref, expires_at, resolved_at, created_at`

func scanDispatch(scan func(dest ...interface{}) error) (*domain.ChargeDispatch, error) {
	var d domain.ChargeDispatch
	var providerRef sql.NullString
	var expiresAt, resolvedAt sql.NullTime
	err := scan(&d.OrderRef, &d.AppID, &providerRef, &expiresAt, &resolvedAt, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.ProviderRef = providerRef.String
	if expiresAt.Valid {
		d.ExpiresAt = expiresAt.Time
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		d.ResolvedAt = &t
	}
	return &d, nil
}

func (r *MySQLDispatchRepository) FindByOrderRef(ctx context.Context, orderRef string) (*domain.ChargeDispatch, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+dispatchColumns+` FROM payment_charge_dispatch WHERE order_ref = ? LIMIT 1`,
		orderRef,
	)
	return scanDispatch(row.Scan)
}

// ClaimUnresolved atomically marks a dispatch as resolved, returning whether
// this call actually won the claim (false means someone else — a concurrent
// webhook delivery or the reconciliation sweep — already resolved it first).
// This is the single choke point that guarantees a terminal outcome (paid/
// expired/failed/refund) is ever dispatched to a consumer exactly once,
// regardless of whether it arrived via webhook or was discovered by polling
// Tripay directly.
func (r *MySQLDispatchRepository) ClaimUnresolved(ctx context.Context, orderRef string) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE payment_charge_dispatch SET resolved_at = ? WHERE order_ref = ? AND resolved_at IS NULL`,
		time.Now(), orderRef,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// ListUnresolvedForSweep returns dispatches with no terminal outcome yet,
// created before `olderThan` (so brand-new charges aren't checked before
// they've had a realistic chance to be paid), oldest first.
func (r *MySQLDispatchRepository) ListUnresolvedForSweep(ctx context.Context, olderThan time.Time, limit int) ([]domain.ChargeDispatch, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+dispatchColumns+` FROM payment_charge_dispatch
		 WHERE resolved_at IS NULL AND created_at < ?
		 ORDER BY created_at ASC LIMIT ?`,
		olderThan, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ChargeDispatch
	for rows.Next() {
		d, err := scanDispatch(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *d)
	}
	return result, rows.Err()
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
