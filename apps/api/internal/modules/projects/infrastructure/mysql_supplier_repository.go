package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

// MySQLSupplierRepository stores project_suppliers: the vendors and venues of
// a project ("Vendor & venue").
type MySQLSupplierRepository struct {
	db *sql.DB
}

func NewMySQLSupplierRepository(db *sql.DB) *MySQLSupplierRepository {
	return &MySQLSupplierRepository{db: db}
}

const supplierColumns = `id, tenant_id, project_id, supplier_kind, supplier_id, supplier_name, IFNULL(category_id, 0), paid_by, show_name,
	capacity_standing, capacity_seated, agreed_cost, engagement_status, booking_date, due_date, IFNULL(pic_staff_id, 0), scope, notes,
	deleted_at, updated_at`

func scanSupplier(scan func(dest ...interface{}) error) (*domain.ProjectSupplier, error) {
	var s domain.ProjectSupplier
	var kind, paidBy, status string
	var standing, seated sql.NullInt64
	var cost sql.NullInt64
	var bookingDate, dueDate, deletedAt sql.NullTime
	var notes sql.NullString
	err := scan(&s.ID, &s.TenantID, &s.ProjectID, &kind, &s.SupplierID, &s.Name, &s.CategoryID, &paidBy, &s.ShowName,
		&standing, &seated, &cost, &status, &bookingDate, &dueDate, &s.PICStaffID, &s.Scope, &notes,
		&deletedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Kind = domain.SupplierKind(kind)
	s.PaidBy = domain.Payer(paidBy)
	s.EngagementStatus = domain.EngagementStatus(status)
	s.CapacityStanding = nullInt(standing)
	s.CapacitySeated = nullInt(seated)
	if cost.Valid {
		v := cost.Int64
		s.AgreedCost = &v
	}
	if bookingDate.Valid {
		s.BookingDate = &bookingDate.Time
	}
	if dueDate.Valid {
		s.DueDate = &dueDate.Time
	}
	s.Notes = notes.String
	s.DeletedAt = deletedStamp(deletedAt)
	return &s, nil
}

func nullInt(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func (r *MySQLSupplierRepository) list(ctx context.Context, query string, args ...any) ([]domain.ProjectSupplier, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.ProjectSupplier
	for rows.Next() {
		s, err := scanSupplier(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *s)
	}
	return list, rows.Err()
}

func (r *MySQLSupplierRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectSupplier, error) {
	return r.list(ctx, `SELECT `+supplierColumns+` FROM project_suppliers WHERE project_id = ? AND deleted_at IS NULL ORDER BY id`, projectID)
}

func (r *MySQLSupplierRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectSupplier, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	return r.list(ctx, `SELECT `+supplierColumns+` FROM project_suppliers WHERE deleted_at IS NULL AND project_id IN (`+
		utils.Placeholders(len(projectIDs))+`) ORDER BY project_id, id`, utils.Int64Args(projectIDs)...)
}

func (r *MySQLSupplierRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectSupplier, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+supplierColumns+` FROM project_suppliers WHERE project_id = ? AND id = ? AND deleted_at IS NULL LIMIT 1`, projectID, id)
	return scanSupplier(row.Scan)
}

func (r *MySQLSupplierRepository) FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.ProjectSupplier, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+supplierColumns+` FROM project_suppliers WHERE project_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, projectID, id)
	return scanSupplier(row.Scan)
}

func (r *MySQLSupplierRepository) FindLiveVenue(ctx context.Context, projectID, venueID int64) (*domain.ProjectSupplier, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+supplierColumns+` FROM project_suppliers
		WHERE project_id = ? AND supplier_kind = 'venue' AND supplier_id = ? AND deleted_at IS NULL ORDER BY id LIMIT 1`, projectID, venueID)
	return scanSupplier(row.Scan)
}

// Create inserts the supplier with the project's tenant (read in the same
// statement, so the row can never land in another workspace).
func (r *MySQLSupplierRepository) Create(ctx context.Context, s *domain.ProjectSupplier) error {
	q := conn(ctx, r.db)
	result, err := q.ExecContext(ctx,
		`INSERT INTO project_suppliers (tenant_id, project_id, supplier_kind, supplier_id, supplier_name, category_id, paid_by, show_name,
		 capacity_standing, capacity_seated, agreed_cost, engagement_status, booking_date, due_date, pic_staff_id, scope, notes)
		 SELECT tenant_id, id, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? FROM projects WHERE id = ?`,
		string(s.Kind), s.SupplierID, s.Name, nullableID(s.CategoryID), string(s.PaidBy), s.ShowName,
		s.CapacityStanding, s.CapacitySeated, s.AgreedCost, string(s.EngagementStatus), s.BookingDate, s.DueDate,
		nullableID(s.PICStaffID), s.Scope, s.Notes, s.ProjectID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = id
	return q.QueryRowContext(ctx, `SELECT tenant_id, updated_at FROM project_suppliers WHERE id = ?`, id).Scan(&s.TenantID, &s.UpdatedAt)
}

var supplierPatchColumns = map[string]column[domain.ProjectSupplier]{
	"paidBy":           col("paid_by", func(s *domain.ProjectSupplier) any { return string(s.PaidBy) }),
	"showName":         col("show_name", func(s *domain.ProjectSupplier) any { return s.ShowName }),
	"agreedCost":       col("agreed_cost", func(s *domain.ProjectSupplier) any { return s.AgreedCost }),
	"capacityStanding": col("capacity_standing", func(s *domain.ProjectSupplier) any { return s.CapacityStanding }),
	"capacitySeated":   col("capacity_seated", func(s *domain.ProjectSupplier) any { return s.CapacitySeated }),
	"engagementStatus": col("engagement_status", func(s *domain.ProjectSupplier) any { return string(s.EngagementStatus) }),
	"bookingDate":      col("booking_date", func(s *domain.ProjectSupplier) any { return s.BookingDate }),
	"dueDate":          col("due_date", func(s *domain.ProjectSupplier) any { return s.DueDate }),
	"picStaffId":       col("pic_staff_id", func(s *domain.ProjectSupplier) any { return nullableID(s.PICStaffID) }),
	"scope":            col("scope", func(s *domain.ProjectSupplier) any { return s.Scope }),
	"notes":            col("notes", func(s *domain.ProjectSupplier) any { return s.Notes }),
}

// Patch writes the changed fields of a live supplier of the project (see patchRow).
func (r *MySQLSupplierRepository) Patch(ctx context.Context, s *domain.ProjectSupplier, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "project_suppliers", supplierPatchColumns, s, changed,
		"project_id = ? AND id = ? AND deleted_at IS NULL", []any{s.ProjectID, s.ID}, expected)
}

func (r *MySQLSupplierRepository) SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error) {
	return softDeleteRow(ctx, r.db, "project_suppliers", projectID, id, at)
}

func (r *MySQLSupplierRepository) Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error) {
	return restoreRow(ctx, r.db, "project_suppliers", projectID, id, notBefore)
}

// HardDelete removes a supplier nothing hangs off: a venue an event chose and
// then let go of before anything was recorded against it.
func (r *MySQLSupplierRepository) HardDelete(ctx context.Context, projectID, id int64) error {
	// Bills of it still waiting in the "deleted" state go with it (a live bill
	// keeps the supplier from being removed: SupplierAttachments.Empty).
	if _, err := conn(ctx, r.db).ExecContext(ctx,
		`DELETE FROM supplier_bill_terms WHERE project_id = ? AND project_supplier_id = ? AND deleted_at IS NOT NULL`, projectID, id); err != nil {
		return err
	}
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM project_suppliers WHERE project_id = ? AND id = ?`, projectID, id)
	return err
}

