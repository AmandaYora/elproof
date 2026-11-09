// Package contracts is the ONLY package other modules may import from vendors.
package contracts

import (
	"context"
	"strings"

	"elproof/internal/modules/vendors/application"
	"elproof/internal/shared/apperror"
)

// VenueSummary is the public-safe subset of a venue's data (ADR-0016) --
// deliberately excludes RentalPrice/Charge/PIC contact, which stay
// staff-only (read directly via GET /venues/{id}, never through this
// cross-module contract). Consumed by `projects` to resolve a project's
// attached venue_id for both its own Project Detail tab and Client Portal's
// Venue tab.
type VenueSummary struct {
	ID                   int64
	Name                 string
	Address              *string
	City                 *string
	Capacity             *int
	Facilities           *string
	SocialMedia          *string
	HasVisibleAttachment bool
}

// VenueAttachInfo is what `projects` needs to attach a venue to a project: its
// name (shown as the project's venue) and its current prices (the starting
// point of the project's own cost snapshot). Kept apart from VenueSummary,
// which is public-safe and reaches clients: prices stay staff-only.
type VenueAttachInfo struct {
	ID          int64
	Name        string
	RentalPrice int64
	Charge      int64
}

// Contracts is what other modules depend on to trigger vendors-owned
// behavior for their own principals — seeding a new tenant's default vendor
// categories on registration (`platform`), and resolving a project's venue_id
// into display data (`projects`, ADR-0016).
type Contracts interface {
	SeedDefaultCategories(ctx context.Context, tenantID int64) error
	GetVenueSummary(ctx context.Context, tenantID, venueID int64) (VenueSummary, error)
	// GetVenueForAttach is the venue's name and current prices, for attaching it
	// to a project. A venue of another workspace is NotFound, like a missing one.
	GetVenueForAttach(ctx context.Context, tenantID, venueID int64) (VenueAttachInfo, error)
	// VendorExists and CategoryExists answer whether the id is a vendor or a
	// vendor category of this workspace (another workspace's id is "no").
	VendorExists(ctx context.Context, tenantID, vendorID int64) (bool, error)
	CategoryExists(ctx context.Context, tenantID, categoryID int64) (bool, error)
	// VendorCategoryID is the category the vendor belongs to (NotFound for a
	// vendor of another workspace, like a missing one).
	VendorCategoryID(ctx context.Context, tenantID, vendorID int64) (int64, error)
	// VendorNames and CategoryNames resolve display names for the given ids in
	// one query, scoped to the workspace: an id that is not this workspace's is
	// absent from the map, so a name can never cross a tenant boundary.
	VendorNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
	CategoryNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
}

type impl struct {
	categories *application.VendorCategoryService
	venues     *application.VenueService
	vendors    *application.VendorService
}

func New(categories *application.VendorCategoryService, venues *application.VenueService, vendors *application.VendorService) Contracts {
	return &impl{categories: categories, venues: venues, vendors: vendors}
}

func (c *impl) VendorExists(ctx context.Context, tenantID, vendorID int64) (bool, error) {
	_, err := c.vendors.Get(ctx, tenantID, vendorID)
	return foundOrNot(err)
}

func (c *impl) VendorNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	return c.vendors.Names(ctx, tenantID, ids)
}

func (c *impl) CategoryNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	return c.categories.Names(ctx, tenantID, ids)
}

func (c *impl) CategoryExists(ctx context.Context, tenantID, categoryID int64) (bool, error) {
	_, err := c.categories.Get(ctx, tenantID, categoryID)
	return foundOrNot(err)
}

// foundOrNot turns a not-found error into a plain "no"; anything else is a real failure.
func foundOrNot(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
		return false, nil
	}
	return false, err
}

func (c *impl) SeedDefaultCategories(ctx context.Context, tenantID int64) error {
	return c.categories.SeedDefaultCategories(ctx, tenantID)
}

// GetVenueSummary is down to one query (c.venues.Get) since there's no more
// photo gallery to separately list -- HasVisibleAttachment is derived
// entirely from the venue row itself.
func (c *impl) GetVenueSummary(ctx context.Context, tenantID, venueID int64) (VenueSummary, error) {
	venue, err := c.venues.Get(ctx, tenantID, venueID)
	if err != nil {
		return VenueSummary{}, err
	}
	hasVisibleAttachment := venue.AttachmentPath != nil && venue.AttachmentMimeType != nil && strings.HasPrefix(*venue.AttachmentMimeType, "image/")
	return VenueSummary{
		ID: venue.ID, Name: venue.Name, Address: venue.Address, City: venue.City, Capacity: venue.Capacity,
		Facilities: venue.Facilities, SocialMedia: venue.SocialMedia, HasVisibleAttachment: hasVisibleAttachment,
	}, nil
}

// GetVenueForAttach reads the venue inside the workspace; an unset price is 0.
func (c *impl) GetVenueForAttach(ctx context.Context, tenantID, venueID int64) (VenueAttachInfo, error) {
	venue, err := c.venues.Get(ctx, tenantID, venueID)
	if err != nil {
		return VenueAttachInfo{}, err
	}
	info := VenueAttachInfo{ID: venue.ID, Name: venue.Name}
	if venue.RentalPrice != nil {
		info.RentalPrice = *venue.RentalPrice
	}
	if venue.Charge != nil {
		info.Charge = *venue.Charge
	}
	return info, nil
}

func (c *impl) VendorCategoryID(ctx context.Context, tenantID, vendorID int64) (int64, error) {
	vendor, err := c.vendors.Get(ctx, tenantID, vendorID)
	if err != nil {
		return 0, err
	}
	return vendor.CategoryID, nil
}
