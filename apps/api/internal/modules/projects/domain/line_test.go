package domain

import (
	"testing"
	"time"
)

func i64(v int64) *int64 { return &v }
func iv(v int) *int      { return &v }

func sampleRule() *LinePackageRule {
	return &LinePackageRule{PackageID: 7, Name: "Gold", Price: 150_000_000, EventKindIDs: []int64{2},
		IncludedGuests: iv(500), ExtraGuestStep: iv(50), ExtraGuestStepPrice: i64(4_000_000)}
}

// One line of each kind, as stored.
func sampleLines() []ProjectLine {
	return []ProjectLine{
		{ID: 1, Kind: LinePackage, PackageRule: sampleRule(), Description: "Gold", Amount: 150_000_000, SupplierRef: i64(9)},
		{ID: 2, Kind: LineComponent, ParentLineID: i64(1), Description: "Dekorasi", Amount: 0, SupplierRef: i64(9)},
		{ID: 3, Kind: LineExtraGuests, ParentLineID: i64(1), Description: "Tambahan tamu", Amount: 12_000_000},
		{ID: 4, Kind: LineItem, Description: "Photobooth", Quantity: i64(2), Unit: "unit", UnitPrice: i64(3_000_000), Amount: 6_000_000, SupplierRef: i64(11)},
		{ID: 5, Kind: LineDiscount, Description: "Potongan", Amount: 5_000_000},
	}
}

// The portal redaction table (PLAN §4.4): Pasangan (vendor + finance) sees
// names and amounts, Keluarga (vendor, no finance) names only, a role without
// portal.vendor.view nothing. Nobody sees a component's price or the supplier
// behind a line.
func TestPortalPolicyLineTable(t *testing.T) {
	cases := []struct {
		name       string
		policy     PortalPolicy
		visible    bool
		showAmount bool
	}{
		{"Pasangan", allBut(), true, true},
		{"Keluarga", allBut(PermPortalFinance), true, false},
		{"tanpa layanan", allBut(PermPortalVendor), false, false},
		{"tanpa izin", NewPortalPolicy(nil), false, false},
	}
	for _, c := range cases {
		for _, l := range sampleLines() {
			pl, ok := c.policy.Line(l)
			if ok != c.visible {
				t.Errorf("%s %s: visible=%v, want %v", c.name, l.Kind, ok, c.visible)
				continue
			}
			if !ok {
				continue
			}
			if pl.Line.SupplierRef != nil {
				t.Errorf("%s %s: the supplier behind a line must never reach the portal", c.name, l.Kind)
			}
			if pl.Line.Description != l.Description {
				t.Errorf("%s %s: the name is always shown", c.name, l.Kind)
			}
			wantAmount := c.showAmount && l.Kind != LineComponent
			if pl.ShowAmount != wantAmount {
				t.Errorf("%s %s: ShowAmount=%v, want %v", c.name, l.Kind, pl.ShowAmount, wantAmount)
			}
			if !wantAmount && (pl.Line.Amount != 0 || pl.Line.UnitPrice != nil) {
				t.Errorf("%s %s: amount %d / unit price %v must be hidden", c.name, l.Kind, pl.Line.Amount, pl.Line.UnitPrice)
			}
			if wantAmount && pl.Line.Amount != l.Amount {
				t.Errorf("%s %s: amount %d, want %d", c.name, l.Kind, pl.Line.Amount, l.Amount)
			}
			if r := pl.Line.PackageRule; r != nil && !c.showAmount && (r.Price != 0 || r.ExtraGuestStepPrice != nil) {
				t.Errorf("%s: the package rule's prices must be hidden: %+v", c.name, r)
			}
		}
	}
	// The policy works on a copy: the stored rule keeps its prices.
	lines := sampleLines()
	allBut(PermPortalFinance).Line(lines[0])
	if lines[0].PackageRule.Price != 150_000_000 || lines[0].SupplierRef == nil {
		t.Error("redaction must not change the caller's line")
	}
	deleted := lines[3]
	now := time.Now()
	deleted.DeletedAt = &now
	if _, ok := allBut().Line(deleted); ok {
		t.Error("a deleted line is never shown")
	}
}

