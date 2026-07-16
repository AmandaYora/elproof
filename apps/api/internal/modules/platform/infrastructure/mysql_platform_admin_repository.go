package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/pagination"
)

type MySQLPlatformAdminRepository struct {
	db *sql.DB
}

func NewMySQLPlatformAdminRepository(db *sql.DB) *MySQLPlatformAdminRepository {
	return &MySQLPlatformAdminRepository{db: db}
}

const platformAdminColumns = `id, name, title, role, username, email, phone, is_active, created_at, updated_at`

func scanPlatformAdmin(scan func(dest ...interface{}) error) (*domain.PlatformAdmin, error) {
	var a domain.PlatformAdmin
	var role string
	err := scan(&a.ID, &a.Name, &a.Title, &role, &a.Username, &a.Email, &a.Phone, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Role = domain.PlatformAdminRole(role)
	return &a, nil
}

func (r *MySQLPlatformAdminRepository) List(ctx context.Context) ([]domain.PlatformAdmin, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+platformAdminColumns+` FROM platform_admins ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []domain.PlatformAdmin
	for rows.Next() {
		a, err := scanPlatformAdmin(rows.Scan)
		if err != nil {
			return nil, err
		}
		admins = append(admins, *a)
	}
	return admins, rows.Err()
}

// ListPaginated backs the real `GET /platform-admins` list page — List above
// stays as-is for any full-roster consumer.
func (r *MySQLPlatformAdminRepository) ListPaginated(ctx context.Context, params pagination.Params, search, role string) ([]domain.PlatformAdmin, int64, error) {
	countQuery := `SELECT COUNT(*) FROM platform_admins`
	listQuery := `SELECT ` + platformAdminColumns + ` FROM platform_admins`
	var args []interface{}
	var conditions []string
	if search != "" {
		conditions = append(conditions, `(name LIKE ? OR email LIKE ?)`)
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	if role != "" {
		conditions = append(conditions, `role = ?`)
		args = append(args, role)
	}
	if len(conditions) > 0 {
		where := ` WHERE ` + strings.Join(conditions, " AND ")
		countQuery += where
		listQuery += where
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

	var admins []domain.PlatformAdmin
	for rows.Next() {
		a, err := scanPlatformAdmin(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		admins = append(admins, *a)
	}
	return admins, total, rows.Err()
}

func (r *MySQLPlatformAdminRepository) FindByID(ctx context.Context, id int64) (*domain.PlatformAdmin, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+platformAdminColumns+` FROM platform_admins WHERE id = ? LIMIT 1`, id)
	return scanPlatformAdmin(row.Scan)
}

func (r *MySQLPlatformAdminRepository) Create(ctx context.Context, admin *domain.PlatformAdmin) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO platform_admins (name, title, role, username, email, phone, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		admin.Name, admin.Title, string(admin.Role), admin.Username, admin.Email, admin.Phone, admin.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	admin.ID = id
	return nil
}

func (r *MySQLPlatformAdminRepository) Update(ctx context.Context, admin *domain.PlatformAdmin) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE platform_admins SET name = ?, title = ?, role = ?, email = ?, phone = ? WHERE id = ?`,
		admin.Name, admin.Title, string(admin.Role), admin.Email, admin.Phone, admin.ID,
	)
	return err
}

func (r *MySQLPlatformAdminRepository) SetActive(ctx context.Context, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE platform_admins SET is_active = ? WHERE id = ?`, isActive, id)
	return err
}
