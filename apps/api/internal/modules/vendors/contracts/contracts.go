// Package contracts is the ONLY package other modules may import from vendors.
package contracts

import (
	"context"

	"elproof/internal/modules/vendors/application"
)

// Contracts is what other modules depend on to trigger vendors-owned
// behavior for their own principals — currently just seeding a new tenant's
// default vendor categories on registration (see `platform`'s TenantService).
type Contracts interface {
	SeedDefaultCategories(ctx context.Context, tenantID int64) error
}

type impl struct {
	categories *application.VendorCategoryService
}

func New(categories *application.VendorCategoryService) Contracts {
	return &impl{categories: categories}
}

func (c *impl) SeedDefaultCategories(ctx context.Context, tenantID int64) error {
	return c.categories.SeedDefaultCategories(ctx, tenantID)
}
