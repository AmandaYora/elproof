package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

// MySQLListMoneyRepository reads, for a page of projects, the two sources of the
// list's derived columns (remaining, nextDueDate): their live client payments
// and their payment schedule — one query each, whatever the page size.
type MySQLListMoneyRepository struct {
	db *sql.DB
}

func NewMySQLListMoneyRepository(db *sql.DB) *MySQLListMoneyRepository {
	return &MySQLListMoneyRepository{db: db}
}

// PaymentLinesByProjects returns the live (not deleted) client payments of the
// projects as the lines the ledger rules read. The caller passes ids of its own
// tenant's projects (a page it has just listed).
func (r *MySQLListMoneyRepository) PaymentLinesByProjects(ctx context.Context, projectIDs []int64) (map[int64][]domain.PaymentLine, error) {
	out := make(map[int64][]domain.PaymentLine, len(projectIDs))
	if len(projectIDs) == 0 {
		return out, nil
	}
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT project_id, type, amount FROM client_payments WHERE project_id IN (`+utils.Placeholders(len(projectIDs))+`) AND deleted_at IS NULL`,
		utils.Int64Args(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var projectID, amount int64
		var typ string
		if err := rows.Scan(&projectID, &typ, &amount); err != nil {
			return nil, err
		}
		out[projectID] = append(out[projectID], domain.PaymentLine{Type: domain.PaymentType(typ), Amount: amount})
	}
	return out, rows.Err()
}

// TermsByProjects returns the payment schedules of the projects.
func (r *MySQLListMoneyRepository) TermsByProjects(ctx context.Context, projectIDs []int64) (map[int64][]domain.PaymentTerm, error) {
	out := make(map[int64][]domain.PaymentTerm, len(projectIDs))
	if len(projectIDs) == 0 {
		return out, nil
	}
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT id, project_id, label, amount, due_date FROM client_payment_terms WHERE project_id IN (`+utils.Placeholders(len(projectIDs))+`) ORDER BY project_id, due_date, id`,
		utils.Int64Args(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t domain.PaymentTerm
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Label, &t.Amount, &t.DueDate); err != nil {
			return nil, err
		}
		out[t.ProjectID] = append(out[t.ProjectID], t)
	}
	return out, rows.Err()
}
