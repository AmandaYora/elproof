// Package contracts is the ONLY package other modules may import from vendors.
package contracts

import (
	"context"
	"strings"

	"elproof/internal/modules/vendors/application"
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

// Contracts is what other modules depend on to trigger vendors-owned
// behavior for their own principals — seeding a new tenant's default vendor
// categories on registration (`platform`), and resolving a project's venue_id
// into display data (`projects`, ADR-0016).
type Contracts interface {
	SeedDefaultCategories(ctx context.Context, tenantID int64) error
	GetVenueSummary(ctx context.Context, tenantID, venueID int64) (VenueSummary, error)
}

type impl struct {
	categories *application.VendorCategoryService
	venues     *application.VenueService
}

func New(categories *application.VendorCategoryService, venues *application.VenueService) Contracts {
	return &impl{categories: categories, venues: venues}
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
