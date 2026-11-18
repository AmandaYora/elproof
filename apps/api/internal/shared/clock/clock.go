// Package clock is the API's business clock: what "now" and "today" mean for
// domain rules (H-n, overdue installments, lagging preparation, trial days,
// undo windows, the purge of deleted rows, records' own timestamps).
//
// In every real environment it is exactly time.Now. Only an end-to-end test
// run moves it, with APP_CLOCK_START (RFC 3339, e.g.
// "2026-11-09T10:00:00+07:00"): the clock then starts at that instant when
// the process starts and keeps ticking at normal speed, so the server and the
// test browser (frozen at the same instant) agree on the calendar day however
// long after that date the suite runs. cmd/server refuses the variable when
// APP_ENV=production.
//
// Security and wall-clock timing deliberately stay on time.Now and never use
// this package: JWT and refresh-token lifetimes, TOTP, sign-in links, Google
// ID token checks, rate limits, signed storage URLs and payment gateway
// requests are checked against parties that only know real time.
package clock

import (
	"fmt"
	"sync/atomic"
	"time"
)

// offset is added to time.Now; 0 (the default) means the real clock.
var offset atomic.Int64

// Now is the business "now".
func Now() time.Time {
	d := offset.Load()
	if d == 0 {
		return time.Now()
	}
	return time.Now().Add(time.Duration(d))
}

// Shifted reports whether the clock was moved away from real time.
func Shifted() bool { return offset.Load() != 0 }

// StartAt shifts the clock so that Now() returns start at this moment and
// advances normally from there. Only cmd/server calls it, once, at startup.
func StartAt(start time.Time) {
	offset.Store(int64(time.Until(start)))
}

// Parse reads an APP_CLOCK_START value (RFC 3339 with an explicit offset).
func Parse(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("APP_CLOCK_START must be RFC 3339 like 2026-11-09T10:00:00+07:00: %w", err)
	}
	return t, nil
}

// Reset puts the real clock back (tests).
func Reset() { offset.Store(0) }
