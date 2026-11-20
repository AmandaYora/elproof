package application

import (
	"context"
	"net/http"
	"testing"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
)

// fakeOfferRefs answers "referenced" for the ids it holds.
type fakeOfferRefs struct{ vendors, venues, categories map[int64]bool }

func (f fakeOfferRefs) VendorReferenced(_ context.Context, _, id int64) (bool, error) {
	return f.vendors[id], nil
}
func (f fakeOfferRefs) VenueReferenced(_ context.Context, _, id int64) (bool, error) {
	return f.venues[id], nil
}
func (f fakeOfferRefs) CategoryReferenced(_ context.Context, _, id int64) (bool, error) {
	return f.categories[id], nil
}

// Repositories that record a delete; anything else panics via the nil embed.
type fakeVenueRepo struct {
	VenueRepository
	deleted []int64
}

func (f *fakeVenueRepo) FindByID(_ context.Context, tenantID, id int64) (*domain.Venue, error) {
	return &domain.Venue{ID: id, TenantID: tenantID, Name: "Gedung Melati"}, nil
}
func (f *fakeVenueRepo) Delete(_ context.Context, _, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeVendorRepo struct {
	VendorRepository
	deleted []int64
}

func (f *fakeVendorRepo) FindByID(_ context.Context, tenantID, id int64) (*domain.Vendor, error) {
	return &domain.Vendor{ID: id, TenantID: tenantID, Name: "Katering Sari"}, nil
}
func (f *fakeVendorRepo) Delete(_ context.Context, _, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type noVendorsInCategory struct{}

func (noVendorsInCategory) CountByCategoryID(context.Context, int64, int64) (int64, error) {
	return 0, nil
}

type deletingCategoryRepo struct {
	fakeCategoryRepo
	deleted []int64
}

func (f *deletingCategoryRepo) Delete(_ context.Context, _, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func wantInOffers(t *testing.T, err error, code, msg string) {
	t.Helper()
	ae, ok := apperror.As(err)
	if !ok || apperror.HTTPStatus(err) != http.StatusConflict || ae.Code != code || ae.Message != msg {
		t.Fatalf("err = %v (%+v), want 409 %s", err, ae, code)
	}
}

// PLAN paket-wo §3.3: a vendor, venue, or category a package or quotation still
// refers to is not deleted; one nothing refers to is, and an unset port refers
// to nothing.
func TestDeleteRefusedWhileReferencedByOffers(t *testing.T) {
	ctx := context.Background()
	refs := fakeOfferRefs{vendors: map[int64]bool{7: true}, venues: map[int64]bool{3: true}, categories: map[int64]bool{2: true}}

	venueRepo := &fakeVenueRepo{}
	venues := NewVenueService(venueRepo, nil, nil)
	if err := venues.Delete(ctx, 1, 3); err != nil {
		t.Fatalf("unset port blocked a venue delete: %v", err)
	}
	venues.SetOfferReferences(refs)
	wantInOffers(t, venues.Delete(ctx, 1, 3), CodeVenueInOffers, MsgVenueInOffers)
	if err := venues.Delete(ctx, 1, 4); err != nil {
		t.Fatalf("unreferenced venue refused: %v", err)
	}
	if len(venueRepo.deleted) != 2 || venueRepo.deleted[1] != 4 {
		t.Fatalf("venue deletes = %v", venueRepo.deleted)
	}

	vendorRepo := &fakeVendorRepo{}
	vendors := NewVendorService(vendorRepo, nil, nil, nil)
	vendors.SetOfferReferences(refs)
	wantInOffers(t, vendors.Delete(ctx, 1, 7), CodeVendorInOffers, MsgVendorInOffers)
	if err := vendors.Delete(ctx, 1, 8); err != nil || len(vendorRepo.deleted) != 1 {
		t.Fatalf("unreferenced vendor: err %v, deletes %v", err, vendorRepo.deleted)
	}

	catRepo := &deletingCategoryRepo{fakeCategoryRepo: fakeCategoryRepo{rows: []domain.VendorCategory{{ID: 2, Name: "Katering"}, {ID: 5, Name: "Dekorasi"}}}}
	categories := NewVendorCategoryService(catRepo, noVendorsInCategory{})
	categories.SetOfferReferences(refs)
	wantInOffers(t, categories.Delete(ctx, 1, 2), CodeCategoryInOffers, MsgCategoryInOffers)
	if err := categories.Delete(ctx, 1, 5); err != nil || len(catRepo.deleted) != 1 {
		t.Fatalf("unreferenced category: err %v, deletes %v", err, catRepo.deleted)
	}
}
