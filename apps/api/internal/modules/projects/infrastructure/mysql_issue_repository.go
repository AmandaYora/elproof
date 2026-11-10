package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLIssueRepository struct {
	db *sql.DB
}

func NewMySQLIssueRepository(db *sql.DB) *MySQLIssueRepository {
	return &MySQLIssueRepository{db: db}
}

const issueColumns = `id, project_id, IFNULL(project_vendor_id, 0), vendor_milestone_id, title, description, impact, found_date, status,
	resolution_plan, pic_staff_id, target_resolution_date, resolved_date, resolution_notes, is_client_visible, updated_at`

func scanIssue(scan func(dest ...interface{}) error) (*domain.VendorIssue, error) {
	var i domain.VendorIssue
	var impact, status string
	var vendorMilestoneID sql.NullInt64
	var resolutionPlan, resolutionNotes sql.NullString
	var targetResolutionDate, resolvedDate sql.NullTime
	err := scan(&i.ID, &i.ProjectID, &i.ProjectVendorID, &vendorMilestoneID, &i.Title, &i.Description, &impact, &i.FoundDate, &status,
		&resolutionPlan, &i.PICStaffID, &targetResolutionDate, &resolvedDate, &resolutionNotes, &i.IsClientVisible, &i.UpdatedAt)
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
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+issueColumns+` FROM vendor_issues WHERE project_id = ? AND `+liveIssueWhere+` ORDER BY found_date DESC, id DESC`, projectID)
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
	query := `SELECT ` + issueColumns + ` FROM vendor_issues WHERE ` + liveIssueWhere + ` AND project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, found_date DESC, id DESC`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
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
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+issueColumns+` FROM vendor_issues WHERE project_id = ? AND id = ? AND `+liveIssueWhere+` LIMIT 1`, projectID, id)
	return scanIssue(row.Scan)
}

func (r *MySQLIssueRepository) Create(ctx context.Context, issue *domain.VendorIssue) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO vendor_issues (project_id, project_vendor_id, vendor_milestone_id, title, description, impact, found_date, status,
		 resolution_plan, pic_staff_id, target_resolution_date, is_client_visible)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		issue.ProjectID, nullableID(issue.ProjectVendorID), issue.VendorMilestoneID, issue.Title, issue.Description, string(issue.Impact), issue.FoundDate,
		string(issue.Status), issue.ResolutionPlan, issue.PICStaffID, issue.TargetResolutionDate, issue.IsClientVisible,
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

var issuePatchColumns = map[string]column[domain.VendorIssue]{
	"projectVendorId":      col("project_vendor_id", func(i *domain.VendorIssue) any { return nullableID(i.ProjectVendorID) }),
	"vendorMilestoneId":    col("vendor_milestone_id", func(i *domain.VendorIssue) any { return i.VendorMilestoneID }),
	"title":                col("title", func(i *domain.VendorIssue) any { return i.Title }),
	"description":          col("description", func(i *domain.VendorIssue) any { return i.Description }),
	"impact":               col("impact", func(i *domain.VendorIssue) any { return string(i.Impact) }),
	"foundDate":            col("found_date", func(i *domain.VendorIssue) any { return i.FoundDate }),
	"status":               col("status", func(i *domain.VendorIssue) any { return string(i.Status) }),
	"resolutionPlan":       col("resolution_plan", func(i *domain.VendorIssue) any { return i.ResolutionPlan }),
	"picStaffId":           col("pic_staff_id", func(i *domain.VendorIssue) any { return i.PICStaffID }),
	"targetResolutionDate": col("target_resolution_date", func(i *domain.VendorIssue) any { return i.TargetResolutionDate }),
	"resolvedDate":         col("resolved_date", func(i *domain.VendorIssue) any { return i.ResolvedDate }),
	"isClientVisible":      col("is_client_visible", func(i *domain.VendorIssue) any { return i.IsClientVisible }),
}

// Patch writes the changed fields of a kendala of the project (see patchRow).
func (r *MySQLIssueRepository) Patch(ctx context.Context, issue *domain.VendorIssue, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "vendor_issues", issuePatchColumns, issue, changed,
		"project_id = ? AND id = ? AND "+liveIssueWhere, []any{issue.ProjectID, issue.ID}, expected)
}
