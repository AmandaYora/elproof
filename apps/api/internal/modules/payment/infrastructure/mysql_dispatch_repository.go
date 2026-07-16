package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/payment/domain"
)

// MySQLDispatchRepository backs the Charge Dispatch Index — a thin
// order_ref -> app_id lookup, never a ledger (see MODULE_PAYMENT.md §4).
type MySQLDispatchRepository struct {
	db *sql.DB
}

func NewMySQLDispatchRepository(db *sql.DB) *MySQLDispatchRepository {
	return &MySQLDispatchRepository{db: db}
}

func (r *MySQLDispatchRepository) Create(ctx context.Context, d *domain.ChargeDispatch) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO payment_charge_dispatch (order_ref, app_id, provider_ref) VALUES (?, ?, ?)`,
		d.OrderRef, d.AppID, nullableString(d.ProviderRef),
	)
	return err
}

func (r *MySQLDispatchRepository) FindByOrderRef(ctx context.Context, orderRef string) (*domain.ChargeDispatch, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT order_ref, app_id, provider_ref, created_at FROM payment_charge_dispatch WHERE order_ref = ? LIMIT 1`,
		orderRef,
	)
	var d domain.ChargeDispatch
	var providerRef sql.NullString
	err := row.Scan(&d.OrderRef, &d.AppID, &providerRef, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.ProviderRef = providerRef.String
	return &d, nil
}
