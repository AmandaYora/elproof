package domain

import (
	"sort"
	"time"
)

// PaymentTerm is one agreed installment of what the client owes.
type PaymentTerm struct {
	ID        int64
	ProjectID int64
	Label     string
	Amount    int64
	DueDate   time.Time
}

// TermStatus is derived, never stored (see migration 000060).
type TermStatus string

const (
	TermPaid     TermStatus = "paid"
	TermOverdue  TermStatus = "overdue"
	TermPartial  TermStatus = "partial"
	TermDueSoon  TermStatus = "due_soon"
	TermUpcoming TermStatus = "upcoming"
)

// dueSoonDays: a term due within this many days (and not paid) is "Segera jatuh tempo".
const dueSoonDays = 7

// TermProgress is a term with how much of it has been covered and its status.
type TermProgress struct {
	Term      PaymentTerm
	Paid      int64
	Remaining int64
	Status    TermStatus
}

// AllocateTerms spreads what the client has paid (DP + Termin + Pelunasan -
// Refund, see AppliedToContract) over the terms, earliest due date first, and
// gives each a status as of `today`: paid; overdue (due date passed and not
// fully paid, even when partly paid); partial; due soon (within 7 days);
// upcoming. Money beyond the schedule's total is simply not allocated.
func AllocateTerms(terms []PaymentTerm, applied int64, today time.Time) []TermProgress {
	sorted := append([]PaymentTerm(nil), terms...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].DueDate.Equal(sorted[j].DueDate) {
			return sorted[i].DueDate.Before(sorted[j].DueDate)
		}
		return sorted[i].ID < sorted[j].ID
	})
	if applied < 0 {
		applied = 0
	}
	out := make([]TermProgress, 0, len(sorted))
	left := applied
	for _, t := range sorted {
		paid := t.Amount
		if left < paid {
			paid = left
		}
		left -= paid
		tp := TermProgress{Term: t, Paid: paid, Remaining: t.Amount - paid}
		days := daysBetween(today, t.DueDate)
		switch {
		case tp.Remaining == 0:
			tp.Status = TermPaid
		case days < 0:
			tp.Status = TermOverdue
		case paid > 0:
			tp.Status = TermPartial
		case days <= dueSoonDays:
			tp.Status = TermDueSoon
		default:
			tp.Status = TermUpcoming
		}
		out = append(out, tp)
	}
	return out
}
