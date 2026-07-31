package application

import (
	"context"
	"encoding/base64"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/pagination"
)

// vendorImportRowCap/vendorImportBatchSize mirror Venue's own bulk-import
// caps exactly (PLAN.md's "Vendor Field Adjustment" §7) -- same reasoning,
// same numbers.
const (
	vendorImportRowCap    = 1000
	vendorImportBatchSize = 200
)

// maxVendorAttachmentDecodedSize/allowedVendorAttachmentMimeTypes mirror
// Venue's own single-attachment slot exactly -- a document or a photo, same
// size cap and mime whitelist.
const maxVendorAttachmentDecodedSize = 15 * 1024 * 1024

var allowedVendorAttachmentMimeTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/webp": true, "application/pdf": true,
}

type VendorRepository interface {
	List(ctx context.Context, tenantID int64, categoryID *int64) ([]domain.Vendor, error)
	ListPaginated(ctx context.Context, tenantID int64, categoryID *int64, params pagination.Params, search string) ([]domain.Vendor, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Vendor, error)
	Create(ctx context.Context, vendor *domain.Vendor) error
	CreateBatch(ctx context.Context, vendors []domain.Vendor) error
	Update(ctx context.Context, vendor *domain.Vendor) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	UpdateAttachment(ctx context.Context, tenantID, id int64, path, mimeType *string) error
}

// VendorObjectStorage is the narrow slice of internal/shared/storage.Client
// this service depends on -- package-local per this module's own boundary,
// same idiom as VenueObjectStorage in venue_service.go.
type VendorObjectStorage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type VendorService struct {
	repo         VendorRepository
	categoryRepo VendorCategoryRepository
	storage      VendorObjectStorage
	buildKey     func(tenantID, projectID, category, filename string) string
}

func NewVendorService(repo VendorRepository, categoryRepo VendorCategoryRepository, storage VendorObjectStorage, buildKey func(string, string, string, string) string) *VendorService {
	return &VendorService{repo: repo, categoryRepo: categoryRepo, storage: storage, buildKey: buildKey}
}

func (s *VendorService) List(ctx context.Context, tenantID int64, categoryID *int64) ([]domain.Vendor, error) {
	return s.repo.List(ctx, tenantID, categoryID)
}

func (s *VendorService) ListPaginated(ctx context.Context, tenantID int64, categoryID *int64, params pagination.Params, search string) ([]domain.Vendor, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, categoryID, params, search)
}

func (s *VendorService) Get(ctx context.Context, tenantID, id int64) (*domain.Vendor, error) {
	vendor, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if vendor == nil {
		return nil, apperror.NotFound("Vendor tidak ditemukan")
	}
	return vendor, nil
}

// VendorInput is shared by Create and Update. City/PriceAkad/PriceAkadResepsi
// are enforced as required only by the frontend's create schema (see the
// "DATA VENDOR" slide plan) -- this layer doesn't re-check non-emptiness,
// same as every other "required" text field in this codebase. It DOES
// validate City against domain.IsValidCity whenever non-empty, matching
// Venue's validateVenueCity rule exactly.
type VendorInput struct {
	Name             string
	CategoryID       int64
	PICName          string
	Phone            string
	Email            string
	SocialMedia      string
	City             string
	Address          string
	PriceAkad        *int64
	PriceAkadResepsi *int64
	Notes            string
}

func validateVendorCity(city string) error {
	if city != "" && !domain.IsValidCity(city) {
		return apperror.Validation("Kota tidak valid", map[string][]string{"city": {"Pilih kota dari daftar yang tersedia"}})
	}
	return nil
}

func (s *VendorService) validateCategory(ctx context.Context, tenantID, categoryID int64) error {
	category, err := s.categoryRepo.FindByID(ctx, tenantID, categoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return apperror.Validation("Kategori vendor tidak valid", map[string][]string{"categoryId": {"Kategori vendor tidak ditemukan"}})
	}
	return nil
}

func (s *VendorService) Create(ctx context.Context, tenantID int64, input VendorInput) (*domain.Vendor, error) {
	if err := s.validateCategory(ctx, tenantID, input.CategoryID); err != nil {
		return nil, err
	}
	if err := validateVendorCity(input.City); err != nil {
		return nil, err
	}
	vendor := &domain.Vendor{
		TenantID: tenantID, CategoryID: input.CategoryID, Name: input.Name, PICName: input.PICName,
		Phone: input.Phone, Email: stringPtrOrNil(input.Email), SocialMedia: stringPtrOrNil(input.SocialMedia),
		City: stringPtrOrNil(input.City), Address: stringPtrOrNil(input.Address),
		PriceAkad: input.PriceAkad, PriceAkadResepsi: input.PriceAkadResepsi, Notes: input.Notes, IsActive: true,
	}
	if err := s.repo.Create(ctx, vendor); err != nil {
		return nil, err
	}
	return vendor, nil
}

