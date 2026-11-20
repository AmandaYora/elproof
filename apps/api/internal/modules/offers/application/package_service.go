package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/pagination"
)

// PackageService manages the WO's packages (Katalog › Paket).
type PackageService struct {
	repo      PackageRepository
	kinds     EventKindRepository
	directory VendorDirectory
	usage     QuotationUsage
}

func NewPackageService(repo PackageRepository, kinds EventKindRepository, directory VendorDirectory) *PackageService {
	return &PackageService{repo: repo, kinds: kinds, directory: directory}
}

// Messages answered by the package service.
const (
	MsgPackageNotFound   = "Paket tidak ditemukan"
	MsgPackageInvalid    = "Data paket belum sesuai"
	MsgPackageName       = "Isi nama paket"
	MsgPackageNameTaken  = "Paket dengan nama ini sudah ada"
	MsgVenueNotFound     = "Venue tidak ditemukan"
	MsgVenueInactive     = "Venue ini nonaktif. Pilih venue lain atau aktifkan dulu di Venue."
	MsgEventKindUnknown  = "Jenis acara tidak ditemukan"
	MsgComponentText     = "Isi keterangan isi paket"
	MsgCategoryNotFound  = "Kategori vendor tidak ditemukan"
	MsgVendorNotFound    = "Vendor tidak ditemukan"
	MsgVendorInactive    = "Vendor ini nonaktif. Pilih vendor lain atau aktifkan dulu di Vendor."
	MsgEstimatedCostNeg  = "Perkiraan biaya tidak boleh kurang dari 0"
	MsgTooManyComponents = "Isi paket paling banyak 50 baris"
	MsgPreviewGuestsNeg  = "Jumlah tamu tidak boleh kurang dari 0"
)

var packageFieldOrder = []string{"name", "price", "eventKindIds", "includedGuests", "extraGuestStep", "extraGuestStepPrice", "defaultVenueId", "components"}

// ComponentInput is one row of "Isi paket". ID is the row's id when it was
// loaded from the package (0 for a new row): it lets an editor without
// finance.margin.view keep the estimated cost they cannot see.
type ComponentInput struct {
	ID            int64
	Description   string
	CategoryID    *int64
	VendorID      *int64
	EstimatedCost *int64
}

// PackageInput is the package form.
type PackageInput struct {
	Name                string
	Price               int64
	EventKindIDs        []int64
	IncludedGuests      *int
	ExtraGuestStep      *int
	ExtraGuestStepPrice *int64
	DefaultVenueID      *int64
	Components          []ComponentInput
}

func (s *PackageService) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, filter PackageListFilter, sort pagination.Sort) ([]domain.Package, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, params, filter, sort)
}

func (s *PackageService) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.ListSortSpec()
}

func (s *PackageService) Get(ctx context.Context, tenantID, id int64) (*domain.Package, error) {
	p, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperror.NotFound(MsgPackageNotFound)
	}
	return p, nil
}

type problems map[string][]string

func (p problems) add(field, msg string) { p[field] = append(p[field], msg) }

func (p problems) err() error {
	if len(p) == 0 {
		return nil
	}
	msg := MsgPackageInvalid
	for _, k := range packageFieldOrder {
		if m := p[k]; len(m) > 0 {
			msg = m[0]
			break
		}
	}
	if msg == MsgPackageInvalid {
		for _, m := range p {
			msg = m[0]
			break
		}
	}
	return apperror.Validation(msg, p)
}

func isNotFound(err error) bool {
	ae, ok := apperror.As(err)
	return ok && ae.Kind == apperror.KindNotFound
}

