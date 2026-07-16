package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/pagination"
)

type MySQLProjectRepository struct {
	db *sql.DB
}

func NewMySQLProjectRepository(db *sql.DB) *MySQLProjectRepository {
	return &MySQLProjectRepository{db: db}
}

const projectColumns = `id, tenant_id, name, bride_name, groom_name, event_date, venue, prep_start_date,
	package_name, contract_value, status, pic_staff_id, description, created_at, updated_at`

func scanProject(scan func(dest ...interface{}) error) (*domain.Project, error) {
	var p domain.Project
	var status string
	var description sql.NullString
	err := scan(&p.ID, &p.TenantID, &p.Name, &p.BrideName, &p.GroomName, &p.EventDate, &p.Venue, &p.PrepStartDate,
		&p.PackageName, &p.ContractValue, &status, &p.PICStaffID, &description, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Status = domain.ProjectStatus(status)
	p.Description = description.String
	return &p, nil
}

func (r *MySQLProjectRepository) List(ctx context.Context, tenantID int64) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE tenant_id = ? ORDER BY event_date DESC, id DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProject(rows.Scan)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, rows.Err()
}

// ListPaginated backs the real `GET /projects` list page — List above stays
// as-is for dashboard/global-search consumers that need the full roster.
func (r *MySQLProjectRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, status string) ([]domain.Project, int64, error) {
	countQuery := `SELECT COUNT(*) FROM projects WHERE tenant_id = ?`
	listQuery := `SELECT ` + projectColumns + ` FROM projects WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	var conditions []string
	if search != "" {
		conditions = append(conditions, `(name LIKE ? OR bride_name LIKE ? OR groom_name LIKE ? OR venue LIKE ?)`)
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if status != "" {
		conditions = append(conditions, `status = ?`)
		args = append(args, status)
	}
	if len(conditions) > 0 {
		where := ` AND ` + strings.Join(conditions, " AND ")
		countQuery += where
		listQuery += where
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY event_date DESC, id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, listQuery, append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProject(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		projects = append(projects, *p)
	}
	return projects, total, rows.Err()
}

func (r *MySQLProjectRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanProject(row.Scan)
}

func (r *MySQLProjectRepository) Create(ctx context.Context, p *domain.Project) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (tenant_id, name, bride_name, groom_name, event_date, venue, prep_start_date,
		 package_name, contract_value, status, pic_staff_id, description)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.TenantID, p.Name, p.BrideName, p.GroomName, p.EventDate, p.Venue, p.PrepStartDate,
		p.PackageName, p.ContractValue, string(p.Status), p.PICStaffID, p.Description,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

func (r *MySQLProjectRepository) Update(ctx context.Context, p *domain.Project) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, bride_name = ?, groom_name = ?, event_date = ?, venue = ?, prep_start_date = ?,
		 package_name = ?, contract_value = ?, status = ?, pic_staff_id = ?, description = ? WHERE tenant_id = ? AND id = ?`,
		p.Name, p.BrideName, p.GroomName, p.EventDate, p.Venue, p.PrepStartDate,
		p.PackageName, p.ContractValue, string(p.Status), p.PICStaffID, p.Description, p.TenantID, p.ID,
	)
	return err
}

func (r *MySQLProjectRepository) SetStatus(ctx context.Context, tenantID, id int64, status domain.ProjectStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE projects SET status = ? WHERE tenant_id = ? AND id = ?`, string(status), tenantID, id)
	return err
}

// --- Project milestones ---

type MySQLMilestoneRepository struct {
	db *sql.DB
}

func NewMySQLMilestoneRepository(db *sql.DB) *MySQLMilestoneRepository {
	return &MySQLMilestoneRepository{db: db}
}

const milestoneColumns = `id, project_id, sort_order, name, status, target_date, completed_date`

func scanMilestone(scan func(dest ...interface{}) error) (*domain.ProjectMilestone, error) {
	var m domain.ProjectMilestone
	var status string
	var completedDate sql.NullTime
	err := scan(&m.ID, &m.ProjectID, &m.SortOrder, &m.Name, &status, &m.TargetDate, &completedDate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Status = domain.MilestoneStatus(status)
	if completedDate.Valid {
		m.CompletedDate = &completedDate.Time
	}
	return &m, nil
}

func (r *MySQLMilestoneRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectMilestone, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+milestoneColumns+` FROM project_milestones WHERE project_id = ? ORDER BY sort_order`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var milestones []domain.ProjectMilestone
	for rows.Next() {
		m, err := scanMilestone(rows.Scan)
		if err != nil {
			return nil, err
		}
		milestones = append(milestones, *m)
	}
	return milestones, rows.Err()
}

func (r *MySQLMilestoneRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectMilestone, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM project_milestones WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanMilestone(row.Scan)
}

func (r *MySQLMilestoneRepository) Create(ctx context.Context, m *domain.ProjectMilestone) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO project_milestones (project_id, sort_order, name, status, target_date) VALUES (?, ?, ?, ?, ?)`,
		m.ProjectID, m.SortOrder, m.Name, string(m.Status), m.TargetDate,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

func (r *MySQLMilestoneRepository) Update(ctx context.Context, m *domain.ProjectMilestone) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE project_milestones SET name = ?, status = ?, target_date = ?, completed_date = ? WHERE id = ?`,
		m.Name, string(m.Status), m.TargetDate, m.CompletedDate, m.ID,
	)
	return err
}

func (r *MySQLMilestoneRepository) NextSortOrder(ctx context.Context, projectID int64) (int, error) {
	var maxOrder sql.NullInt64
	row := r.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM project_milestones WHERE project_id = ?`, projectID)
	if err := row.Scan(&maxOrder); err != nil {
		return 0, err
	}
	return int(maxOrder.Int64) + 1, nil
}
