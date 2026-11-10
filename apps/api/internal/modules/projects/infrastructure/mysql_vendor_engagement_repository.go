package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLVendorEngagementRepository struct {
	db *sql.DB
}

func NewMySQLVendorEngagementRepository(db *sql.DB) *MySQLVendorEngagementRepository {
	return &MySQLVendorEngagementRepository{db: db}
}

const projectVendorColumns = `id, project_id, vendor_id, category_id, scope, contract_value, pricing_tier, engagement_status,
	booking_date, event_date, event_start_time, event_end_time, dp_amount, due_date, pic_staff_id, notes, deleted_at, updated_at`

// timeToHHMM/hhmmToTime convert between MySQL's TIME wire format
// ("15:04:05") and the "HH:MM" shape ProjectVendor.EventStartTime/
// EventEndTime carry at the domain/API boundary -- mirrors how every other
// nullable field in this file converts at the scan/exec boundary rather than
// leaking a driver-specific shape upward.
func timeToHHMM(s string) string {
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}

func scanProjectVendor(scan func(dest ...interface{}) error) (*domain.ProjectVendor, error) {
	var pv domain.ProjectVendor
	var pricingTier, status string
	var bookingDate, dueDate sql.NullTime
	var eventStartTime, eventEndTime sql.NullString
	var notes sql.NullString
	var deletedAt sql.NullTime
	err := scan(&pv.ID, &pv.ProjectID, &pv.VendorID, &pv.CategoryID, &pv.Scope, &pv.ContractValue, &pricingTier, &status,
		&bookingDate, &pv.EventDate, &eventStartTime, &eventEndTime, &pv.DPAmount, &dueDate, &pv.PICStaffID, &notes, &deletedAt, &pv.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	pv.PricingTier = domain.VendorPricingTier(pricingTier)
	pv.EngagementStatus = domain.EngagementStatus(status)
	if bookingDate.Valid {
		pv.BookingDate = &bookingDate.Time
	}
	if dueDate.Valid {
		pv.DueDate = &dueDate.Time
	}
	if eventStartTime.Valid {
		v := timeToHHMM(eventStartTime.String)
		pv.EventStartTime = &v
	}
	if eventEndTime.Valid {
		v := timeToHHMM(eventEndTime.String)
		pv.EventEndTime = &v
	}
	pv.Notes = notes.String
	pv.DeletedAt = deletedStamp(deletedAt)
	return &pv, nil
}

func (r *MySQLVendorEngagementRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectVendor, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+projectVendorColumns+` FROM project_vendors WHERE project_id = ? AND deleted_at IS NULL ORDER BY id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.ProjectVendor
	for rows.Next() {
		pv, err := scanProjectVendor(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *pv)
	}
	return list, rows.Err()
}

// ListByProjects backs ComputeProgressBatch — one query across every id in
// projectIDs instead of one query per project (PLAN.md "Performance
// remediation").
func (r *MySQLVendorEngagementRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectVendor, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + projectVendorColumns + ` FROM project_vendors WHERE deleted_at IS NULL AND project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, id`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.ProjectVendor
	for rows.Next() {
		pv, err := scanProjectVendor(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *pv)
	}
	return list, rows.Err()
}

func (r *MySQLVendorEngagementRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectVendor, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+projectVendorColumns+` FROM project_vendors WHERE project_id = ? AND id = ? AND deleted_at IS NULL LIMIT 1`, projectID, id)
	return scanProjectVendor(row.Scan)
}

// FindDeletedByID is the restore lookup: the engagement while it waits in the
// "deleted" state (nil when it is live, never existed or is another project's).
func (r *MySQLVendorEngagementRepository) FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.ProjectVendor, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+projectVendorColumns+` FROM project_vendors WHERE project_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, projectID, id)
	return scanProjectVendor(row.Scan)
}

// SoftDelete hides the engagement (and with it its milestones, payments and
// kendala) until it is restored or purged.
func (r *MySQLVendorEngagementRepository) SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error) {
	return softDeleteRow(ctx, r.db, "project_vendors", projectID, id, at)
}

// Restore brings back an engagement deleted after notBefore.
func (r *MySQLVendorEngagementRepository) Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error) {
	return restoreRow(ctx, r.db, "project_vendors", projectID, id, notBefore)
}

