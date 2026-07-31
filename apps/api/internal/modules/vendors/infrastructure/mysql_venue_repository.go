package infrastructure

import (
	"context"
	"database/sql"
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

// ListPaginated backs the real `GET /venues` list page -- List above stays
// as-is for the venue-picker (attach-to-project) dropdown and the bulk
// import's prefetch, both of which need the full roster in one call.
func (r *MySQLVenueRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.Venue, int64, error) {
	countQuery := `SELECT COUNT(*) FROM venues WHERE tenant_id = ?`
	listQuery := `SELECT ` + venueColumns + ` FROM venues WHERE tenant_id = ?`
	args := []interface{}{tenantID}
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
