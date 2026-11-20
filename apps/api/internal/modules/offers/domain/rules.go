package domain

import (
	"errors"
	"fmt"
)

// PackageRule is the copy of a package's pricing rule kept on a quotation or
// project line (PLAN §2.1, §3.1 package_rule): editing the catalog afterwards
// never changes a line that already holds a copy. Its JSON form is what the
// line stores.
type PackageRule struct {
	PackageID           int64   `json:"packageId"`
	Name                string  `json:"name"`
	Price               int64   `json:"price"`
	EventKindIDs        []int64 `json:"eventKindIds"`
	IncludedGuests      *int    `json:"includedGuests"`
	ExtraGuestStep      *int    `json:"extraGuestStep"`
	ExtraGuestStepPrice *int64  `json:"extraGuestStepPrice"`
}

// RuleProblem is one thing wrong with a rule, keyed by the form field it
// belongs to.
type RuleProblem struct {
	Field   string
	Message string
}

// Messages of ValidateRule, kept as constants so tests and the form agree.
const (
	MsgPriceNegative          = "Harga paket tidak boleh kurang dari 0"
	MsgEventKindsRequired     = "Pilih minimal satu acara yang dicakup"
	MsgIncludedGuestsPositive = "Tamu termasuk minimal 1. Kosongkan bila harga tidak bergantung tamu."
	MsgStepPositive           = "Isi jumlah tamu per tambahan, minimal 1"
	MsgStepPriceNegative      = "Harga tambahan tidak boleh kurang dari 0"
	MsgStepPriceRequired      = "Isi harga untuk setiap tambahan tamu"
	MsgExtraNeedsIncluded     = "Isi tamu termasuk dulu, baru tambahan tamu bisa dihitung"
)

// ValidateRule checks a rule the way the package form needs it: a price of 0
// or more, at least one covered event kind, and an extra-guest charge that is
// either absent or complete (a step > 0 with a price, on top of an included
// number of guests).
func ValidateRule(r PackageRule) []RuleProblem {
	var out []RuleProblem
	add := func(field, msg string) { out = append(out, RuleProblem{field, msg}) }
	if r.Price < 0 {
		add("price", MsgPriceNegative)
	}
	if len(r.EventKindIDs) == 0 {
		add("eventKindIds", MsgEventKindsRequired)
	}
	if r.IncludedGuests != nil && *r.IncludedGuests < 1 {
		add("includedGuests", MsgIncludedGuestsPositive)
	}
	hasStep := r.ExtraGuestStep != nil
	hasPrice := r.ExtraGuestStepPrice != nil
	if hasStep && *r.ExtraGuestStep < 1 {
		add("extraGuestStep", MsgStepPositive)
	}
	if hasPrice && *r.ExtraGuestStepPrice < 0 {
		add("extraGuestStepPrice", MsgStepPriceNegative)
	}
	if hasPrice && !hasStep {
		add("extraGuestStep", MsgStepPositive)
	}
	if hasStep && !hasPrice {
		add("extraGuestStepPrice", MsgStepPriceRequired)
	}
	if (hasStep || hasPrice) && r.IncludedGuests == nil {
		add("includedGuests", MsgExtraNeedsIncluded)
	}
	return out
}

// EventGuests is the total number of guests of one event (Σ of its sessions)
// with the kind of that event.
type EventGuests struct {
	EventKindID int64
	Guests      int
}

// ExtraGuestsResult is the charge for guests above what a package includes,
// with every number that explains it ("620 − 500 termasuk = 120 → 3 × 50 tamu
// × Rp4.000.000").
type ExtraGuestsResult struct {
	// Guests is the largest total of the events the package covers.
	Guests int `json:"guests"`
	// Over is how many of them are above the included number (never < 0).
	Over int `json:"over"`
	// Steps is how many started steps Over spans.
	Steps  int   `json:"steps"`
	Amount int64 `json:"amount"`
}

