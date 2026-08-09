package application

import (
	"context"
	"fmt"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type VendorCategoryRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.VendorCategory, error)
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.VendorCategory, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.VendorCategory, error)
	Create(ctx context.Context, category *domain.VendorCategory) error
	Update(ctx context.Context, category *domain.VendorCategory) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	// Delete is a deliberate, guarded exception to this codebase's soft-state
	// convention (PLAN.md's hard-delete plan) -- only ever reached after
	// VendorCategoryService.Delete confirms zero vendors reference the row.
	Delete(ctx context.Context, tenantID, id int64) error
}

// VendorCategoryUsageChecker is the narrow slice of VendorRepository this
// service needs to guard hard delete -- same-module (no cross-module call
// needed, unlike Vendor/Venue/Staff's hard delete, since vendors.category_id
// is a real same-module SQL FK, see PLAN.md).
type VendorCategoryUsageChecker interface {
	CountByCategoryID(ctx context.Context, tenantID, categoryID int64) (int64, error)
}

type VendorCategoryService struct {
	repo    VendorCategoryRepository
	vendors VendorCategoryUsageChecker
}

func NewVendorCategoryService(repo VendorCategoryRepository, vendors VendorCategoryUsageChecker) *VendorCategoryService {
	return &VendorCategoryService{repo: repo, vendors: vendors}
}

func (s *VendorCategoryService) List(ctx context.Context, tenantID int64) ([]domain.VendorCategory, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *VendorCategoryService) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.VendorCategory, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, params, search)
}

func (s *VendorCategoryService) Get(ctx context.Context, tenantID, id int64) (*domain.VendorCategory, error) {
	category, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, apperror.NotFound("Kategori vendor tidak ditemukan")
	}
	return category, nil
}

type VendorCategoryInput struct {
	Name        string
	Description string
}

func (s *VendorCategoryService) Create(ctx context.Context, tenantID int64, input VendorCategoryInput) (*domain.VendorCategory, error) {
	category := &domain.VendorCategory{TenantID: tenantID, Name: input.Name, Description: input.Description, IsActive: true}
	if err := s.repo.Create(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *VendorCategoryService) Update(ctx context.Context, tenantID, id int64, input VendorCategoryInput) (*domain.VendorCategory, error) {
	category, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	category.Name = input.Name
	category.Description = input.Description
	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

// defaultCategoryTemplate is the checklist a new tenant starts with, so the
// Vendor Categories page isn't blank by default — the near-universal vendor
// types every WO deals with. A tenant that doesn't need one can deactivate it
// via SetActive; there's no delete, same as project milestones.
//
// "Venue" was removed from this list (ADR-0016) — it's now its own directory
// (see vendors/domain/venue.go), not a vendor category. Existing tenants'
// already-seeded "Venue" category is deactivated by migration
// 000022_migrate_venue_vendor_data, not by this code change.
var defaultCategoryTemplate = []struct {
	Name        string
	Description string
}{
	{"Katering", "Penyedia konsumsi untuk tamu dan keluarga"},
	{"Dekorasi", "Dekorasi pelaminan, panggung, dan area acara"},
	{"Fotografi & Videografi", "Dokumentasi foto dan video acara"},
	{"MUA & Hairdo", "Make up artist dan penata rambut pengantin"},
	{"Busana Pengantin", "Gaun, jas, dan busana adat pengantin"},
	{"MC & Entertainment", "Pembawa acara, band, atau hiburan lainnya"},
	{"Percetakan", "Undangan, souvenir, dan kebutuhan cetak lainnya"},
}

// SeedDefaultCategories is called once, from `platform`'s tenant registration
// flow (via vendors/contracts), to give a newly registered tenant a starting
// set of vendor categories instead of an empty list.
func (s *VendorCategoryService) SeedDefaultCategories(ctx context.Context, tenantID int64) error {
	for _, tmpl := range defaultCategoryTemplate {
		category := &domain.VendorCategory{TenantID: tenantID, Name: tmpl.Name, Description: tmpl.Description, IsActive: true}
		if err := s.repo.Create(ctx, category); err != nil {
			return err
		}
	}
	return nil
}

// Delete permanently removes a vendor category -- an explicit, guarded
// exception to this codebase's soft-state convention (PLAN.md's hard-delete
// plan). Unlike Vendor/Venue/Staff, this is a hard BLOCK, never a
// confirm-and-proceed dialog: vendors.category_id is a real, same-module SQL
// FK on a NOT NULL column, so a vendor row cannot exist without a category --
// there's no "orphaned but readable" state to let the user opt into.
func (s *VendorCategoryService) Delete(ctx context.Context, tenantID, id int64) error {
	if _, err := s.Get(ctx, tenantID, id); err != nil {
		return err
	}
	count, err := s.vendors.CountByCategoryID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperror.Conflict(fmt.Sprintf("Kategori masih digunakan oleh %d vendor dan tidak dapat dihapus. Ubah kategori pada vendor tersebut terlebih dahulu.", count))
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *VendorCategoryService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.VendorCategory, error) {
	category, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	category.IsActive = isActive
	return category, nil
}
