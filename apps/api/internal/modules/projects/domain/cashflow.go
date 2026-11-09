package domain

import (
	"sort"
	"time"
)

// Pure rules behind the three money sections of /dashboard (E13): the 8-week
// cash flow, vendor payments that are due, and vendors still unconfirmed close
// to the event.

// CashFlowWeeks is how many weeks the cash flow looks ahead.
const CashFlowWeeks = 8

// MaxDashboardRows caps vendorPaymentsDue and unconfirmedVendors.
const MaxDashboardRows = 10

// dashboardHorizonDays is the "within 30 days" window of vendorPaymentsDue and
// unconfirmedVendors.
const dashboardHorizonDays = 30

// DueAmount is an unpaid amount with the date it falls due.
type DueAmount struct {
	DueDate   time.Time
	Remaining int64
}

// CashFlowWeek is one week of expected money in and out.
type CashFlowWeek struct {
	WeekStart   time.Time
	ExpectedIn  int64
	ExpectedOut int64
}

// MondayOf is the Monday of the calendar week containing d (weeks run Monday to
// Sunday), at midnight UTC.
func MondayOf(d time.Time) time.Time {
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	back := (int(day.Weekday()) + 6) % 7 // Monday = 0 ... Sunday = 6
	return day.AddDate(0, 0, -back)
}

// BuildCashFlow buckets what is still unpaid into the next CashFlowWeeks weeks,
// the first being the week containing today. Money owed to us ("in": client
// installments) and owed by us ("out": vendors) are bucketed by due date; anything
// already past due lands in week 1, anything beyond the last week is left out.
// Amounts that are zero or negative count for nothing.
func BuildCashFlow(today time.Time, in, out []DueAmount) []CashFlowWeek {
	start := MondayOf(today)
	weeks := make([]CashFlowWeek, CashFlowWeeks)
	for i := range weeks {
		weeks[i].WeekStart = start.AddDate(0, 0, 7*i)
	}
	put := func(list []DueAmount, add func(w *CashFlowWeek, n int64)) {
		for _, a := range list {
			if a.Remaining <= 0 {
				continue
			}
			due := time.Date(a.DueDate.Year(), a.DueDate.Month(), a.DueDate.Day(), 0, 0, 0, 0, time.UTC)
			idx := int(due.Sub(start).Hours()/24) / 7
			if due.Before(start) {
				idx = 0
			}
			if idx >= CashFlowWeeks {
				continue
			}
			add(&weeks[idx], a.Remaining)
		}
	}
	put(in, func(w *CashFlowWeek, n int64) { w.ExpectedIn += n })
	put(out, func(w *CashFlowWeek, n int64) { w.ExpectedOut += n })
	return weeks
}

// VendorDue is one unpaid vendor engagement balance with its due date.
type VendorDue struct {
	ProjectID   int64
	ProjectName string
	VendorID    int64
	DueDate     time.Time
	Remaining   int64
}

// SelectVendorPaymentsDue keeps the unpaid vendor balances that are overdue or
// due within 30 days, nearest due date first (overdue first), at most
// MaxDashboardRows.
func SelectVendorPaymentsDue(today time.Time, rows []VendorDue) []VendorDue {
	var out []VendorDue
	for _, r := range rows {
		if r.Remaining <= 0 {
			continue
		}
		if daysBetween(today, r.DueDate) <= dashboardHorizonDays {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DueDate.Before(out[j].DueDate) })
	if len(out) > MaxDashboardRows {
		out = out[:MaxDashboardRows]
	}
	return out
}

// OutstandingOf is what is still owed on an engagement given what was paid
// against it (never negative).
func OutstandingOf(contractValue int64, lines []PaymentLine) int64 {
	return ComputeLedger(lines, contractValue).Remaining
}

// EngagementBalanceRow is one vendor engagement with what is agreed and what
// has been applied to it, as the dashboard reads them across the workspace.
type EngagementBalanceRow struct {
	EngagementID  int64
	ProjectID     int64
	ProjectName   string
	VendorID      int64
	ContractValue int64
	DueDate       time.Time
	Applied       int64
}

// Remaining is what is still owed on the engagement (never negative).
func (r EngagementBalanceRow) Remaining() int64 {
	return OutstandingOf(r.ContractValue, []PaymentLine{{Type: PaymentDP, Amount: max64(r.Applied, 0)}})
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// UnconfirmedVendorRow is a vendor still Planned/Negotiation close to the event.
type UnconfirmedVendorRow struct {
	EngagementID     int64
	ProjectID        int64
	ProjectName      string
	VendorID         int64
	EngagementStatus EngagementStatus
	EventDate        time.Time
}
