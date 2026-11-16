package application

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// ImportCellError is one problem of one column of an import row, named by the
// column header of the template (FQ-07: a refused row says exactly which cells
// to fix instead of one sentence listing every required column).
type ImportCellError struct {
	Column  string
	Message string
}

// rowCheck collects the problems of one import row, in column order.
type rowCheck struct{ errs []ImportCellError }

func (c *rowCheck) add(column, message string) {
	c.errs = append(c.errs, ImportCellError{Column: column, Message: message})
}

func (c *rowCheck) failed() bool { return len(c.errs) > 0 }

// summary is the row's problems as one sentence (the row's `message`, kept for
// clients that only show that).
func (c *rowCheck) summary() string {
	parts := make([]string, 0, len(c.errs))
	for _, e := range c.errs {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, "; ")
}

// required: an empty cell is refused; otherwise it must fit its column.
func (c *rowCheck) required(column, value string, max int) {
	if value == "" {
		c.add(column, column+" wajib diisi")
		return
	}
	c.maxLen(column, value, max)
}

func (c *rowCheck) maxLen(column, value string, max int) {
	if max > 0 && utf8.RuneCountInString(value) > max {
		c.add(column, column+" maksimal "+strconv.Itoa(max)+" karakter")
	}
}

// minLen refuses a filled cell shorter than min characters (an empty one is
// required's business).
func (c *rowCheck) minLen(column, value string, min int, unit string) {
	if value != "" && utf8.RuneCountInString(value) < min {
		c.add(column, column+" minimal "+strconv.Itoa(min)+" "+unit)
	}
}

// numberRule says how a numeric column is judged.
type numberRule struct {
	required bool  // an empty cell is refused
	positive bool  // 0 is refused too (the form's "lebih dari 0")
	max      int64 // 0 = no bound beyond int64
}

// number judges a numeric cell: notNumber is true when the cell held text that
// is not a whole number (the parser then leaves value nil).
func (c *rowCheck) number(column string, value *int64, notNumber bool, rule numberRule) {
	switch {
	case notNumber:
		c.add(column, column+" harus berupa angka bulat")
	case value == nil:
		if rule.required {
			c.add(column, column+" wajib diisi")
		}
	case *value < 0:
		c.add(column, column+" tidak boleh negatif")
	case rule.positive && *value == 0:
		c.add(column, column+" harus lebih dari 0")
	case rule.max > 0 && *value > rule.max:
		c.add(column, column+" terlalu besar. Periksa lagi angkanya.")
	}
}

// hasColumn reports whether the parser flagged column as not a number.
func hasColumn(columns []string, column string) bool {
	for _, c := range columns {
		if c == column {
			return true
		}
	}
	return false
}
