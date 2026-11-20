// Package application holds the offers module's use cases: the catalog of
// event kinds and packages (Tahap 1), later quotations.
package application

import (
	"context"
	"time"

	"elproof/internal/modules/offers/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/pagination"
)

// VendorDirectory is the slice of vendors' contract the catalog needs: is a
// venue, vendor, or category this workspace's (and active), the venue's
// capacity, and display names. vendorscontracts.Contracts satisfies it; an id
// of another workspace answers like a missing one (ADR-0004).
type VendorDirectory interface {
	GetVenueRef(ctx context.Context, tenantID, venueID int64) (vendorscontracts.VenueRef, error)
	GetVendorRef(ctx context.Context, tenantID, vendorID int64) (vendorscontracts.VendorRef, error)
	CategoryExists(ctx context.Context, tenantID, categoryID int64) (bool, error)
	VenueNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
	VendorNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
	CategoryNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
}

// EventKindRepository stores event kinds; every call is scoped to a tenant.
type EventKindRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.EventKind, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.EventKind, error)
	Create(ctx context.Context, k *domain.EventKind) error
	// Rename writes the name when the row's version is still expected (nil =
	// unconditionally); false means it was not (someone else changed it).
	Rename(ctx context.Context, tenantID, id int64, name string, expected *time.Time) (bool, error)
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	Delete(ctx context.Context, tenantID, id int64) error
	// PackageCounts is how many packages cover each kind of the tenant.
	PackageCounts(ctx context.Context, tenantID int64) (map[int64]int, error)
}

// PackageListFilter narrows GET /packages.
type PackageListFilter struct {
	Search   string
	IsActive *bool
}

// PackageRepository stores packages with their covered kinds and components;
// every call is scoped to a tenant.
type PackageRepository interface {
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, filter PackageListFilter, sort pagination.Sort) ([]domain.Package, int64, error)
	ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// FindByID loads the package with its kinds and components.
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Package, error)
	// NameTaken reports whether another package of the tenant (not selfID) has
	// this name, compared without regard to case.
	NameTaken(ctx context.Context, tenantID int64, name string, selfID int64) (bool, error)
	// Create inserts the package, its kinds, and its components in one
	// transaction.
	Create(ctx context.Context, p *domain.Package) error
	// Update rewrites the package, its kinds, and its components in one
	// transaction when its version is still expected (nil = unconditionally);
	// false means it was not.
	Update(ctx context.Context, p *domain.Package, expected *time.Time) (bool, error)
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	Delete(ctx context.Context, tenantID, id int64) error
	// VendorReferenced, VenueReferenced, CategoryReferenced answer whether a
	// package of the tenant still points at the directory row.
	VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error)
	VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error)
	CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error)
}
