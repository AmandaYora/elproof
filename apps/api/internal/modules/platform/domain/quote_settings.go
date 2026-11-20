package domain

import (
	"strings"
	"unicode"
)

// QuoteSettings is the "Penawaran" part of the WO's business profile
// (paket-wo PLAN §3.3): the document code that starts every quotation number,
// whether the WO is a PKP (PPN is added), how many days a new quotation stays
// valid, and the default terms copied into each new quotation.
type QuoteSettings struct {
	DocumentCode        string
	IsPKP               bool
	DefaultValidityDays int
	DefaultTermsText    string
}

const (
	DocumentCodeMinLen         = 2
	DocumentCodeMaxLen         = 5
	DefaultQuoteValidityDays   = 14
	MaxQuoteValidityDays       = 365
	MaxDefaultTermsTextLen     = 5000
	fallbackDocumentCode       = "WO"
	singleWordDocumentCodeSize = 3
)

// DefaultQuoteTerms is the example a new WO starts with. The WO edits it in
// Profil usaha; anything a package does not include is written here too.
const DefaultQuoteTerms = `1. Penawaran ini berlaku sampai tanggal yang tertulis di atas.
2. Tanggal acara terkunci setelah tanda jadi (DP) diterima.
3. Tanda jadi yang sudah dibayar tidak dapat dikembalikan bila acara dibatalkan oleh klien.
4. Penundaan acara dikonfirmasi paling lambat 60 hari sebelum tanggal acara dan mengikuti ketersediaan vendor dan venue.
5. Jumlah tamu final dikonfirmasi paling lambat 14 hari sebelum acara. Tambahan tamu setelah itu ditagihkan sesuai harga paket.
6. Yang tidak termasuk penawaran ini: biaya izin keramaian, tip kru, dan biaya di luar rincian.`

// DocumentCodeFromName derives a document code from a business name: the
// initials of its words (Melati Organizer → MO, Bunga Wedding Organizer → BWO),
// or the first three letters of a single-word name (Lavanya → LAV), capped at
// five characters. Only letters and digits count.
func DocumentCodeFromName(name string) string {
	var words []string
	for _, w := range strings.FieldsFunc(name, func(r rune) bool { return !isCodeRune(r) }) {
		words = append(words, strings.ToUpper(w))
	}
	var code []rune
	switch len(words) {
	case 0:
		return fallbackDocumentCode
	case 1:
		for _, r := range words[0] {
			if len(code) == singleWordDocumentCodeSize {
				break
			}
			code = append(code, r)
		}
	default:
		for _, w := range words {
			if len(code) == DocumentCodeMaxLen {
				break
			}
			code = append(code, []rune(w)[0])
		}
	}
	if len(code) < DocumentCodeMinLen {
		return fallbackDocumentCode
	}
	return string(code)
}

// NormalizeDocumentCode trims and upper-cases what the WO typed.
func NormalizeDocumentCode(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// ValidDocumentCode reports whether code is 2–5 ASCII capital letters or digits.
func ValidDocumentCode(code string) bool {
	if len(code) < DocumentCodeMinLen || len(code) > DocumentCodeMaxLen {
		return false
	}
	for _, r := range code {
		if !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func isCodeRune(r rune) bool {
	return r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r))
}