func (s *VendorService) Update(ctx context.Context, tenantID, id int64, input VendorInput) (*domain.Vendor, error) {
	vendor, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateCategory(ctx, tenantID, input.CategoryID); err != nil {
		return nil, err
	}
	if err := validateVendorCity(input.City); err != nil {
		return nil, err
	}
	vendor.Name = input.Name
	vendor.CategoryID = input.CategoryID
	vendor.PICName = input.PICName
	vendor.Phone = input.Phone
	vendor.Email = stringPtrOrNil(input.Email)
	vendor.SocialMedia = stringPtrOrNil(input.SocialMedia)
	vendor.City = stringPtrOrNil(input.City)
	vendor.Address = stringPtrOrNil(input.Address)
	vendor.PriceAkad = input.PriceAkad
	vendor.PriceAkadResepsi = input.PriceAkadResepsi
	vendor.Notes = input.Notes
	if err := s.repo.Update(ctx, vendor); err != nil {
		return nil, err
	}
	return vendor, nil
}

func (s *VendorService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.Vendor, error) {
	vendor, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	vendor.IsActive = isActive
	return vendor, nil
}

// --- Attachment (single slot per vendor -- document or photo) ---

type UploadVendorAttachmentInput struct {
	FileName   string
	MimeType   string
	Base64Data string
}

func (s *VendorService) UploadAttachment(ctx context.Context, tenantID, vendorID int64, input UploadVendorAttachmentInput) (*domain.Vendor, error) {
	if !allowedVendorAttachmentMimeTypes[input.MimeType] {
		return nil, apperror.Validation("Format lampiran tidak didukung", map[string][]string{"mimeType": {"Gunakan PNG, JPEG, WebP, atau PDF"}})
	}
	vendor, err := s.Get(ctx, tenantID, vendorID)
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
	if len(decoded) > maxVendorAttachmentDecodedSize {
		return nil, apperror.Validation("Ukuran lampiran terlalu besar", map[string][]string{"base64Data": {"Maksimal 15 MB"}})
	}

	// compress.Image passes non-image mimeTypes (e.g. application/pdf)
	// through unchanged -- see internal/shared/compress's own doc comment.
	processed, err := compress.Image(decoded, input.MimeType)
	if err != nil {
		return nil, apperror.Internal("Gagal memproses lampiran")
	}

	key := s.buildKey(
		strconv.FormatInt(tenantID, 10), strconv.FormatInt(vendorID, 10), "vendor-attachment",
		uuid.NewString()+"-"+sanitizeVenueFileName(input.FileName),
	)
	if _, err := s.storage.Save(ctx, key, processed, input.MimeType); err != nil {
		return nil, apperror.Internal("Gagal mengunggah lampiran ke object storage")
	}

	mimeType := input.MimeType
	if err := s.repo.UpdateAttachment(ctx, tenantID, vendorID, &key, &mimeType); err != nil {
		return nil, err
	}
	vendor.AttachmentPath = &key
	vendor.AttachmentMimeType = &mimeType
	return vendor, nil
}

func (s *VendorService) DownloadAttachment(ctx context.Context, tenantID, vendorID int64) (*domain.Vendor, io.ReadCloser, error) {
	vendor, err := s.Get(ctx, tenantID, vendorID)
	if err != nil {
		return nil, nil, err
	}
	if vendor.AttachmentPath == nil {
		return nil, nil, apperror.NotFound("Vendor belum memiliki lampiran")
	}
	reader, err := s.storage.Open(ctx, *vendor.AttachmentPath)
	if err != nil {
		return nil, nil, apperror.Internal("Gagal mengambil lampiran dari object storage")
	}
	return vendor, reader, nil
}

// --- Bulk import (Excel) ---

// VendorImportRow is one parsed spreadsheet row, handed up from the handler
// (which owns the excelize-specific parsing) -- this layer stays format
// agnostic, same separation as Venue's VenueImportRow.
type VendorImportRow struct {
	Name             string
	CategoryName     string
	PICName          string
	Phone            string
	Email            string
	SocialMedia      string
	City             string
	Address          string
	PriceAkad        *int64
	PriceAkadResepsi *int64
	Notes            string
}

type VendorImportRowError struct {
	Row     int
	Message string
}

type VendorImportResult struct {
	InsertedCount int
	UpdatedCount  int
	Errors        []VendorImportRowError
}

// vendorImportKey is the case-insensitive (name, city, categoryId) dedupe key
// -- unlike Venue's (name, city), Vendor keeps CategoryID as a real identity
// dimension (PLAN.md's "Vendor Field Adjustment" §1), so two vendors sharing
// a name+city under different categories are legitimately distinct records.
func vendorImportKey(name, city string, categoryID int64) string {
	return strings.ToLower(name) + "|" + strings.ToLower(city) + "|" + strconv.FormatInt(categoryID, 10)
}

type vendorImportUpdateItem struct {
	rowNum int
	vendor domain.Vendor
}

