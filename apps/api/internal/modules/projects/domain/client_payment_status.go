package domain

import "time"

// ClientPaymentStatus is a project-level aggregate (contract value vs. total
// client payments received) -- not a per-payment-row property, see PLAN.md.
type ClientPaymentStatus string

const (
	StatusUnpaid        ClientPaymentStatus = "Unpaid"
	StatusPartiallyPaid ClientPaymentStatus = "PartiallyPaid"
	StatusPaid          ClientPaymentStatus = "Paid"
	StatusOverdue       ClientPaymentStatus = "Overdue"
)

// ComputeClientPaymentStatus is a pure function -- `today` must always be
// the Go server's own clock (ADR-0007), passed in explicitly by the caller
// (mirroring IsMilestoneOverdue's own asOf parameter above) rather than read
// here, so the caller stays the single place responsible for "now". Never
// ported to the frontend -- the backend computes this once, per GET
// /projects/{id} response, and the frontend only ever renders the resulting
// string.
//
// Evaluation is an explicit short-circuit sequence, not independent
// conditions: Paid is checked first and stops immediately, so a project that
// has since become overdue-looking (a stale next_due_date) can never be
// misreported once its outstanding balance is already settled.
func ComputeClientPaymentStatus(outstanding int64, hasAnyPayment bool, nextDueDate *time.Time, today time.Time) ClientPaymentStatus {
	if outstanding <= 0 {
		return StatusPaid
	}
	if nextDueDate != nil && nextDueDate.Before(today) {
		return StatusOverdue
	}
	if hasAnyPayment {
		return StatusPartiallyPaid
	}
	return StatusUnpaid
}
