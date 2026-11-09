package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLPaymentTermRepository struct {
	db *sql.DB
}

func NewMySQLPaymentTermRepository(db *sql.DB) *MySQLPaymentTermRepository {
	return &MySQLPaymentTermRepository{db: db}
}

func (r *MySQLPaymentTermRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.PaymentTerm, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT id, project_id, label, amount, due_date FROM client_payment_terms WHERE project_id = ? ORDER BY due_date, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.PaymentTerm
	for rows.Next() {
		var t domain.PaymentTerm
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Label, &t.Amount, &t.DueDate); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// ReplaceAll swaps the project's whole schedule for `terms` in one
// transaction, so a failed save never leaves half a schedule behind. When the
// caller already runs inside a transaction (atomic project creation) it joins
// that one instead of opening its own.
func (r *MySQLPaymentTermRepository) ReplaceAll(ctx context.Context, projectID int64, terms []domain.PaymentTerm) error {
	return runInTx(ctx, r.db, func(q dbtx) error {
		if _, err := q.ExecContext(ctx, `DELETE FROM client_payment_terms WHERE project_id = ?`, projectID); err != nil {
			return err
		}
		for _, t := range terms {
			if _, err := q.ExecContext(ctx,
				`INSERT INTO client_payment_terms (project_id, label, amount, due_date) VALUES (?, ?, ?, ?)`,
				projectID, t.Label, t.Amount, t.DueDate); err != nil {
				return err
			}
		}
		return nil
	})
}
