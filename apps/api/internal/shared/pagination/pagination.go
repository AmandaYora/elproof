// Package pagination provides the shared list-pagination shape used by every
// paginated endpoint: meta: {page, limit, total, total_pages}.
package pagination

import (
	"math"
	"net/http"
	"strconv"
)

const (
	DefaultPage  = 1
	DefaultLimit = 10
	MaxLimit     = 100
)

type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type Params struct {
	Page  int
	Limit int
}

// Offset returns the SQL OFFSET for these params.
func (p Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

// FromRequest parses ?page= and ?limit= from the query string, applying defaults
// and clamping limit to MaxLimit.
func FromRequest(r *http.Request) Params {
	page := parsePositiveInt(r.URL.Query().Get("page"), DefaultPage)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), DefaultLimit)
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return Params{Page: page, Limit: limit}
}

// BuildMeta computes total_pages from the total row count.
func BuildMeta(params Params, total int64) Meta {
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	if totalPages < 1 {
		totalPages = 1
	}
	return Meta{Page: params.Page, Limit: params.Limit, Total: total, TotalPages: totalPages}
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
