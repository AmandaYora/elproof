package infrastructure

import (
	"context"
	"database/sql"
	"elproof/internal/shared/utils"
	"strings"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/pagination"
)

type MySQLVenueRepository struct {
	db *sql.DB
}

func NewMySQLVenueRepository(db *sql.DB) *MySQLVenueRepository {
	return &MySQLVenueRepository{db: db}
}

const venueColumns = `id, tenant_id, name, pic_name, phone_pic, phone_venue, email, address, city,
	rental_price, charge, capacity, facilities, social_media, notes, attachment_path,
	attachment_mime_type, is_active, created_at, updated_at`

// scanVenue is a direct nullable-column mapping, same style as scanVendor --
// facilities/social_media are plain TEXT now (not JSON), so they scan exactly
// like notes, no marshal/unmarshal code needed anywhere in this file.
func scanVenue(scan func(dest ...interface{}) error) (*domain.Venue, error) {
	var v domain.Venue
	var phoneVenue, email, address, city, notes sql.NullString
	var rentalPrice, charge, capacity sql.NullInt64
	var facilities, socialMedia sql.NullString
	var attachmentPath, attachmentMimeType sql.NullString

	err := scan(
		&v.ID, &v.TenantID, &v.Name, &v.PICName, &v.PhonePIC, &phoneVenue, &email, &address, &city,
		&rentalPrice, &charge, &capacity, &facilities, &socialMedia, &notes, &attachmentPath,
		&attachmentMimeType, &v.IsActive, &v.CreatedAt, &v.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	v.PhoneVenue = nullStringPtr(phoneVenue)
	v.Email = nullStringPtr(email)
	v.Address = nullStringPtr(address)
	v.City = nullStringPtr(city)
	v.RentalPrice = nullInt64Ptr(rentalPrice)
	v.Charge = nullInt64Ptr(charge)
	v.Capacity = nullIntPtr(capacity)
	v.Facilities = nullStringPtr(facilities)
	v.SocialMedia = nullStringPtr(socialMedia)
	v.Notes = notes.String
	v.AttachmentPath = nullStringPtr(attachmentPath)
	v.AttachmentMimeType = nullStringPtr(attachmentMimeType)
	return &v, nil
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func nullInt64Ptr(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}

func nullIntPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int64)
	return &v
}

