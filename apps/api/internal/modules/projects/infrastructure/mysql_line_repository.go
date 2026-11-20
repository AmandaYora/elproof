package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

// MySQLLineRepository stores project_lines (the Rincian, PLAN §3.1) and keeps
// the contract cache on projects in step with them. Every write runs inside
// the caller's transaction (the line service always writes under one), so the
// cache is rewritten in the same unit of work as the change it reflects.
type MySQLLineRepository struct {
	db *sql.DB
}

func NewMySQLLineRepository(db *sql.DB) *MySQLLineRepository {
	return &MySQLLineRepository{db: db}
}

const lineColumns = `id, tenant_id, project_id, kind, parent_line_id, package_rule, description, category_id, event_id,
	quantity, unit, unit_price, amount, is_bonus, supplier_ref, sort_order, deleted_at, created_at, updated_at`

func scanLine(scan func(dest ...any) error) (*domain.ProjectLine, error) {
	var l domain.ProjectLine
	var kind string
	var parent, category, event, quantity, unitPrice, supplier sql.NullInt64
	var rule []byte
	var deleted sql.NullTime
	err := scan(&l.ID, &l.TenantID, &l.ProjectID, &kind, &parent, &rule, &l.Description, &category, &event,
		&quantity, &l.Unit, &unitPrice, &l.Amount, &l.IsBonus, &supplier, &l.SortOrder, &deleted, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l.Kind = domain.LineKind(kind)
	l.ParentLineID = nullInt64(parent)
	l.CategoryID = nullInt64(category)
	l.EventID = nullInt64(event)
	l.Quantity = nullInt64(quantity)
	l.UnitPrice = nullInt64(unitPrice)
	l.SupplierRef = nullInt64(supplier)
	l.DeletedAt = deletedStamp(deleted)
	if len(rule) > 0 && string(rule) != "null" {
		var r domain.LinePackageRule
		if err := json.Unmarshal(rule, &r); err != nil {
			return nil, err
		}
		l.PackageRule = &r
	}
	return &l, nil
}

func nullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func ruleJSON(r *domain.LinePackageRule) (any, error) {
	if r == nil {
		return nil, nil
	}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// List is the project's lines in screen order; live only unless withDeleted.
func (r *MySQLLineRepository) List(ctx context.Context, tenantID, projectID int64, withDeleted bool) ([]domain.ProjectLine, error) {
	q := `SELECT ` + lineColumns + ` FROM project_lines WHERE tenant_id = ? AND project_id = ?`
	if !withDeleted {
		q += ` AND deleted_at IS NULL`
	}
	rows, err := conn(ctx, r.db).QueryContext(ctx, q+` ORDER BY sort_order, id`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []domain.ProjectLine{}
	for rows.Next() {
		l, err := scanLine(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *l)
	}
	return list, rows.Err()
}

// Get is one line of the project, deleted or not; nil when there is none.
func (r *MySQLLineRepository) Get(ctx context.Context, tenantID, projectID, id int64) (*domain.ProjectLine, error) {
	return scanLine(conn(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+lineColumns+` FROM project_lines WHERE tenant_id = ? AND project_id = ? AND id = ?`,
		tenantID, projectID, id).Scan)
}

// Insert stores a new line and sets its ID.
func (r *MySQLLineRepository) Insert(ctx context.Context, l *domain.ProjectLine) error {
	rule, err := ruleJSON(l.PackageRule)
	if err != nil {
		return err
	}
	res, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO project_lines (tenant_id, project_id, kind, parent_line_id, package_rule, description, category_id, event_id,
		 quantity, unit, unit_price, amount, is_bonus, supplier_ref, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.TenantID, l.ProjectID, string(l.Kind), l.ParentLineID, rule, l.Description, l.CategoryID, l.EventID,
		l.Quantity, l.Unit, l.UnitPrice, l.Amount, l.IsBonus, l.SupplierRef, l.SortOrder)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	l.ID = id
	return nil
}

// Update rewrites a live line's editable fields (its kind, parent and rule
// copy never change).
func (r *MySQLLineRepository) Update(ctx context.Context, l *domain.ProjectLine) error {
	_, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE project_lines SET description = ?, category_id = ?, event_id = ?, quantity = ?, unit = ?, unit_price = ?,
		 amount = ?, is_bonus = ?, supplier_ref = ?, sort_order = ?
		 WHERE tenant_id = ? AND project_id = ? AND id = ? AND deleted_at IS NULL`,
		l.Description, l.CategoryID, l.EventID, l.Quantity, l.Unit, l.UnitPrice,
		l.Amount, l.IsBonus, l.SupplierRef, l.SortOrder, l.TenantID, l.ProjectID, l.ID)
	return err
}

// SetAmount rewrites a computed amount (the ExtraGuests line).
func (r *MySQLLineRepository) SetAmount(ctx context.Context, tenantID, projectID, id, amount int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE project_lines SET amount = ? WHERE tenant_id = ? AND project_id = ? AND id = ?`, amount, tenantID, projectID, id)
	return err
}

// SetSortOrder moves a line.
func (r *MySQLLineRepository) SetSortOrder(ctx context.Context, tenantID, projectID, id int64, order int) error {
	_, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE project_lines SET sort_order = ? WHERE tenant_id = ? AND project_id = ? AND id = ?`, order, tenantID, projectID, id)
	return err
}

// SoftDelete stamps a live line and its live children with the same
// deleted_at, so a restore brings exactly them back; false when there was none.
func (r *MySQLLineRepository) SoftDelete(ctx context.Context, tenantID, projectID, id int64, at time.Time) (bool, error) {
	q := conn(ctx, r.db)
	res, err := q.ExecContext(ctx,
		`UPDATE project_lines SET deleted_at = ? WHERE tenant_id = ? AND project_id = ? AND id = ? AND deleted_at IS NULL`,
		at, tenantID, projectID, id)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return false, err
	}
	_, err = q.ExecContext(ctx,
		`UPDATE project_lines SET deleted_at = ? WHERE tenant_id = ? AND project_id = ? AND parent_line_id = ? AND deleted_at IS NULL`,
		at, tenantID, projectID, id)
	return err == nil, err
}

// Restore clears the stamp of a line deleted after notBefore (the purge
// cutoff) and of the children deleted together with it; false when there was
// none.
func (r *MySQLLineRepository) Restore(ctx context.Context, tenantID, projectID, id int64, notBefore time.Time) (bool, error) {
	q := conn(ctx, r.db)
	var at time.Time
	err := q.QueryRowContext(ctx,
		`SELECT deleted_at FROM project_lines WHERE tenant_id = ? AND project_id = ? AND id = ? AND deleted_at IS NOT NULL AND deleted_at > ?`,
		tenantID, projectID, id, notBefore).Scan(&at)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE project_lines SET deleted_at = NULL WHERE tenant_id = ? AND project_id = ? AND parent_line_id = ? AND deleted_at = ?`,
		tenantID, projectID, id, at); err != nil {
		return false, err
	}
	_, err = q.ExecContext(ctx,
		`UPDATE project_lines SET deleted_at = NULL WHERE tenant_id = ? AND project_id = ? AND id = ?`, tenantID, projectID, id)
	return err == nil, err
}

// contractCacheSQL rewrites the project's contract cache from its live lines
// with one statement (PLAN §3.2): MySQL assigns left to right, so the tax and
// total read the subtotal just written. updated_at keeps its value: the cache
// is not part of the project form (ADR-0036 versions what PATCH edits).
const contractCacheSQL = `UPDATE projects p SET
	contract_subtotal = (
		SELECT COALESCE(SUM(CASE
			WHEN l.kind IN ('Package', 'ExtraGuests', 'Item') THEN CAST(l.amount AS SIGNED)
			WHEN l.kind = 'Discount' THEN -CAST(l.amount AS SIGNED)
			ELSE 0 END), 0)
		FROM project_lines l
		WHERE l.tenant_id = p.tenant_id AND l.project_id = p.id AND l.deleted_at IS NULL),
	contract_tax = CASE WHEN p.is_pkp AND contract_subtotal > 0 THEN FLOOR(contract_subtotal * 11 / 100) ELSE 0 END,
	contract_total = contract_subtotal + contract_tax,
	updated_at = updated_at`

// RefreshContract rewrites the contract cache; with bumpVersion it also moves
// the lines' own version (a change of the lines themselves, not of a guest
// number).
func (r *MySQLLineRepository) RefreshContract(ctx context.Context, tenantID, projectID int64, bumpVersion bool) error {
	q := contractCacheSQL
	if bumpVersion {
		q += `, lines_version = CURRENT_TIMESTAMP(6)`
	}
	_, err := conn(ctx, r.db).ExecContext(ctx, q+` WHERE p.tenant_id = ? AND p.id = ?`, tenantID, projectID)
	return err
}

// Contract is the contract cache and the lines' version as stored.
func (r *MySQLLineRepository) Contract(ctx context.Context, tenantID, projectID int64) (domain.ContractCache, error) {
	var c domain.ContractCache
	err := conn(ctx, r.db).QueryRowContext(ctx,
		`SELECT is_pkp, contract_subtotal, contract_tax, contract_total, lines_version FROM projects WHERE tenant_id = ? AND id = ?`,
		tenantID, projectID).Scan(&c.IsPKP, &c.Subtotal, &c.Tax, &c.Total, &c.LinesVersion)
	if err == sql.ErrNoRows {
		return c, nil
	}
	return c, err
}

// PackageNames is, per project, the descriptions of its live Package lines in
// screen order (the client home and lists show them instead of a typed name).
func (r *MySQLLineRepository) PackageNames(ctx context.Context, tenantID int64, projectIDs []int64) (map[int64][]string, error) {
	out := map[int64][]string{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	args := append([]any{tenantID}, utils.Int64Args(projectIDs)...)
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT project_id, description FROM project_lines
		 WHERE tenant_id = ? AND kind = 'Package' AND deleted_at IS NULL AND project_id IN `+inList(projectIDs)+`
		 ORDER BY project_id, sort_order, id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var name string
		if err := rows.Scan(&pid, &name); err != nil {
			return nil, err
		}
		out[pid] = append(out[pid], name)
	}
	return out, rows.Err()
}
