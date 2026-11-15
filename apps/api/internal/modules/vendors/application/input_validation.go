package application

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"elproof/internal/shared/apperror"
)

// Column widths of the vendors/venues tables (migrations 000006, 000020,
// 000023). A value past them, or a negative number in a BIGINT/INT UNSIGNED
// column, makes MySQL refuse the write -- which surfaced as a 500 instead of
// a 422 the form can show next to the field (QA EP-03).
const (
	maxDirectoryNameLen    = 150
	maxDirectoryPICNameLen = 150
	maxDirectoryPhoneLen   = 30
	maxDirectoryEmailLen   = 150
	maxDirectoryAddressLen = 255
	maxVenueCapacity       = math.MaxUint32
)

// fieldErrors collects every problem of one input so the response names them
// all at once, keyed by the JSON field name the form uses.
type fieldErrors map[string][]string

func (f fieldErrors) add(key, msg string) { f[key] = append(f[key], msg) }

// err returns nil when nothing was collected, otherwise a 422 whose message is
// the first problem in `order` (the form's reading order).
func (f fieldErrors) err(order []string, fallback string) error {
	if len(f) == 0 {
		return nil
	}
	msg := fallback
	for _, k := range order {
		if m := f[k]; len(m) > 0 {
			msg = m[0]
			break
		}
	}
	return apperror.Validation(msg, f)
}

func (f fieldErrors) maxLen(key, value, label string, max int) {
	if utf8.RuneCountInString(value) > max {
		f.add(key, label+" maksimal "+strconv.Itoa(max)+" karakter")
	}
}

func (f fieldErrors) nonNegative(key string, value *int64, label string) {
	if value != nil && *value < 0 {
		f.add(key, label+" tidak boleh kurang dari 0. Isi 0 atau lebih, atau kosongkan.")
	}
}

var vendorFieldOrder = []string{"name", "categoryId", "picName", "phone", "email", "city", "address", "priceAkad", "priceAkadResepsi", "priceResepsi"}

// validateVendorInput trims the name in place and checks what the database
// would otherwise refuse: a name (required), column widths, and prices that
// are never negative. Other "required" fields stay a form rule (see VendorInput).
func validateVendorInput(in *VendorInput) error {
	f := fieldErrors{}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		f.add("name", "Isi nama vendor")
	} else {
		f.maxLen("name", in.Name, "Nama vendor", maxDirectoryNameLen)
	}
	f.maxLen("picName", in.PICName, "Nama PIC", maxDirectoryPICNameLen)
	f.maxLen("phone", in.Phone, "No. telepon", maxDirectoryPhoneLen)
	f.maxLen("email", in.Email, "Email", maxDirectoryEmailLen)
	f.maxLen("address", in.Address, "Alamat", maxDirectoryAddressLen)
	f.nonNegative("priceAkad", in.PriceAkad, "Harga Akad")
	f.nonNegative("priceAkadResepsi", in.PriceAkadResepsi, "Harga Akad + Resepsi")
	f.nonNegative("priceResepsi", in.PriceResepsi, "Harga Resepsi")
	return f.err(vendorFieldOrder, "Data vendor belum lengkap atau tidak sesuai")
}

var venueFieldOrder = []string{"name", "picName", "phonePic", "phoneVenue", "email", "city", "address", "rentalPrice", "charge", "capacity"}

// validateVenueInput is validateVendorInput's twin for venues: name required,
// column widths, and rental price, charge and capacity never negative.
func validateVenueInput(in *VenueInput) error {
	f := fieldErrors{}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		f.add("name", "Isi nama venue")
	} else {
		f.maxLen("name", in.Name, "Nama venue", maxDirectoryNameLen)
	}
	f.maxLen("picName", in.PICName, "Nama PIC", maxDirectoryPICNameLen)
	f.maxLen("phonePic", in.PhonePIC, "No. telepon PIC", maxDirectoryPhoneLen)
	f.maxLen("phoneVenue", in.PhoneVenue, "No. telepon venue", maxDirectoryPhoneLen)
	f.maxLen("email", in.Email, "Email", maxDirectoryEmailLen)
	f.maxLen("address", in.Address, "Alamat", maxDirectoryAddressLen)
	f.nonNegative("rentalPrice", in.RentalPrice, "Harga sewa")
	f.nonNegative("charge", in.Charge, "Charge")
	if in.Capacity != nil {
		if *in.Capacity < 0 {
			f.add("capacity", "Kapasitas tidak boleh kurang dari 0. Isi 0 atau lebih, atau kosongkan.")
		} else if int64(*in.Capacity) > maxVenueCapacity {
			f.add("capacity", "Kapasitas terlalu besar. Periksa lagi angkanya.")
		}
	}
	return f.err(venueFieldOrder, "Data venue belum lengkap atau tidak sesuai")
}

// importRowProblem is the first reason an import row would be refused by the
// database (a name too long for its column, a negative number); "" when none.
// The row-level message is shown in the import result list.
func importRowProblem(name string, numbers ...labeledNumber) string {
	if utf8.RuneCountInString(name) > maxDirectoryNameLen {
		return "Nama maksimal " + strconv.Itoa(maxDirectoryNameLen) + " karakter"
	}
	for _, n := range numbers {
		if n.value == nil {
			continue
		}
		if *n.value < 0 {
			return n.label + " tidak boleh kurang dari 0"
		}
		if n.max > 0 && *n.value > n.max {
			return n.label + " terlalu besar"
		}
	}
	return ""
}

type labeledNumber struct {
	label string
	value *int64
	max   int64 // 0 = no upper bound beyond int64
}

func capacityNumber(c *int) labeledNumber {
	n := labeledNumber{label: "Kapasitas", max: maxVenueCapacity}
	if c != nil {
		v := int64(*c)
		n.value = &v
	}
	return n
}