// ExtraGuests prices the guests above what the rule includes (PLAN §1):
// ceil((guests − included) / step) × price per step, where guests is the
// largest total among the events of a kind the package covers. Below the
// included number the price stays as it is. A rule without an included number
// or without a complete step charges nothing.
func ExtraGuests(rule PackageRule, events []EventGuests) ExtraGuestsResult {
	covered := make(map[int64]bool, len(rule.EventKindIDs))
	for _, id := range rule.EventKindIDs {
		covered[id] = true
	}
	guests := 0
	for _, e := range events {
		if covered[e.EventKindID] && e.Guests > guests {
			guests = e.Guests
		}
	}
	return ExtraGuestsFor(rule, guests)
}

// ExtraGuestsFor is ExtraGuests for a number of guests already known (the
// "Contoh hitung" box of the package form).
func ExtraGuestsFor(rule PackageRule, guests int) ExtraGuestsResult {
	res := ExtraGuestsResult{Guests: guests}
	if rule.IncludedGuests == nil || guests <= *rule.IncludedGuests {
		return res
	}
	res.Over = guests - *rule.IncludedGuests
	if rule.ExtraGuestStep == nil || *rule.ExtraGuestStep < 1 || rule.ExtraGuestStepPrice == nil {
		return res
	}
	step := *rule.ExtraGuestStep
	res.Steps = (res.Over + step - 1) / step
	res.Amount = int64(res.Steps) * *rule.ExtraGuestStepPrice
	return res
}

// LineKind is the kind of a quotation or project line (PLAN §3.1). Every
// amount is positive; the kind says whether it adds or takes away.
type LineKind string

const (
	LinePackage     LineKind = "Package"
	LineComponent   LineKind = "Component"
	LineExtraGuests LineKind = "ExtraGuests"
	LineItem        LineKind = "Item"
	LineDiscount    LineKind = "Discount"
)

// Line is what LineTotals needs of one line.
type Line struct {
	Kind   LineKind
	Amount int64
}

// Totals is what a set of lines adds up to.
type Totals struct {
	// Gross is packages + extra guests + items, before discounts.
	Gross    int64 `json:"gross"`
	Discount int64 `json:"discount"`
	// Subtotal is Gross − Discount: the base of PPN and of the margin.
	Subtotal int64 `json:"subtotal"`
	Tax      int64 `json:"tax"`
	Total    int64 `json:"total"`
}

// Errors of LineTotals.
var (
	ErrNegativeAmount          = errors.New("nominal baris tidak boleh kurang dari 0")
	ErrDiscountExceedsSubtotal = errors.New("potongan melebihi subtotal")
	ErrUnknownLineKind         = errors.New("jenis baris tidak dikenal")
)

// LineTotals adds the lines up (PLAN §1): subtotal = packages + extra guests
// + items − discounts; PPN only for a PKP workspace, 12% of a DPP of 11/12 of
// the subtotal (PMK 131/2024, effectively 11%), rounded down to the rupiah;
// total = subtotal + PPN. A component line carries no amount of its own. A
// negative amount, or discounts above what they are taken from, is refused —
// the contract value can never fall below 0.
func LineTotals(lines []Line, isPKP bool) (Totals, error) {
	var t Totals
	for _, l := range lines {
		if l.Amount < 0 {
			return Totals{}, ErrNegativeAmount
		}
		switch l.Kind {
		case LinePackage, LineExtraGuests, LineItem:
			t.Gross += l.Amount
		case LineDiscount:
			t.Discount += l.Amount
		case LineComponent:
		default:
			return Totals{}, ErrUnknownLineKind
		}
	}
	if t.Discount > t.Gross {
		return Totals{}, ErrDiscountExceedsSubtotal
	}
	t.Subtotal = t.Gross - t.Discount
	if isPKP {
		t.Tax = TaxOf(t.Subtotal)
	}
	t.Total = t.Subtotal + t.Tax
	return t, nil
}

// TaxOf is the PPN of a subtotal: 12% × (11/12 × subtotal), rounded down.
// 12 × 11 / (12 × 100) reduces to 11/100, computed in integers so no
// floating-point error creeps into a contract value.
func TaxOf(subtotal int64) int64 {
	return subtotal * 11 / 100
}

