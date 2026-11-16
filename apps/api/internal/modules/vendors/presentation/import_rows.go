package presentation

import (
	"strconv"
	"strings"

	"elproof/internal/modules/vendors/application"
)

// importCells reads one spreadsheet row by column index (the template's fixed
// order) and remembers which numeric cells held text that is not a whole
// number, so the row check can name them (FQ-07).
type importCells struct {
	cells     []string
	headers   []string
	notNumber []string
}

func (c *importCells) text(i int) string {
	if i < len(c.cells) {
		return strings.TrimSpace(c.cells[i])
	}
	return ""
}

// number is the whole number in column i; nil for an empty cell, and nil plus a
// note in notNumber for a cell that is not a whole number.
func (c *importCells) number(i int) *int64 {
	s := c.text(i)
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(stripThousandsSeparators(s), 10, 64)
	if err != nil {
		c.notNumber = append(c.notNumber, c.headers[i])
		return nil
	}
	return &n
}

func (c *importCells) smallNumber(i int) *int {
	n := c.number(i)
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

// importCellErrorBody is one problem of one column of a refused import row.
type importCellErrorBody struct {
	Column  string `json:"column"`
	Message string `json:"message"`
}

// importRowErrorBody is one refused import row: `message` is every problem in
// one sentence (what older clients show), `errors` the same per column (empty
// for a row the database refused as a whole, and for the batch line, row 0).
type importRowErrorBody struct {
	Row     int                   `json:"row"`
	Message string                `json:"message"`
	Errors  []importCellErrorBody `json:"errors"`
}

type importResultBody struct {
	InsertedCount int                  `json:"insertedCount"`
	UpdatedCount  int                  `json:"updatedCount"`
	Errors        []importRowErrorBody `json:"errors"`
}

func toImportRowErrorBody(row int, message string, cells []application.ImportCellError) importRowErrorBody {
	out := importRowErrorBody{Row: row, Message: message, Errors: make([]importCellErrorBody, 0, len(cells))}
	for _, c := range cells {
		out.Errors = append(out.Errors, importCellErrorBody{Column: c.Column, Message: c.Message})
	}
	return out
}
