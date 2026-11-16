package application

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"elproof/internal/shared/apperror"
)

func i64(v int64) *int64 { return &v }
func iptr(v int) *int    { return &v }

// fieldsOf asserts err is a 422 and returns its per-field messages.
func fieldsOf(t *testing.T, err error) map[string][]string {
	t.Helper()
	if err == nil {
		t.Fatal("want a validation error, got nil")
	}
	if got := apperror.HTTPStatus(err); got != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (err %v)", got, err)
	}
	ae, _ := apperror.As(err)
	return ae.Fields
}

func TestValidateVendorInput(t *testing.T) {
	valid := func() VendorInput {
		return VendorInput{Name: "  Sanggar Rias Ayu ", CategoryID: 1, PriceAkad: i64(5_000_000), PriceAkadResepsi: i64(0), PriceResepsi: nil}
	}

	in := valid()
	if err := validateVendorInput(&in); err != nil {
		t.Fatalf("valid input refused: %v", err)
	}
	if in.Name != "Sanggar Rias Ayu" {
		t.Fatalf("name not trimmed: %q", in.Name)
	}

	cases := []struct {
		name   string
		mutate func(*VendorInput)
		fields []string
	}{
		{"empty name", func(v *VendorInput) { v.Name = "" }, []string{"name"}},
		{"blank name", func(v *VendorInput) { v.Name = " \t " }, []string{"name"}},
		{"name too long", func(v *VendorInput) { v.Name = strings.Repeat("é", 151) }, []string{"name"}},
		{"negative akad", func(v *VendorInput) { v.PriceAkad = i64(-5000) }, []string{"priceAkad"}},
		{"negative akad resepsi", func(v *VendorInput) { v.PriceAkadResepsi = i64(-1) }, []string{"priceAkadResepsi"}},
		{"negative resepsi", func(v *VendorInput) { v.PriceResepsi = i64(-1) }, []string{"priceResepsi"}},
		{"phone too long", func(v *VendorInput) { v.Phone = strings.Repeat("1", 31) }, []string{"phone"}},
		{"everything at once", func(v *VendorInput) { v.Name = ""; v.PriceAkad = i64(-1); v.PriceResepsi = i64(-2) }, []string{"name", "priceAkad", "priceResepsi"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := valid()
			tc.mutate(&in)
			fields := fieldsOf(t, validateVendorInput(&in))
			if len(fields) != len(tc.fields) {
				t.Fatalf("fields = %v, want keys %v", fields, tc.fields)
			}
			for _, k := range tc.fields {
				if len(fields[k]) == 0 {
					t.Fatalf("no message for %q in %v", k, fields)
				}
			}
		})
	}

	// A name of exactly the column width is fine (counted in characters, not bytes).
	in = valid()
	in.Name = strings.Repeat("é", 150)
	if err := validateVendorInput(&in); err != nil {
		t.Fatalf("150-character name refused: %v", err)
	}
}

func TestValidateVenueInput(t *testing.T) {
	valid := func() VenueInput {
		return VenueInput{Name: "Gedung Serbaguna", PhonePIC: "081234567", RentalPrice: i64(10_000_000), Charge: i64(0), Capacity: iptr(500)}
	}
	in := valid()
	if err := validateVenueInput(&in); err != nil {
		t.Fatalf("valid input refused: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*VenueInput)
		field  string
	}{
		{"empty name", func(v *VenueInput) { v.Name = "  " }, "name"},
		{"negative rental", func(v *VenueInput) { v.RentalPrice = i64(-1) }, "rentalPrice"},
		{"negative charge", func(v *VenueInput) { v.Charge = i64(-1) }, "charge"},
		{"negative capacity", func(v *VenueInput) { v.Capacity = iptr(-10) }, "capacity"},
		{"capacity past INT UNSIGNED", func(v *VenueInput) { v.Capacity = iptr(maxVenueCapacity + 1) }, "capacity"},
		{"no PIC phone (FQ-06)", func(v *VenueInput) { v.PhonePIC = "" }, "phonePic"},
		{"blank PIC phone", func(v *VenueInput) { v.PhonePIC = "      " }, "phonePic"},
		{"PIC phone too short", func(v *VenueInput) { v.PhonePIC = "12345" }, "phonePic"},
		{"PIC phone too long", func(v *VenueInput) { v.PhonePIC = strings.Repeat("1", 31) }, "phonePic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := valid()
			tc.mutate(&in)
			fields := fieldsOf(t, validateVenueInput(&in))
			if len(fields[tc.field]) == 0 || len(fields) != 1 {
				t.Fatalf("fields = %v, want only %q", fields, tc.field)
			}
		})
	}
}

// Create/Update refuse a bad input before touching any repository: the
// services below have none, so reaching one would panic.
func TestServicesValidateBeforeStoring(t *testing.T) {
	ctx := context.Background()
	vendors := &VendorService{}
	if _, err := vendors.Create(ctx, 1, VendorInput{Name: "", CategoryID: 1, PriceAkad: i64(-5000)}); err == nil {
		t.Fatal("vendor create accepted a nameless vendor with a negative price")
	} else if f := fieldsOf(t, err); len(f["name"]) == 0 || len(f["priceAkad"]) == 0 {
		t.Fatalf("vendor create fields = %v", f)
	}
	if _, err := vendors.Update(ctx, 1, 2, VendorInput{Name: "Ayu", PriceResepsi: i64(-1)}); err == nil {
		t.Fatal("vendor update accepted a negative price")
	} else if f := fieldsOf(t, err); len(f["priceResepsi"]) == 0 {
		t.Fatalf("vendor update fields = %v", f)
	}

	venues := &VenueService{}
	if _, err := venues.Create(ctx, 1, VenueInput{Name: "", PhonePIC: "081234567", RentalPrice: i64(-1)}); err == nil {
		t.Fatal("venue create accepted a nameless venue")
	} else if f := fieldsOf(t, err); len(f["name"]) == 0 || len(f["rentalPrice"]) == 0 {
		t.Fatalf("venue create fields = %v", f)
	}
	if _, err := venues.Update(ctx, 1, 2, VenueInput{Name: "Gedung", PhonePIC: "081234567", Capacity: iptr(-1)}); err == nil {
		t.Fatal("venue update accepted a negative capacity")
	} else if f := fieldsOf(t, err); len(f["capacity"]) == 0 {
		t.Fatalf("venue update fields = %v", f)
	}
}
