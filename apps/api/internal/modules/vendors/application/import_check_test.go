package application

import (
	"reflect"
	"strings"
	"testing"
)

func validVendorRow() VendorImportRow {
	return VendorImportRow{
		Name: "Sanggar Ayu", CategoryName: "katering", PICName: "Ayu", Phone: "081234567890", City: "Kota Bandung",
		PriceAkad: i64(1_000_000), PriceAkadResepsi: i64(2_000_000),
	}
}

// FQ-07: a refused import row names each column and what is wrong with it, all
// problems of the row at once, instead of one sentence listing seven columns.
func TestCheckVendorImportRow(t *testing.T) {
	categories := map[string]int64{"katering": 7}
	id, c := checkVendorImportRow(validVendorRow(), categories)
	if c.failed() || id != 7 {
		t.Fatalf("valid row refused: id=%d %+v", id, c.errs)
	}

	row := validVendorRow()
	row.CategoryName = "Bunga"
	row.PriceAkad = i64(-5)
	row.PriceAkadResepsi = nil
	row.NotNumber = []string{"Harga Akad+Resepsi"}
	row.City = "Gotham"
	row.PICName = ""
	id, c = checkVendorImportRow(row, categories)
	want := []ImportCellError{
		{"Kategori", `Kategori "Bunga" tidak ditemukan. Pilih dari daftar kategori di template.`},
		{"Nama PIC", "Nama PIC wajib diisi"},
		{"Kota", `Kota "Gotham" tidak ada di daftar kota. Pilih dari daftar di template.`},
		{"Harga Akad", "Harga Akad tidak boleh negatif"},
		{"Harga Akad+Resepsi", "Harga Akad+Resepsi harus berupa angka bulat"},
	}
	if id != 0 || !reflect.DeepEqual(c.errs, want) {
		t.Fatalf("errors = %+v", c.errs)
	}
	if s := c.summary(); !strings.HasPrefix(s, `Kategori "Bunga" tidak ditemukan`) || strings.Count(s, "; ") != 4 {
		t.Errorf("summary = %q", s)
	}

	for _, tc := range []struct {
		name   string
		mutate func(*VendorImportRow)
		column string
		msg    string
	}{
		{"empty name", func(r *VendorImportRow) { r.Name = "" }, "Nama Vendor", "Nama Vendor wajib diisi"},
		{"long name", func(r *VendorImportRow) { r.Name = strings.Repeat("a", 151) }, "Nama Vendor", "Nama Vendor maksimal 150 karakter"},
		{"no category", func(r *VendorImportRow) { r.CategoryName = "" }, "Kategori", "Kategori wajib diisi"},
		{"no phone", func(r *VendorImportRow) { r.Phone = "" }, "No Tlp Vendor", "No Tlp Vendor wajib diisi"},
		{"no city", func(r *VendorImportRow) { r.City = "" }, "Kota", "Kota wajib diisi"},
		{"zero price", func(r *VendorImportRow) { r.PriceAkad = i64(0) }, "Harga Akad", "Harga Akad harus lebih dari 0"},
		{"no price", func(r *VendorImportRow) { r.PriceAkadResepsi = nil }, "Harga Akad+Resepsi", "Harga Akad+Resepsi wajib diisi"},
		{"long address", func(r *VendorImportRow) { r.Address = strings.Repeat("x", 256) }, "Alamat", "Alamat maksimal 255 karakter"},
	} {
		row := validVendorRow()
		tc.mutate(&row)
		_, c := checkVendorImportRow(row, categories)
		if len(c.errs) != 1 || c.errs[0].Column != tc.column || c.errs[0].Message != tc.msg {
			t.Errorf("%s: %+v", tc.name, c.errs)
		}
	}
}

func TestCheckVenueImportRow(t *testing.T) {
	valid := func() VenueImportRow {
		return VenueImportRow{Name: "Gedung", PICName: "Budi", PhonePIC: "081234567", City: "Kota Bandung", PriceAkadResepsi: i64(0), CapacityStanding: iptr(300)}
	}
	if c := checkVenueImportRow(valid()); c.failed() {
		t.Fatalf("valid row refused: %+v", c.errs)
	}
	// Only a seated capacity is enough too.
	row := valid()
	row.CapacityStanding, row.CapacitySeated = nil, iptr(200)
	if c := checkVenueImportRow(row); c.failed() {
		t.Fatalf("seated-only row refused: %+v", c.errs)
	}
	row = valid()
	row.PhonePIC = "123"
	row.Charge = i64(-1)
	row.CapacityStanding = iptr(maxVenueCapacity + 1)
	row.CapacitySeated = iptr(0)
	row.NotNumber = []string{"Harga Resepsi"}
	row.PriceAkadResepsi = i64(-5)
	want := []ImportCellError{
		{"No Tlp PIC", "No Tlp PIC minimal 6 karakter"},
		{"Kapasitas Berdiri", "Kapasitas Berdiri terlalu besar. Periksa lagi angkanya."},
		{"Kapasitas Duduk", "Kapasitas Duduk harus lebih dari 0"},
		{"Harga Resepsi", "Harga Resepsi harus berupa angka bulat"},
		{"Harga Akad+Resepsi", "Harga Akad+Resepsi tidak boleh negatif"},
		{"Charge", "Charge tidak boleh negatif"},
	}
	if c := checkVenueImportRow(row); !reflect.DeepEqual(c.errs, want) {
		t.Fatalf("errors = %+v", c.errs)
	}
	row = valid()
	row.PhonePIC, row.Name = "", ""
	if c := checkVenueImportRow(row); len(c.errs) != 2 || c.errs[0].Column != "Nama Venue" || c.errs[1].Message != "No Tlp PIC wajib diisi" {
		t.Errorf("empty name and phone: %+v", c.errs)
	}
	// Neither capacity: one problem, named on Kapasitas Berdiri.
	row = valid()
	row.CapacityStanding = nil
	want = []ImportCellError{{"Kapasitas Berdiri", "Isi Kapasitas Berdiri atau Kapasitas Duduk, minimal satu"}}
	if c := checkVenueImportRow(row); !reflect.DeepEqual(c.errs, want) {
		t.Errorf("no capacity: %+v", c.errs)
	}
}
