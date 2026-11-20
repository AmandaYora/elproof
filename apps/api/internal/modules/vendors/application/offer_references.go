package application

import "context"

// OfferReferences answers whether packages or quotations (the `offers` module,
// docs/plan/paket-wo/PLAN.md §3.3) still refer to a vendor, a venue, or a
// vendor category. A permanent delete is refused while one does: the row should
// be deactivated instead, so it can no longer be picked for a new line. It is a
// port this module owns; an adapter over offers' contract is handed in through
// vendors.Module.SetOfferReferences in the composition root. Unset (nil) means
// nothing refers to anything.
type OfferReferences interface {
	VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error)
	VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error)
	CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error)
}
