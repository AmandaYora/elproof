package application

import (
	"context"

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
}

type VendorCategoryService struct {
	repo VendorCategoryRepository
}

func NewVendorCategoryService(repo VendorCategoryRepository) *VendorCategoryService {
	return &VendorCategoryService{repo: repo}
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
