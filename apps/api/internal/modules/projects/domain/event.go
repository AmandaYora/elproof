package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Layout is how guests are placed at an event ("Tata letak"), which picks the
// venue capacity that applies.
type Layout string

const (
	LayoutStanding Layout = "Standing"
	LayoutSeated   Layout = "Seated"
)

// IsValidLayout reports whether l is one of the two layouts.
func IsValidLayout(l Layout) bool { return l == LayoutStanding || l == LayoutSeated }

// ProjectEvent is one event of a project ("Acara": akad, resepsi, ...). The
// kind's name is copied, so renaming the kind in the catalog leaves the project
// as it was agreed. Exactly one event of a project is its main event ("Acara
// utama"): its date is the project's event date, the Hari H the schedule counts
// from (H-n), and the one the client's home shows.
type ProjectEvent struct {
	ID          int64
	TenantID    int64
	ProjectID   int64
	EventKindID int64
	KindName    string
	EventDate   time.Time
	StartTime   *string // "HH:MM"
	EndTime     *string
	// VenueSupplierID is the project supplier (kind venue) the event takes place
	// at; nil while no venue is chosen (guests are then "not checked yet").
	VenueSupplierID *int64
	Layout          Layout
	IsMain          bool
	SortOrder       int
	Sessions        []EventSession
	UpdatedAt       time.Time
}

// EventSession is one session of an event ("Sesi 1", its hours, its guests).
type EventSession struct {
	ID        int64
	EventID   int64
	Label     string
	StartTime *string
	EndTime   *string
	Guests    int
	SortOrder int
}

// TotalGuests is the sum of the sessions' guests.
func (e ProjectEvent) TotalGuests() int {
	n := 0
	for _, s := range e.Sessions {
		n += s.Guests
	}
	return n
}

// SortEvents orders events the way every screen lists them: by date, then by
// the order the WO set (Pindah ke atas/bawah), then by id.
func SortEvents(events []ProjectEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		a, b := events[i], events[j]
		if !a.EventDate.Equal(b.EventDate) {
			return a.EventDate.Before(b.EventDate)
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		return a.ID < b.ID
	})
}

// MainEvent is the project's main event (nil when the project has none yet).
func MainEvent(events []ProjectEvent) *ProjectEvent {
	for i := range events {
		if events[i].IsMain {
			return &events[i]
		}
	}
	return nil
}

// LocationLine is the place a project happens, derived from its events' venues
// (PLAN §3.2): "Akad: Masjid Al-Ikhlas · Resepsi: Gedung Melati". One venue for
// every event is just its name. venueNames maps a venue supplier id to the name
// to show; an event without a venue is left out. Empty when no event has one.
func LocationLine(events []ProjectEvent, venueNames map[int64]string) string {
	sorted := append([]ProjectEvent(nil), events...)
	SortEvents(sorted)
	var parts []string
	distinct := map[string]bool{}
	for _, e := range sorted {
		if e.VenueSupplierID == nil {
			continue
		}
		name := venueNames[*e.VenueSupplierID]
		if name == "" {
			continue
		}
		distinct[name] = true
		parts = append(parts, e.KindName+": "+name)
	}
	if len(distinct) == 1 {
		for name := range distinct {
			return name
		}
	}
	return strings.Join(parts, " · ")
}

// EventsLine is the short list of a project's events for a second line under
// the main event ("2 acara: Akad 13 Mar · Resepsi 14 Mar"). Empty for one event.
func EventsLine(events []ProjectEvent) string {
	if len(events) < 2 {
		return ""
	}
	sorted := append([]ProjectEvent(nil), events...)
	SortEvents(sorted)
	parts := make([]string, len(sorted))
	for i, e := range sorted {
		parts[i] = fmt.Sprintf("%s %d %s", e.KindName, e.EventDate.Day(), shortMonths[e.EventDate.Month()-1])
	}
	return fmt.Sprintf("%d acara: %s", len(sorted), strings.Join(parts, " · "))
}

var shortMonths = [12]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
