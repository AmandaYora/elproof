package infrastructure

import (
	"context"
	"database/sql"
	"elproof/internal/shared/utils"
	"errors"

	"elproof/internal/modules/staff/domain"
	"elproof/internal/shared/pagination"
)

type MySQLStaffRepository struct {
	db *sql.DB
}

func NewMySQLStaffRepository(db *sql.DB) *MySQLStaffRepository {
	return &MySQLStaffRepository{db: db}
}

const staffColumns = `id, tenant_id, name, title, initials, role, IFNULL(role_id, 0), username, email, phone, is_active, created_at, updated_at`

func scanStaff(scan func(dest ...interface{}) error) (*domain.StaffMember, error) {
	var m domain.StaffMember
	var role string
	err := scan(&m.ID, &m.TenantID, &m.Name, &m.Title, &m.Initials, &role, &m.RoleID, &m.Username, &m.Email, &m.Phone, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Role = domain.StaffRole(role)
	return &m, nil
}

func (r *MySQLStaffRepository) List(ctx context.Context, tenantID int64) ([]domain.StaffMember, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+staffColumns+` FROM staff_members WHERE tenant_id = ? ORDER BY id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.StaffMember
	for rows.Next() {
		m, err := scanStaff(rows.Scan)
		if err != nil {
			return nil, err
		}
		members = append(members, *m)
	}
	return members, rows.Err()
}

// staffSortColumns is the sort whitelist of GET /staff's own columns (ADR-0035).
// roleName is the role's CURRENT name, which `access` owns: the application layer
// adds it as a column ranked by name (pagination.RankedCol).
var staffSortColumns = map[string]pagination.SortColumn{
	"name":      pagination.Col("name"),
	"title":     pagination.Col("title"),
	"isActive":  pagination.Col("is_active"),
	"createdAt": pagination.Col("created_at"),
}

// ListSortSpec returns the SQL-orderable columns of the list.
func (r *MySQLStaffRepository) ListSortSpec() map[string]pagination.SortColumn {
	return staffSortColumns
}

func staffWhere(tenantID int64, f domain.StaffListFilter) (string, []interface{}) {
	where := ` WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if f.Search != "" {
		where += ` AND (name LIKE ? OR email LIKE ?)`
		like := "%" + utils.EscapeLike(f.Search) + "%"
		args = append(args, like, like)
	}
	if f.RoleID > 0 {
		// Filter by workspace role id (Pengguna page's role filter).
		where += ` AND role_id = ?`
		args = append(args, f.RoleID)
	}
	if f.IsActive != nil {
		where += ` AND is_active = ?`
		args = append(args, *f.IsActive)
	}
	return where, args
}

// ListPaginated backs the real `GET /staff` list page (table + server
// pagination) — List above stays as-is for the many PIC/staff dropdown
// pickers across the app that need the full roster in one call (see
// ADR/Fase 7 implementation note on why both exist).
func (r *MySQLStaffRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, f domain.StaffListFilter, sort pagination.Sort) ([]domain.StaffMember, int64, error) {
	where, args := staffWhere(tenantID, f)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM staff_members`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+staffColumns+` FROM staff_members`+where+sort.OrderBy("id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var members []domain.StaffMember
	for rows.Next() {
		m, err := scanStaff(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		members = append(members, *m)
	}
	return members, total, rows.Err()
}

func (r *MySQLStaffRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.StaffMember, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+staffColumns+` FROM staff_members WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanStaff(row.Scan)
}

func (r *MySQLStaffRepository) Create(ctx context.Context, member *domain.StaffMember) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO staff_members (tenant_id, name, title, initials, role, role_id, username, email, phone, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		member.TenantID, member.Name, member.Title, member.Initials, string(member.Role), nullableRoleID(member.RoleID), member.Username, member.Email, member.Phone, member.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	member.ID = id
	return nil
}

func (r *MySQLStaffRepository) Update(ctx context.Context, member *domain.StaffMember) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE staff_members SET name = ?, title = ?, role = ?, role_id = ?, initials = ?, email = ?, phone = ? WHERE tenant_id = ? AND id = ?`,
		member.Name, member.Title, string(member.Role), nullableRoleID(member.RoleID), member.Initials, member.Email, member.Phone, member.TenantID, member.ID,
	)
	return err
}

func (r *MySQLStaffRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE staff_members SET is_active = ? WHERE tenant_id = ? AND id = ?`,
		isActive, tenantID, id,
	)
	return err
}

// Delete permanently removes a staff member. Originally only ever called by
// StaffService.Create as a compensating rollback when identity.CreateCredential
// fails after this row already committed; now also reachable via
// StaffService.Delete's guarded, Owner-only hard-delete flow (PLAN.md's
// hard-delete plan, an explicit exception to this codebase's soft-state
// convention).
func (r *MySQLStaffRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM staff_members WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

// CountActiveNonOwner counts active staff other than the Owner (trial cap).
func (r *MySQLStaffRepository) CountActiveNonOwner(ctx context.Context, tenantID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM staff_members WHERE tenant_id = ? AND is_active = TRUE AND role <> 'Owner'`, tenantID).Scan(&n)
	return n, err
}

// FindOwnerID returns the Owner's staff id (nil when the tenant has none).
func (r *MySQLStaffRepository) FindOwnerID(ctx context.Context, tenantID int64) (*int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM staff_members WHERE tenant_id = ? AND role = 'Owner' ORDER BY id LIMIT 1`, tenantID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func nullableRoleID(id int64) sql.NullInt64 {
	return sql.NullInt64{Int64: id, Valid: id > 0}
}

// NamesByIDs resolves names for the given staff ids, tenant-scoped, in one
// query. Ids that do not exist in this tenant are simply absent.
func (r *MySQLStaffRepository) NamesByIDs(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := append([]interface{}{tenantID}, utils.Int64Args(ids)...)
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM staff_members WHERE tenant_id = ? AND id IN (`+utils.Placeholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}
