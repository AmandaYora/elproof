package domain

import "time"

// ProjectListFilter is everything GET /projects can narrow by besides the
// caller's own scope (which the repository takes separately and always applies).
type ProjectListFilter struct {
	Search       string
	Status       string
	ShowArchived bool
	// PICStaffID narrows to one PIC (a manager looking at one planner's
	// projects). It is ANDed with the caller's scope, never replacing it.
	PICStaffID *int64
	// EventFrom/EventTo bound event_date, both inclusive.
	EventFrom, EventTo *time.Time
}

// ListMoney is the two derived columns of a row of the project list.
type ListMoney struct {
	// Remaining is what the client still owes on the contract: the contract
	// value minus the payments applied to it (DP + Termin + Pelunasan - Refund,
	// a Tambahan excluded, deleted payments not counted), never negative.
	Remaining int64
	// NextDueDate is the due date of the earliest installment not fully paid
	// (nil: no schedule, or every installment is paid).
	NextDueDate *time.Time
}

// ComputeListMoney derives a project's list columns from its raw numbers with
// the very rules the overview and the portal use (ComputeLedger, AllocateTerms):
// the payments are spread over the installments earliest due date first, and the
// first installment left with something owing gives the next due date.
func ComputeListMoney(contractValue int64, payments []PaymentLine, terms []PaymentTerm) ListMoney {
	ledger := ComputeLedger(payments, contractValue)
	out := ListMoney{Remaining: ledger.Remaining}
	// The status needs a "today"; the due date of the next open installment does not.
	if next := NextOpenTerm(AllocateTerms(terms, AppliedToContract(payments), time.Time{})); next != nil {
		d := next.Term.DueDate
		out.NextDueDate = &d
	}
	return out
}

// ProjectLabel is the name and event date of a project: what other modules show
// next to a row that only stores the project's id (the clients list).
type ProjectLabel struct {
	ID        int64
	Name      string
	EventDate time.Time
}
