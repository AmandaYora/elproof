package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

type MySQLPackageRepository struct {
	db *sql.DB
}

func NewMySQLPackageRepository(db *sql.DB) *MySQLPackageRepository {
	return &MySQLPackageRepository{db: db}
}

const packageColumns = `id, tenant_id, name, price, included_guests, extra_guest_step, extra_guest_step_price, default_venue_id, is_active, created_at, updated_at`

func scanPackage(scan func(dest ...any) error) (*domain.Package, error) {
	var p domain.Package
	var included, step sql.NullInt32
	var stepPrice, venue sql.NullInt64
	err := scan(&p.ID, &p.TenantID, &p.Name, &p.Price, &included, &step, &stepPrice, &venue, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IncludedGuests = intPtr(included)
	p.ExtraGuestStep = intPtr(step)
	p.ExtraGuestStepPrice = int64Ptr(stepPrice)
	p.DefaultVenueID = int64Ptr(venue)
	return &p, nil
}

func intPtr(v sql.NullInt32) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int32)
	return &i
}

func int64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	i := v.Int64
	return &i
}

func nullable[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// packageSortColumns is the ?sort= whitelist of GET /packages.
var packageSortColumns = map[string]pagination.SortColumn{
	"name":           pagination.Col("name"),
	"price":          pagination.Col("price"),
	"includedGuests": pagination.NullableCol("included_guests"),
	"createdAt":      pagination.Col("created_at"),
}

func (r *MySQLPackageRepository) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return packageSortColumns, pagination.By("name", false)
}

func (r *MySQLPackageRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, filter application.PackageListFilter, sort pagination.Sort) ([]domain.Package, int64, error) {
	where := ` WHERE tenant_id = ?`
	args := []any{tenantID}
	if filter.Search != "" {
		where += ` AND name LIKE ?`
		args = append(args, "%"+filter.Search+"%")
	}
	if filter.IsActive != nil {
		where += ` AND is_active = ?`
		args = append(args, *filter.IsActive)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM packages`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+packageColumns+` FROM packages`+where+sort.OrderBy("id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Package
	for rows.Next() {
		p, err := scanPackage(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.loadChildren(ctx, tenantID, out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// loadChildren fills the covered kinds and components of the packages, two
// queries for the whole page.
func (r *MySQLPackageRepository) loadChildren(ctx context.Context, tenantID int64, packages []domain.Package) error {
	if len(packages) == 0 {
		return nil
	}
	index := make(map[int64]int, len(packages))
	ids := make([]int64, 0, len(packages))
	for i, p := range packages {
		index[p.ID] = i
		ids = append(ids, p.ID)
	}
	args := append([]any{tenantID}, utils.Int64Args(ids)...)
	in := `(` + utils.Placeholders(len(ids)) + `)`

	rows, err := r.db.QueryContext(ctx, `SELECT pek.package_id, pek.event_kind_id FROM package_event_kinds pek
		JOIN event_kinds k ON k.id = pek.event_kind_id AND k.tenant_id = pek.tenant_id
		WHERE pek.tenant_id = ? AND pek.package_id IN `+in+` ORDER BY k.sort_order, k.id`, args...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pkg, kind int64
		if err := rows.Scan(&pkg, &kind); err != nil {
			rows.Close()
			return err
		}
		packages[index[pkg]].EventKindIDs = append(packages[index[pkg]].EventKindIDs, kind)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	rows, err = r.db.QueryContext(ctx, `SELECT package_id, id, description, category_id, vendor_id, estimated_cost, sort_order
		FROM package_components WHERE tenant_id = ? AND package_id IN `+in+` ORDER BY sort_order, id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var pkg int64
		var c domain.PackageComponent
		var category, vendor, cost sql.NullInt64
		if err := rows.Scan(&pkg, &c.ID, &c.Description, &category, &vendor, &cost, &c.SortOrder); err != nil {
			return err
		}
		c.CategoryID, c.VendorID, c.EstimatedCost = int64Ptr(category), int64Ptr(vendor), int64Ptr(cost)
		packages[index[pkg]].Components = append(packages[index[pkg]].Components, c)
	}
	return rows.Err()
}

