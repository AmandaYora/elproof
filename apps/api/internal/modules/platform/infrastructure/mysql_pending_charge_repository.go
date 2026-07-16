package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/platform/domain"
)

type MySQLPendingChargeRepository struct {
	db *sql.DB
}

func NewMySQLPendingChargeRepository(db *sql.DB) *MySQLPendingChargeRepository {
	return &MySQLPendingChargeRepository{db: db}
}

func (r *MySQLPendingChargeRepository) Create(ctx context.Context, orderRef string, tenantID, planID int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO pending_subscription_charges (order_ref, tenant_id, plan_id) VALUES (?, ?, ?)`,
		orderRef, tenantID, planID,
	)
	return err
}

func (r *MySQLPendingChargeRepository) FindByOrderRef(ctx context.Context, orderRef string) (*domain.PendingCharge, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT order_ref, tenant_id, plan_id FROM pending_subscription_charges WHERE order_ref = ? LIMIT 1`,
		orderRef,
	)
	var p domain.PendingCharge
	err := row.Scan(&p.OrderRef, &p.TenantID, &p.PlanID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *MySQLPendingChargeRepository) Delete(ctx context.Context, orderRef string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM pending_subscription_charges WHERE order_ref = ?`, orderRef)
	return err
}