// attachmentCountsSQL counts, per supplier of a project, what hangs off it.
// Files are the live evidence of the supplier itself, of its payments and of
// its steps.
const attachmentCountsSQL = `
	SELECT s.id,
		(SELECT COUNT(*) FROM supplier_payments sp WHERE sp.project_supplier_id = s.id AND sp.deleted_at IS NULL),
		(SELECT COUNT(*) FROM evidence e WHERE e.project_id = s.project_id AND (
			(e.related_kind = 'projectSupplier' AND e.related_id = s.id)
			OR (e.related_kind = 'supplierPayment' AND e.related_id IN
				(SELECT sp.id FROM supplier_payments sp WHERE sp.project_supplier_id = s.id AND sp.deleted_at IS NULL))
			OR (e.related_kind = 'vendorMilestone' AND e.related_id IN
				(SELECT vm.id FROM vendor_milestones vm WHERE vm.project_supplier_id = s.id)))),
		(SELECT COUNT(*) FROM vendor_milestones vm WHERE vm.project_supplier_id = s.id),
		(SELECT COUNT(*) FROM vendor_issues vi WHERE vi.project_supplier_id = s.id),
		(SELECT COUNT(*) FROM supplier_bill_terms bt WHERE bt.project_supplier_id = s.id AND bt.deleted_at IS NULL)
	FROM project_suppliers s
	WHERE s.project_id = ? AND s.deleted_at IS NULL`

