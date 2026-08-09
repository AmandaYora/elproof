package presentation

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// templateDropdownRows mirrors vendorImportRowCap/venueImportRowCap
// (application package, unexported there) -- the dropdown only needs to
// reach as far down as Import will ever actually read.
const templateDropdownRows = 1000

// templateDropdown constrains one Import-template column (matched by its
// exact header text) to a fixed set of values via an in-file Excel dropdown.
type templateDropdown struct {
	header string
	values []string
}

// addTemplateDropdowns turns every non-free-text column in an Import
// template (e.g. Kategori -> the tenant's own registered vendor categories,
// Kota -> the fixed AllowedCities list) into an actual Excel dropdown, so a
// value the backend would reject anyway (VendorService.ImportVendors /
// VenueService.ImportVenues already validate both -- see
// domain.IsValidCity and the category-name lookup in ImportVendors) is
// caught in Excel itself before the file is ever uploaded, not just after.
//
// The values can't be inlined into the dropdown formula directly --
// excelize's SetDropList caps a literal formula at 255 characters, and the
// 128-entry AllowedCities list alone is already over that -- so each list is
// written into its own column of a hidden "Referensi" helper sheet and the
// dropdown references that range instead (SetSqrefDropList), the pattern
// excelize's own docs recommend for exactly this case. The helper sheet is
// hidden, not deleted, since a dropdown's source range must stay resolvable
// for as long as the workbook exists.
func addTemplateDropdowns(f *excelize.File, sheet string, headers []string, dropdowns []templateDropdown) error {
	if len(dropdowns) == 0 {
		return nil
	}
	const helperSheet = "Referensi"
	if _, err := f.NewSheet(helperSheet); err != nil {
		return err
	}

	added := false
	for i, dd := range dropdowns {
		if len(dd.values) == 0 {
			// Nothing valid to offer yet (e.g. a tenant with no vendor
			// categories registered) -- leave that column plain free-text
			// rather than wiring up a dropdown with an empty source range.
			continue
		}
		colIndex := indexOfHeader(headers, dd.header)
		if colIndex < 0 {
			continue // this sheet doesn't have that column at all
		}

		helperCol, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		for rowIdx, v := range dd.values {
			if err := f.SetCellValue(helperSheet, fmt.Sprintf("%s%d", helperCol, rowIdx+1), v); err != nil {
				return err
			}
		}

		targetCol, err := excelize.ColumnNumberToName(colIndex + 1)
		if err != nil {
			return err
		}
		dv := excelize.NewDataValidation(true)
		dv.SetSqref(fmt.Sprintf("%s2:%s%d", targetCol, targetCol, templateDropdownRows+1))
		dv.SetSqrefDropList(fmt.Sprintf("'%s'!$%s$1:$%s$%d", helperSheet, helperCol, helperCol, len(dd.values)))
		dv.SetError(excelize.DataValidationErrorStyleStop, "Nilai tidak valid",
			fmt.Sprintf("Pilih %s dari daftar dropdown yang tersedia -- nilai bebas tidak diterima saat impor.", dd.header))
		dv.SetInput(dd.header, "Klik panah di sisi kanan sel untuk memilih dari daftar.")
		if err := f.AddDataValidation(sheet, dv); err != nil {
			return err
		}
		added = true
	}

	if !added {
		// Nothing was actually wired up (e.g. brand-new tenant with zero
		// categories) -- drop the now-empty helper sheet rather than
		// shipping a pointless hidden blank sheet in the file.
		return f.DeleteSheet(helperSheet)
	}
	return f.SetSheetVisible(helperSheet, false)
}

func indexOfHeader(headers []string, name string) int {
	for i, h := range headers {
		if h == name {
			return i
		}
	}
	return -1
}
