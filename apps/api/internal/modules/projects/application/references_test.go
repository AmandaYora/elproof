package application

import (
	"context"
	"testing"

	"elproof/internal/modules/projects/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
)

// staffByTenant knows which staff ids exist in which workspace.
type staffByTenant map[int64]map[int64]string

func (s staffByTenant) GetName(_ context.Context, tenantID, staffID int64) (string, bool, error) {
	name, ok := s[tenantID][staffID]
	return name, ok, nil
}
func (s staffByTenant) GetPhone(context.Context, int64, int64) (string, bool, error) {
	return "", false, nil
}

func (s staffByTenant) StaffNames(_ context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	for _, id := range ids {
		if name, ok := s[tenantID][id]; ok {
			out[id] = name
		}
	}
	return out, nil
}

type vendorsByTenant struct {
	vendors, categories map[int64]map[int64]bool
	refs                map[int64]map[int64]vendorscontracts.VendorRef
}

func (v vendorsByTenant) VendorExists(_ context.Context, t, id int64) (bool, error) {
	return v.vendors[t][id], nil
}
func (v vendorsByTenant) CategoryExists(_ context.Context, t, id int64) (bool, error) {
	return v.categories[t][id], nil
}

func (v vendorsByTenant) VendorCategoryID(_ context.Context, t, id int64) (int64, error) {
	if !v.vendors[t][id] {
		return 0, apperror.NotFound("Vendor tidak ditemukan")
	}
	return 0, nil
}

func (v vendorsByTenant) VendorNames(context.Context, int64, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}
func (v vendorsByTenant) CategoryNames(context.Context, int64, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}

func (v vendorsByTenant) GetVendorRef(_ context.Context, t, id int64) (vendorscontracts.VendorRef, error) {
	if ref, ok := v.refs[t][id]; ok {
		return ref, nil
	}
	return vendorscontracts.VendorRef{}, apperror.NotFound("Vendor tidak ditemukan")
}
func (v vendorsByTenant) GetVenueRef(context.Context, int64, int64) (vendorscontracts.VenueRef, error) {
	return vendorscontracts.VenueRef{}, apperror.NotFound("Venue tidak ditemukan")
}
func (v vendorsByTenant) GetVenueSummary(context.Context, int64, int64) (vendorscontracts.VenueSummary, error) {
	return vendorscontracts.VenueSummary{}, apperror.NotFound("Venue tidak ditemukan")
}

func TestReferencesOnlyAcceptTheCallersOwnWorkspace(t *testing.T) {
	ctx := context.Background()
	refs := NewReferences(staffByTenant{1: {10: "Ani"}, 2: {20: "Budi"}})
	refs.SetVendors(vendorsByTenant{
		vendors:    map[int64]map[int64]bool{1: {5: true}, 2: {6: true}},
		categories: map[int64]map[int64]bool{1: {7: true}, 2: {8: true}},
	})

	if err := refs.Staff(ctx, 1, "picStaffId", 10, 0); err != nil {
		t.Errorf("own staff (and 0 = none) must pass: %v", err)
	}
	for name, err := range map[string]error{
		"another workspace's staff":    refs.Staff(ctx, 1, "picStaffId", 20),
		"a staff id nobody has":        refs.Staff(ctx, 1, "picStaffId", 999),
		"one bad id among good":        refs.Staff(ctx, 1, "picStaffId", 10, 20),
		"another workspace's vendor":   refs.Vendor(ctx, 1, "vendorId", 6),
		"another workspace's category": refs.Category(ctx, 1, "categoryId", 8),
	} {
		ae, ok := apperror.As(err)
		if err == nil || !ok || ae.Kind != apperror.KindValidation {
			t.Errorf("%s must be refused with a validation error, got %v", name, err)
		}
	}
	if err := refs.Vendor(ctx, 1, "vendorId", 5); err != nil {
		t.Errorf("own vendor: %v", err)
	}
	if err := refs.Category(ctx, 1, "categoryId", 7); err != nil {
		t.Errorf("own category: %v", err)
	}
}

// oneSupplier knows which supplier ids are live on which project.
type oneSupplier struct {
	SupplierRepository
	byProject map[int64]map[int64]bool
}

func (r oneSupplier) FindByID(_ context.Context, projectID, id int64) (*domain.ProjectSupplier, error) {
	if r.byProject[projectID][id] {
		return &domain.ProjectSupplier{ID: id, ProjectID: projectID}, nil
	}
	return nil, nil
}

func TestIssueMustUseASupplierOfTheSameProject(t *testing.T) {
	s := &IssueService{suppliers: oneSupplier{byProject: map[int64]map[int64]bool{5: {1: true}, 6: {2: true}}}}
	if err := s.validateSupplier(context.Background(), 5, 1); err != nil {
		t.Errorf("own supplier: %v", err)
	}
	for _, c := range []struct{ project, sup int64 }{{5, 2}, {6, 1}, {5, 99}} {
		err := s.validateSupplier(context.Background(), c.project, c.sup)
		if ae, ok := apperror.As(err); !ok || len(ae.Fields["projectSupplierId"]) == 0 {
			t.Errorf("project %d with supplier %d must be refused on projectSupplierId, got %v", c.project, c.sup, err)
		}
	}
	// 0 is "no vendor": a kendala about the event in general (E13).
	if err := s.validateSupplier(context.Background(), 5, 0); err != nil {
		t.Errorf("a kendala without a vendor must pass: %v", err)
	}
}

func TestSupplierVendorMustBeOwnAndActive(t *testing.T) {
	ctx := context.Background()
	refs := NewReferences(staffByTenant{})
	refs.SetVendors(vendorsByTenant{refs: map[int64]map[int64]vendorscontracts.VendorRef{
		1: {5: {ID: 5, Name: "Dekor Indah", CategoryID: 3, IsActive: true}, 6: {ID: 6, Name: "Dekor Lama", IsActive: false}},
		2: {7: {ID: 7, Name: "Vendor lawan", IsActive: true}},
	}})
	ref, err := refs.SupplierVendor(ctx, 1, "vendorId", 5)
	if err != nil || ref.Name != "Dekor Indah" || ref.CategoryID != 3 {
		t.Fatalf("own active vendor: %+v %v", ref, err)
	}
	for name, id := range map[string]int64{"none picked": 0, "inactive": 6, "another workspace's": 7, "unknown": 99} {
		_, err := refs.SupplierVendor(ctx, 1, "vendorId", id)
		if ae, ok := apperror.As(err); !ok || ae.Kind != apperror.KindValidation || len(ae.Fields["vendorId"]) == 0 {
			t.Errorf("%s vendor must be a 422 on vendorId, got %v", name, err)
		}
	}
}