// ImportVendors mirrors Venue's ImportVenues exactly (one prefetch of the
// tenant's full roster, in-memory dedupe map, batched inserts, per-row
// updates, no all-file transaction -- see PLAN.md §7), plus one extra
// prefetch: the tenant's vendor_categories, resolved once into a
// name-\>ID map so each row's free-text "Kategori" column can be validated
// without a query per row.
func (s *VendorService) ImportVendors(ctx context.Context, tenantID int64, rows []VendorImportRow) (VendorImportResult, error) {
	if len(rows) > vendorImportRowCap {
		return VendorImportResult{}, apperror.Validation("Berkas terlalu besar", map[string][]string{"file": {"Maksimal 1000 baris data"}})
	}

	existing, err := s.repo.List(ctx, tenantID, nil)
	if err != nil {
		return VendorImportResult{}, err
	}
	byKey := make(map[string]domain.Vendor, len(existing))
	for _, v := range existing {
		city := ""
		if v.City != nil {
			city = *v.City
		}
		byKey[vendorImportKey(v.Name, city, v.CategoryID)] = v
	}

	categories, err := s.categoryRepo.List(ctx, tenantID)
	if err != nil {
		return VendorImportResult{}, err
	}
	categoryByName := make(map[string]int64, len(categories))
	for _, c := range categories {
		categoryByName[strings.ToLower(c.Name)] = c.ID
	}

	var result VendorImportResult
	var toInsert []domain.Vendor
	var toUpdate []vendorImportUpdateItem
	pendingInsertIndex := make(map[string]int)

	for i, row := range rows {
		rowNum := i + 2 // header occupies row 1
		if row.Name == "" || row.PICName == "" || row.Phone == "" || row.City == "" || row.CategoryName == "" ||
			row.PriceAkad == nil || *row.PriceAkad < 1 || row.PriceAkadResepsi == nil || *row.PriceAkadResepsi < 1 {
			result.Errors = append(result.Errors, VendorImportRowError{Row: rowNum, Message: "Nama Vendor, Kategori, Nama PIC, No Tlp Vendor, Kota, Harga Akad, dan Harga Akad+Resepsi wajib diisi dengan nilai yang valid"})
			continue
		}
		if !domain.IsValidCity(row.City) {
			result.Errors = append(result.Errors, VendorImportRowError{Row: rowNum, Message: "Kota tidak valid: " + row.City})
			continue
		}
		categoryID, ok := categoryByName[strings.ToLower(row.CategoryName)]
		if !ok {
			result.Errors = append(result.Errors, VendorImportRowError{Row: rowNum, Message: "Kategori tidak ditemukan: " + row.CategoryName})
			continue
		}

		vendor := domain.Vendor{
			TenantID: tenantID, CategoryID: categoryID, Name: row.Name, PICName: row.PICName, Phone: row.Phone,
			Email: stringPtrOrNil(row.Email), SocialMedia: stringPtrOrNil(row.SocialMedia), City: stringPtrOrNil(row.City),
			Address: stringPtrOrNil(row.Address), PriceAkad: row.PriceAkad, PriceAkadResepsi: row.PriceAkadResepsi,
			Notes: row.Notes, IsActive: true,
		}

		key := vendorImportKey(row.Name, row.City, categoryID)
		if match, ok := byKey[key]; ok {
			// attachment_path/attachment_mime_type/is_active are never
			// touched by an import-triggered update -- same reasoning as
			// Venue's ImportVenues.
			vendor.ID = match.ID
			vendor.IsActive = match.IsActive
			toUpdate = append(toUpdate, vendorImportUpdateItem{rowNum: rowNum, vendor: vendor})
		} else if idx, ok := pendingInsertIndex[key]; ok {
			// Same (name, city, category) as an earlier row in this same
			// file, neither matching an existing DB vendor -- keep exactly
			// one insert, the later row's data wins (last-row-wins).
			toInsert[idx] = vendor
		} else {
			pendingInsertIndex[key] = len(toInsert)
			toInsert = append(toInsert, vendor)
		}
	}

	for start := 0; start < len(toInsert); start += vendorImportBatchSize {
		end := start + vendorImportBatchSize
		if end > len(toInsert) {
			end = len(toInsert)
		}
		chunk := toInsert[start:end]
		if err := s.repo.CreateBatch(ctx, chunk); err != nil {
			result.Errors = append(result.Errors, VendorImportRowError{
				Row:     0,
				Message: "Gagal menyimpan sekumpulan baris baru (baris data ke-" + strconv.Itoa(start+1) + " s.d. " + strconv.Itoa(end) + "): " + err.Error(),
			})
			continue
		}
		result.InsertedCount += len(chunk)
	}

	for _, item := range toUpdate {
		vendor := item.vendor
		if err := s.repo.Update(ctx, &vendor); err != nil {
			result.Errors = append(result.Errors, VendorImportRowError{Row: item.rowNum, Message: "Gagal memperbarui vendor: " + err.Error()})
			continue
		}
		result.UpdatedCount++
	}

	return result, nil
}
