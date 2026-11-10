// Package etag turns a row's `updated_at` into the HTTP validator the API uses
// for optimistic concurrency (ADR-0036) and reads it back from `If-Match`.
//
// It is purely technical: it knows a time and a header, never a domain.
package etag

import (
	"errors"
	"strings"
	"time"
)

// layout is RFC 3339 in UTC with exactly six fractional digits, so the same
// instant always renders the same bytes: "2026-11-10T08:15:30.123456Z".
const layout = "2006-01-02T15:04:05.000000Z"

// ErrMalformed is returned for an If-Match value that is not one quoted (or bare)
// RFC 3339 timestamp.
var ErrMalformed = errors.New("etag: If-Match is not an RFC 3339 timestamp")

// Format renders a version as the `updatedAt` field of a response.
func Format(t time.Time) string { return t.UTC().Format(layout) }

// Header renders the same instant as an ETag header value (a quoted string).
func Header(t time.Time) string { return `"` + Format(t) + `"` }

// Same reports whether two versions are the same instant, at the microsecond
// precision the column stores.
func Same(a, b time.Time) bool {
	return a.Truncate(time.Microsecond).Equal(b.Truncate(time.Microsecond))
}

// Truncate drops everything below the microsecond: what a TIMESTAMP(6) holds.
func Truncate(t time.Time) time.Time { return t.Truncate(time.Microsecond) }

// ParseIfMatch reads an If-Match header. An absent (empty) header is (nil, nil):
// the caller then writes unconditionally. A value that is not exactly one
// timestamp (quotes optional) is ErrMalformed; lists, `*` and weak validators
// are not supported and count as malformed.
func ParseIfMatch(header string) (*time.Time, error) {
	v := strings.TrimSpace(header)
	if v == "" {
		return nil, nil
	}
	if strings.HasPrefix(v, `"`) {
		if len(v) < 2 || !strings.HasSuffix(v, `"`) {
			return nil, ErrMalformed
		}
		v = v[1 : len(v)-1]
	}
	if v == "" || strings.ContainsAny(v, `", *`) || strings.HasPrefix(v, "W/") {
		return nil, ErrMalformed
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return nil, ErrMalformed
	}
	t = Truncate(t.UTC())
	return &t, nil
}
