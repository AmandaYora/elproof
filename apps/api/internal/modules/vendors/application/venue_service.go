package application

import (
	"context"
	"encoding/base64"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/pagination"
)

// venueImportRowCap/venueImportBatchSize bound a single import request (§7 of
// PLAN.md's Venue Extraction Revisi 2): the row cap keeps one pathological
// file from tying up a request indefinitely (same spirit as this codebase's
// existing per-upload size caps, just measured in rows instead of bytes); the
// batch size keeps a single multi-row INSERT from growing unbounded and
// staying clear of MySQL's max packet size.
const (
	venueImportRowCap    = 1000
	venueImportBatchSize = 200
)

// maxVenueAttachmentDecodedSize matches evidence's cap (ADR-0010) -- this is
// a real document/photo, not a small header image like the tenant logo.
const maxVenueAttachmentDecodedSize = 15 * 1024 * 1024

// allowedVenueAttachmentMimeTypes accepts images and PDF -- a venue's single
// attachment can be either a photo or a contract-style document.
var allowedVenueAttachmentMimeTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/webp": true, "application/pdf": true,
}

type VenueRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.Venue, error)
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.Venue, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Venue, error)
	Create(ctx context.Context, venue *domain.Venue) error
	CreateBatch(ctx context.Context, venues []domain.Venue) error
	Update(ctx context.Context, venue *domain.Venue) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	UpdateAttachment(ctx context.Context, tenantID, id int64, path, mimeType *string) error
}

// VenueObjectStorage is the narrow slice of internal/shared/storage.Client
// this service depends on -- package-local per this module's own boundary,
// same idiom as evidence_service.go/tenant_service.go's own ObjectStorage.
type VenueObjectStorage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type VenueService struct {
	repo     VenueRepository
	storage  VenueObjectStorage
	buildKey func(tenantID, projectID, category, filename string) string
}

func NewVenueService(repo VenueRepository, storage VenueObjectStorage, buildKey func(string, string, string, string) string) *VenueService {
	return &VenueService{repo: repo, storage: storage, buildKey: buildKey}
}

func (s *VenueService) List(ctx context.Context, tenantID int64) ([]domain.Venue, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *VenueService) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.Venue, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, params, search)
}

func (s *VenueService) Get(ctx context.Context, tenantID, id int64) (*domain.Venue, error) {
	venue, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if venue == nil {
		return nil, apperror.NotFound("Venue tidak ditemukan")
	}
	return venue, nil
}

