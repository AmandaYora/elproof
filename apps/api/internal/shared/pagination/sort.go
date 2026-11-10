package pagination

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"elproof/internal/shared/apperror"
)

// Sorting for the paginated lists: ?sort=field:asc|desc.
//
// Rules every list shares (ADR-0035):
//   - a single column; `field` alone means ascending;
//   - `field` must be a key of the endpoint's whitelist, which maps the public
//     name to a FIXED SQL fragment written in this codebase. The request never
//     contributes text to the SQL: a value that is not a key (injection
//     attempts included) is a 422 and nothing else;
//   - the primary key is always appended as tie-breaker, in the same direction,
//     so equal values page in a stable order.

// SortColumn is one whitelisted, sortable column.
type SortColumn struct {
	// Expr is the SQL expression to order by (a constant of the calling
	// package, never request input), e.g. "name" or "p.event_date".
	Expr string
	// NullsLast keeps NULLs after every real value whatever the direction
	// (a vendor without a price, a project without an unpaid term).
	NullsLast bool
	// nullExpr, when set, is what is tested for NULL (RankedCol: FIELD() never
	// returns NULL for a NULL column, so the column itself is tested).
	nullExpr string
}

// Col is a plain NOT NULL column.
func Col(expr string) SortColumn { return SortColumn{Expr: expr} }

// NullableCol is a column that may be NULL: NULLs always sort last.
func NullableCol(expr string) SortColumn { return SortColumn{Expr: expr, NullsLast: true} }

// RankedCol orders by a column whose natural order is not the one the user wants
// because the display value lives in another module (a role's or a plan's NAME,
// while this table stores its id): rankedIDs are those ids listed in the order
// of the names, and the column sorts by position in that list via FIELD().
// The ids are int64 values, rendered as numbers, never text from a request. An
// id that is not in the list ranks first, NULL last. With no ids it degrades to
// the plain column.
func RankedCol(expr string, rankedIDs []int64) SortColumn {
	if len(rankedIDs) == 0 {
		return SortColumn{Expr: expr, NullsLast: true}
	}
	parts := make([]string, len(rankedIDs))
	for i, id := range rankedIDs {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return SortColumn{Expr: "FIELD(" + expr + ", " + strings.Join(parts, ", ") + ")", NullsLast: false, nullExpr: expr}
}

// Sort is the effective order of a list.
type Sort struct {
	// Field is the public name ("eventDate").
	Field string
	Desc  bool
	col   SortColumn
}

// With returns the sort ordering by col instead of the whitelisted column of the
// same field: for the few fields whose SQL depends on data read at request time
// (RankedCol).
func (s Sort) With(col SortColumn) Sort {
	s.col = col
	return s
}

// By names a default sort; SortFromRequest resolves it against the whitelist.
func By(field string, desc bool) Sort { return Sort{Field: field, Desc: desc} }

// Dir is "asc" or "desc".
func (s Sort) Dir() string {
	if s.Desc {
		return "desc"
	}
	return "asc"
}

// String is the "field:dir" echoed in meta.sort.
func (s Sort) String() string { return s.Field + ":" + s.Dir() }

// OrderBy renders the ORDER BY clause (with a leading space), the primary key
// `tieBreaker` last in the same direction. tieBreaker is a constant of the
// calling package ("id", "p.id").
func (s Sort) OrderBy(tieBreaker string) string {
	dir := " ASC"
	if s.Desc {
		dir = " DESC"
	}
	var b strings.Builder
	b.WriteString(" ORDER BY ")
	switch {
	case s.col.nullExpr != "":
		b.WriteString("(" + s.col.nullExpr + " IS NULL), ")
	case s.col.NullsLast:
		b.WriteString("(" + s.col.Expr + " IS NULL), ")
	}
	b.WriteString(s.col.Expr + dir)
	if tieBreaker != "" && tieBreaker != s.col.Expr {
		b.WriteString(", " + tieBreaker + dir)
	}
	return b.String()
}

// SortFromRequest reads ?sort= against the endpoint's whitelist. No parameter
// gives the default (which must itself be in the whitelist). An unknown field,
// a bad direction or more than one sort parameter is a validation error keyed
// "sort" (422) listing what is allowed.
func SortFromRequest(r *http.Request, allowed map[string]SortColumn, def Sort) (Sort, error) {
	resolve := func(field string, desc bool) (Sort, bool) {
		col, ok := allowed[field]
		if !ok {
			return Sort{}, false
		}
		return Sort{Field: field, Desc: desc, col: col}, true
	}
	values := r.URL.Query()["sort"]
	if len(values) == 0 || (len(values) == 1 && strings.TrimSpace(values[0]) == "") {
		s, ok := resolve(def.Field, def.Desc)
		if !ok {
			return Sort{}, apperror.Internal("daftar urutan bawaan tidak ada di daftar putih: " + def.Field)
		}
		return s, nil
	}
	if len(values) > 1 {
		return Sort{}, badSort("Urutkan menurut satu kolom saja. Kolom yang boleh: " + allowedList(allowed))
	}
	field, dir := strings.TrimSpace(values[0]), "asc"
	if i := strings.Index(field, ":"); i >= 0 {
		field, dir = strings.TrimSpace(field[:i]), strings.ToLower(strings.TrimSpace(field[i+1:]))
	}
	if dir != "asc" && dir != "desc" {
		return Sort{}, badSort("Arah urutan harus asc atau desc")
	}
	s, ok := resolve(field, dir == "desc")
	if !ok {
		return Sort{}, badSort("Urutkan menurut salah satu dari: " + allowedList(allowed))
	}
	return s, nil
}

func badSort(message string) error {
	return apperror.Validation("Parameter urutan tidak valid", map[string][]string{"sort": {message}})
}

func allowedList(allowed map[string]SortColumn) string {
	names := make([]string, 0, len(allowed))
	for name := range allowed {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// SearchFromRequest reads the free-text search of a list: ?q= (the common name)
// or ?search= (what every list accepted first); q wins when both are sent. The
// value is trimmed; callers escape it for LIKE.
func SearchFromRequest(r *http.Request) string {
	q := r.URL.Query()
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		return v
	}
	return strings.TrimSpace(q.Get("search"))
}
