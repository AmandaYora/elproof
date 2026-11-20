package presentation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
)

func p64(v int64) *int64 { return &v }
func pint(v int) *int    { return &v }

// linesFixture is a Rincian with one line of each kind (amounts chosen so
// each is a unique, searchable number).
func linesFixture() *application.LinesView {
	rule := &domain.LinePackageRule{PackageID: 7, Name: "Paket Gold", Price: 150_000_111, EventKindIDs: []int64{2},
		IncludedGuests: pint(500), ExtraGuestStep: pint(50), ExtraGuestStepPrice: p64(4_000_111)}
	at := time.Date(2026, 11, 20, 1, 2, 3, 456000, time.UTC)
	lines := []application.LineView{
		{ProjectLine: domain.ProjectLine{ID: 1, Kind: domain.LinePackage, PackageRule: rule, Description: "Paket Gold", Amount: 150_000_111, SupplierRef: p64(9), UpdatedAt: at}},
		{ProjectLine: domain.ProjectLine{ID: 2, Kind: domain.LineComponent, ParentLineID: p64(1), Description: "Dekorasi pelaminan", Amount: 0, SupplierRef: p64(9), UpdatedAt: at}},
		{ProjectLine: domain.ProjectLine{ID: 3, Kind: domain.LineExtraGuests, ParentLineID: p64(1), Description: "Tambahan tamu", Amount: 8_000_222, UpdatedAt: at},
			ExtraGuests: &application.ExtraGuestsView{EventID: p64(4), EventKindName: "Resepsi", Guests: 600, IncludedGuests: 500, Over: 100, Step: 50, StepPrice: 4_000_111, Steps: 2}},
		{ProjectLine: domain.ProjectLine{ID: 4, Kind: domain.LineItem, Description: "Photobooth", Quantity: p64(2), Unit: "unit", UnitPrice: p64(3_000_333), Amount: 6_000_666, SupplierRef: p64(11), UpdatedAt: at}},
		{ProjectLine: domain.ProjectLine{ID: 5, Kind: domain.LineDiscount, Description: "Potongan", Amount: 5_000_444, UpdatedAt: at}},
	}
	return &application.LinesView{
		ProjectID: 12, Lines: lines, IsPKP: true, LinesVersion: at,
		Totals:     offerscontracts.Totals{Gross: 164_000_999, Discount: 5_000_444, Subtotal: 159_000_555, Tax: 17_490_061, Total: 176_490_616},
		TermsTotal: 100_000_000, HasTerms: true, TermsMismatch: true,
		Margin: domain.Margin{Subtotal: 159_000_555, SupplierCost: 20_000_000, MissingCost: 2, Margin: 139_000_555},
	}
}

var fixtureAmounts = []string{"150000111", "8000222", "3000333", "6000666", "5000444", "4000111", "164000999", "159000555", "17490061", "176490616"}

func portalJSON(t *testing.T, can ...string) (string, map[string]any) {
	t.Helper()
	allowed := map[string]bool{}
	for _, p := range can {
		allowed[p] = true
	}
	policy := domain.NewPortalPolicy(func(p string) bool { return allowed[p] })
	raw, err := json.Marshal(toPortalLinesResponse(linesFixture(), policy))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return string(raw), m
}

// Keluarga (portal.vendor.view without portal.finance.view) sees the names of
// the packages and what they contain, and not one amount.
func TestPortalLinesKeluargaSeesNoAmounts(t *testing.T) {
	raw, m := portalJSON(t, domain.PermPortalVendor)
	if n := len(m["lines"].([]any)); n != 5 {
		t.Fatalf("every line's name is shown: %d", n)
	}
	for _, amount := range fixtureAmounts {
		if strings.Contains(raw, amount) {
			t.Errorf("Keluarga must not see %s: %s", amount, raw)
		}
	}
	for _, key := range []string{`"amount"`, `"unitPrice"`, `"price"`, `"extraGuestStepPrice"`, `"stepPrice"`, `"totals"`,
		`"supplierRef"`, `"linesVersion"`, `"termsTotal"`, `"termsMismatch"`, `"hasTerms"`, `"margin"`, `"updatedAt"`} {
		if strings.Contains(raw, key) {
			t.Errorf("Keluarga's Rincian carries %s: %s", key, raw)
		}
	}
	if !strings.Contains(raw, "Dekorasi pelaminan") || !strings.Contains(raw, "Paket Gold") {
		t.Errorf("the names are shown: %s", raw)
	}
}

