package domain

import "time"

// Package is a product the WO offers a client: a price, the kinds of event it
// covers, how many guests the price includes and what each extra step of
// guests costs, an optional default venue, and what it contains.
type Package struct {
	ID       int64
	TenantID int64
	Name     string
	Price    int64
	// IncludedGuests is how many guests Price covers; nil = the price does not
	// depend on the number of guests.
	IncludedGuests *int
	// ExtraGuestStep / ExtraGuestStepPrice: every started step of this many
	// guests above IncludedGuests costs this much ("Tambahan per 50 tamu
	// seharga Rp4.000.000"). Both nil when there is no charge for extra guests.
	ExtraGuestStep      *int
	ExtraGuestStepPrice *int64
	// DefaultVenueID is a venue of the vendors directory (a primitive id, no
	// foreign key — ADR-0004/0049); nil = no default venue.
	DefaultVenueID *int64
	EventKindIDs   []int64
	Components     []PackageComponent
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PackageComponent is one thing the package contains ("Katering buffet 450
// porsi"). It is always paid by the WO as part of the package. CategoryID and
// VendorID point into the vendors directory (primitive ids); EstimatedCost is
// the WO's own estimate, shown only to holders of finance.margin.view.
type PackageComponent struct {
	ID            int64
	Description   string
	CategoryID    *int64
	VendorID      *int64
	EstimatedCost *int64
	SortOrder     int
}

// Column widths and limits of the packages tables (migration 000086).
const (
	MaxPackageNameLen          = 150
	MaxComponentDescriptionLen = 255
	MaxPackageComponents       = 50
)

// Rule is the package's pricing rule, the part a quotation or project line
// copies when the package is added (PackageRule in PLAN §2.1).
func (p Package) Rule() PackageRule {
	ids := make([]int64, len(p.EventKindIDs))
	copy(ids, p.EventKindIDs)
	return PackageRule{
		PackageID:           p.ID,
		Name:                p.Name,
		Price:               p.Price,
		EventKindIDs:        ids,
		IncludedGuests:      p.IncludedGuests,
		ExtraGuestStep:      p.ExtraGuestStep,
		ExtraGuestStepPrice: p.ExtraGuestStepPrice,
	}
}