func (r *MySQLSupplierRepository) Attachments(ctx context.Context, projectID, id int64) (domain.SupplierAttachments, error) {
	var a domain.SupplierAttachments
	var sid int64
	err := conn(ctx, r.db).QueryRowContext(ctx, attachmentCountsSQL+` AND s.id = ?`, projectID, id).
		Scan(&sid, &a.Payments, &a.Files, &a.Milestones, &a.Issues, &a.BillTerms)
	if err == sql.ErrNoRows {
		return a, nil
	}
	return a, err
}

func (r *MySQLSupplierRepository) AttachmentsByProject(ctx context.Context, projectID int64) (map[int64]domain.SupplierAttachments, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, attachmentCountsSQL, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]domain.SupplierAttachments{}
	for rows.Next() {
		var id int64
		var a domain.SupplierAttachments
		if err := rows.Scan(&id, &a.Payments, &a.Files, &a.Milestones, &a.Issues, &a.BillTerms); err != nil {
			return nil, err
		}
		out[id] = a
	}
	return out, rows.Err()
}

// ListBySupplier backs the directory's "Lihat proyek" — a same-module join
// (project_suppliers + projects, both owned by `projects`); `vendors` only sees
// the rows through the contract.
func (r *MySQLSupplierRepository) ListBySupplier(ctx context.Context, tenantID int64, kind domain.SupplierKind, supplierID int64) ([]domain.SupplierHistoryRow, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT s.project_id, p.name, p.event_date, s.engagement_status
		 FROM project_suppliers s
		 JOIN projects p ON p.id = s.project_id
		 WHERE s.tenant_id = ? AND s.supplier_kind = ? AND s.supplier_id = ? AND s.deleted_at IS NULL
		 ORDER BY p.event_date DESC, s.project_id DESC`,
		tenantID, string(kind), supplierID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.SupplierHistoryRow
	for rows.Next() {
		var row domain.SupplierHistoryRow
		var status string
		if err := rows.Scan(&row.ProjectID, &row.ProjectName, &row.EventDate, &status); err != nil {
			return nil, err
		}
		row.EngagementStatus = domain.EngagementStatus(status)
		list = append(list, row)
	}
	return list, rows.Err()
}

// --- Steps of a supplier ---

type MySQLVendorMilestoneRepository struct {
	db *sql.DB
}

func NewMySQLVendorMilestoneRepository(db *sql.DB) *MySQLVendorMilestoneRepository {
	return &MySQLVendorMilestoneRepository{db: db}
}

const vendorMilestoneColumns = `id, project_supplier_id, sort_order, name, description, status, target_date, completed_date, pic_staff_id, notes, updated_at`

func scanVendorMilestone(scan func(dest ...interface{}) error) (*domain.VendorMilestone, error) {
	var m domain.VendorMilestone
	var status string
	var description, notes sql.NullString
	var completedDate sql.NullTime
	err := scan(&m.ID, &m.ProjectSupplierID, &m.SortOrder, &m.Name, &description, &status, &m.TargetDate, &completedDate, &m.PICStaffID, &notes, &m.UpdatedAt)
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

func (r *MySQLVendorMilestoneRepository) list(ctx context.Context, query string, args ...any) ([]domain.VendorMilestone, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, args...)
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

func (r *MySQLVendorMilestoneRepository) ListBySupplier(ctx context.Context, projectSupplierID int64) ([]domain.VendorMilestone, error) {
	return r.list(ctx, `SELECT `+vendorMilestoneColumns+` FROM vendor_milestones WHERE project_supplier_id = ? ORDER BY sort_order, id`, projectSupplierID)
}

func (r *MySQLVendorMilestoneRepository) ListBySuppliers(ctx context.Context, ids []int64) ([]domain.VendorMilestone, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return r.list(ctx, `SELECT `+vendorMilestoneColumns+` FROM vendor_milestones WHERE project_supplier_id IN (`+
		utils.Placeholders(len(ids))+`) ORDER BY project_supplier_id, sort_order, id`, utils.Int64Args(ids)...)
}

func (r *MySQLVendorMilestoneRepository) FindByID(ctx context.Context, projectSupplierID, id int64) (*domain.VendorMilestone, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+vendorMilestoneColumns+` FROM vendor_milestones WHERE project_supplier_id = ? AND id = ? LIMIT 1`, projectSupplierID, id)
	return scanVendorMilestone(row.Scan)
}

// Create inserts the step with its supplier's tenant.
func (r *MySQLVendorMilestoneRepository) Create(ctx context.Context, m *domain.VendorMilestone) error {
	q := conn(ctx, r.db)
	result, err := q.ExecContext(ctx,
		`INSERT INTO vendor_milestones (tenant_id, project_supplier_id, sort_order, name, description, status, target_date, pic_staff_id, notes)
		 SELECT tenant_id, id, ?, ?, ?, ?, ?, ?, ? FROM project_suppliers WHERE id = ?`,
		m.SortOrder, m.Name, m.Description, string(m.Status), m.TargetDate, m.PICStaffID, m.Notes, m.ProjectSupplierID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return readVersion(ctx, q, "vendor_milestones", id, &m.UpdatedAt)
}

var vendorMilestonePatchColumns = map[string]column[domain.VendorMilestone]{
	"status":        col("status", func(m *domain.VendorMilestone) any { return string(m.Status) }),
	"targetDate":    col("target_date", func(m *domain.VendorMilestone) any { return m.TargetDate }),
	"completedDate": col("completed_date", func(m *domain.VendorMilestone) any { return m.CompletedDate }),
	"picStaffId":    col("pic_staff_id", func(m *domain.VendorMilestone) any { return m.PICStaffID }),
	"description":   col("description", func(m *domain.VendorMilestone) any { return m.Description }),
	"notes":         col("notes", func(m *domain.VendorMilestone) any { return m.Notes }),
}

func (r *MySQLVendorMilestoneRepository) Patch(ctx context.Context, m *domain.VendorMilestone, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "vendor_milestones", vendorMilestonePatchColumns, m, changed,
		"project_supplier_id = ? AND id = ?", []any{m.ProjectSupplierID, m.ID}, expected)
}

func (r *MySQLVendorMilestoneRepository) NextSortOrder(ctx context.Context, projectSupplierID int64) (int, error) {
	var maxOrder sql.NullInt64
	if err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT MAX(sort_order) FROM vendor_milestones WHERE project_supplier_id = ?`, projectSupplierID).Scan(&maxOrder); err != nil {
		return 0, err
	}
	return int(maxOrder.Int64) + 1, nil
}