func (r *MySQLVendorEngagementRepository) Create(ctx context.Context, pv *domain.ProjectVendor) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO project_vendors (project_id, vendor_id, category_id, scope, contract_value, pricing_tier, engagement_status,
		 booking_date, event_date, event_start_time, event_end_time, dp_amount, due_date, pic_staff_id, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		pv.ProjectID, pv.VendorID, pv.CategoryID, pv.Scope, pv.ContractValue, string(pv.PricingTier), string(pv.EngagementStatus),
		pv.BookingDate, pv.EventDate, pv.EventStartTime, pv.EventEndTime, pv.DPAmount, pv.DueDate, pv.PICStaffID, pv.Notes,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	pv.ID = id
	return nil
}

var projectVendorPatchColumns = map[string]column[domain.ProjectVendor]{
	"vendorId":         col("vendor_id", func(p *domain.ProjectVendor) any { return p.VendorID }),
	"categoryId":       col("category_id", func(p *domain.ProjectVendor) any { return p.CategoryID }),
	"scope":            col("scope", func(p *domain.ProjectVendor) any { return p.Scope }),
	"contractValue":    col("contract_value", func(p *domain.ProjectVendor) any { return p.ContractValue }),
	"pricingTier":      col("pricing_tier", func(p *domain.ProjectVendor) any { return string(p.PricingTier) }),
	"engagementStatus": col("engagement_status", func(p *domain.ProjectVendor) any { return string(p.EngagementStatus) }),
	"bookingDate":      col("booking_date", func(p *domain.ProjectVendor) any { return p.BookingDate }),
	"eventDate":        col("event_date", func(p *domain.ProjectVendor) any { return p.EventDate }),
	"eventStartTime":   col("event_start_time", func(p *domain.ProjectVendor) any { return p.EventStartTime }),
	"eventEndTime":     col("event_end_time", func(p *domain.ProjectVendor) any { return p.EventEndTime }),
	"dpAmount":         col("dp_amount", func(p *domain.ProjectVendor) any { return p.DPAmount }),
	"dueDate":          col("due_date", func(p *domain.ProjectVendor) any { return p.DueDate }),
	"picStaffId":       col("pic_staff_id", func(p *domain.ProjectVendor) any { return p.PICStaffID }),
	"notes":            col("notes", func(p *domain.ProjectVendor) any { return p.Notes }),
}

// Patch writes the changed fields of a live engagement of the project (see patchRow).
func (r *MySQLVendorEngagementRepository) Patch(ctx context.Context, pv *domain.ProjectVendor, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "project_vendors", projectVendorPatchColumns, pv, changed,
		"project_id = ? AND id = ? AND deleted_at IS NULL", []any{pv.ProjectID, pv.ID}, expected)
}

func (r *MySQLVendorEngagementRepository) SetStatus(ctx context.Context, projectID, id int64, status domain.EngagementStatus) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE project_vendors SET engagement_status = ? WHERE project_id = ? AND id = ?`, string(status), projectID, id)
	return err
}

// ListByVendor backs the `vendors` module's "Lihat Project" history — a
// same-module join (project_vendors + projects, both owned by `projects`),
// never exposed as a cross-module join; `vendors` only ever sees the
// resulting rows through `contracts.Contracts.ListVendorEngagementHistory`.
func (r *MySQLVendorEngagementRepository) ListByVendor(ctx context.Context, tenantID, vendorID int64) ([]domain.VendorEngagementHistoryRow, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT pv.project_id, p.name, p.event_date, p.venue, pv.engagement_status
		 FROM project_vendors pv
		 JOIN projects p ON p.id = pv.project_id
		 WHERE pv.vendor_id = ? AND p.tenant_id = ? AND pv.deleted_at IS NULL
		 ORDER BY p.event_date DESC`,
		vendorID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VendorEngagementHistoryRow
	for rows.Next() {
		var row domain.VendorEngagementHistoryRow
		var status string
		if err := rows.Scan(&row.ProjectID, &row.ProjectName, &row.EventDate, &row.Venue, &status); err != nil {
			return nil, err
		}
		row.EngagementStatus = domain.EngagementStatus(status)
		list = append(list, row)
	}
	return list, rows.Err()
}

// --- Vendor milestones ---

type MySQLVendorMilestoneRepository struct {
	db *sql.DB
}