// Layout is how guests stand at an event, which picks the venue capacity
// that applies.
type Layout string

const (
	LayoutStanding Layout = "Standing"
	LayoutSeated   Layout = "Seated"
)

// ValidLayout reports whether l is one of the two layouts.
func ValidLayout(l Layout) bool { return l == LayoutStanding || l == LayoutSeated }

func layoutWord(l Layout) string {
	if l == LayoutSeated {
		return "duduk"
	}
	return "berdiri"
}

// VenueCapacity is what the capacity check needs of a venue: its name and how
// many people it holds per layout (nil = no number for that layout).
type VenueCapacity struct {
	VenueID  int64
	Name     string
	Standing *int
	Seated   *int
}

// For is the venue's capacity for a layout (nil when it has no number for it).
func (v VenueCapacity) For(l Layout) *int {
	if l == LayoutSeated {
		return v.Seated
	}
	return v.Standing
}

// SessionGuests is one session of an event.
type SessionGuests struct {
	Label  string
	Guests int
}

// EventCapacity is one event as the capacity check sees it.
type EventCapacity struct {
	KindName string
	Layout   Layout
	// VenueID is nil while the event has no venue: it is then "not checked
	// against a venue yet", never a problem.
	VenueID  *int64
	Sessions []SessionGuests
}

// CapacityProblem is one event or session that does not fit. Session is the
// index of the session, or -1 when the problem is the venue itself (it has no
// number for the event's layout).
type CapacityProblem struct {
	Event   int
	Session int
	Message string
}

// CheckCapacity refuses every session whose guests exceed the capacity of its
// event's venue for that event's layout (PLAN §4.1), with a message that names
// the numbers ("Gedung Melati muat 600 orang berdiri, Sesi 1 Resepsi 650"). A
// venue without a number for the layout is a problem of its own ("Isi
// kapasitas duduk Gedung Melati di Venue"). An event without a venue, or whose
// venue is not in venues, is skipped.
func CheckCapacity(events []EventCapacity, venues map[int64]VenueCapacity) []CapacityProblem {
	var out []CapacityProblem
	for ei, e := range events {
		if e.VenueID == nil {
			continue
		}
		v, ok := venues[*e.VenueID]
		if !ok {
			continue
		}
		limit := v.For(e.Layout)
		if limit == nil {
			out = append(out, CapacityProblem{ei, -1, MissingCapacityMessage(v, e.Layout)})
			continue
		}
		for si, s := range e.Sessions {
			if s.Guests > *limit {
				out = append(out, CapacityProblem{ei, si, fmt.Sprintf("%s muat %d orang %s, %s %s %d",
					v.Name, *limit, layoutWord(e.Layout), s.Label, e.KindName, s.Guests)})
			}
		}
	}
	return out
}

// MissingCapacityMessage asks for the venue's number for a layout.
func MissingCapacityMessage(v VenueCapacity, l Layout) string {
	return fmt.Sprintf("Isi kapasitas %s %s di Venue", layoutWord(l), v.Name)
}

// PackageVenueProblem checks a package's included guests against its default
// venue. The catalog has no layout yet, so the venue's largest number applies;
// the exact layout is checked again when the package is used for an event.
// It returns "" when the guests fit, the package does not depend on guests, or
// the venue has no number at all (a venue always has one: the directory
// requires it).
func PackageVenueProblem(included *int, v VenueCapacity) string {
	if included == nil {
		return ""
	}
	limit, layout := 0, LayoutStanding
	if v.Standing != nil {
		limit = *v.Standing
	}
	if v.Seated != nil && *v.Seated > limit {
		limit, layout = *v.Seated, LayoutSeated
	}
	if limit == 0 || *included <= limit {
		return ""
	}
	return fmt.Sprintf("%s muat paling banyak %d orang %s, tamu termasuk %d", v.Name, limit, layoutWord(layout), *included)
}
