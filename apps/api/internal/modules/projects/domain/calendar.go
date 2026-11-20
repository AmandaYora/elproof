package domain

import (
	"sort"
	"strings"
	"time"
)

// CalendarKind is what a calendar entry is about (GET /calendar, RISET WO
// Console §3 #2): a wedding day, a schedule step, or a client installment.
type CalendarKind string

const (
	CalendarEvent     CalendarKind = "event"
	CalendarMilestone CalendarKind = "milestone"
	CalendarPayment   CalendarKind = "payment"
)

// CalendarMaxDays bounds one request: a month grid shows six weeks (42 days)
// and the agenda one month, so three months is plenty and keeps the query small.
const CalendarMaxDays = 93

// CalendarItem is one entry of the cross-project calendar. Status is the
// project status for an event, the step status for a milestone, and the
// derived TermStatus for a payment. Amount/Remaining are set for payments only.
type CalendarItem struct {
	Date        time.Time
	Kind        CalendarKind
	ProjectID   int64
	ProjectName string
	// RefID is the event, step or installment id: it keeps the order of
	// same-day entries stable.
	RefID     int64
	Title     string
	Status    string
	Amount    *int64
	Remaining *int64
}

// CalendarKinds is the set of kinds a request asks for.
type CalendarKinds map[CalendarKind]bool

// Has reports whether the kind was asked for.
func (k CalendarKinds) Has(kind CalendarKind) bool { return k[kind] }

// ParseCalendarKinds reads the `kinds` filter ("event,milestone,payment"). An
// empty value means every kind; an unknown name is refused (ok=false) so a
// typo is not silently an empty calendar.
func ParseCalendarKinds(raw string) (CalendarKinds, bool) {
	all := CalendarKinds{CalendarEvent: true, CalendarMilestone: true, CalendarPayment: true}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return all, true
	}
	out := CalendarKinds{}
	for _, part := range strings.Split(raw, ",") {
		kind := CalendarKind(strings.TrimSpace(part))
		if kind == "" {
			continue
		}
		if !all[kind] {
			return nil, false
		}
		out[kind] = true
	}
	if len(out) == 0 {
		return all, true
	}
	return out, true
}

// CalendarRangeProblem says what is wrong with a from/to pair ("" when it is
// fine): `to` before `from`, or longer than CalendarMaxDays (both inclusive).
func CalendarRangeProblem(from, to time.Time) string {
	from, to = dayOf(from), dayOf(to)
	if to.Before(from) {
		return "Tanggal akhir tidak boleh sebelum tanggal awal"
	}
	if days := int(to.Sub(from).Hours()/24) + 1; days > CalendarMaxDays {
		return "Rentang kalender paling lama 3 bulan"
	}
	return ""
}

// InCalendarRange reports whether the calendar day of t falls from..to (inclusive).
func InCalendarRange(t, from, to time.Time) bool {
	d := dayOf(t)
	return !d.Before(dayOf(from)) && !d.After(dayOf(to))
}

// CalendarPaymentItems turns the installments of the projects that have one
// due in from..to into calendar entries with the status the project's payment
// schedule gives them as of today (AllocateTerms, WIB day): the payments of a
// project are spread over all its installments, earliest first, so a project's
// whole schedule is needed even when only one installment is in the range.
// Only installments due in the range are returned.
func CalendarPaymentItems(rows []TermWithProject, applied map[int64]int64, from, to, today time.Time) []CalendarItem {
	byProject := make(map[int64][]PaymentTerm)
	names := make(map[int64]string)
	var order []int64
	for _, r := range rows {
		id := r.Term.ProjectID
		if _, seen := byProject[id]; !seen {
			order = append(order, id)
			names[id] = r.ProjectName
		}
		byProject[id] = append(byProject[id], r.Term)
	}
	var out []CalendarItem
	for _, id := range order {
		for _, tp := range AllocateTerms(byProject[id], applied[id], today) {
			if !InCalendarRange(tp.Term.DueDate, from, to) {
				continue
			}
			amount, remaining := tp.Term.Amount, tp.Remaining
			out = append(out, CalendarItem{
				Date: dayOf(tp.Term.DueDate), Kind: CalendarPayment, ProjectID: id, ProjectName: names[id],
				RefID: tp.Term.ID, Title: tp.Term.Label, Status: string(tp.Status), Amount: &amount, Remaining: &remaining,
			})
		}
	}
	return out
}

var calendarKindOrder = map[CalendarKind]int{CalendarEvent: 0, CalendarPayment: 1, CalendarMilestone: 2}

// SortCalendarItems orders entries by day; on one day the wedding days first
// (they are what the day is about), then installments, then schedule steps,
// then by project name and id so the order never changes between requests.
func SortCalendarItems(items []CalendarItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if !a.Date.Equal(b.Date) {
			return a.Date.Before(b.Date)
		}
		if a.Kind != b.Kind {
			return calendarKindOrder[a.Kind] < calendarKindOrder[b.Kind]
		}
		if a.ProjectName != b.ProjectName {
			return a.ProjectName < b.ProjectName
		}
		if a.ProjectID != b.ProjectID {
			return a.ProjectID < b.ProjectID
		}
		return a.RefID < b.RefID
	})
}
