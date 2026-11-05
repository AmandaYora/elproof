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

func (r *MySQLManualActivationRepository) Create(ctx context.Context, a *domain.ManualActivation) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO manual_activations (tenant_id, plan_id, actor_admin_id, reason, proof_reference, note, transaction_reference, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.TenantID, a.PlanID, a.ActorAdminID, string(a.Reason), a.ProofReference, a.Note, a.TransactionReference, a.ExpiresAt,
	)
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}
