package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLIssueRepository struct {
	db *sql.DB
}

func NewMySQLIssueRepository(db *sql.DB) *MySQLIssueRepository {
	return &MySQLIssueRepository{db: db}
}

const issueColumns = `id, project_id, project_vendor_id, vendor_milestone_id, title, description, impact, found_date, status,
	resolution_plan, pic_staff_id, target_resolution_date, resolved_date, resolution_notes`

func scanIssue(scan func(dest ...interface{}) error) (*domain.VendorIssue, error) {
	var i domain.VendorIssue
	var impact, status string
	var vendorMilestoneID sql.NullInt64
	var resolutionPlan, resolutionNotes sql.NullString
	var targetResolutionDate, resolvedDate sql.NullTime
	err := scan(&i.ID, &i.ProjectID, &i.ProjectVendorID, &vendorMilestoneID, &i.Title, &i.Description, &impact, &i.FoundDate, &status,
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
	if vendorMilestoneID.Valid {
		i.VendorMilestoneID = &vendorMilestoneID.Int64
	}
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

// ListByProjects backs ComputeProgressBatch — one query across every id in
// projectIDs instead of one query per project (PLAN.md "Performance
// remediation").
func (r *MySQLIssueRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.VendorIssue, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + issueColumns + ` FROM vendor_issues WHERE project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, found_date DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
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
		`INSERT INTO vendor_issues (project_id, project_vendor_id, vendor_milestone_id, title, description, impact, found_date, status,
		 resolution_plan, pic_staff_id, target_resolution_date)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		issue.ProjectID, issue.ProjectVendorID, issue.VendorMilestoneID, issue.Title, issue.Description, string(issue.Impact), issue.FoundDate,
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

// Update overwrites every editable field (PLAN.md "Retire the standalone
// Kendala tab") — previously this only ever wrote status/resolved_date/
// resolution_notes; a full overwrite is needed now that issues can be
// edited (vendor, milestone, impact, etc.), not just status-changed. The
// frontend's "quick status change" dropdown now calls this same method with
// every existing field spread plus the new status, mirroring how vendor
// milestone updates already work.
func (r *MySQLIssueRepository) Update(ctx context.Context, issue *domain.VendorIssue) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vendor_issues SET project_vendor_id = ?, vendor_milestone_id = ?, title = ?, description = ?, impact = ?,
		 found_date = ?, target_resolution_date = ?, pic_staff_id = ?, resolution_plan = ?, status = ?, resolved_date = ?,
		 resolution_notes = ? WHERE id = ?`,
		issue.ProjectVendorID, issue.VendorMilestoneID, issue.Title, issue.Description, string(issue.Impact),
		issue.FoundDate, issue.TargetResolutionDate, issue.PICStaffID, issue.ResolutionPlan, string(issue.Status),
		issue.ResolvedDate, issue.ResolutionNotes, issue.ID,
	)
	return err
}
