package application

import (
	"context"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type VendorRepository interface {
	List(ctx context.Context, tenantID int64, categoryID *int64) ([]domain.Vendor, error)
	ListPaginated(ctx context.Context, tenantID int64, categoryID *int64, params pagination.Params, search string) ([]domain.Vendor, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Vendor, error)
	Create(ctx context.Context, vendor *domain.Vendor) error
	Update(ctx context.Context, vendor *domain.Vendor) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
}

type VendorService struct {
	repo         VendorRepository
	categoryRepo VendorCategoryRepository
}

func NewVendorService(repo VendorRepository, categoryRepo VendorCategoryRepository) *VendorService {
	return &VendorService{repo: repo, categoryRepo: categoryRepo}
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

type VendorInput struct {
	Name       string
	CategoryID int64
	PICName    string
	Phone      string
	Email      string
	Address    string
	Notes      string
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
	vendor := &domain.Vendor{
		TenantID: tenantID, CategoryID: input.CategoryID, Name: input.Name, PICName: input.PICName,
		Phone: input.Phone, Email: input.Email, Address: input.Address, Notes: input.Notes, IsActive: true,
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
	vendor.Name = input.Name
	vendor.CategoryID = input.CategoryID
	vendor.PICName = input.PICName
	vendor.Phone = input.Phone
	vendor.Email = input.Email
	vendor.Address = input.Address
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
