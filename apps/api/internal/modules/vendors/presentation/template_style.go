package presentation

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// styleTemplateSheet gives an Import template a clean, professional look
// without going past what a spreadsheet user already expects from one: a
// bold header row that stays pinned while scrolling, columns wide enough
// that a real value doesn't get clipped, and thousand-separated formatting
// on the currency columns so a typed price reads back clearly. No borders on
// data rows, no alternating row banding, no instructions sheet -- there's no
// data yet for any of that to describe.
func styleTemplateSheet(f *excelize.File, sheet string, headers []string, currencyHeaders []string) error {
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1E3A5F"}},
		Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "left"},
		Border:    []excelize.Border{{Type: "bottom", Color: "172741", Style: 1}},
	})
	if err != nil {
		return err
	}
	currencyStyle, err := f.NewStyle(&excelize.Style{NumFmt: 3}) // built-in "#,##0"
	if err != nil {
		return err
	}

	currencySet := make(map[string]struct{}, len(currencyHeaders))
	for _, h := range currencyHeaders {
		currencySet[h] = struct{}{}
	}

	for i, header := range headers {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, col, col, columnWidthFor(header)); err != nil {
			return err
		}
		headerCell := fmt.Sprintf("%s1", col)
		if err := f.SetCellStyle(sheet, headerCell, headerCell, headerStyle); err != nil {
			return err
		}
		if _, ok := currencySet[header]; ok {
			rangeStart := fmt.Sprintf("%s2", col)
			rangeEnd := fmt.Sprintf("%s%d", col, templateDropdownRows+1)
			if err := f.SetCellStyle(sheet, rangeStart, rangeEnd, currencyStyle); err != nil {
				return err
			}
		}
	}

	if err := f.SetRowHeight(sheet, 1, 20); err != nil {
		return err
	}
	// Keeps the header row visible no matter how far down a long list of
	// vendors/venues scrolls -- a plain, expected spreadsheet convenience.
	return f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, Split: false, XSplit: 0, YSplit: 1,
		TopLeftCell: "A2", ActivePane: "bottomLeft",
	})
}

// stripThousandsSeparators undoes styleTemplateSheet's own "#,##0" display
// formatting on a currency cell -- excelize's row reader (Import uses
// f.Rows()/Columns()) returns a numeric cell's *formatted* display string,
// not its raw value, so a currency cell showing "5,000,000" reads back as
// that exact string, commas included, not "5000000". Every numeric import
// column that styleTemplateSheet ever applies a number format to must run
// its cell text through this before strconv.ParseInt/Atoi. A plain comma is
// the only separator stripped -- id-ID's period-as-thousands convention was
// deliberately not chosen for the template's format, so callers reading a
// price field are never ambiguous with a decimal point.
func stripThousandsSeparators(s string) string {
	return strings.ReplaceAll(s, ",", "")
}

// columnWidthFor sizes a column for the value it actually holds, not just its
// header text -- an "Email" or "Alamat" header is short, but real emails and
// addresses run much longer than that, so those columns get a fixed, roomier
// width instead of the header-length heuristic every other column uses.
func columnWidthFor(header string) float64 {
	switch header {
	case "Email", "Sosial Media":
		return 26
	case "Alamat", "Fasilitas", "Catatan":
		return 32
	case "Kota", "Nama Vendor", "Nama Venue":
		return 22
	case "No Tlp Vendor", "No Tlp PIC", "No Tlp Venue":
		return 16
	case "Harga Akad", "Harga Akad+Resepsi", "Harga Sewa", "Charge", "Kapasitas":
		return 16
	default:
		if width := float64(len(header)) + 6; width >= 14 {
			return width
		}
		return 14
	}
}