// VenueInput is shared by Create and Update, mirroring VendorInput's shape.
// City/RentalPrice are enforced as required only by the frontend's create
// schema (see ADR-0016) -- this layer doesn't re-check non-emptiness, same as
// every other "required" text field in this codebase (e.g. VendorInput.Name).
// It DOES validate City against domain.IsValidCity whenever non-empty,
// since that's a fixed-enum correctness check, not a mere required-ness rule.
type VenueInput struct {
	Name        string
	PICName     string
	PhonePIC    string
	PhoneVenue  string
	Email       string
	Address     string
	City        string
	RentalPrice *int64
	Charge      *int64
	Capacity    *int
	Facilities  string
	SocialMedia string
	Notes       string
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func validateVenueCity(city string) error {
	if city != "" && !domain.IsValidCity(city) {
		return apperror.Validation("Kota tidak valid", map[string][]string{"city": {"Pilih kota dari daftar yang tersedia"}})
	}
	return nil
}

func (s *VenueService) Create(ctx context.Context, tenantID int64, input VenueInput) (*domain.Venue, error) {
	if err := validateVenueCity(input.City); err != nil {
		return nil, err
	}
	venue := &domain.Venue{
		TenantID: tenantID, Name: input.Name, PICName: input.PICName, PhonePIC: input.PhonePIC,
		PhoneVenue: stringPtrOrNil(input.PhoneVenue), Email: stringPtrOrNil(input.Email),
		Address: stringPtrOrNil(input.Address), City: stringPtrOrNil(input.City),
		RentalPrice: input.RentalPrice, Charge: input.Charge, Capacity: input.Capacity,
		Facilities: stringPtrOrNil(input.Facilities), SocialMedia: stringPtrOrNil(input.SocialMedia),
		Notes: input.Notes, IsActive: true,
	}
	if err := s.repo.Create(ctx, venue); err != nil {
		return nil, err
	}
	return venue, nil
}

func (s *VenueService) Update(ctx context.Context, tenantID, id int64, input VenueInput) (*domain.Venue, error) {
	if err := validateVenueCity(input.City); err != nil {
		return nil, err
	}
	venue, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	venue.Name = input.Name
	venue.PICName = input.PICName
	venue.PhonePIC = input.PhonePIC
	venue.PhoneVenue = stringPtrOrNil(input.PhoneVenue)
	venue.Email = stringPtrOrNil(input.Email)
	venue.Address = stringPtrOrNil(input.Address)
	venue.City = stringPtrOrNil(input.City)
	venue.RentalPrice = input.RentalPrice
	venue.Charge = input.Charge
	venue.Capacity = input.Capacity
	venue.Facilities = stringPtrOrNil(input.Facilities)
	venue.SocialMedia = stringPtrOrNil(input.SocialMedia)
	venue.Notes = input.Notes
	if err := s.repo.Update(ctx, venue); err != nil {
		return nil, err
	}
	return venue, nil
}

func (s *VenueService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.Venue, error) {
	venue, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	venue.IsActive = isActive
	return venue, nil
}

// --- Attachment (single slot per venue -- document or photo) ---

type UploadVenueAttachmentInput struct {
	FileName   string
	MimeType   string
	Base64Data string
}

func (s *VenueService) UploadAttachment(ctx context.Context, tenantID, venueID int64, input UploadVenueAttachmentInput) (*domain.Venue, error) {
	if !allowedVenueAttachmentMimeTypes[input.MimeType] {
		return nil, apperror.Validation("Format lampiran tidak didukung", map[string][]string{"mimeType": {"Gunakan PNG, JPEG, WebP, atau PDF"}})
	}
	venue, err := s.Get(ctx, tenantID, venueID)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(input.Base64Data)
	if err != nil {
		return nil, apperror.Validation("Data lampiran tidak valid", map[string][]string{"base64Data": {"Gagal membaca data file"}})
	}
	if len(decoded) == 0 {
		return nil, apperror.Validation("File lampiran kosong", map[string][]string{"base64Data": {"File tidak boleh kosong"}})
	}
	if len(decoded) > maxVenueAttachmentDecodedSize {
		return nil, apperror.Validation("Ukuran lampiran terlalu besar", map[string][]string{"base64Data": {"Maksimal 15 MB"}})
	}

	// compress.Image passes non-image mimeTypes (e.g. application/pdf)
	// through unchanged -- see internal/shared/compress's own doc comment.
	processed, err := compress.Image(decoded, input.MimeType)
	if err != nil {
		return nil, apperror.Internal("Gagal memproses lampiran")
	}

	key := s.buildKey(
		strconv.FormatInt(tenantID, 10), strconv.FormatInt(venueID, 10), "venue-attachment",
		uuid.NewString()+"-"+sanitizeVenueFileName(input.FileName),
	)
	if _, err := s.storage.Save(ctx, key, processed, input.MimeType); err != nil {
		return nil, apperror.Internal("Gagal mengunggah lampiran ke object storage")
	}

	mimeType := input.MimeType
	if err := s.repo.UpdateAttachment(ctx, tenantID, venueID, &key, &mimeType); err != nil {
		return nil, err
	}
	venue.AttachmentPath = &key
	venue.AttachmentMimeType = &mimeType
	return venue, nil
}

func (s *VenueService) DownloadAttachment(ctx context.Context, tenantID, venueID int64) (*domain.Venue, io.ReadCloser, error) {
	venue, err := s.Get(ctx, tenantID, venueID)
	if err != nil {
		return nil, nil, err
	}
	if venue.AttachmentPath == nil {
		return nil, nil, apperror.NotFound("Venue belum memiliki lampiran")
	}
	reader, err := s.storage.Open(ctx, *venue.AttachmentPath)
	if err != nil {
		return nil, nil, apperror.Internal("Gagal mengambil lampiran dari object storage")
	}
	return venue, reader, nil
}

var unsafeVenueFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeVenueFileName(name string) string {
	cleaned := unsafeVenueFileNameChars.ReplaceAllString(name, "-")
	if cleaned == "" {
		return "file"
	}
	return cleaned
}

// --- Bulk import (Excel) ---

// VenueImportRow is one parsed spreadsheet row, handed up from the handler
// (which owns the excelize-specific parsing -- this layer stays format
// agnostic, same separation as every other "presentation decodes, service
// validates" upload in this codebase).
type VenueImportRow struct {
	Name        string
	PICName     string
	PhonePIC    string
	PhoneVenue  string
	Email       string
	Address     string
	City        string
	RentalPrice *int64
	Charge      *int64
	Capacity    *int
	Facilities  string
	SocialMedia string
	Notes       string
}

type VenueImportRowError struct {
	Row     int
	Message string
}

type VenueImportResult struct {
	InsertedCount int
	UpdatedCount  int
	Errors        []VenueImportRowError
}

// venueImportKey is the case-insensitive (name, city) dedupe key shared by
// the prefetch map and every row's lookup -- see PLAN.md §7.
func venueImportKey(name, city string) string {
	return strings.ToLower(name) + "|" + strings.ToLower(city)
}

type venueImportUpdateItem struct {
	rowNum int
	venue  domain.Venue
}

// ImportVenues implements PLAN.md §7's O(1)-round-trip bulk upsert: one
// prefetch of the tenant's full roster (already just `WHERE tenant_id = ?`
// against the indexed column), an in-memory map keyed by venueImportKey for
// every row's duplicate check, batched multi-row inserts for brand-new rows,
// and one indexed-PK UPDATE per matched row -- no per-row SELECT anywhere.
// Neither path runs inside one all-file transaction, so a row/chunk that
// fails at the database level is reported as a row error without undoing
// everything else already committed -- partial success is a real guarantee,
// not an accident of skipping validation-only failures.
func (s *VenueService) ImportVenues(ctx context.Context, tenantID int64, rows []VenueImportRow) (VenueImportResult, error) {
	if len(rows) > venueImportRowCap {
		return VenueImportResult{}, apperror.Validation("Berkas terlalu besar", map[string][]string{"file": {"Maksimal 1000 baris data"}})
	}

	existing, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return VenueImportResult{}, err
	}
	byKey := make(map[string]domain.Venue, len(existing))
	for _, v := range existing {
		city := ""
		if v.City != nil {
			city = *v.City
		}
		byKey[venueImportKey(v.Name, city)] = v
	}

	var result VenueImportResult
	var toInsert []domain.Venue
	var toUpdate []venueImportUpdateItem
	// pendingInsertIndex tracks keys already staged in toInsert during THIS
	// run -- byKey alone only reflects the DB's state before the import
	// started, so without this a file with two brand-new rows sharing the
	// same (name, city) would insert two duplicate venues instead of one
	// (the same upsert-by-name+city rule the user asked for must also apply
	// within a single file, not just against pre-existing rows).
	pendingInsertIndex := make(map[string]int)

	for i, row := range rows {
		rowNum := i + 2 // header occupies row 1
		if row.Name == "" || row.PICName == "" || row.PhonePIC == "" || row.City == "" {
			result.Errors = append(result.Errors, VenueImportRowError{Row: rowNum, Message: "Nama Venue, Nama PIC, No Tlp PIC, dan Kota wajib diisi"})
			continue
		}
		if !domain.IsValidCity(row.City) {
			result.Errors = append(result.Errors, VenueImportRowError{Row: rowNum, Message: "Kota tidak valid: " + row.City})
			continue
		}

		venue := domain.Venue{
			TenantID: tenantID, Name: row.Name, PICName: row.PICName, PhonePIC: row.PhonePIC,
			PhoneVenue: stringPtrOrNil(row.PhoneVenue), Email: stringPtrOrNil(row.Email),
			Address: stringPtrOrNil(row.Address), City: stringPtrOrNil(row.City),
			RentalPrice: row.RentalPrice, Charge: row.Charge, Capacity: row.Capacity,
			Facilities: stringPtrOrNil(row.Facilities), SocialMedia: stringPtrOrNil(row.SocialMedia),
			Notes: row.Notes, IsActive: true,
		}

		key := venueImportKey(row.Name, row.City)
		if match, ok := byKey[key]; ok {
			// attachment_path/attachment_mime_type/is_active are never
			// touched by an import-triggered update -- the row has no
			// attachment data and no active-status column, so carrying the
			// existing venue's values forward (not clearing them) is the
			// only sane behavior. repo.Update's own SQL already leaves
			// attachment columns alone; IsActive is copied explicitly since
			// domain.Venue is otherwise fully overwritten from the row.
			venue.ID = match.ID
			venue.IsActive = match.IsActive
			toUpdate = append(toUpdate, venueImportUpdateItem{rowNum: rowNum, venue: venue})
		} else if idx, ok := pendingInsertIndex[key]; ok {
			// Same (name, city) as an earlier row in this same file, neither
			// matching an existing DB venue -- keep exactly one insert, the
			// later row's data wins (last-row-wins, same effective semantics
			// as two rows both matching an existing DB venue above).
			toInsert[idx] = venue
		} else {
			pendingInsertIndex[key] = len(toInsert)
			toInsert = append(toInsert, venue)
		}
	}

	for start := 0; start < len(toInsert); start += venueImportBatchSize {
		end := start + venueImportBatchSize
		if end > len(toInsert) {
			end = len(toInsert)
		}
		chunk := toInsert[start:end]
		if err := s.repo.CreateBatch(ctx, chunk); err != nil {
			result.Errors = append(result.Errors, VenueImportRowError{
				Row:     0,
				Message: "Gagal menyimpan sekumpulan baris baru (baris data ke-" + strconv.Itoa(start+1) + " s.d. " + strconv.Itoa(end) + "): " + err.Error(),
			})
			continue
		}
		result.InsertedCount += len(chunk)
	}

	for _, item := range toUpdate {
		venue := item.venue
		if err := s.repo.Update(ctx, &venue); err != nil {
			result.Errors = append(result.Errors, VenueImportRowError{Row: item.rowNum, Message: "Gagal memperbarui venue: " + err.Error()})
			continue
		}
		result.UpdatedCount++
	}

	return result, nil
}
