package domain

import (
	"sort"
	"time"
)

// LineKind is the kind of a project line ("Rincian", PLAN §3.1). Every amount
// is positive; the kind says whether it adds to the contract or takes away.
type LineKind string

const (
	// LinePackage is a package from the catalog, with a copy of its rule.
	LinePackage LineKind = "Package"
	// LineComponent is one thing a package contains ("Termasuk paket"); it
	// never carries an amount of its own.
	LineComponent LineKind = "Component"
	// LineExtraGuests is the charge for guests above what a package includes:
	// always computed from the package's rule copy and the events' guests,
	// never typed.
	LineExtraGuests LineKind = "ExtraGuests"
	// LineItem is a typed line: quantity x unit price (a bonus is an item at 0).
	LineItem LineKind = "Item"
	// LineDiscount takes its amount off the subtotal ("Potongan").
	LineDiscount LineKind = "Discount"
)

// IsValidLineKind reports whether k is one of the five kinds.
func IsValidLineKind(k LineKind) bool {
	switch k {
	case LinePackage, LineComponent, LineExtraGuests, LineItem, LineDiscount:
		return true
	}
	return false
}

// IsChild reports whether a line of this kind hangs off a package.
func (k LineKind) IsChild() bool { return k == LineComponent || k == LineExtraGuests }

// Limits of a project's lines (migration 000088 widths).
const (
	MaxProjectLines       = 300
	MaxLineDescriptionLen = 255
	MaxLineUnitLen        = 30
	MaxLineQuantity       = 1_000_000
)

// LinePackageRule is the copy of a package's pricing rule kept on a Package
// line (the JSON of offers' PackageRule): editing the catalog afterwards never
// changes a project that already holds a copy.
type LinePackageRule struct {
	PackageID           int64   `json:"packageId"`
	Name                string  `json:"name"`
	Price               int64   `json:"price"`
	EventKindIDs        []int64 `json:"eventKindIds"`
	IncludedGuests      *int    `json:"includedGuests"`
	ExtraGuestStep      *int    `json:"extraGuestStep"`
	ExtraGuestStepPrice *int64  `json:"extraGuestStepPrice"`
}

// ChargesExtraGuests reports whether the rule prices guests above an included
// number (it then has an ExtraGuests line).
func (r LinePackageRule) ChargesExtraGuests() bool {
	return r.IncludedGuests != nil && r.ExtraGuestStep != nil && *r.ExtraGuestStep >= 1 && r.ExtraGuestStepPrice != nil
}

// Covers reports whether the rule covers events of this kind.
func (r LinePackageRule) Covers(eventKindID int64) bool {
	for _, id := range r.EventKindIDs {
		if id == eventKindID {
			return true
		}
	}
	return false
}

// ProjectLine is one line of a project's Rincian.
type ProjectLine struct {
	ID        int64
	TenantID  int64
	ProjectID int64
	Kind      LineKind
	// ParentLineID is the Package a Component or ExtraGuests line belongs to.
	ParentLineID *int64
	// PackageRule is the rule copy (Package lines only).
	PackageRule *LinePackageRule
	Description string
	CategoryID  *int64
	EventID     *int64
	// Quantity, Unit and UnitPrice describe an Item (Amount = Quantity x UnitPrice).
	Quantity  *int64
	Unit      string
	UnitPrice *int64
	// Amount is always >= 0: Package = the rule's price, Component = 0,
	// ExtraGuests = computed, Item = quantity x unit price, Discount = typed.
	Amount  int64
	IsBonus bool
	// SupplierRef is the project supplier (paid by the WO) that provides it.
	SupplierRef *int64
	SortOrder   int
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SortLines puts lines in screen order: sort order, then id.
func SortLines(lines []ProjectLine) {
	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].SortOrder != lines[j].SortOrder {
			return lines[i].SortOrder < lines[j].SortOrder
		}
		return lines[i].ID < lines[j].ID
	})
}

// EventGuestTotal is the guests of one event (Σ of its sessions).
func EventGuestTotal(e ProjectEvent) int {
	n := 0
	for _, s := range e.Sessions {
		n += s.Guests
	}
	return n
}

// DecisiveEvent is the event the extra-guest charge of a rule is computed from:
// the one with the most guests among the events of a kind the rule covers (the
// first in screen order on a tie); nil when none is covered.
func DecisiveEvent(rule LinePackageRule, events []ProjectEvent) *ProjectEvent {
	var best *ProjectEvent
	bestGuests := -1
	for i := range events {
		if !rule.Covers(events[i].EventKindID) {
			continue
		}
		if g := EventGuestTotal(events[i]); g > bestGuests {
			best, bestGuests = &events[i], g
		}
	}
	return best
}

// TermsMismatch reports whether a payment schedule no longer adds up to the
// contract value (PLAN §3.2): only a schedule that exists can be off. The
// schedule is never rewritten by itself (ADR-0042): the screen offers
// "Hitung ulang termin".
func TermsMismatch(terms []PaymentTerm, contractTotal int64) bool {
	return len(terms) > 0 && TermsTotal(terms) != contractTotal
}

// Margin is what a project earns before the WO's own costs: the subtotal (PPN
// belongs to the state, so it is left out) minus the agreed cost of the vendors
// and venues the WO pays. It is incomplete while MissingCost of them have no
// cost yet ("Margin belum lengkap (n vendor)"). Shown only with
// finance.margin.view.
type Margin struct {
	Subtotal     int64
	SupplierCost int64
	MissingCost  int
	Margin       int64
}

// ComputeMargin applies the margin rule to a project's subtotal and suppliers.
func ComputeMargin(subtotal int64, suppliers []ProjectSupplier) Margin {
	cost, missing := SupplierCost(suppliers)
	return Margin{Subtotal: subtotal, SupplierCost: cost, MissingCost: missing, Margin: subtotal - cost}
}

// ContractCache is the contract as stored on the project (rewritten from the
// live lines in the same transaction as every change) with the lines' version.
type ContractCache struct {
	IsPKP        bool
	Subtotal     int64
	Tax          int64
	Total        int64
	LinesVersion time.Time
}
