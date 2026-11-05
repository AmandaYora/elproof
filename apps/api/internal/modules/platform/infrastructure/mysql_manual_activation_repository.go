package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/platform/domain"
)

type MySQLManualActivationRepository struct {
	db *sql.DB
}

func NewMySQLManualActivationRepository(db *sql.DB) *MySQLManualActivationRepository {
	return &MySQLManualActivationRepository{db: db}
}

// ApplyGrant extends the tenant's subscription and records why, in one
// transaction (both tables belong to `platform`), so a grant can never exist
// without its reason and proof — or the record without the grant.
func (r *MySQLManualActivationRepository) ApplyGrant(ctx context.Context, a *domain.ManualActivation, status domain.SubscriptionStatus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE tenants SET plan_id = ?, subscription_status = ?, subscription_expires_at = ? WHERE id = ?`,
		a.PlanID, string(status), a.ExpiresAt, a.TenantID,
	); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO manual_activations (tenant_id, plan_id, actor_admin_id, reason, proof_reference, note, transaction_reference, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.TenantID, a.PlanID, a.ActorAdminID, string(a.Reason), a.ProofReference, a.Note, a.TransactionReference, a.ExpiresAt,
	)
	if err != nil {
		return err
	}
	if a.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return tx.Commit()
}
