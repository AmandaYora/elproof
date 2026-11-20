// Package contracts is the ONLY package other modules may import from offers
// (ADR-0049). Today it serves the composition root, which hands it to
// `platform` (EventKindSeeder) and `vendors` (OfferReferences) through their
// setters.
package contracts

import (
	"context"
	"sort"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
)

// Contracts is offers' cross-module surface.
type Contracts interface {
	// SeedDefaultEventKinds gives a new workspace the six built-in event
	// kinds; idempotent.
	SeedDefaultEventKinds(ctx context.Context, tenantID int64) error
	// VendorReferenced, VenueReferenced, CategoryReferenced answer whether a
	// package (later also a quotation) of the workspace still points at the
	// directory row, so its permanent delete is refused.
	VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error)
	VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error)
	CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error)
	// GetEventKind is one event kind of the workspace (a not found otherwise):
	// a project event copies its name.
	GetEventKind(ctx context.Context, tenantID, id int64) (EventKindRef, error)
	// CheckCapacity is the one capacity rule (PLAN §4.1) for a project's events
	// as for an offer's: every session whose guests exceed its venue's number
	// for the event's layout, and every venue without a number for it.
	CheckCapacity(events []CapacityEvent, venues map[int64]CapacityVenue) []CapacityProblem
	// PackageCopy is what a project line copies when a package of the
	// workspace is added (a not found otherwise): its rule and its components.
	PackageCopy(ctx context.Context, tenantID, packageID int64) (PackageCopy, error)
	// ExtraGuests and LineTotals are the pricing rules of PLAN §2.1, the same
	// functions a quotation uses (pure: no database).
	ExtraGuests(rule PackageRule, events []EventGuests) ExtraGuestsResult
	LineTotals(lines []Line, isPKP bool) (Totals, error)
}

// The pricing types of offers/domain, re-exported so another module never
// imports offers/domain itself (ADR-0049).
type (
	PackageRule       = domain.PackageRule
	EventGuests       = domain.EventGuests
	ExtraGuestsResult = domain.ExtraGuestsResult
	LineKind          = domain.LineKind
	Line              = domain.Line
	Totals            = domain.Totals
)

// Line kinds (PLAN §3.1).
const (
	LinePackage     = domain.LinePackage
	LineComponent   = domain.LineComponent
	LineExtraGuests = domain.LineExtraGuests
	LineItem        = domain.LineItem
	LineDiscount    = domain.LineDiscount
)

// Errors of LineTotals.
var (
	ErrNegativeAmount          = domain.ErrNegativeAmount
	ErrDiscountExceedsSubtotal = domain.ErrDiscountExceedsSubtotal
	ErrUnknownLineKind         = domain.ErrUnknownLineKind
)

// PackageCopy is a package as a line copies it.
type PackageCopy struct {
	Rule       PackageRule
	IsActive   bool
	Components []PackageComponentCopy
}

// PackageComponentCopy is one component as a line copies it: never its
// estimated cost (a project line has no per-component price).
type PackageComponentCopy struct {
	Description string
	CategoryID  *int64
}

// ExtraGuestsOf and LineTotalsOf are the pure functions themselves, for a test
// double of the port.
func ExtraGuestsOf(rule PackageRule, events []EventGuests) ExtraGuestsResult {
	return domain.ExtraGuests(rule, events)
}

func LineTotalsOf(lines []Line, isPKP bool) (Totals, error) { return domain.LineTotals(lines, isPKP) }

// EventKindRef is what another module may know of an event kind.
type EventKindRef struct {
	ID         int64
	Name       string
	DefaultKey string // "akad", "resepsi", ... ("" for one the WO added)
	IsActive   bool
}

// CapacityVenue is a venue's name and capacities (nil = no number).
type CapacityVenue struct {
	Name     string
	Standing *int
	Seated   *int
}

// CapacityEvent is one event as the capacity rule sees it. Layout is
// "Standing" or "Seated"; VenueID nil means no venue yet (never a problem).
type CapacityEvent struct {
	KindName string
	Layout   string
	VenueID  *int64
	Sessions []CapacitySession
}

// CapacitySession is one session's label and guests.
type CapacitySession struct {
	Label  string
	Guests int
}

// CapacityProblem is one event (Session -1: its venue has no number for the
// layout) or session that does not fit, with the message to show.
type CapacityProblem struct {
	Event   int
	Session int
	Message string
}

type impl struct {
	kinds    *application.EventKindService
	packages *application.PackageService
}

func New(kinds *application.EventKindService, packages *application.PackageService) Contracts {
	return &impl{kinds: kinds, packages: packages}
}

func (c *impl) SeedDefaultEventKinds(ctx context.Context, tenantID int64) error {
	return c.kinds.SeedDefaultEventKinds(ctx, tenantID)
}

func (c *impl) VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error) {
	return c.packages.VendorReferenced(ctx, tenantID, vendorID)
}

func (c *impl) VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error) {
	return c.packages.VenueReferenced(ctx, tenantID, venueID)
}

func (c *impl) CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error) {
	return c.packages.CategoryReferenced(ctx, tenantID, categoryID)
}

func (c *impl) GetEventKind(ctx context.Context, tenantID, id int64) (EventKindRef, error) {
	k, err := c.kinds.Get(ctx, tenantID, id)
	if err != nil {
		return EventKindRef{}, err
	}
	return EventKindRef{ID: k.ID, Name: k.Name, DefaultKey: k.DefaultKey, IsActive: k.IsActive}, nil
}

func (c *impl) CheckCapacity(events []CapacityEvent, venues map[int64]CapacityVenue) []CapacityProblem {
	in := make([]domain.EventCapacity, len(events))
	for i, e := range events {
		sessions := make([]domain.SessionGuests, len(e.Sessions))
		for j, s := range e.Sessions {
			sessions[j] = domain.SessionGuests{Label: s.Label, Guests: s.Guests}
		}
		in[i] = domain.EventCapacity{KindName: e.KindName, Layout: domain.Layout(e.Layout), VenueID: e.VenueID, Sessions: sessions}
	}
	vs := make(map[int64]domain.VenueCapacity, len(venues))
	for id, v := range venues {
		vs[id] = domain.VenueCapacity{VenueID: id, Name: v.Name, Standing: v.Standing, Seated: v.Seated}
	}
	problems := domain.CheckCapacity(in, vs)
	out := make([]CapacityProblem, len(problems))
	for i, p := range problems {
		out[i] = CapacityProblem{Event: p.Event, Session: p.Session, Message: p.Message}
	}
	return out
}

func (c *impl) PackageCopy(ctx context.Context, tenantID, packageID int64) (PackageCopy, error) {
	p, err := c.packages.Get(ctx, tenantID, packageID)
	if err != nil {
		return PackageCopy{}, err
	}
	out := PackageCopy{Rule: p.Rule(), IsActive: p.IsActive, Components: make([]PackageComponentCopy, 0, len(p.Components))}
	comps := append([]domain.PackageComponent(nil), p.Components...)
	sort.SliceStable(comps, func(i, j int) bool { return comps[i].SortOrder < comps[j].SortOrder })
	for _, pc := range comps {
		out.Components = append(out.Components, PackageComponentCopy{Description: pc.Description, CategoryID: pc.CategoryID})
	}
	return out, nil
}

func (c *impl) ExtraGuests(rule PackageRule, events []EventGuests) ExtraGuestsResult {
	return domain.ExtraGuests(rule, events)
}

func (c *impl) LineTotals(lines []Line, isPKP bool) (Totals, error) {
	return domain.LineTotals(lines, isPKP)
}
