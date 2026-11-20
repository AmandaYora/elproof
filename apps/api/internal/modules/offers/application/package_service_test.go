package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/offers/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

// --- fakes -------------------------------------------------------------

type fakeKinds struct {
	rows   []domain.EventKind
	counts map[int64]int
	nextID int64
}

func (f *fakeKinds) List(_ context.Context, tenantID int64) ([]domain.EventKind, error) {
	var out []domain.EventKind
	for _, k := range f.rows {
		if k.TenantID == tenantID {
			out = append(out, k)
		}
	}
	return out, nil
}
func (f *fakeKinds) FindByID(_ context.Context, tenantID, id int64) (*domain.EventKind, error) {
	for _, k := range f.rows {
		if k.TenantID == tenantID && k.ID == id {
			c := k
			return &c, nil
		}
	}
	return nil, nil
}
func (f *fakeKinds) Create(_ context.Context, k *domain.EventKind) error {
	f.nextID++
	k.ID = 100 + f.nextID
	f.rows = append(f.rows, *k)
	return nil
}
func (f *fakeKinds) Rename(context.Context, int64, int64, string, *time.Time) (bool, error) {
	return true, nil
}
func (f *fakeKinds) SetActive(context.Context, int64, int64, bool) error { return nil }
func (f *fakeKinds) Delete(_ context.Context, tenantID, id int64) error {
	for i, k := range f.rows {
		if k.TenantID == tenantID && k.ID == id {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return nil
}
func (f *fakeKinds) PackageCounts(context.Context, int64) (map[int64]int, error) {
	if f.counts == nil {
		return map[int64]int{}, nil
	}
	return f.counts, nil
}

type fakePackages struct {
	saved *domain.Package
	byID  map[int64]*domain.Package
}

func (f *fakePackages) ListPaginated(context.Context, int64, pagination.Params, PackageListFilter, pagination.Sort) ([]domain.Package, int64, error) {
	return nil, 0, nil
}
func (f *fakePackages) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return nil, pagination.Sort{}
}
func (f *fakePackages) FindByID(_ context.Context, tenantID, id int64) (*domain.Package, error) {
	if p, ok := f.byID[id]; ok && p.TenantID == tenantID {
		c := *p
		return &c, nil
	}
	return nil, nil
}
func (f *fakePackages) NameTaken(context.Context, int64, string, int64) (bool, error) {
	return false, nil
}
func (f *fakePackages) Create(_ context.Context, p *domain.Package) error {
	p.ID = 1
	f.saved = p
	if f.byID == nil {
		f.byID = map[int64]*domain.Package{}
	}
	f.byID[1] = p
	return nil
}
func (f *fakePackages) Update(_ context.Context, p *domain.Package, _ *time.Time) (bool, error) {
	f.saved = p
	f.byID[p.ID] = p
	return true, nil
}
func (f *fakePackages) SetActive(context.Context, int64, int64, bool) error { return nil }
func (f *fakePackages) Delete(context.Context, int64, int64) error          { return nil }
func (f *fakePackages) VendorReferenced(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (f *fakePackages) VenueReferenced(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (f *fakePackages) CategoryReferenced(context.Context, int64, int64) (bool, error) {
	return false, nil
}

// fakeDirectory holds venues and vendors per workspace; another workspace's
// id answers NotFound, as vendors' contract does.
type fakeDirectory struct {
	venues     map[int64]vendorscontracts.VenueRef
	venueOwner map[int64]int64
	vendors    map[int64]vendorscontracts.VendorRef
	categories map[int64]bool
}

func (d fakeDirectory) GetVenueRef(_ context.Context, tenantID, id int64) (vendorscontracts.VenueRef, error) {
	v, ok := d.venues[id]
	if !ok || d.venueOwner[id] != tenantID {
		return vendorscontracts.VenueRef{}, apperror.NotFound("Venue tidak ditemukan")
	}
	return v, nil
}
func (d fakeDirectory) GetVendorRef(_ context.Context, _ int64, id int64) (vendorscontracts.VendorRef, error) {
	v, ok := d.vendors[id]
	if !ok {
		return vendorscontracts.VendorRef{}, apperror.NotFound("Vendor tidak ditemukan")
	}
	return v, nil
}
func (d fakeDirectory) CategoryExists(_ context.Context, _ int64, id int64) (bool, error) {
	return d.categories[id], nil
}
func (d fakeDirectory) VenueNames(context.Context, int64, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}
func (d fakeDirectory) VendorNames(context.Context, int64, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}
func (d fakeDirectory) CategoryNames(context.Context, int64, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}

func ip(v int) *int       { return &v }
func i64p(v int64) *int64 { return &v }

const (
	woA int64 = 1
	woB int64 = 2
)

func newFixture() (*PackageService, *fakePackages) {
	kinds := &fakeKinds{rows: []domain.EventKind{
		{ID: 4, TenantID: woA, Name: "Akad", DefaultKey: "akad", IsActive: true},
		{ID: 5, TenantID: woA, Name: "Resepsi", DefaultKey: "resepsi", IsActive: true},
		{ID: 9, TenantID: woA, Name: "Siraman", IsActive: false},
		{ID: 14, TenantID: woB, Name: "Akad", IsActive: true},
	}}
	dir := fakeDirectory{
		venues: map[int64]vendorscontracts.VenueRef{
			10: {ID: 10, Name: "Gedung Melati", CapacityStanding: ip(600), CapacitySeated: ip(400), IsActive: true},
			20: {ID: 20, Name: "Gedung WO B", CapacityStanding: ip(2000), IsActive: true},
		},
		venueOwner: map[int64]int64{10: woA, 20: woB},
		vendors:    map[int64]vendorscontracts.VendorRef{30: {ID: 30, Name: "Dapur Ibu", CategoryID: 7, IsActive: true}},
		categories: map[int64]bool{7: true},
	}
	repo := &fakePackages{}
	return NewPackageService(repo, kinds, dir), repo
}

func validInput() PackageInput {
	return PackageInput{Name: "Paket Resepsi 500 tamu", Price: 120_000_000, EventKindIDs: []int64{4, 5},
		IncludedGuests: ip(500), ExtraGuestStep: ip(50), ExtraGuestStepPrice: i64p(4_000_000), DefaultVenueID: i64p(10),
		Components: []ComponentInput{{Description: "Katering buffet 450 porsi", VendorID: i64p(30), EstimatedCost: i64p(60_000_000)}}}
}

func fieldsOf(t *testing.T, err error) map[string][]string {
	t.Helper()
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation {
		t.Fatalf("want a validation error, got %v", err)
	}
	return ae.Fields
}

// --- tests -------------------------------------------------------------

func TestCreatePackageCopiesVendorCategory(t *testing.T) {
	svc, repo := newFixture()
	if _, err := svc.Create(context.Background(), woA, validInput(), true); err != nil {
		t.Fatal(err)
	}
	c := repo.saved.Components[0]
	if c.CategoryID == nil || *c.CategoryID != 7 {
		t.Fatalf("category not taken from the vendor: %+v", c)
	}
	if c.EstimatedCost == nil || *c.EstimatedCost != 60_000_000 {
		t.Fatalf("estimated cost not kept: %+v", c)
	}
}

func TestPackageIncludedGuestsAboveDefaultVenue(t *testing.T) {
	svc, _ := newFixture()
	in := validInput()
	in.IncludedGuests = ip(650)
	_, err := svc.Create(context.Background(), woA, in, true)
	f := fieldsOf(t, err)
	if got := strings.Join(f["includedGuests"], ""); !strings.Contains(got, "Gedung Melati muat paling banyak 600 orang berdiri, tamu termasuk 650") {
		t.Fatalf("includedGuests = %q", got)
	}
}

func TestPackageCannotUseAnotherWorkspacesVenue(t *testing.T) {
	svc, _ := newFixture()
	in := validInput()
	in.DefaultVenueID = i64p(20) // WO B's venue
	_, err := svc.Create(context.Background(), woA, in, true)
	if got := fieldsOf(t, err)["defaultVenueId"]; len(got) == 0 || got[0] != MsgVenueNotFound {
		t.Fatalf("defaultVenueId = %v", got)
	}
}

func TestPackageCannotUseAnotherWorkspacesEventKind(t *testing.T) {
	svc, _ := newFixture()
	in := validInput()
	in.EventKindIDs = []int64{14} // WO B's Akad
	_, err := svc.Create(context.Background(), woA, in, true)
	if got := fieldsOf(t, err)["eventKindIds"]; len(got) == 0 || got[0] != MsgEventKindUnknown {
		t.Fatalf("eventKindIds = %v", got)
	}
}

func TestPackageRefusesInactiveKindOnlyWhenNew(t *testing.T) {
	svc, repo := newFixture()
	in := validInput()
	in.EventKindIDs = []int64{4, 9}
	if _, err := svc.Create(context.Background(), woA, in, true); err == nil {
		t.Fatal("a newly picked inactive kind must be refused")
	}
	// A package that already covers the kind may keep it after it was deactivated.
	repo.byID = map[int64]*domain.Package{1: {ID: 1, TenantID: woA, Name: "Lama", EventKindIDs: []int64{4, 9}, UpdatedAt: time.Unix(1, 0)}}
	if _, err := svc.Update(context.Background(), woA, 1, in, nil, true); err != nil {
		t.Fatalf("keeping an inactive kind: %v", err)
	}
}

func TestPackageEditWithoutMarginKeepsEstimatedCost(t *testing.T) {
	svc, repo := newFixture()
	repo.byID = map[int64]*domain.Package{1: {ID: 1, TenantID: woA, Name: "Paket", EventKindIDs: []int64{4},
		Components: []domain.PackageComponent{{ID: 77, Description: "Katering", EstimatedCost: i64p(60_000_000)}}, UpdatedAt: time.Unix(1, 0)}}
	in := validInput()
	in.Components = []ComponentInput{
		{ID: 77, Description: "Katering 500 porsi", EstimatedCost: i64p(1)}, // ignored: cannot see costs
		{Description: "Dekorasi", EstimatedCost: i64p(2)},                   // new row, no cost
	}
	if _, err := svc.Update(context.Background(), woA, 1, in, nil, false); err != nil {
		t.Fatal(err)
	}
	got := repo.saved.Components
	if got[0].EstimatedCost == nil || *got[0].EstimatedCost != 60_000_000 {
		t.Fatalf("kept cost = %v", got[0].EstimatedCost)
	}
	if got[1].EstimatedCost != nil {
		t.Fatalf("new row got a cost without the permission: %v", *got[1].EstimatedCost)
	}
}

func TestPackageUpdateStale(t *testing.T) {
	svc, repo := newFixture()
	repo.byID = map[int64]*domain.Package{1: {ID: 1, TenantID: woA, Name: "Paket", EventKindIDs: []int64{4}, UpdatedAt: time.Unix(100, 0)}}
	old := time.Unix(50, 0)
	_, err := svc.Update(context.Background(), woA, 1, validInput(), &old, true)
	if _, ok := apperror.IsStale(err); !ok {
		t.Fatalf("want stale, got %v", err)
	}
}

func TestPackageOfAnotherWorkspaceIsNotFound(t *testing.T) {
	svc, repo := newFixture()
	repo.byID = map[int64]*domain.Package{1: {ID: 1, TenantID: woA, Name: "Paket"}}
	_, err := svc.Get(context.Background(), woB, 1)
	if ae, ok := apperror.As(err); !ok || ae.Kind != apperror.KindNotFound {
		t.Fatalf("want 404, got %v", err)
	}
}

func TestPreview(t *testing.T) {
	svc, _ := newFixture()
	p, err := svc.Preview(PreviewInput{Price: 120_000_000, IncludedGuests: ip(500), ExtraGuestStep: ip(50), ExtraGuestStepPrice: i64p(4_000_000), Guests: 620})
	if err != nil {
		t.Fatal(err)
	}
	if p.Extra.Steps != 3 || p.Extra.Amount != 12_000_000 || p.Total != 132_000_000 {
		t.Fatalf("preview = %+v", p)
	}
	if _, err := svc.Preview(PreviewInput{Price: 1, IncludedGuests: ip(500), ExtraGuestStepPrice: i64p(1)}); err == nil {
		t.Fatal("a price per step without a step must be refused")
	}
}

func TestSeedDefaultEventKindsIsIdempotent(t *testing.T) {
	kinds := &fakeKinds{}
	svc := NewEventKindService(kinds)
	for i := 0; i < 2; i++ {
		if err := svc.SeedDefaultEventKinds(context.Background(), woA); err != nil {
			t.Fatal(err)
		}
	}
	if len(kinds.rows) != 6 {
		t.Fatalf("got %d kinds, want 6", len(kinds.rows))
	}
	// A renamed built-in kind is not added again.
	kinds.rows[3].Name = "Akad nikah"
	if err := svc.SeedDefaultEventKinds(context.Background(), woA); err != nil {
		t.Fatal(err)
	}
	if len(kinds.rows) != 6 {
		t.Fatalf("after rename: %d kinds", len(kinds.rows))
	}
}

func TestEventKindInUseCannotBeDeleted(t *testing.T) {
	kinds := &fakeKinds{rows: []domain.EventKind{{ID: 4, TenantID: woA, Name: "Akad", IsActive: true}}, counts: map[int64]int{4: 2}}
	svc := NewEventKindService(kinds)
	err := svc.Delete(context.Background(), woA, 4)
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindConflict || ae.Code != CodeEventKindInUse {
		t.Fatalf("want 409 %s, got %v", CodeEventKindInUse, err)
	}
	kinds.counts = nil
	if err := svc.Delete(context.Background(), woA, 4); err != nil {
		t.Fatalf("unused kind: %v", err)
	}
}

func TestEventKindNameMustBeUnique(t *testing.T) {
	kinds := &fakeKinds{rows: []domain.EventKind{{ID: 4, TenantID: woA, Name: "Akad", IsActive: true}}}
	svc := NewEventKindService(kinds)
	if _, err := svc.Create(context.Background(), woA, " akad "); err == nil {
		t.Fatal("duplicate name must be refused")
	}
	if _, err := svc.Create(context.Background(), woB, "Akad"); err != nil {
		t.Fatalf("another workspace may use the same name: %v", err)
	}
}