func (r *MySQLPackageRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Package, error) {
	p, err := scanPackage(r.db.QueryRowContext(ctx, `SELECT `+packageColumns+` FROM packages WHERE tenant_id = ? AND id = ?`, tenantID, id).Scan)
	if err != nil || p == nil {
		return p, err
	}
	list := []domain.Package{*p}
	if err := r.loadChildren(ctx, tenantID, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (r *MySQLPackageRepository) NameTaken(ctx context.Context, tenantID int64, name string, selfID int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM packages WHERE tenant_id = ? AND name = ? AND id <> ?`, tenantID, name, selfID).Scan(&n)
	return n > 0, err
}

func (r *MySQLPackageRepository) Create(ctx context.Context, p *domain.Package) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO packages (tenant_id, name, price, included_guests, extra_guest_step, extra_guest_step_price, default_venue_id, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.TenantID, p.Name, p.Price, nullable(p.IncludedGuests), nullable(p.ExtraGuestStep), nullable(p.ExtraGuestStepPrice), nullable(p.DefaultVenueID), p.IsActive)
	if isDuplicate(err) {
		return errNameTaken("Paket dengan nama ini sudah ada")
	}
	if err != nil {
		return err
	}
	if p.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	if err := writeChildren(ctx, tx, p); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLPackageRepository) Update(ctx context.Context, p *domain.Package, expected *time.Time) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	query := `UPDATE packages SET name = ?, price = ?, included_guests = ?, extra_guest_step = ?, extra_guest_step_price = ?, default_venue_id = ?,
		updated_at = CURRENT_TIMESTAMP(6) WHERE tenant_id = ? AND id = ?`
	args := []any{p.Name, p.Price, nullable(p.IncludedGuests), nullable(p.ExtraGuestStep), nullable(p.ExtraGuestStepPrice), nullable(p.DefaultVenueID), p.TenantID, p.ID}
	if expected != nil {
		query += ` AND updated_at = ?`
		args = append(args, *expected)
	}
	res, err := tx.ExecContext(ctx, query, args...)
	if isDuplicate(err) {
		return false, errNameTaken("Paket dengan nama ini sudah ada")
	}
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM package_event_kinds WHERE tenant_id = ? AND package_id = ?`, p.TenantID, p.ID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM package_components WHERE tenant_id = ? AND package_id = ?`, p.TenantID, p.ID); err != nil {
		return false, err
	}
	if err := writeChildren(ctx, tx, p); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func writeChildren(ctx context.Context, tx *sql.Tx, p *domain.Package) error {
	for _, kind := range p.EventKindIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO package_event_kinds (tenant_id, package_id, event_kind_id) VALUES (?, ?, ?)`, p.TenantID, p.ID, kind); err != nil {
			return err
		}
	}
	for _, c := range p.Components {
		if _, err := tx.ExecContext(ctx, `INSERT INTO package_components (tenant_id, package_id, description, category_id, vendor_id, estimated_cost, sort_order)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			p.TenantID, p.ID, c.Description, nullable(c.CategoryID), nullable(c.VendorID), nullable(c.EstimatedCost), c.SortOrder); err != nil {
			return err
		}
	}
	return nil
}

func (r *MySQLPackageRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE packages SET is_active = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}

// Delete removes the package; its kinds and components go with it (ON DELETE
// CASCADE, same module).
func (r *MySQLPackageRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM packages WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

func (r *MySQLPackageRepository) exists(ctx context.Context, query string, args ...any) (bool, error) {
	var found int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *MySQLPackageRepository) VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error) {
	return r.exists(ctx, `SELECT 1 FROM package_components WHERE tenant_id = ? AND vendor_id = ? LIMIT 1`, tenantID, vendorID)
}

func (r *MySQLPackageRepository) VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error) {
	return r.exists(ctx, `SELECT 1 FROM packages WHERE tenant_id = ? AND default_venue_id = ? LIMIT 1`, tenantID, venueID)
}

func (r *MySQLPackageRepository) CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error) {
	return r.exists(ctx, `SELECT 1 FROM package_components WHERE tenant_id = ? AND category_id = ? LIMIT 1`, tenantID, categoryID)
}
