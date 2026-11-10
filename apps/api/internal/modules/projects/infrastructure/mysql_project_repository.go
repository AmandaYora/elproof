package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

type MySQLProjectRepository struct {
	db *sql.DB
}

func NewMySQLProjectRepository(db *sql.DB) *MySQLProjectRepository {
	return &MySQLProjectRepository{db: db}
}

const projectColumns = `id, tenant_id, name, bride_name, groom_name, event_date, venue, venue_id, venue_rental_price, venue_charge,
	prep_start_date, package_name, contract_value, status, pic_staff_id, pic_sales_staff_id, description, is_archived, created_at, updated_at`

func scanProject(scan func(dest ...interface{}) error) (*domain.Project, error) {
	var p domain.Project
	var status string
	var description sql.NullString
	var venueID sql.NullInt64
	var venueRentalPrice, venueCharge sql.NullInt64
	err := scan(&p.ID, &p.TenantID, &p.Name, &p.BrideName, &p.GroomName, &p.EventDate, &p.Venue, &venueID, &venueRentalPrice, &venueCharge,
		&p.PrepStartDate, &p.PackageName, &p.ContractValue, &status, &p.PICStaffID, &p.PICSalesStaffID, &description, &p.IsArchived, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Status = domain.ProjectStatus(status)
	p.Description = description.String
	if venueID.Valid {
		p.VenueID = &venueID.Int64
	}
	if venueRentalPrice.Valid {
		p.VenueRentalPrice = &venueRentalPrice.Int64
	}
	if venueCharge.Valid {
		p.VenueCharge = &venueCharge.Int64
	}
	return &p, nil
}

// picStaffID, when non-nil, scopes the result to a single PIC's own projects
// — the Wedding Planner ("Staff" role) role restriction; picSalesStaffID,
// when non-nil, scopes it to a single Sales staff member's own projects
// instead (mutually exclusive with picStaffID in practice -- a caller is
// either role). nil/nil means every project in the tenant (Owner/Admin, and
// internal callers like Dashboard).
func (r *MySQLProjectRepository) List(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64) ([]domain.Project, error) {
	query := `SELECT ` + projectColumns + ` FROM projects WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if cond, condArgs := picFilter(picStaffID, picSalesStaffID); cond != "" {
		query += cond
		args = append(args, condArgs...)
	}
	query += ` ORDER BY event_date DESC, id DESC`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, args...)
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

// ListByVenueID backs the hard-delete "impact" endpoint for Venue (PLAN.md) —
// every project this venue is currently attached to (projects.venue_id has no
// SQL FK, cross-module by design, so this is the one place that can answer
// "what still points at this venue").
func (r *MySQLProjectRepository) ListByVenueID(ctx context.Context, tenantID, venueID int64) ([]domain.ProjectRef, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT id, name FROM projects WHERE tenant_id = ? AND venue_id = ?`, tenantID, venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []domain.ProjectRef
	for rows.Next() {
		var ref domain.ProjectRef
		if err := rows.Scan(&ref.ID, &ref.Name); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// ListByStaffPIC backs the hard-delete "impact" endpoint for Staff (PLAN.md)
// — every project this staff member is a live PIC assignment on, across the
// four columns that represent an active "responsible person" role
// (projects.pic_staff_id, project_vendors.pic_staff_id,
// vendor_milestones.pic_staff_id, vendor_issues.pic_staff_id). Deliberately
// excludes activity_log.actor_staff_id/evidence.uploaded_by_staff_id — those
// are immutable historical audit trail, not a live assignment, so a staff
// member who merely triggered one of those years ago must not block their
// own deletion (see PLAN.md's reasoning). UNION already de-duplicates a
// project referenced through more than one of the four columns.
func (r *MySQLProjectRepository) ListByStaffPIC(ctx context.Context, tenantID, staffID int64) ([]domain.ProjectRef, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT p.id, p.name FROM projects p WHERE p.tenant_id = ? AND p.pic_staff_id = ?
		 UNION
		 SELECT p.id, p.name FROM project_vendors pv JOIN projects p ON p.id = pv.project_id
		   WHERE p.tenant_id = ? AND pv.pic_staff_id = ? AND pv.deleted_at IS NULL
		 UNION
		 SELECT p.id, p.name FROM vendor_milestones vm
		   JOIN project_vendors pv ON pv.id = vm.project_vendor_id
		   JOIN projects p ON p.id = pv.project_id
		   WHERE p.tenant_id = ? AND vm.pic_staff_id = ? AND pv.deleted_at IS NULL
		 UNION
		 SELECT p.id, p.name FROM vendor_issues vi JOIN projects p ON p.id = vi.project_id
		   WHERE p.tenant_id = ? AND vi.pic_staff_id = ? AND (vi.project_vendor_id IS NULL OR vi.project_vendor_id NOT IN (`+deletedEngagementIDs+`))`,
		tenantID, staffID, tenantID, staffID, tenantID, staffID, tenantID, staffID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []domain.ProjectRef
	for rows.Next() {
		var ref domain.ProjectRef
		if err := rows.Scan(&ref.ID, &ref.Name); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// CountAll backs the dashboard's TotalProjects stat — a plain COUNT(*)
// instead of loading every row just to take len() (PLAN.md "Performance
// remediation").
func (r *MySQLProjectRepository) CountAll(ctx context.Context, tenantID int64) (int64, error) {
	var total int64
	err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE tenant_id = ?`, tenantID).Scan(&total)
	return total, err
}

// ListForDashboard backs every other dashboard computation (trends,
// near-D-day, upcoming, lagging) — bounded to rows that could actually
// matter for any of them, instead of List's entire unbounded tenant history
// (PLAN.md "Performance remediation"): a project still open (non-terminal
// status) regardless of age, or a project whose created_at/event_date falls
// within the trend window (`since`, the earliest month ProjectTrend/
// RevenueTrend can ever display). A terminal-status project older than
// `since` on both dates can't affect any of the four computations, so it's
// excluded at the query level rather than fetched and then ignored in Go.
func (r *MySQLProjectRepository) ListForDashboard(ctx context.Context, tenantID int64, since time.Time) ([]domain.Project, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT `+projectColumns+` FROM projects
		 WHERE tenant_id = ? AND (status NOT IN ('Completed', 'Cancelled') OR created_at >= ? OR event_date >= ?)
		 ORDER BY event_date DESC, id DESC`,
		tenantID, since, since,
	)
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

// projectSortColumns is the sort whitelist of GET /projects: public name ->
// fixed SQL (ADR-0035). "remaining" and "nextDueDate" are the derived columns
// of the list, computed here by the same rules as domain.ComputeListMoney (the
// integration tests keep the two in step): the applied payments are the live
// DP/Termin/Pelunasan minus Refund (Tambahan out), floored at 0, and an
// installment is unpaid while the running total of the schedule (earliest due
// date, then id) exceeds them.
var projectSortColumns = map[string]pagination.SortColumn{
	"name":          pagination.Col("projects.name"),
	"eventDate":     pagination.Col("projects.event_date"),
	"status":        pagination.Col("projects.status"),
	"contractValue": pagination.Col("projects.contract_value"),
	"createdAt":     pagination.Col("projects.created_at"),
	"remaining":     pagination.Col(projectRemainingSQL),
	"nextDueDate":   pagination.NullableCol(projectNextDueSQL),
}

// ListSortSpec is the whitelist and the default order of GET /projects (today's
// order: latest event first), for the presentation layer to read ?sort= against.
func (r *MySQLProjectRepository) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return projectSortColumns, pagination.By("eventDate", true)
}

const projectAppliedSQL = `GREATEST(COALESCE((SELECT SUM(CASE WHEN cp.type IN ('DP','Termin','Pelunasan') THEN CAST(cp.amount AS SIGNED)
		WHEN cp.type = 'Refund' THEN -CAST(cp.amount AS SIGNED) ELSE 0 END)
	FROM client_payments cp WHERE cp.project_id = projects.id AND cp.deleted_at IS NULL), 0), 0)`

const projectRemainingSQL = `GREATEST(CAST(projects.contract_value AS SIGNED) - ` + projectAppliedSQL + `, 0)`

const projectNextDueSQL = `(SELECT MIN(t.due_date) FROM client_payment_terms t WHERE t.project_id = projects.id AND
	(SELECT SUM(t2.amount) FROM client_payment_terms t2 WHERE t2.project_id = t.project_id
	   AND (t2.due_date < t.due_date OR (t2.due_date = t.due_date AND t2.id <= t.id))) > ` + projectAppliedSQL + `)`

// ListPaginated backs the real `GET /projects` list page — List above stays
// as-is for dashboard/global-search consumers that need the full roster
// (including archived projects; this method's showArchived split doesn't
// apply there — see ProjectService.ListPaginated's doc comment). The caller's
// own scope (picStaffID/picSalesStaffID) is always ANDed with the filters.
func (r *MySQLProjectRepository) ListPaginated(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64, params pagination.Params, f domain.ProjectListFilter, sort pagination.Sort) ([]domain.Project, int64, error) {
	where := ` WHERE tenant_id = ? AND is_archived = ?`
	args := []interface{}{tenantID, f.ShowArchived}
	if cond, condArgs := picFilter(picStaffID, picSalesStaffID); cond != "" {
		where += cond
		args = append(args, condArgs...)
	}
	if f.Search != "" {
		where += ` AND (name LIKE ? OR bride_name LIKE ? OR groom_name LIKE ? OR venue LIKE ?)`
		like := "%" + utils.EscapeLike(f.Search) + "%"
		args = append(args, like, like, like, like)
	}
	if f.Status != "" {
		where += ` AND status = ?`
		args = append(args, f.Status)
	}
	if f.PICStaffID != nil {
		where += ` AND pic_staff_id = ?`
		args = append(args, *f.PICStaffID)
	}
	if f.EventFrom != nil {
		where += ` AND event_date >= ?`
		args = append(args, f.EventFrom.Format("2006-01-02"))
	}
	if f.EventTo != nil {
		where += ` AND event_date <= ?`
		args = append(args, f.EventTo.Format("2006-01-02"))
	}

	var total int64
	if err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*) FROM projects`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT ` + projectColumns + ` FROM projects` + where + sort.OrderBy("projects.id") + ` LIMIT ? OFFSET ?`
	rows, err := conn(ctx, r.db).QueryContext(ctx, listQuery, append(args, params.Limit, params.Offset())...)
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

// ListLabels backs contracts.ProjectLabels: id, name and event date of the
// tenant's projects among ids, in one query (the tenant filter keeps another
// workspace's ids from resolving).
func (r *MySQLProjectRepository) ListLabels(ctx context.Context, tenantID int64, ids []int64) ([]domain.ProjectLabel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT id, name, event_date FROM projects WHERE tenant_id = ? AND id IN (`+utils.Placeholders(len(ids))+`)`,
		append([]interface{}{tenantID}, utils.Int64Args(ids)...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.ProjectLabel
	for rows.Next() {
		var l domain.ProjectLabel
		if err := rows.Scan(&l.ID, &l.Name, &l.EventDate); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

func (r *MySQLProjectRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Project, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanProject(row.Scan)
}

func (r *MySQLProjectRepository) Create(ctx context.Context, p *domain.Project) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO projects (tenant_id, name, bride_name, groom_name, event_date, venue, venue_id, venue_rental_price, venue_charge,
		 prep_start_date, package_name, contract_value, status, pic_staff_id, pic_sales_staff_id, description)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.TenantID, p.Name, p.BrideName, p.GroomName, p.EventDate, p.Venue, p.VenueID, p.VenueRentalPrice, p.VenueCharge,
		p.PrepStartDate, p.PackageName, p.ContractValue, string(p.Status), p.PICStaffID, p.PICSalesStaffID, p.Description,
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

// projectPatchColumns maps each patchable field of a project to its column.
var projectPatchColumns = map[string]column[domain.Project]{
	"name":             col("name", func(p *domain.Project) any { return p.Name }),
	"brideName":        col("bride_name", func(p *domain.Project) any { return p.BrideName }),
	"groomName":        col("groom_name", func(p *domain.Project) any { return p.GroomName }),
	"eventDate":        col("event_date", func(p *domain.Project) any { return p.EventDate }),
	"venue":            col("venue", func(p *domain.Project) any { return p.Venue }),
	"venueId":          col("venue_id", func(p *domain.Project) any { return p.VenueID }),
	"venueRentalPrice": col("venue_rental_price", func(p *domain.Project) any { return p.VenueRentalPrice }),
	"venueCharge":      col("venue_charge", func(p *domain.Project) any { return p.VenueCharge }),
	"prepStartDate":    col("prep_start_date", func(p *domain.Project) any { return p.PrepStartDate }),
	"packageName":      col("package_name", func(p *domain.Project) any { return p.PackageName }),
	"contractValue":    col("contract_value", func(p *domain.Project) any { return p.ContractValue }),
	"picStaffId":       col("pic_staff_id", func(p *domain.Project) any { return p.PICStaffID }),
	"picSalesStaffId":  col("pic_sales_staff_id", func(p *domain.Project) any { return p.PICSalesStaffID }),
	"description":      col("description", func(p *domain.Project) any { return p.Description }),
	"isArchived":       col("is_archived", func(p *domain.Project) any { return p.IsArchived }),
}

// Patch writes the changed fields of p in one statement; see patchRow for the
// compare-and-set meaning of expected and of the result.
func (r *MySQLProjectRepository) Patch(ctx context.Context, p *domain.Project, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "projects", projectPatchColumns, p, changed, "tenant_id = ? AND id = ?", []any{p.TenantID, p.ID}, expected)
}

// ChangeStatus moves a project from one status to another only if it is still
// in `from`, and says whether it did. Two people acting on a stale page can
// therefore not skip or undo each other's step: the loser gets false.
func (r *MySQLProjectRepository) ChangeStatus(ctx context.Context, tenantID, id int64, from, to domain.ProjectStatus) (bool, error) {
	res, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE projects SET status = ? WHERE tenant_id = ? AND id = ? AND status = ?`, string(to), tenantID, id, string(from))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// DeleteCascade permanently removes a project and every row across this
// module's 8 sub-entity tables that references it, in one transaction —
// see ADR-0013. Deletion order matters: vendor_payments references evidence
// (invoice_evidence_id/proof_evidence_id), so it must go first; everything
// that references project_vendors must go before project_vendors itself;
// everything goes before the projects row. client_payments has its own
// direct FK straight to projects (no project_vendor_id, unlike
// vendor_payments) so it has no ordering dependency on anything else here —
// it just needs to go before the final projects delete. Evidence's
// object-storage files are NOT touched here — the caller
// (ProjectService.Delete) deletes those only after this transaction
// commits, so a storage failure never leaves a dangling DB reference (see
// its own doc comment).
func (r *MySQLProjectRepository) DeleteCascade(ctx context.Context, tenantID, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM activity_log WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM vendor_payments WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM client_payments WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM client_payment_terms WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM vendor_issues WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE vm FROM vendor_milestones vm INNER JOIN project_vendors pv ON vm.project_vendor_id = pv.id WHERE pv.project_id = ?`,
		id,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM evidence WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_vendors WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_milestones WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE tenant_id = ? AND id = ?`, tenantID, id); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Project milestones ---

type MySQLMilestoneRepository struct {
	db *sql.DB
}

func NewMySQLMilestoneRepository(db *sql.DB) *MySQLMilestoneRepository {
	return &MySQLMilestoneRepository{db: db}
}

const milestoneColumns = `id, project_id, sort_order, name, status, target_date, completed_date, updated_at`

func scanMilestone(scan func(dest ...interface{}) error) (*domain.ProjectMilestone, error) {
	var m domain.ProjectMilestone
	var status string
	var completedDate sql.NullTime
	err := scan(&m.ID, &m.ProjectID, &m.SortOrder, &m.Name, &status, &m.TargetDate, &completedDate, &m.UpdatedAt)
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
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+milestoneColumns+` FROM project_milestones WHERE project_id = ? ORDER BY sort_order`, projectID)
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

// ListByProjects backs ComputeProgressBatch (PLAN.md "Performance
// remediation"): one query across every id in projectIDs instead of one
// query per project.
func (r *MySQLMilestoneRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectMilestone, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + milestoneColumns + ` FROM project_milestones WHERE project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, sort_order`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
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
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM project_milestones WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanMilestone(row.Scan)
}

func (r *MySQLMilestoneRepository) Create(ctx context.Context, m *domain.ProjectMilestone) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
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

var milestonePatchColumns = map[string]column[domain.ProjectMilestone]{
	"status":        col("status", func(m *domain.ProjectMilestone) any { return string(m.Status) }),
	"targetDate":    col("target_date", func(m *domain.ProjectMilestone) any { return m.TargetDate }),
	"completedDate": col("completed_date", func(m *domain.ProjectMilestone) any { return m.CompletedDate }),
}

// Patch writes the changed fields of a project milestone (see patchRow).
func (r *MySQLMilestoneRepository) Patch(ctx context.Context, m *domain.ProjectMilestone, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "project_milestones", milestonePatchColumns, m, changed, "project_id = ? AND id = ?", []any{m.ProjectID, m.ID}, expected)
}

func (r *MySQLMilestoneRepository) NextSortOrder(ctx context.Context, projectID int64) (int, error) {
	var maxOrder sql.NullInt64
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT MAX(sort_order) FROM project_milestones WHERE project_id = ?`, projectID)
	if err := row.Scan(&maxOrder); err != nil {
		return 0, err
	}
	return int(maxOrder.Int64) + 1, nil
}

// Reorder rewrites sort_order to match each ID's position in orderedIDs
// (1-based) — the application layer has already validated orderedIDs is an
// exact permutation of this project's milestone IDs.
func (r *MySQLMilestoneRepository) Reorder(ctx context.Context, projectID int64, orderedIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range orderedIDs {
		if _, err := tx.ExecContext(ctx,
			`UPDATE project_milestones SET sort_order = ? WHERE id = ? AND project_id = ?`,
			i+1, id, projectID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CountActive counts projects still in progress: not completed, cancelled or
// archived (trial cap, PLAN.md decision K6).
func (r *MySQLProjectRepository) CountActive(ctx context.Context, tenantID int64) (int, error) {
	var n int
	err := conn(ctx, r.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM projects WHERE tenant_id = ? AND is_archived = FALSE AND status NOT IN ('Completed', 'Cancelled')`,
		tenantID).Scan(&n)
	return n, err
}

// picFilter scopes a project list. Both set means "projects this person
// holds" (PIC or PIC Sales) — a role scoped to its own projects (ADR-0026).
func picFilter(picStaffID, picSalesStaffID *int64) (string, []interface{}) {
	switch {
	case picStaffID != nil && picSalesStaffID != nil:
		return ` AND (pic_staff_id = ? OR pic_sales_staff_id = ?)`, []interface{}{*picStaffID, *picSalesStaffID}
	case picStaffID != nil:
		return ` AND pic_staff_id = ?`, []interface{}{*picStaffID}
	case picSalesStaffID != nil:
		return ` AND pic_sales_staff_id = ?`, []interface{}{*picSalesStaffID}
	}
	return "", nil
}
