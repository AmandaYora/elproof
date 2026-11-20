package application

import (
	"context"

	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
)

// VendorDirectory is what the reference checks need from `vendors`: whether a
// vendor or a vendor category exists in the caller's own workspace. Bridged
// from main.go (vendors is built after projects), like EventVenues.
type VendorDirectory interface {
	VendorExists(ctx context.Context, tenantID, vendorID int64) (bool, error)
	CategoryExists(ctx context.Context, tenantID, categoryID int64) (bool, error)
	// VendorCategoryID is the category the vendor belongs to (NotFound when the
	// vendor is not this workspace's).
	VendorCategoryID(ctx context.Context, tenantID, vendorID int64) (int64, error)
	// VendorNames/CategoryNames resolve display names in one query, scoped to
	// the workspace (an id that is not this workspace's is absent).
	VendorNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
	CategoryNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
	// GetVendorRef is one vendor of the workspace (NotFound otherwise).
	GetVendorRef(ctx context.Context, tenantID, vendorID int64) (vendorscontracts.VendorRef, error)
	// GetVenueRef is one venue of the workspace with its capacities (NotFound otherwise).
	GetVenueRef(ctx context.Context, tenantID, venueID int64) (vendorscontracts.VenueRef, error)
	// GetVenueSummary is the public-safe data of a venue of the workspace
	// (address, city, facilities) the portal shows with an event.
	GetVenueSummary(ctx context.Context, tenantID, venueID int64) (vendorscontracts.VenueSummary, error)
}

// References confirms that ids a request body names — a PIC, a vendor, a
// category — belong to the caller's own workspace. A request path is already
// scoped to the caller's project; the ids inside a body are not, and without
// this a person could attach another workspace's staff or vendor to their
// own project and read it back through the project's own responses.
// Anything that does not resolve in this workspace answers the same way as
// something that does not exist.
type References struct {
	staff   StaffNameResolver
	vendors VendorDirectory
}

func NewReferences(staff StaffNameResolver) *References { return &References{staff: staff} }

// SetVendors completes the two-phase wiring with the vendors module.
func (r *References) SetVendors(v VendorDirectory) { r.vendors = v }

func notFoundField(field, message string) error {
	return apperror.Validation(message, map[string][]string{field: {message}})
}

// Staff checks each id (0 means "none" and is skipped).
func (r *References) Staff(ctx context.Context, tenantID int64, field string, ids ...int64) error {
	for _, id := range ids {
		if id == 0 {
			continue
		}
		_, ok, err := r.staff.GetName(ctx, tenantID, id)
		if err != nil {
			return err
		}
		if !ok {
			return notFoundField(field, "Anggota tim tidak ditemukan")
		}
	}
	return nil
}

// Vendor checks a vendor id (0 means "none").
func (r *References) Vendor(ctx context.Context, tenantID int64, field string, id int64) error {
	if id == 0 || r.vendors == nil {
		return nil
	}
	ok, err := r.vendors.VendorExists(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if !ok {
		return notFoundField(field, "Vendor tidak ditemukan")
	}
	return nil
}

// Category checks a vendor category id (0 means "none").
func (r *References) Category(ctx context.Context, tenantID int64, field string, id int64) error {
	if id == 0 || r.vendors == nil {
		return nil
	}
	ok, err := r.vendors.CategoryExists(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if !ok {
		return notFoundField(field, "Kategori vendor tidak ditemukan")
	}
	return nil
}

// VendorNames names the given vendors for display (best effort: a failure or
// an unknown id leaves the id out, so an unresolved name shows as empty).
func (r *References) VendorNames(ctx context.Context, tenantID int64, ids []int64) map[int64]string {
	if r.vendors == nil || len(ids) == 0 {
		return map[int64]string{}
	}
	names, err := r.vendors.VendorNames(ctx, tenantID, ids)
	if err != nil || names == nil {
		return map[int64]string{}
	}
	return names
}

// CategoryNames is VendorNames for vendor categories.
func (r *References) CategoryNames(ctx context.Context, tenantID int64, ids []int64) map[int64]string {
	if r.vendors == nil || len(ids) == 0 {
		return map[int64]string{}
	}
	names, err := r.vendors.CategoryNames(ctx, tenantID, ids)
	if err != nil || names == nil {
		return map[int64]string{}
	}
	return names
}

// VendorCategoryID is the category of a vendor of this workspace; ok is false
// when the vendor is not found here. With no directory wired it reports ok=false.
func (r *References) VendorCategoryID(ctx context.Context, tenantID, vendorID int64) (categoryID int64, ok bool, err error) {
	if vendorID == 0 || r.vendors == nil {
		return 0, false, nil
	}
	id, err := r.vendors.VendorCategoryID(ctx, tenantID, vendorID)
	if err != nil {
		if ae, isApp := apperror.As(err); isApp && ae.Kind == apperror.KindNotFound {
			return 0, false, nil
		}
		return 0, false, err
	}
	return id, true, nil
}

// SupplierVendor is the directory vendor a new project supplier is made from:
// it must be this workspace's and active (an inactive vendor stays on the
// projects it already works on, but is not picked for new ones).
func (r *References) SupplierVendor(ctx context.Context, tenantID int64, field string, id int64) (vendorscontracts.VendorRef, error) {
	if id == 0 {
		return vendorscontracts.VendorRef{}, notFoundField(field, "Pilih vendor")
	}
	if r.vendors == nil {
		return vendorscontracts.VendorRef{}, apperror.Internal("Direktori vendor belum tersedia")
	}
	ref, err := r.vendors.GetVendorRef(ctx, tenantID, id)
	if err != nil {
		if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
			return vendorscontracts.VendorRef{}, notFoundField(field, "Vendor tidak ditemukan")
		}
		return vendorscontracts.VendorRef{}, err
	}
	if !ref.IsActive {
		return vendorscontracts.VendorRef{}, notFoundField(field, "Vendor/venue nonaktif tidak bisa dipilih untuk baris baru")
	}
	return ref, nil
}

// VenueSummaries is the public-safe data of the given directory venues, by id
// (best effort: a venue that no longer resolves is left out).
func (r *References) VenueSummaries(ctx context.Context, tenantID int64, ids []int64) map[int64]vendorscontracts.VenueSummary {
	out := map[int64]vendorscontracts.VenueSummary{}
	if r.vendors == nil {
		return out
	}
	for _, id := range ids {
		if _, done := out[id]; done || id == 0 {
			continue
		}
		v, err := r.vendors.GetVenueSummary(ctx, tenantID, id)
		if err != nil {
			continue
		}
		out[id] = v
	}
	return out
}

// VenueRef is the directory venue as it is now (capacities for "Perbarui
// kapasitas"); a venue no longer in the directory is a 422 on field.
func (r *References) VenueRef(ctx context.Context, tenantID int64, field string, id int64) (vendorscontracts.VenueRef, error) {
	if r.vendors == nil {
		return vendorscontracts.VenueRef{}, apperror.Internal("Direktori venue belum tersedia")
	}
	ref, err := r.vendors.GetVenueRef(ctx, tenantID, id)
	if err != nil {
		if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
			return vendorscontracts.VenueRef{}, notFoundField(field, "Venue sudah tidak ada di direktori")
		}
		return vendorscontracts.VenueRef{}, err
	}
	return ref, nil
}
