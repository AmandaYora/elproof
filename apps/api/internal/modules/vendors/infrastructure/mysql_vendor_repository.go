package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/pagination"
)

type MySQLVendorRepository struct {
	db *sql.DB
}

func NewMySQLVendorRepository(db *sql.DB) *MySQLVendorRepository {
	return &MySQLVendorRepository{db: db}
}

const vendorColumns = `id, tenant_id, category_id, name, pic_name, phone, email, address, notes, is_active, created_at, updated_at`

func scanVendor(scan func(dest ...interface{}) error) (*domain.Vendor, error) {
	var v domain.Vendor
	var notes sql.NullString
	err := scan(&v.ID, &v.TenantID, &v.CategoryID, &v.Name, &v.PICName, &v.Phone, &v.Email, &v.Address, &notes, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v.Notes = notes.String
	return &v, nil
}

func (r *MySQLVendorRepository) List(ctx context.Context, tenantID int64, categoryID *int64) ([]domain.Vendor, error) {
	query := `SELECT ` + vendorColumns + ` FROM vendors WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if categoryID != nil {
		query += ` AND category_id = ?`
		args = append(args, *categoryID)
	}
	query += ` ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vendors []domain.Vendor
	for rows.Next() {
		v, err := scanVendor(rows.Scan)
		if err != nil {
			return nil, err
		}
		vendors = append(vendors, *v)
	}
	return vendors, rows.Err()
}

// ListPaginated backs the real `GET /vendors` list page — List above stays
// as-is for the vendor-picker dropdowns that need the full roster in one call.
func (r *MySQLVendorRepository) ListPaginated(ctx context.Context, tenantID int64, categoryID *int64, params pagination.Params, search string) ([]domain.Vendor, int64, error) {
	countQuery := `SELECT COUNT(*) FROM vendors WHERE tenant_id = ?`
	listQuery := `SELECT ` + vendorColumns + ` FROM vendors WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if categoryID != nil {
		countQuery += ` AND category_id = ?`
		listQuery += ` AND category_id = ?`
		args = append(args, *categoryID)
	}
	if search != "" {
		countQuery += ` AND (name LIKE ? OR pic_name LIKE ? OR email LIKE ?)`
		listQuery += ` AND (name LIKE ? OR pic_name LIKE ? OR email LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
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

	var vendors []domain.Vendor
	for rows.Next() {
		v, err := scanVendor(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		vendors = append(vendors, *v)
	}
	return vendors, total, rows.Err()
}

func (r *MySQLVendorRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Vendor, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+vendorColumns+` FROM vendors WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanVendor(row.Scan)
}

func (r *MySQLVendorRepository) Create(ctx context.Context, vendor *domain.Vendor) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO vendors (tenant_id, category_id, name, pic_name, phone, email, address, notes, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		vendor.TenantID, vendor.CategoryID, vendor.Name, vendor.PICName, vendor.Phone, vendor.Email, vendor.Address, vendor.Notes, vendor.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	vendor.ID = id
	return nil
}

func (r *MySQLVendorRepository) Update(ctx context.Context, vendor *domain.Vendor) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vendors SET category_id = ?, name = ?, pic_name = ?, phone = ?, email = ?, address = ?, notes = ? WHERE tenant_id = ? AND id = ?`,
		vendor.CategoryID, vendor.Name, vendor.PICName, vendor.Phone, vendor.Email, vendor.Address, vendor.Notes, vendor.TenantID, vendor.ID,
	)
	return err
}

func (r *MySQLVendorRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE vendors SET is_active = ? WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}
