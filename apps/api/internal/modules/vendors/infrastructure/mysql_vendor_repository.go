package infrastructure

import (
	"context"
	"database/sql"
	"elproof/internal/shared/utils"
	"strings"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/pagination"
)

type MySQLVendorRepository struct {
	db *sql.DB
}

func NewMySQLVendorRepository(db *sql.DB) *MySQLVendorRepository {
	return &MySQLVendorRepository{db: db}
}

const vendorColumns = `id, tenant_id, category_id, name, pic_name, phone, email, social_media, city,
	address, price_akad, price_akad_resepsi, price_resepsi, notes, attachment_path, attachment_mime_type,
	is_active, created_at, updated_at`

func scanVendor(scan func(dest ...interface{}) error) (*domain.Vendor, error) {
	var v domain.Vendor
	var email, socialMedia, city, address, notes sql.NullString
	var priceAkad, priceAkadResepsi, priceResepsi sql.NullInt64
	var attachmentPath, attachmentMimeType sql.NullString

	err := scan(
		&v.ID, &v.TenantID, &v.CategoryID, &v.Name, &v.PICName, &v.Phone, &email, &socialMedia, &city,
		&address, &priceAkad, &priceAkadResepsi, &priceResepsi, &notes, &attachmentPath, &attachmentMimeType,
		&v.IsActive, &v.CreatedAt, &v.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	v.Email = nullStringPtr(email)
	v.SocialMedia = nullStringPtr(socialMedia)
	v.City = nullStringPtr(city)
	v.Address = nullStringPtr(address)
	v.PriceAkad = nullInt64Ptr(priceAkad)
	v.PriceAkadResepsi = nullInt64Ptr(priceAkadResepsi)
	v.PriceResepsi = nullInt64Ptr(priceResepsi)
	v.Notes = notes.String
	v.AttachmentPath = nullStringPtr(attachmentPath)
	v.AttachmentMimeType = nullStringPtr(attachmentMimeType)
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

// vendorSortColumns is the sort whitelist of GET /vendors (ADR-0035). The list
// joins vendor_categories (this module's own table) as c for categoryName.
var vendorSortColumns = map[string]pagination.SortColumn{
	"name":         pagination.Col("v.name"),
	"categoryName": pagination.NullableCol("c.name"),
	"city":         pagination.NullableCol("v.city"),
	"priceAkad":    pagination.NullableCol("v.price_akad"),
	"isActive":     pagination.Col("v.is_active"),
	"createdAt":    pagination.Col("v.created_at"),
}

// ListSortSpec is the whitelist and the default order of GET /vendors (today's:
// oldest first).
func (r *MySQLVendorRepository) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return vendorSortColumns, pagination.By("createdAt", false)
}

// ListPaginated backs the real `GET /vendors` list page — List above stays
// as-is for the vendor-picker dropdowns and the bulk import's prefetch, both
// of which need the full roster in one call.
func (r *MySQLVendorRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, f domain.VendorListFilter, sort pagination.Sort) ([]domain.VendorRow, int64, error) {
	where := ` WHERE v.tenant_id = ?`
	args := []interface{}{tenantID}
	if f.CategoryID != nil {
		where += ` AND v.category_id = ?`
		args = append(args, *f.CategoryID)
	}
	if f.Search != "" {
		where += ` AND (v.name LIKE ? OR v.pic_name LIKE ? OR v.email LIKE ?)`
		like := "%" + utils.EscapeLike(f.Search) + "%"
		args = append(args, like, like, like)
	}
	if f.City != "" {
		where += ` AND v.city = ?`
		args = append(args, f.City)
	}
	if f.IsActive != nil {
		where += ` AND v.is_active = ?`
		args = append(args, *f.IsActive)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vendors v`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+qualifyColumns("v", vendorColumns)+`, COALESCE(c.name, '')
		 FROM vendors v LEFT JOIN vendor_categories c ON c.id = v.category_id AND c.tenant_id = v.tenant_id`+where+
			sort.OrderBy("v.id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var vendors []domain.VendorRow
	for rows.Next() {
		var categoryName string
		v, err := scanVendor(func(dest ...interface{}) error { return rows.Scan(append(dest, &categoryName)...) })
		if err != nil {
			return nil, 0, err
		}
		vendors = append(vendors, domain.VendorRow{Vendor: *v, CategoryName: categoryName})
	}
	return vendors, total, rows.Err()
}

// BulkSetActive activates or deactivates several vendors of the tenant in one
// transaction (POST /vendors/bulk).
func (r *MySQLVendorRepository) BulkSetActive(ctx context.Context, tenantID int64, ids []int64, active bool) (domain.BulkResult, error) {
	return bulkSetActive(ctx, r.db, "vendors", tenantID, ids, active)
}

// ListFiltered backs Export -- the same category/search/city filters as
// ListPaginated, but unpaginated (Export always returns the whole matching
// set, per PLAN.md's Export design).
func (r *MySQLVendorRepository) ListFiltered(ctx context.Context, tenantID int64, categoryID *int64, search, city string) ([]domain.Vendor, error) {
	query := `SELECT ` + vendorColumns + ` FROM vendors WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if categoryID != nil {
		query += ` AND category_id = ?`
		args = append(args, *categoryID)
	}
	if search != "" {
		query += ` AND (name LIKE ? OR pic_name LIKE ? OR email LIKE ?)`
		like := "%" + utils.EscapeLike(search) + "%"
		args = append(args, like, like, like)
	}
	if city != "" {
		query += ` AND city = ?`
		args = append(args, city)
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

// CountByCategoryID backs Vendor Category's hard-delete guard (PLAN.md) --
// `vendors.category_id` is a real same-module SQL FK (RESTRICT by default,
// NOT NULL column), so a category still assigned to any vendor can never be
// hard-deleted; this turns that into a friendly count-based message instead
// of a raw FK-violation error.
func (r *MySQLVendorRepository) CountByCategoryID(ctx context.Context, tenantID, categoryID int64) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vendors WHERE tenant_id = ? AND category_id = ?`, tenantID, categoryID).Scan(&count)
	return count, err
}

func (r *MySQLVendorRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Vendor, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+vendorColumns+` FROM vendors WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanVendor(row.Scan)
}

func (r *MySQLVendorRepository) Create(ctx context.Context, vendor *domain.Vendor) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO vendors (tenant_id, category_id, name, pic_name, phone, email, social_media, city,
		 address, price_akad, price_akad_resepsi, price_resepsi, notes, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		vendor.TenantID, vendor.CategoryID, vendor.Name, vendor.PICName, vendor.Phone, vendor.Email, vendor.SocialMedia, vendor.City,
		vendor.Address, vendor.PriceAkad, vendor.PriceAkadResepsi, vendor.PriceResepsi, vendor.Notes, vendor.IsActive,
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

// CreateBatch inserts multiple vendors in one multi-row INSERT statement --
// used only by the bulk-import flow, which has already chunked the caller's
// slice to a safe size before calling this. Mirrors Venue's own CreateBatch.
func (r *MySQLVendorRepository) CreateBatch(ctx context.Context, vendors []domain.Vendor) error {
	if len(vendors) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString(`INSERT INTO vendors (tenant_id, category_id, name, pic_name, phone, email, social_media, city,
		 address, price_akad, price_akad_resepsi, price_resepsi, notes, is_active) VALUES `)
	args := make([]interface{}, 0, len(vendors)*14)
	for i, v := range vendors {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(`(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		args = append(args, v.TenantID, v.CategoryID, v.Name, v.PICName, v.Phone, v.Email, v.SocialMedia, v.City,
			v.Address, v.PriceAkad, v.PriceAkadResepsi, v.PriceResepsi, v.Notes, v.IsActive)
	}
	_, err := r.db.ExecContext(ctx, sb.String(), args...)
	return err
}

func (r *MySQLVendorRepository) Update(ctx context.Context, vendor *domain.Vendor) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE vendors SET category_id = ?, name = ?, pic_name = ?, phone = ?, email = ?, social_media = ?, city = ?,
		 address = ?, price_akad = ?, price_akad_resepsi = ?, price_resepsi = ?, notes = ? WHERE tenant_id = ? AND id = ?`,
		vendor.CategoryID, vendor.Name, vendor.PICName, vendor.Phone, vendor.Email, vendor.SocialMedia, vendor.City,
		vendor.Address, vendor.PriceAkad, vendor.PriceAkadResepsi, vendor.PriceResepsi, vendor.Notes, vendor.TenantID, vendor.ID,
	)
	return err
}

func (r *MySQLVendorRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE vendors SET is_active = ? WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}

func (r *MySQLVendorRepository) UpdateAttachment(ctx context.Context, tenantID, id int64, path, mimeType *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE vendors SET attachment_path = ?, attachment_mime_type = ? WHERE tenant_id = ? AND id = ?`, path, mimeType, tenantID, id)
	return err
}

// Delete permanently removes a vendor -- a deliberate, guarded exception to
// this codebase's soft-state convention (PLAN.md's hard-delete plan:
// informed-consent, not a blocking precondition). Never touches
// project_vendors or any other row that might reference this vendor --
// those are cross-module, no SQL FK, and are left to degrade gracefully via
// the frontend's existing "Vendor tidak dikenal" fallback, exactly the
// tradeoff the confirmation dialog exists to make explicit.
func (r *MySQLVendorRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM vendors WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

// NamesByIDs resolves vendor names for the given ids, tenant-scoped, in one
// query. Ids that are not this tenant's vendors are absent.
func (r *MySQLVendorRepository) NamesByIDs(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	return namesByIDs(ctx, r.db, "vendors", tenantID, ids)
}
