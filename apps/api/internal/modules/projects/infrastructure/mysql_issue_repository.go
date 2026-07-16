package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLIssueRepository struct {
	db *sql.DB
}

func NewMySQLIssueRepository(db *sql.DB) *MySQLIssueRepository {
	return &MySQLIssueRepository{db: db}
}

const issueColumns = `id, project_id, project_vendor_id, title, description, impact, found_date, status,
	resolution_plan, pic_staff_id, target_resolution_date, resolved_date, resolution_notes`

func scanIssue(scan func(dest ...interface{}) error) (*domain.VendorIssue, error) {
	var i domain.VendorIssue
	var impact, status string
	var resolutionPlan, resolutionNotes sql.NullString
	var targetResolutionDate, resolvedDate sql.NullTime
	err := scan(&i.ID, &i.ProjectID, &i.ProjectVendorID, &i.Title, &i.Description, &impact, &i.FoundDate, &status,
		&resolutionPlan, &i.PICStaffID, &targetResolutionDate, &resolvedDate, &resolutionNotes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	i.Impact = domain.IssueImpact(impact)
	i.Status = domain.IssueStatus(status)
	i.ResolutionPlan = resolutionPlan.String
	i.ResolutionNotes = resolutionNotes.String
	if targetResolutionDate.Valid {
		i.TargetResolutionDate = &targetResolutionDate.Time
	}
	if resolvedDate.Valid {
		i.ResolvedDate = &resolvedDate.Time
	}
	return &i, nil
}

func (r *MySQLIssueRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.VendorIssue, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+issueColumns+` FROM vendor_issues WHERE project_id = ? ORDER BY found_date DESC, id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VendorIssue
	for rows.Next() {
		i, err := scanIssue(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *i)
	}
	return list, rows.Err()
}

func (r *MySQLIssueRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.VendorIssue, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+issueColumns+` FROM vendor_issues WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanIssue(row.Scan)
}

func (r *MySQLIssueRepository) Create(ctx context.Context, issue *domain.VendorIssue) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO vendor_issues (project_id, project_vendor_id, title, description, impact, found_date, status,
		 resolution_plan, pic_staff_id, target_resolution_date)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		issue.ProjectID, issue.ProjectVendorID, issue.Title, issue.Description, string(issue.Impact), issue.FoundDate,
		string(issue.Status), issue.ResolutionPlan, issue.PICStaffID, issue.TargetResolutionDate,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	issue.ID = id
	return nil
}

func (r *MySQLIssueRepository) Update(ctx context.Context, issue *domain.VendorIssue) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vendor_issues SET status = ?, resolved_date = ?, resolution_notes = ? WHERE id = ?`,
		string(issue.Status), issue.ResolvedDate, issue.ResolutionNotes, issue.ID,
	)
	return err
}