// build validates the input and turns it into a package. existing is the
// package being edited (nil on create): a kind, venue, or vendor it already
// points at may be inactive, but a newly picked one must be active.
// canSeeCost is finance.margin.view: without it, estimated costs are not read
// from the input and the ones the package already holds are kept.
func (s *PackageService) build(ctx context.Context, tenantID int64, in PackageInput, existing *domain.Package, canSeeCost bool) (*domain.Package, error) {
	f := problems{}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		f.add("name", MsgPackageName)
	} else if utf8.RuneCountInString(in.Name) > domain.MaxPackageNameLen {
		f.add("name", fmt.Sprintf("Nama paket maksimal %d karakter", domain.MaxPackageNameLen))
	} else {
		selfID := int64(0)
		if existing != nil {
			selfID = existing.ID
		}
		taken, err := s.repo.NameTaken(ctx, tenantID, in.Name, selfID)
		if err != nil {
			return nil, err
		}
		if taken {
			f.add("name", MsgPackageNameTaken)
		}
	}

	kindIDs := uniqueIDs(in.EventKindIDs)
	rule := domain.PackageRule{Name: in.Name, Price: in.Price, EventKindIDs: kindIDs,
		IncludedGuests: in.IncludedGuests, ExtraGuestStep: in.ExtraGuestStep, ExtraGuestStepPrice: in.ExtraGuestStepPrice}
	for _, p := range domain.ValidateRule(rule) {
		f.add(p.Field, p.Message)
	}
	if err := s.checkKinds(ctx, tenantID, kindIDs, existing, f); err != nil {
		return nil, err
	}
	if err := s.checkVenue(ctx, tenantID, in, existing, f); err != nil {
		return nil, err
	}
	components, err := s.buildComponents(ctx, tenantID, in.Components, existing, canSeeCost, f)
	if err != nil {
		return nil, err
	}
	if err := f.err(); err != nil {
		return nil, err
	}
	p := &domain.Package{TenantID: tenantID, Name: in.Name, Price: in.Price, IncludedGuests: in.IncludedGuests,
		ExtraGuestStep: in.ExtraGuestStep, ExtraGuestStepPrice: in.ExtraGuestStepPrice, DefaultVenueID: in.DefaultVenueID,
		EventKindIDs: kindIDs, Components: components, IsActive: true}
	if existing != nil {
		p.ID, p.IsActive = existing.ID, existing.IsActive
	}
	return p, nil
}

func uniqueIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (s *PackageService) checkKinds(ctx context.Context, tenantID int64, ids []int64, existing *domain.Package, f problems) error {
	if len(ids) == 0 {
		return nil
	}
	kinds, err := s.kinds.List(ctx, tenantID)
	if err != nil {
		return err
	}
	byID := map[int64]domain.EventKind{}
	for _, k := range kinds {
		byID[k.ID] = k
	}
	had := map[int64]bool{}
	if existing != nil {
		for _, id := range existing.EventKindIDs {
			had[id] = true
		}
	}
	for _, id := range ids {
		k, ok := byID[id]
		switch {
		case !ok:
			f.add("eventKindIds", MsgEventKindUnknown)
			return nil
		case !k.IsActive && !had[id]:
			f.add("eventKindIds", fmt.Sprintf("Jenis acara %s nonaktif. Aktifkan dulu di Jenis acara.", k.Name))
			return nil
		}
	}
	return nil
}

func (s *PackageService) checkVenue(ctx context.Context, tenantID int64, in PackageInput, existing *domain.Package, f problems) error {
	if in.DefaultVenueID == nil {
		return nil
	}
	ref, err := s.directory.GetVenueRef(ctx, tenantID, *in.DefaultVenueID)
	if isNotFound(err) {
		f.add("defaultVenueId", MsgVenueNotFound)
		return nil
	}
	if err != nil {
		return err
	}
	kept := existing != nil && existing.DefaultVenueID != nil && *existing.DefaultVenueID == ref.ID
	if !ref.IsActive && !kept {
		f.add("defaultVenueId", MsgVenueInactive)
		return nil
	}
	capacity := domain.VenueCapacity{VenueID: ref.ID, Name: ref.Name, Standing: ref.CapacityStanding, Seated: ref.CapacitySeated}
	if msg := domain.PackageVenueProblem(in.IncludedGuests, capacity); msg != "" {
		f.add("includedGuests", msg)
	}
	return nil
}

