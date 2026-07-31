package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLMilestoneTemplateRepository struct {
	db *sql.DB
}

func NewMySQLMilestoneTemplateRepository(db *sql.DB) *MySQLMilestoneTemplateRepository {
	return &MySQLMilestoneTemplateRepository{db: db}
}

const milestoneTemplateColumns = `id, tenant_id, sort_order, name, days_before_event, created_at, updated_at`

func scanMilestoneTemplate(scan func(dest ...interface{}) error) (*domain.ProjectMilestoneTemplate, error) {
	var t domain.ProjectMilestoneTemplate
	err := scan(&t.ID, &t.TenantID, &t.SortOrder, &t.Name, &t.DaysBeforeEvent, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *MySQLMilestoneTemplateRepository) List(ctx context.Context, tenantID int64) ([]domain.ProjectMilestoneTemplate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+milestoneTemplateColumns+` FROM project_milestone_templates WHERE tenant_id = ? ORDER BY sort_order`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []domain.ProjectMilestoneTemplate
	for rows.Next() {
		t, err := scanMilestoneTemplate(rows.Scan)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	return templates, rows.Err()
}

func (r *MySQLMilestoneTemplateRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.ProjectMilestoneTemplate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+milestoneTemplateColumns+` FROM project_milestone_templates WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanMilestoneTemplate(row.Scan)
}

func (r *MySQLMilestoneTemplateRepository) Create(ctx context.Context, t *domain.ProjectMilestoneTemplate) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO project_milestone_templates (tenant_id, sort_order, name, days_before_event) VALUES (?, ?, ?, ?)`,
		t.TenantID, t.SortOrder, t.Name, t.DaysBeforeEvent,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	return nil
}

func (r *MySQLMilestoneTemplateRepository) Update(ctx context.Context, t *domain.ProjectMilestoneTemplate) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE project_milestone_templates SET name = ?, days_before_event = ? WHERE tenant_id = ? AND id = ?`,
		t.Name, t.DaysBeforeEvent, t.TenantID, t.ID,
	)
	return err
}

func (r *MySQLMilestoneTemplateRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_milestone_templates WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

func (r *MySQLMilestoneTemplateRepository) NextSortOrder(ctx context.Context, tenantID int64) (int, error) {
	var maxOrder sql.NullInt64
	row := r.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM project_milestone_templates WHERE tenant_id = ?`, tenantID)
	if err := row.Scan(&maxOrder); err != nil {
		return 0, err
	}
	return int(maxOrder.Int64) + 1, nil
}

// Reorder rewrites sort_order to match each ID's position in orderedIDs
// (1-based) -- the application layer has already validated orderedIDs is an
// exact permutation of this tenant's template IDs.
func (r *MySQLMilestoneTemplateRepository) Reorder(ctx context.Context, tenantID int64, orderedIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range orderedIDs {
		if _, err := tx.ExecContext(ctx,
			`UPDATE project_milestone_templates SET sort_order = ? WHERE id = ? AND tenant_id = ?`,
			i+1, id, tenantID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