func TestPortalPolicyProjectHomeAndTerm(t *testing.T) {
	pr := Project{ID: 1, Name: "A & B", PrepStartDate: time.Now(), PICSalesStaffID: 4, IsPKP: true,
		ContractSubtotal: 100, ContractTax: 11, ContractTotal: 111, LinesVersion: time.Now()}
	full := allBut().Project(pr)
	if !full.ShowMoney || full.Project.ContractTotal != 111 || full.Project.ContractTax != 11 {
		t.Errorf("Pasangan sees the contract: %+v", full)
	}
	if !full.Project.PrepStartDate.IsZero() || full.Project.PICSalesStaffID != 0 || !full.Project.LinesVersion.IsZero() {
		t.Errorf("internal fields always cleared: %+v", full.Project)
	}
	family := allBut(PermPortalFinance).Project(pr)
	if family.ShowMoney || family.Project.ContractTotal != 0 || family.Project.ContractSubtotal != 0 || family.Project.ContractTax != 0 {
		t.Errorf("Keluarga sees no numbers: %+v", family)
	}
	if h := allBut(PermPortalFinance).Home(); h.Money || !h.Services || !h.Events {
		t.Errorf("Keluarga home: %+v", h)
	}
	if h := allBut(PermPortalVendor).Home(); h.Services {
		t.Errorf("without portal.vendor.view no package names: %+v", h)
	}
	if !allBut().Term() || allBut(PermPortalFinance).Term() {
		t.Error("the schedule follows portal.finance.view")
	}
}

func TestTermsMismatchOnlyForAnExistingSchedule(t *testing.T) {
	if TermsMismatch(nil, 100) {
		t.Error("no schedule, no mismatch")
	}
	terms := []PaymentTerm{{Amount: 40}, {Amount: 60}}
	if TermsMismatch(terms, 100) {
		t.Error("equal totals are no mismatch")
	}
	if !TermsMismatch(terms, 111) {
		t.Error("a schedule below the contract (PPN added) is a mismatch")
	}
	if !TermsMismatch(terms, 90) {
		t.Error("a schedule above the contract (a discount added) is a mismatch")
	}
}

// "Margin belum lengkap (n vendor)": every active WO-paid supplier without an
// agreed cost is counted; a client-paid, cancelled or deleted one is not.
func TestComputeMarginCountsMissingCosts(t *testing.T) {
	now := time.Now()
	suppliers := []ProjectSupplier{
		{AgreedCost: i64(30), PaidBy: PayerWO, EngagementStatus: EngagementBooked},
		{PaidBy: PayerWO, EngagementStatus: EngagementBooked},
		{PaidBy: PayerWO, EngagementStatus: EngagementPlanned, Kind: SupplierVenue},
		{PaidBy: PayerClient, EngagementStatus: EngagementBooked},
		{PaidBy: PayerWO, EngagementStatus: EngagementCancelled},
		{PaidBy: PayerWO, EngagementStatus: EngagementBooked, DeletedAt: &now},
	}
	m := ComputeMargin(100, suppliers)
	if m.Subtotal != 100 || m.SupplierCost != 30 || m.MissingCost != 2 || m.Margin != 70 {
		t.Errorf("margin = %+v, want subtotal 100, cost 30, 2 missing, margin 70", m)
	}
	if m := ComputeMargin(100, suppliers[:1]); m.MissingCost != 0 {
		t.Errorf("complete margin: %+v", m)
	}
}

func TestDecisiveEventIsTheCoveredOneWithMostGuests(t *testing.T) {
	rule := *sampleRule() // covers kind 2 (Resepsi)
	events := []ProjectEvent{
		{ID: 1, EventKindID: 1, Sessions: []EventSession{{Guests: 900}}},
		{ID: 2, EventKindID: 2, Sessions: []EventSession{{Guests: 300}, {Guests: 350}}},
		{ID: 3, EventKindID: 2, Sessions: []EventSession{{Guests: 640}}},
	}
	e := DecisiveEvent(rule, events)
	if e == nil || e.ID != 2 || EventGuestTotal(*e) != 650 {
		t.Fatalf("decisive = %+v, want event 2 (650 guests over its sessions)", e)
	}
	if DecisiveEvent(LinePackageRule{EventKindIDs: []int64{9}}, events) != nil {
		t.Error("a rule covering no event has no decisive event")
	}
	if !rule.ChargesExtraGuests() || (LinePackageRule{IncludedGuests: iv(500)}).ChargesExtraGuests() {
		t.Error("only a complete rule charges extra guests")
	}
}
