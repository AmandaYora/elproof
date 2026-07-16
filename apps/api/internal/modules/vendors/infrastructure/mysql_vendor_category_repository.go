package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/pagination"
)

type MySQLVendorCategoryRepository struct {
	db *sql.DB
}

func NewMySQLVendorCategoryRepository(db *sql.DB) *MySQLVendorCategoryRepository {
	return &MySQLVendorCategoryRepository{db: db}
}

const vendorCategoryColumns = `id, tenant_id, name, description, is_active, created_at, updated_at`

func scanVendorCategory(scan func(dest ...interface{}) error) (*domain.VendorCategory, error) {
	var c domain.VendorCategory
	err := scan(&c.ID, &c.TenantID, &c.Name, &c.Description, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *MySQLVendorCategoryRepository) List(ctx context.Context, tenantID int64) ([]domain.VendorCategory, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+vendorCategoryColumns+` FROM vendor_categories WHERE tenant_id = ? ORDER BY id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.VendorCategory
	for rows.Next() {
		c, err := scanVendorCategory(rows.Scan)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *c)
	}
	return categories, rows.Err()
}

// ListPaginated backs the real `GET /vendor-categories` list page — List
// above stays as-is for the category-picker dropdowns that need the full list.
func (r *MySQLVendorCategoryRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.VendorCategory, int64, error) {
	countQuery := `SELECT COUNT(*) FROM vendor_categories WHERE tenant_id = ?`
	listQuery := `SELECT ` + vendorCategoryColumns + ` FROM vendor_categories WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if search != "" {
		countQuery += ` AND name LIKE ?`
		listQuery += ` AND name LIKE ?`
		args = append(args, "%"+search+"%")
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

	var categories []domain.VendorCategory
	for rows.Next() {
		c, err := scanVendorCategory(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		categories = append(categories, *c)
	}
	return categories, total, rows.Err()
}

func (r *MySQLVendorCategoryRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.VendorCategory, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+vendorCategoryColumns+` FROM vendor_categories WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanVendorCategory(row.Scan)
}

func (r *MySQLVendorCategoryRepository) Create(ctx context.Context, category *domain.VendorCategory) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO vendor_categories (tenant_id, name, description, is_active) VALUES (?, ?, ?, ?)`,
		category.TenantID, category.Name, category.Description, category.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	category.ID = id
	return nil
}

func (r *MySQLVendorCategoryRepository) Update(ctx context.Context, category *domain.VendorCategory) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vendor_categories SET name = ?, description = ? WHERE tenant_id = ? AND id = ?`,
		category.Name, category.Description, category.TenantID, category.ID,
	)
	return err
}

func (r *MySQLVendorCategoryRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE vendor_categories SET is_active = ? WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}