func (r *MySQLVenueRepository) List(ctx context.Context, tenantID int64) ([]domain.Venue, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+venueColumns+` FROM venues WHERE tenant_id = ? ORDER BY id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var venues []domain.Venue
	for rows.Next() {
		v, err := scanVenue(rows.Scan)
		if err != nil {
			return nil, err
		}
		venues = append(venues, *v)
	}
	return venues, rows.Err()
}

// venueSortColumns is the sort whitelist of GET /venues (ADR-0035).
var venueSortColumns = map[string]pagination.SortColumn{
	"name":        pagination.Col("name"),
	"city":        pagination.NullableCol("city"),
	"capacity":    pagination.NullableCol("capacity"),
	"rentalPrice": pagination.NullableCol("rental_price"),
	"isActive":    pagination.Col("is_active"),
	"createdAt":   pagination.Col("created_at"),
}

// ListSortSpec is the whitelist and the default order of GET /venues (today's:
// oldest first).
func (r *MySQLVenueRepository) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return venueSortColumns, pagination.By("createdAt", false)
}

// ListPaginated backs the real `GET /venues` list page -- List above stays
// as-is for the venue-picker (attach-to-project) dropdown and the bulk
// import's prefetch, both of which need the full roster in one call.
func (r *MySQLVenueRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, f domain.VenueListFilter, sort pagination.Sort) ([]domain.Venue, int64, error) {
	where, args := venueWhere(tenantID, f)

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM venues`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+venueColumns+` FROM venues`+where+sort.OrderBy("id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var venues []domain.Venue
	for rows.Next() {
		v, err := scanVenue(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		venues = append(venues, *v)
	}
	return venues, total, rows.Err()
}

// BulkSetActive activates or deactivates several venues of the tenant in one
// transaction (POST /venues/bulk).
func (r *MySQLVenueRepository) BulkSetActive(ctx context.Context, tenantID int64, ids []int64, active bool) (domain.BulkResult, error) {
	return bulkSetActive(ctx, r.db, "venues", tenantID, ids, active)
}

// venueWhere is the WHERE clause both the paginated list and the export apply.
func venueWhere(tenantID int64, f domain.VenueListFilter) (string, []interface{}) {
	where := ` WHERE tenant_id = ?`
	args := []interface{}{tenantID}
	if f.Search != "" {
		where += ` AND (name LIKE ? OR pic_name LIKE ? OR email LIKE ?)`
		like := "%" + utils.EscapeLike(f.Search) + "%"
		args = append(args, like, like, like)
	}
	if f.City != "" {
		where += ` AND city = ?`
		args = append(args, f.City)
	}
	if f.IsActive != nil {
		where += ` AND is_active = ?`
		args = append(args, *f.IsActive)
	}
	return where, args
}

// ListFiltered backs Export -- the same filters as ListPaginated (one WHERE
// builder), but unpaginated (Export always returns the whole matching set).
func (r *MySQLVenueRepository) ListFiltered(ctx context.Context, tenantID int64, f domain.VenueListFilter) ([]domain.Venue, error) {
	where, args := venueWhere(tenantID, f)
	rows, err := r.db.QueryContext(ctx, `SELECT `+venueColumns+` FROM venues`+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var venues []domain.Venue
	for rows.Next() {
		v, err := scanVenue(rows.Scan)
		if err != nil {
			return nil, err
		}
		venues = append(venues, *v)
	}
	return venues, rows.Err()
}

func (r *MySQLVenueRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Venue, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+venueColumns+` FROM venues WHERE tenant_id = ? AND id = ? LIMIT 1`, tenantID, id)
	return scanVenue(row.Scan)
}

func (r *MySQLVenueRepository) Create(ctx context.Context, venue *domain.Venue) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO venues (tenant_id, name, pic_name, phone_pic, phone_venue, email, address, city,
		 rental_price, charge, capacity, facilities, social_media, notes, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		venue.TenantID, venue.Name, venue.PICName, venue.PhonePIC, venue.PhoneVenue, venue.Email, venue.Address, venue.City,
		venue.RentalPrice, venue.Charge, venue.Capacity, venue.Facilities, venue.SocialMedia, venue.Notes, venue.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	venue.ID = id
	return nil
}

// CreateBatch inserts multiple venues in one multi-row INSERT statement --
// used only by the bulk-import flow (§7 of PLAN.md's Revisi 2), which has
// already chunked the caller's slice to a safe size before calling this.
// IDs are not read back into the caller's slice: the import flow has no
// further use for them once committed.
func (r *MySQLVenueRepository) CreateBatch(ctx context.Context, venues []domain.Venue) error {
	if len(venues) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString(`INSERT INTO venues (tenant_id, name, pic_name, phone_pic, phone_venue, email, address, city,
		 rental_price, charge, capacity, facilities, social_media, notes, is_active) VALUES `)
	args := make([]interface{}, 0, len(venues)*15)
	for i, v := range venues {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(`(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		args = append(args, v.TenantID, v.Name, v.PICName, v.PhonePIC, v.PhoneVenue, v.Email, v.Address, v.City,
			v.RentalPrice, v.Charge, v.Capacity, v.Facilities, v.SocialMedia, v.Notes, v.IsActive)
	}
	_, err := r.db.ExecContext(ctx, sb.String(), args...)
	return err
}

func (r *MySQLVenueRepository) Update(ctx context.Context, venue *domain.Venue) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE venues SET name = ?, pic_name = ?, phone_pic = ?, phone_venue = ?, email = ?, address = ?, city = ?,
		 rental_price = ?, charge = ?, capacity = ?, facilities = ?, social_media = ?, notes = ?
		 WHERE tenant_id = ? AND id = ?`,
		venue.Name, venue.PICName, venue.PhonePIC, venue.PhoneVenue, venue.Email, venue.Address, venue.City,
		venue.RentalPrice, venue.Charge, venue.Capacity, venue.Facilities, venue.SocialMedia, venue.Notes,
		venue.TenantID, venue.ID,
	)
	return err
}

func (r *MySQLVenueRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE venues SET is_active = ? WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}

func (r *MySQLVenueRepository) UpdateAttachment(ctx context.Context, tenantID, id int64, path, mimeType *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE venues SET attachment_path = ?, attachment_mime_type = ? WHERE tenant_id = ? AND id = ?`, path, mimeType, tenantID, id)
	return err
}

// Delete permanently removes a venue -- a deliberate, guarded exception to
// this codebase's soft-state convention (PLAN.md's hard-delete plan:
// informed-consent, not a blocking precondition). Never touches
// `projects.venue_id`, which has no SQL FK to this table (cross-module by
// design) -- left to degrade gracefully via ProjectService.GetVenue's
// graceful-not-found handling, exactly the tradeoff the confirmation dialog
// exists to make explicit.
func (r *MySQLVenueRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM venues WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}
