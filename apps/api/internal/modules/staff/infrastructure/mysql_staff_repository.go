package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/staff/domain"
	"elproof/internal/shared/pagination"
)

type MySQLStaffRepository struct {
	db *sql.DB
}

func NewMySQLStaffRepository(db *sql.DB) *MySQLStaffRepository {
	return &MySQLStaffRepository{db: db}
}

const staffColumns = `id, tenant_id, name, title, initials, role, username, email, phone, is_active, created_at, updated_at`

func scanStaff(scan func(dest ...interface{}) error) (*domain.StaffMember, error) {
	var m domain.StaffMember
	var role string
	err := scan(&m.ID, &m.TenantID, &m.Name, &m.Title, &m.Initials, &role, &m.Username, &m.Email, &m.Phone, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
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

// ListPaginated backs the real `GET /staff` list page (table + server
// pagination) — List above stays as-is for the many PIC/staff dropdown
// pickers across the app that need the full roster in one call (see
// ADR/Fase 7 implementation note on why both exist).
func (r *MySQLStaffRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, role string) ([]domain.StaffMember, int64, error) {
	countQuery := `SELECT COUNT(*) FROM staff_members WHERE tenant_id = ?`
	listQuery := `SELECT ` + staffColumns + ` FROM staff_members WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if search != "" {
		countQuery += ` AND (name LIKE ? OR email LIKE ?)`
		listQuery += ` AND (name LIKE ? OR email LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	if role != "" {
		countQuery += ` AND role = ?`
		listQuery += ` AND role = ?`
		args = append(args, role)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY id LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, listQuery, append(args, params.Limit, params.Offset())...)
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
		`INSERT INTO staff_members (tenant_id, name, title, initials, role, username, email, phone, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		member.TenantID, member.Name, member.Title, member.Initials, string(member.Role), member.Username, member.Email, member.Phone, member.IsActive,
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
		`UPDATE staff_members SET name = ?, title = ?, role = ?, initials = ?, email = ?, phone = ? WHERE tenant_id = ? AND id = ?`,
		member.Name, member.Title, string(member.Role), member.Initials, member.Email, member.Phone, member.TenantID, member.ID,
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

// Delete is only ever called by StaffService.Create as a compensating
// rollback when identity.CreateCredential fails after this row already
// committed (mirrors the clients module's Create/Delete pair).
func (r *MySQLStaffRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM staff_members WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}