func (s *PackageService) buildComponents(ctx context.Context, tenantID int64, rows []ComponentInput, existing *domain.Package, canSeeCost bool, f problems) ([]domain.PackageComponent, error) {
	if len(rows) > domain.MaxPackageComponents {
		f.add("components", MsgTooManyComponents)
		return nil, nil
	}
	old := map[int64]domain.PackageComponent{}
	if existing != nil {
		for _, c := range existing.Components {
			old[c.ID] = c
		}
	}
	knownCategory := map[int64]bool{}
	out := make([]domain.PackageComponent, 0, len(rows))
	for i, row := range rows {
		key := "components." + strconv.Itoa(i)
		c := domain.PackageComponent{Description: strings.TrimSpace(row.Description), CategoryID: row.CategoryID, VendorID: row.VendorID, SortOrder: i + 1}
		prev, hadRow := old[row.ID]
		if c.Description == "" {
			f.add(key+".description", MsgComponentText)
		} else if utf8.RuneCountInString(c.Description) > domain.MaxComponentDescriptionLen {
			f.add(key+".description", fmt.Sprintf("Keterangan maksimal %d karakter", domain.MaxComponentDescriptionLen))
		}
		if c.VendorID != nil {
			ref, err := s.directory.GetVendorRef(ctx, tenantID, *c.VendorID)
			switch {
			case isNotFound(err):
				f.add(key+".vendorId", MsgVendorNotFound)
			case err != nil:
				return nil, err
			case !ref.IsActive && !(hadRow && prev.VendorID != nil && *prev.VendorID == ref.ID):
				f.add(key+".vendorId", MsgVendorInactive)
			case c.CategoryID == nil:
				cat := ref.CategoryID
				c.CategoryID = &cat
				knownCategory[cat] = true
			}
		}
		if c.CategoryID != nil && !knownCategory[*c.CategoryID] {
			ok, err := s.directory.CategoryExists(ctx, tenantID, *c.CategoryID)
			if err != nil {
				return nil, err
			}
			if !ok {
				f.add(key+".categoryId", MsgCategoryNotFound)
			} else {
				knownCategory[*c.CategoryID] = true
			}
		}
		if canSeeCost {
			if row.EstimatedCost != nil && *row.EstimatedCost < 0 {
				f.add(key+".estimatedCost", MsgEstimatedCostNeg)
			}
			c.EstimatedCost = row.EstimatedCost
		} else if hadRow {
			c.EstimatedCost = prev.EstimatedCost
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *PackageService) Create(ctx context.Context, tenantID int64, in PackageInput, canSeeCost bool) (*domain.Package, error) {
	p, err := s.build(ctx, tenantID, in, nil, canSeeCost)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, p.ID)
}

// Update rewrites a package. expected is the If-Match version (ADR-0036).
// Lines that already copied the package's rule are not touched.
func (s *PackageService) Update(ctx context.Context, tenantID, id int64, in PackageInput, expected *time.Time, canSeeCost bool) (*domain.Package, error) {
	existing, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if expected != nil && !etag.Same(existing.UpdatedAt, *expected) {
		return nil, apperror.Stale(existing)
	}
	p, err := s.build(ctx, tenantID, in, existing, canSeeCost)
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.Update(ctx, p, expected)
	if err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Stale(current)
	}
	return current, nil
}

func (s *PackageService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.Package, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p.IsActive == isActive {
		return p, nil
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, id)
}

