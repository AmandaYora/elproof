package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLVendorEngagementRepository struct {
	db *sql.DB
}

func NewMySQLVendorEngagementRepository(db *sql.DB) *MySQLVendorEngagementRepository {
	return &MySQLVendorEngagementRepository{db: db}
}

const projectVendorColumns = `id, project_id, vendor_id, category_id, scope, contract_value, engagement_status,
	booking_date, event_date, dp_amount, paid_amount, due_date, pic_staff_id, notes`

func scanProjectVendor(scan func(dest ...interface{}) error) (*domain.ProjectVendor, error) {
	var pv domain.ProjectVendor
	var status string
	var bookingDate, dueDate sql.NullTime
	var notes sql.NullString
	err := scan(&pv.ID, &pv.ProjectID, &pv.VendorID, &pv.CategoryID, &pv.Scope, &pv.ContractValue, &status,
		&bookingDate, &pv.EventDate, &pv.DPAmount, &pv.PaidAmount, &dueDate, &pv.PICStaffID, &notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	pv.EngagementStatus = domain.EngagementStatus(status)
	if bookingDate.Valid {
		pv.BookingDate = &bookingDate.Time
	}
	if dueDate.Valid {
		pv.DueDate = &dueDate.Time
	}
	pv.Notes = notes.String
	return &pv, nil
}

func (r *MySQLVendorEngagementRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectVendor, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+projectVendorColumns+` FROM project_vendors WHERE project_id = ? ORDER BY id`, projectID)
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
	row := r.db.QueryRowContext(ctx, `SELECT `+projectVendorColumns+` FROM project_vendors WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanProjectVendor(row.Scan)
}

func (r *MySQLVendorEngagementRepository) Create(ctx context.Context, pv *domain.ProjectVendor) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO project_vendors (project_id, vendor_id, category_id, scope, contract_value, engagement_status,
		 booking_date, event_date, dp_amount, paid_amount, due_date, pic_staff_id, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		pv.ProjectID, pv.VendorID, pv.CategoryID, pv.Scope, pv.ContractValue, string(pv.EngagementStatus),
		pv.BookingDate, pv.EventDate, pv.DPAmount, pv.PaidAmount, pv.DueDate, pv.PICStaffID, pv.Notes,
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

func (r *MySQLVendorEngagementRepository) Update(ctx context.Context, pv *domain.ProjectVendor) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE project_vendors SET vendor_id = ?, category_id = ?, scope = ?, contract_value = ?, engagement_status = ?,
		 booking_date = ?, event_date = ?, dp_amount = ?, paid_amount = ?, due_date = ?, pic_staff_id = ?, notes = ?
		 WHERE id = ?`,
		pv.VendorID, pv.CategoryID, pv.Scope, pv.ContractValue, string(pv.EngagementStatus),
		pv.BookingDate, pv.EventDate, pv.DPAmount, pv.PaidAmount, pv.DueDate, pv.PICStaffID, pv.Notes, pv.ID,
	)
	return err
}

func (r *MySQLVendorEngagementRepository) SetStatus(ctx context.Context, projectID, id int64, status domain.EngagementStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE project_vendors SET engagement_status = ? WHERE project_id = ? AND id = ?`, string(status), projectID, id)
	return err
}

// ListByVendor backs the `vendors` module's "Lihat Project" history — a
// same-module join (project_vendors + projects, both owned by `projects`),
// never exposed as a cross-module join; `vendors` only ever sees the
// resulting rows through `contracts.Contracts.ListVendorEngagementHistory`.
func (r *MySQLVendorEngagementRepository) ListByVendor(ctx context.Context, tenantID, vendorID int64) ([]domain.VendorEngagementHistoryRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pv.project_id, p.name, p.event_date, p.venue, pv.engagement_status
		 FROM project_vendors pv
		 JOIN projects p ON p.id = pv.project_id
		 WHERE pv.vendor_id = ? AND p.tenant_id = ?
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

const vendorMilestoneColumns = `id, project_vendor_id, sort_order, name, description, status, target_date, completed_date, pic_staff_id, notes`

func scanVendorMilestone(scan func(dest ...interface{}) error) (*domain.VendorMilestone, error) {
	var m domain.VendorMilestone
	var status string
	var description, notes sql.NullString
	var completedDate sql.NullTime
	err := scan(&m.ID, &m.ProjectVendorID, &m.SortOrder, &m.Name, &description, &status, &m.TargetDate, &completedDate, &m.PICStaffID, &notes)
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
	rows, err := r.db.QueryContext(ctx, `SELECT `+vendorMilestoneColumns+` FROM vendor_milestones WHERE project_vendor_id = ? ORDER BY sort_order`, projectVendorID)
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
	row := r.db.QueryRowContext(ctx, `SELECT `+vendorMilestoneColumns+` FROM vendor_milestones WHERE project_vendor_id = ? AND id = ? LIMIT 1`, projectVendorID, id)
	return scanVendorMilestone(row.Scan)
}

func (r *MySQLVendorMilestoneRepository) Create(ctx context.Context, m *domain.VendorMilestone) error {
	result, err := r.db.ExecContext(ctx,
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

func (r *MySQLVendorMilestoneRepository) Update(ctx context.Context, m *domain.VendorMilestone) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vendor_milestones SET name = ?, description = ?, status = ?, target_date = ?, completed_date = ?,
		 pic_staff_id = ?, notes = ? WHERE id = ?`,
		m.Name, m.Description, string(m.Status), m.TargetDate, m.CompletedDate, m.PICStaffID, m.Notes, m.ID,
	)
	return err
}

func (r *MySQLVendorMilestoneRepository) NextSortOrder(ctx context.Context, projectVendorID int64) (int, error) {
	var maxOrder sql.NullInt64
	row := r.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM vendor_milestones WHERE project_vendor_id = ?`, projectVendorID)
	if err := row.Scan(&maxOrder); err != nil {
		return 0, err
	}
	return int(maxOrder.Int64) + 1, nil
}
