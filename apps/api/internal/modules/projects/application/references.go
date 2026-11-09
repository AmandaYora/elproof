package application

import (
	"context"

	"elproof/internal/shared/apperror"
)

// VendorDirectory is what the reference checks need from `vendors`: whether a
// vendor or a vendor category exists in the caller's own workspace. Bridged
// from main.go (vendors is built after projects), like VenueResolver.
type VendorDirectory interface {
	VendorExists(ctx context.Context, tenantID, vendorID int64) (bool, error)
	CategoryExists(ctx context.Context, tenantID, categoryID int64) (bool, error)
	// VendorNames/CategoryNames resolve display names in one query, scoped to
	// the workspace (an id that is not this workspace's is absent).
	VendorNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
	CategoryNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
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
			return notFoundField(field, "Staf tidak ditemukan")
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