// Delete removes a package permanently. A package a quotation refers to can
// only be deactivated (422 package_in_quotation).
func (s *PackageService) Delete(ctx context.Context, tenantID, id int64) error {
	if _, err := s.Get(ctx, tenantID, id); err != nil {
		return err
	}
	if err := s.packageInQuotation(ctx, tenantID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// PreviewInput is the "Contoh hitung" box: the rule as typed and a number of
// guests.
type PreviewInput struct {
	Price               int64
	IncludedGuests      *int
	ExtraGuestStep      *int
	ExtraGuestStepPrice *int64
	Guests              int
}

// Preview is the price of a package for a number of guests.
type Preview struct {
	Extra domain.ExtraGuestsResult
	Total int64
}

// Preview prices a package that is still being typed, with the same rule a
// quotation will use, so the web never holds a second formula (PLAN §2.2).
// Nothing is stored.
func (s *PackageService) Preview(in PreviewInput) (Preview, error) {
	f := problems{}
	rule := domain.PackageRule{Price: in.Price, EventKindIDs: []int64{0},
		IncludedGuests: in.IncludedGuests, ExtraGuestStep: in.ExtraGuestStep, ExtraGuestStepPrice: in.ExtraGuestStepPrice}
	for _, p := range domain.ValidateRule(rule) {
		f.add(p.Field, p.Message)
	}
	if in.Guests < 0 {
		f.add("guests", MsgPreviewGuestsNeg)
	}
	if err := f.err(); err != nil {
		return Preview{}, err
	}
	extra := domain.ExtraGuestsFor(rule, in.Guests)
	return Preview{Extra: extra, Total: in.Price + extra.Amount}, nil
}

// Names resolves the display names a package response shows: covered kinds
// (this module) and the venue, categories, and vendors (the directory, scoped
// to the workspace).
type Names struct {
	Kinds      map[int64]string
	Venues     map[int64]string
	Categories map[int64]string
	Vendors    map[int64]string
}

func (s *PackageService) Names(ctx context.Context, tenantID int64, packages []domain.Package) (Names, error) {
	n := Names{Kinds: map[int64]string{}}
	kinds, err := s.kinds.List(ctx, tenantID)
	if err != nil {
		return n, err
	}
	for _, k := range kinds {
		n.Kinds[k.ID] = k.Name
	}
	var venues, categories, vendors []int64
	for _, p := range packages {
		if p.DefaultVenueID != nil {
			venues = append(venues, *p.DefaultVenueID)
		}
		for _, c := range p.Components {
			if c.CategoryID != nil {
				categories = append(categories, *c.CategoryID)
			}
			if c.VendorID != nil {
				vendors = append(vendors, *c.VendorID)
			}
		}
	}
	if n.Venues, err = s.directory.VenueNames(ctx, tenantID, venues); err != nil {
		return n, err
	}
	if n.Categories, err = s.directory.CategoryNames(ctx, tenantID, categories); err != nil {
		return n, err
	}
	if n.Vendors, err = s.directory.VendorNames(ctx, tenantID, vendors); err != nil {
		return n, err
	}
	return n, nil
}

// VendorReferenced, VenueReferenced, and CategoryReferenced answer the vendors
// module's OfferReferences port (PLAN §3.3): a directory row a package or a
// quotation still points at cannot be deleted permanently.
func (s *PackageService) VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error) {
	return s.orQuotation(ctx, func() (bool, error) { return s.repo.VendorReferenced(ctx, tenantID, vendorID) },
		func(u QuotationUsage) (bool, error) { return u.VendorReferenced(ctx, tenantID, vendorID) })
}

func (s *PackageService) VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error) {
	return s.orQuotation(ctx, func() (bool, error) { return s.repo.VenueReferenced(ctx, tenantID, venueID) },
		func(u QuotationUsage) (bool, error) { return u.VenueReferenced(ctx, tenantID, venueID) })
}

func (s *PackageService) CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error) {
	return s.orQuotation(ctx, func() (bool, error) { return s.repo.CategoryReferenced(ctx, tenantID, categoryID) },
		func(u QuotationUsage) (bool, error) { return u.CategoryReferenced(ctx, tenantID, categoryID) })
}