func NewMySQLVendorMilestoneRepository(db *sql.DB) *MySQLVendorMilestoneRepository {
	return &MySQLVendorMilestoneRepository{db: db}
}

const vendorMilestoneColumns = `id, project_vendor_id, sort_order, name, description, status, target_date, completed_date, pic_staff_id, notes, updated_at`

func scanVendorMilestone(scan func(dest ...interface{}) error) (*domain.VendorMilestone, error) {
	var m domain.VendorMilestone
	var status string
	var description, notes sql.NullString
	var completedDate sql.NullTime
	err := scan(&m.ID, &m.ProjectVendorID, &m.SortOrder, &m.Name, &description, &status, &m.TargetDate, &completedDate, &m.PICStaffID, &notes, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Status = domain.MilestoneStatus(status)
	m.Description = description.String
	m.Notes = notes.String
	if completedDate.Valid {
		m.CompletedDate = &completedDate.Time
	}
	return &m, nil
}

func (r *MySQLVendorMilestoneRepository) ListByProjectVendor(ctx context.Context, projectVendorID int64) ([]domain.VendorMilestone, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+vendorMilestoneColumns+` FROM vendor_milestones WHERE project_vendor_id = ? ORDER BY sort_order`, projectVendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VendorMilestone
	for rows.Next() {
		m, err := scanVendorMilestone(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *m)
	}
	return list, rows.Err()
}

// ListByProjectVendors backs ComputeProgressBatch — one query across every
// project_vendor_id in the batch instead of one query per engagement
// (PLAN.md "Performance remediation").
func (r *MySQLVendorMilestoneRepository) ListByProjectVendors(ctx context.Context, projectVendorIDs []int64) ([]domain.VendorMilestone, error) {
	if len(projectVendorIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + vendorMilestoneColumns + ` FROM vendor_milestones WHERE project_vendor_id IN (` + utils.Placeholders(len(projectVendorIDs)) + `) ORDER BY project_vendor_id, sort_order`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, utils.Int64Args(projectVendorIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VendorMilestone
	for rows.Next() {
		m, err := scanVendorMilestone(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *m)
	}
	return list, rows.Err()
}

func (r *MySQLVendorMilestoneRepository) FindByID(ctx context.Context, projectVendorID, id int64) (*domain.VendorMilestone, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+vendorMilestoneColumns+` FROM vendor_milestones WHERE project_vendor_id = ? AND id = ? LIMIT 1`, projectVendorID, id)
	return scanVendorMilestone(row.Scan)
}

func (r *MySQLVendorMilestoneRepository) Create(ctx context.Context, m *domain.VendorMilestone) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO vendor_milestones (project_vendor_id, sort_order, name, description, status, target_date, pic_staff_id, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ProjectVendorID, m.SortOrder, m.Name, m.Description, string(m.Status), m.TargetDate, m.PICStaffID, m.Notes,
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

var vendorMilestonePatchColumns = map[string]column[domain.VendorMilestone]{
	"status":        col("status", func(m *domain.VendorMilestone) any { return string(m.Status) }),
	"targetDate":    col("target_date", func(m *domain.VendorMilestone) any { return m.TargetDate }),
	"completedDate": col("completed_date", func(m *domain.VendorMilestone) any { return m.CompletedDate }),
	"picStaffId":    col("pic_staff_id", func(m *domain.VendorMilestone) any { return m.PICStaffID }),
	"description":   col("description", func(m *domain.VendorMilestone) any { return m.Description }),
	"notes":         col("notes", func(m *domain.VendorMilestone) any { return m.Notes }),
}

// Patch writes the changed fields of a vendor milestone (see patchRow).
func (r *MySQLVendorMilestoneRepository) Patch(ctx context.Context, m *domain.VendorMilestone, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "vendor_milestones", vendorMilestonePatchColumns, m, changed,
		"project_vendor_id = ? AND id = ?", []any{m.ProjectVendorID, m.ID}, expected)
}

func (r *MySQLVendorMilestoneRepository) NextSortOrder(ctx context.Context, projectVendorID int64) (int, error) {
	var maxOrder sql.NullInt64
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT MAX(sort_order) FROM vendor_milestones WHERE project_vendor_id = ?`, projectVendorID)
	if err := row.Scan(&maxOrder); err != nil {
		return 0, err
	}
	return int(maxOrder.Int64) + 1, nil
}
