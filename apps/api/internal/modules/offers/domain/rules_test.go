package domain

import (
	"errors"
	"testing"
)

func ip(v int) *int       { return &v }
func i64p(v int64) *int64 { return &v }

const (
	kindAkad    int64 = 1
	kindResepsi int64 = 2
	kindLamaran int64 = 3
)

func TestExtraGuests(t *testing.T) {
	resepsi500 := PackageRule{Price: 120_000_000, EventKindIDs: []int64{kindAkad, kindResepsi},
		IncludedGuests: ip(500), ExtraGuestStep: ip(50), ExtraGuestStepPrice: i64p(4_000_000)}
	perPax := resepsi500
	perPax.ExtraGuestStep, perPax.ExtraGuestStepPrice = ip(1), i64p(150_000)
	flat := PackageRule{Price: 25_000_000, EventKindIDs: []int64{kindAkad}}

	tests := []struct {
		name   string
		rule   PackageRule
		events []EventGuests
		want   ExtraGuestsResult
	}{
		{"620 guests, 50 per step: 3 steps", resepsi500, []EventGuests{{kindResepsi, 620}},
			ExtraGuestsResult{Guests: 620, Over: 120, Steps: 3, Amount: 12_000_000}},
		{"450 guests: below included, nothing", resepsi500, []EventGuests{{kindResepsi, 450}},
			ExtraGuestsResult{Guests: 450}},
		{"exactly included: nothing", resepsi500, []EventGuests{{kindResepsi, 500}},
			ExtraGuestsResult{Guests: 500}},
		{"one over starts a step", resepsi500, []EventGuests{{kindResepsi, 501}},
			ExtraGuestsResult{Guests: 501, Over: 1, Steps: 1, Amount: 4_000_000}},
		{"step 1 is per pax", perPax, []EventGuests{{kindResepsi, 620}},
			ExtraGuestsResult{Guests: 620, Over: 120, Steps: 120, Amount: 18_000_000}},
		{"largest covered event counts, not the sum", resepsi500,
			[]EventGuests{{kindAkad, 150}, {kindResepsi, 550}},
			ExtraGuestsResult{Guests: 550, Over: 50, Steps: 1, Amount: 4_000_000}},
		{"events of other kinds are ignored", resepsi500,
			[]EventGuests{{kindLamaran, 900}, {kindResepsi, 480}},
			ExtraGuestsResult{Guests: 480}},
		{"no included number: never charged", flat, []EventGuests{{kindAkad, 900}},
			ExtraGuestsResult{Guests: 900}},
		{"no events yet", resepsi500, nil, ExtraGuestsResult{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtraGuests(tt.rule, tt.events); got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLineTotals(t *testing.T) {
	tests := []struct {
		name    string
		lines   []Line
		pkp     bool
		want    Totals
		wantErr error
	}{
		{"PKP: subtotal 100 jt gives PPN 11 jt", []Line{{LinePackage, 100_000_000}}, true,
			Totals{Gross: 100_000_000, Subtotal: 100_000_000, Tax: 11_000_000, Total: 111_000_000}, nil},
		{"not PKP: no PPN", []Line{{LinePackage, 100_000_000}}, false,
			Totals{Gross: 100_000_000, Subtotal: 100_000_000, Total: 100_000_000}, nil},
		{"package, extra guests, item, discount, component", []Line{
			{LinePackage, 120_000_000}, {LineComponent, 0}, {LineExtraGuests, 12_000_000},
			{LineItem, 3_600_000}, {LineDiscount, 5_000_000},
		}, true, Totals{Gross: 135_600_000, Discount: 5_000_000, Subtotal: 130_600_000, Tax: 14_366_000, Total: 144_966_000}, nil},
		{"PPN rounds down to the rupiah", []Line{{LineItem, 1_234_567}}, true,
			Totals{Gross: 1_234_567, Subtotal: 1_234_567, Tax: 135_802, Total: 1_370_369}, nil},
		{"discount equal to the subtotal is allowed", []Line{{LineItem, 5_000_000}, {LineDiscount, 5_000_000}}, true,
			Totals{Gross: 5_000_000, Discount: 5_000_000}, nil},
		{"discount above the subtotal is refused", []Line{{LinePackage, 10_000_000}, {LineDiscount, 10_000_001}}, false,
			Totals{}, ErrDiscountExceedsSubtotal},
		{"negative amount is refused", []Line{{LineItem, -1}}, false, Totals{}, ErrNegativeAmount},
		{"unknown kind is refused", []Line{{"Other", 1}}, false, Totals{}, ErrUnknownLineKind},
		{"no lines", nil, true, Totals{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LineTotals(tt.lines, tt.pkp)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCheckCapacity(t *testing.T) {
	gedung := int64(10)
	masjid := int64(11)
	venues := map[int64]VenueCapacity{
		gedung: {VenueID: gedung, Name: "Gedung Melati", Standing: ip(600)},
		masjid: {VenueID: masjid, Name: "Masjid Al-Ikhlas", Seated: ip(150)},
	}
	events := []EventCapacity{
		{KindName: "Akad", Layout: LayoutSeated, VenueID: &masjid, Sessions: []SessionGuests{{"Sesi 1", 120}}},
		{KindName: "Resepsi", Layout: LayoutStanding, VenueID: &gedung, Sessions: []SessionGuests{{"Sesi 1", 650}, {"Sesi 2", 550}}},
		{KindName: "Lamaran", Layout: LayoutSeated, VenueID: &gedung, Sessions: []SessionGuests{{"Sesi 1", 40}}},
		{KindName: "Pengajian", Layout: LayoutSeated, Sessions: []SessionGuests{{"Sesi 1", 5000}}},
	}
	got := CheckCapacity(events, venues)
	want := []CapacityProblem{
		{1, 0, "Gedung Melati muat 600 orang berdiri, Sesi 1 Resepsi 650"},
		{2, -1, "Isi kapasitas duduk Gedung Melati di Venue"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d problems %+v, want %+v", len(got), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("problem %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPackageVenueProblem(t *testing.T) {
	gedung := VenueCapacity{Name: "Gedung Melati", Standing: ip(600), Seated: ip(400)}
	if msg := PackageVenueProblem(ip(650), gedung); msg != "Gedung Melati muat paling banyak 600 orang berdiri, tamu termasuk 650" {
		t.Fatalf("650 in a 600 venue: %q", msg)
	}
	if msg := PackageVenueProblem(ip(600), gedung); msg != "" {
		t.Fatalf("600 fits: %q", msg)
	}
	if msg := PackageVenueProblem(nil, gedung); msg != "" {
		t.Fatalf("no included guests: %q", msg)
	}
	seatedOnly := VenueCapacity{Name: "Masjid", Seated: ip(150)}
	if msg := PackageVenueProblem(ip(200), seatedOnly); msg != "Masjid muat paling banyak 150 orang duduk, tamu termasuk 200" {
		t.Fatalf("seated only: %q", msg)
	}
}

func TestValidateRule(t *testing.T) {
	ok := PackageRule{Price: 1, EventKindIDs: []int64{1}, IncludedGuests: ip(500), ExtraGuestStep: ip(50), ExtraGuestStepPrice: i64p(4_000_000)}
	if p := ValidateRule(ok); len(p) != 0 {
		t.Fatalf("valid rule: %+v", p)
	}
	flat := PackageRule{Price: 0, EventKindIDs: []int64{1}}
	if p := ValidateRule(flat); len(p) != 0 {
		t.Fatalf("flat rule: %+v", p)
	}
	cases := []struct {
		name  string
		rule  PackageRule
		field string
	}{
		{"no event kinds", PackageRule{Price: 1}, "eventKindIds"},
		{"negative price", PackageRule{Price: -1, EventKindIDs: []int64{1}}, "price"},
		{"step 0 with a price", PackageRule{EventKindIDs: []int64{1}, IncludedGuests: ip(500), ExtraGuestStep: ip(0), ExtraGuestStepPrice: i64p(1)}, "extraGuestStep"},
		{"price without a step", PackageRule{EventKindIDs: []int64{1}, IncludedGuests: ip(500), ExtraGuestStepPrice: i64p(1)}, "extraGuestStep"},
		{"step without a price", PackageRule{EventKindIDs: []int64{1}, IncludedGuests: ip(500), ExtraGuestStep: ip(50)}, "extraGuestStepPrice"},
		{"extra without included", PackageRule{EventKindIDs: []int64{1}, ExtraGuestStep: ip(50), ExtraGuestStepPrice: i64p(1)}, "includedGuests"},
		{"included 0", PackageRule{EventKindIDs: []int64{1}, IncludedGuests: ip(0)}, "includedGuests"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			found := false
			for _, p := range ValidateRule(c.rule) {
				if p.Field == c.field {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected a problem on %s, got %+v", c.field, ValidateRule(c.rule))
			}
		})
	}
}
