package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/pagination"
)

type MySQLClientRepository struct {
	db *sql.DB
}

func NewMySQLClientRepository(db *sql.DB) *MySQLClientRepository {
	return &MySQLClientRepository{db: db}
}

const clientColumns = `id, tenant_id, project_id, role, relation_note, name, phone, email, is_active,
	last_credential_reset_at, created_at, updated_at`

func scanClient(scan func(dest ...interface{}) error) (*domain.Client, error) {
	var c domain.Client
	var role string
	var lastReset sql.NullTime
	err := scan(&c.ID, &c.TenantID, &c.ProjectID, &role, &c.RelationNote, &c.Name, &c.Phone, &c.Email, &c.IsActive,
		&lastReset, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Role = domain.ClientRole(role)
	if lastReset.Valid {
		c.LastCredentialResetAt = &lastReset.Time
	}
	return &c, nil
}

func (r *MySQLClientRepository) ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? AND project_id = ? ORDER BY id`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

func (r *MySQLClientRepository) ListByTenant(ctx context.Context, tenantID int64) ([]domain.Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? ORDER BY id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

// ListByTenantPaginated backs the real `GET /clients` list page (no
// projectId filter) — ListByTenant above stays as-is for global search.
func (r *MySQLClientRepository) ListByTenantPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.Client, int64, error) {
	countQuery := `SELECT COUNT(*) FROM clients WHERE tenant_id = ?`
	listQuery := `SELECT ` + clientColumns + ` FROM clients WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if search != "" {
		countQuery += ` AND (name LIKE ? OR email LIKE ?)`
		listQuery += ` AND (name LIKE ? OR email LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
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

	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *c)
	}
	return list, total, rows.Err()
}

func (r *MySQLClientRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanClient(row.Scan)
}

func (r *MySQLClientRepository) Create(ctx context.Context, c *domain.Client) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO clients (tenant_id, project_id, role, relation_note, name, phone, email, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.TenantID, c.ProjectID, string(c.Role), c.RelationNote, c.Name, c.Phone, c.Email, c.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

func (r *MySQLClientRepository) Update(ctx context.Context, c *domain.Client) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clients SET name = ?, phone = ?, email = ?, relation_note = ? WHERE tenant_id = ? AND id = ?`,
		c.Name, c.Phone, c.Email, c.RelationNote, c.TenantID, c.ID,
	)
	return err
}

func (r *MySQLClientRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE clients SET is_active = ? WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}

func (r *MySQLClientRepository) SetCredentialResetAt(ctx context.Context, tenantID, id int64, when time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE clients SET last_credential_reset_at = ? WHERE tenant_id = ? AND id = ?`, when, tenantID, id)
	return err
}
