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
	// minVenuePhonePICLen mirrors the venue form (venue.schema.ts phonePic min 6).
	minVenuePhonePICLen = 6
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
	f.nonNegative("priceResepsi", in.PriceResepsi, "Harga Resepsi saja")
	return f.err(vendorFieldOrder, "Data vendor belum lengkap atau tidak sesuai")
}

var venueFieldOrder = []string{"name", "picName", "phonePic", "phoneVenue", "email", "city", "address", "capacityStanding", "capacitySeated", "priceAkad", "priceResepsi", "priceAkadResepsi", "charge"}

// MsgVenueCapacityRequired is the message both capacity fields carry when
// neither is filled (PLAN paket-wo §3.3: at least one of them, at least 1).
const MsgVenueCapacityRequired = "Isi kapasitas berdiri atau duduk, minimal satu"

// venueCapacityProblems judges the two capacities of a venue: each filled one
// is at least 1 and fits INT UNSIGNED, and at least one is filled. The result is
// keyed by JSON field name; a missing pair names both fields.
func venueCapacityProblems(standing, seated *int) map[string]string {
	out := map[string]string{}
	check := func(key, label string, v *int) {
		switch {
		case v == nil:
		case *v < 1:
			out[key] = "Kapasitas " + label + " minimal 1 orang. Kosongkan bila tidak ada."
		case int64(*v) > maxVenueCapacity:
			out[key] = "Kapasitas " + label + " terlalu besar. Periksa lagi angkanya."
		}
	}
	check("capacityStanding", "berdiri", standing)
	check("capacitySeated", "duduk", seated)
	if standing == nil && seated == nil {
		out["capacityStanding"] = MsgVenueCapacityRequired
		out["capacitySeated"] = MsgVenueCapacityRequired
	}
	return out
}

// validateVenueInput is validateVendorInput's twin for venues: name required,
// column widths, a capacity per layout (at least one), and prices and charge
// never negative.
func validateVenueInput(in *VenueInput) error {
	f := fieldErrors{}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		f.add("name", "Isi nama venue")
	} else {
		f.maxLen("name", in.Name, "Nama venue", maxDirectoryNameLen)
	}
	f.maxLen("picName", in.PICName, "Nama PIC", maxDirectoryPICNameLen)
	// No. telepon PIC is required, the same rule as the venue form (FQ-06):
	// at least minVenuePhonePICLen characters once trimmed.
	in.PhonePIC = strings.TrimSpace(in.PhonePIC)
	if utf8.RuneCountInString(in.PhonePIC) < minVenuePhonePICLen {
		f.add("phonePic", "Isi nomor telepon PIC, minimal "+strconv.Itoa(minVenuePhonePICLen)+" angka")
	} else {
		f.maxLen("phonePic", in.PhonePIC, "No. telepon PIC", maxDirectoryPhoneLen)
	}
	f.maxLen("phoneVenue", in.PhoneVenue, "No. telepon venue", maxDirectoryPhoneLen)
	f.maxLen("email", in.Email, "Email", maxDirectoryEmailLen)
	f.maxLen("address", in.Address, "Alamat", maxDirectoryAddressLen)
	capacity := venueCapacityProblems(in.CapacityStanding, in.CapacitySeated)
	for _, key := range []string{"capacityStanding", "capacitySeated"} {
		if msg, ok := capacity[key]; ok {
			f.add(key, msg)
		}
	}
	f.nonNegative("priceAkad", in.PriceAkad, "Harga Akad")
	f.nonNegative("priceResepsi", in.PriceResepsi, "Harga Resepsi")
	f.nonNegative("priceAkadResepsi", in.PriceAkadResepsi, "Harga Akad + Resepsi")
	f.nonNegative("charge", in.Charge, "Charge tambahan")
	return f.err(venueFieldOrder, "Data venue belum lengkap atau tidak sesuai")
}
