package infrastructure

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

// ListOpenProjectTerms returns every payment-schedule installment of the
// workspace's open projects (not completed, cancelled or archived).
func (r *MySQLDashboardRepository) ListOpenProjectTerms(ctx context.Context, tenantID int64) ([]domain.TermWithProject, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.project_id, t.label, t.amount, t.due_date, p.name
		FROM client_payment_terms t
		JOIN projects p ON p.id = t.project_id
		WHERE p.tenant_id = ? AND p.is_archived = 0 AND p.status NOT IN ('Completed','Cancelled')
		ORDER BY t.project_id, t.due_date, t.id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.TermWithProject
	for rows.Next() {
		var row domain.TermWithProject
		if err := rows.Scan(&row.Term.ID, &row.Term.ProjectID, &row.Term.Label, &row.Term.Amount, &row.Term.DueDate, &row.ProjectName); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// ListAppliedToContract returns, per open project, what the client has paid
// against the contract (DP + Termin + Pelunasan - Refund; Tambahan excluded,
// same rule as domain.AppliedToContract).
func (r *MySQLDashboardRepository) ListAppliedToContract(ctx context.Context, tenantID int64) (map[int64]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT cp.project_id,
		       COALESCE(SUM(CASE WHEN cp.type IN ('DP','Termin','Pelunasan') THEN cp.amount
		                         WHEN cp.type = 'Refund' THEN -CAST(cp.amount AS SIGNED) ELSE 0 END), 0)
		FROM client_payments cp
		JOIN projects p ON p.id = cp.project_id
		WHERE p.tenant_id = ? AND p.is_archived = 0 AND p.status NOT IN ('Completed','Cancelled')
		GROUP BY cp.project_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := make(map[int64]int64)
	for rows.Next() {
		var projectID, total int64
		if err := rows.Scan(&projectID, &total); err != nil {
			return nil, err
		}
		applied[projectID] = total
	}
	return applied, rows.Err()
}

// ListProjectMilestonesDue returns open projects' Timeline items whose target
// date falls from `from` to `to` (inclusive) and that are not finished.
func (r *MySQLDashboardRepository) ListProjectMilestonesDue(ctx context.Context, tenantID int64, from, to time.Time) ([]domain.DashboardTaskRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.project_id, p.name, m.name, m.target_date
		FROM project_milestones m
		JOIN projects p ON p.id = m.project_id
		WHERE p.tenant_id = ? AND p.is_archived = 0 AND p.status NOT IN ('Completed','Cancelled')
		  AND m.status NOT IN ('Completed','Cancelled') AND m.target_date BETWEEN ? AND ?
		ORDER BY m.target_date, m.id`, tenantID, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.DashboardTaskRow
	for rows.Next() {
		var row domain.DashboardTaskRow
		if err := rows.Scan(&row.ProjectID, &row.ProjectName, &row.Name, &row.TargetDate); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}