// Pasangan (vendor + finance) sees the amounts, except a component's (it has
// no price of its own) and never the supplier behind a line or the WO's
// internals (version, schedule check, margin).
func TestPortalLinesPasanganSeesAmountsButNoComponentPrice(t *testing.T) {
	raw, m := portalJSON(t, domain.PermPortalVendor, domain.PermPortalFinance)
	lines := m["lines"].([]any)
	for _, l := range lines {
		line := l.(map[string]any)
		_, hasAmount := line["amount"]
		if line["kind"] == "Component" {
			if hasAmount || line["unitPrice"] != nil {
				t.Errorf("a component's price is never shown: %v", line)
			}
			continue
		}
		if !hasAmount {
			t.Errorf("%s: the amount is shown with portal.finance.view: %v", line["kind"], line)
		}
		if _, has := line["supplierRef"]; has {
			t.Errorf("the supplier behind a line never reaches the portal: %v", line)
		}
	}
	if m["totals"] == nil || m["totals"].(map[string]any)["total"] != float64(176_490_616) {
		t.Errorf("totals with portal.finance.view: %v", m["totals"])
	}
	for _, key := range []string{`"linesVersion"`, `"termsTotal"`, `"termsMismatch"`, `"margin"`, `"supplierRef"`} {
		if strings.Contains(raw, key) {
			t.Errorf("the portal never carries %s: %s", key, raw)
		}
	}
}

func TestPortalLinesNeedVendorView(t *testing.T) {
	_, m := portalJSON(t, domain.PermPortalFinance)
	if n := len(m["lines"].([]any)); n != 0 || m["totals"] == nil {
		t.Errorf("without portal.vendor.view no line is listed (the totals follow finance): %v", m)
	}
}

// Staff see everything; the margin (and its "belum lengkap" count) only with
// finance.margin.view.
func TestStaffLinesMarginOnlyWithThePermission(t *testing.T) {
	with := asMap(t, toLinesResponse(linesFixture(), true))
	margin, ok := with["margin"].(map[string]any)
	if !ok || margin["missingCost"] != float64(2) || margin["incomplete"] != true || margin["margin"] != float64(139_000_555) {
		t.Errorf("margin: %v", with["margin"])
	}
	if with["termsMismatch"] != true || with["termsTotal"] != float64(100_000_000) || with["linesVersion"] != "2026-11-20T01:02:03.000456Z" {
		t.Errorf("schedule check and version: %v", with)
	}
	without := asMap(t, toLinesResponse(linesFixture(), false))
	if _, has := without["margin"]; has {
		t.Errorf("without finance.margin.view the margin is absent: %v", without)
	}
	line := without["lines"].([]any)[1].(map[string]any)
	if line["amount"] != float64(0) || line["parentId"] != float64(1) {
		t.Errorf("staff see a component at 0 under its package: %v", line)
	}
	extra := without["lines"].([]any)[2].(map[string]any)["extraGuests"].(map[string]any)
	if extra["over"] != float64(100) || extra["steps"] != float64(2) || extra["stepPrice"] != float64(4_000_111) || extra["eventKindName"] != "Resepsi" {
		t.Errorf("the extra-guest arithmetic: %v", extra)
	}
}

// The project as the portal shows it: Keluarga gets no contract value or PPN
// keys at all, Pasangan gets them; the lines' version never.
func TestPortalProjectMoneyFollowsFinance(t *testing.T) {
	pr := domain.Project{ID: 12, Name: "A & B", IsPKP: true, ContractSubtotal: 100, ContractTax: 11, ContractTotal: 111,
		LinesVersion: time.Now(), PrepStartDate: time.Now(), PICSalesStaffID: 4}
	family := asMap(t, toPortalProjectResponse(domain.NewPortalPolicy(func(p string) bool { return p == domain.PermPortalVendor }).Project(pr)))
	for _, k := range []string{"contractValue", "contractSubtotal", "contractTax", "linesVersion", "prepStartDate"} {
		if v, has := family[k]; has && v != "" {
			t.Errorf("Keluarga's project carries %s: %v", k, family)
		}
	}
	couple := asMap(t, toPortalProjectResponse(domain.NewPortalPolicy(func(string) bool { return true }).Project(pr)))
	if couple["contractValue"] != float64(111) || couple["contractTax"] != float64(11) {
		t.Errorf("Pasangan sees the contract: %v", couple)
	}
	if v, has := couple["linesVersion"]; has && v != "" {
		t.Errorf("the lines' version is internal: %v", couple)
	}
}
