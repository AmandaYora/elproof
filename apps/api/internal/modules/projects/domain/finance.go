package domain

import "time"

// Pure rules behind the Keuangan page (/finance/*): client installments across
// projects (receivables), what is still owed to vendors and venues (payables),
// and the page's headline figures. The installment rule is AllocateTerms'
// (TermStatusOf); the cash flow is BuildCashFlow, the very function /dashboard
// uses.

// FinanceScope is whose projects a finance read covers: one workspace and, for
// a role limited to its own projects, only the projects where HolderStaffID is
// PIC or PIC Sales (the same scope GET /projects applies).
type FinanceScope struct {
	TenantID      int64
	HolderStaffID *int64
}

// FinanceHorizonDays is the "jatuh tempo 30 hari" window, and ReceivedWindowDays
// the "uang masuk 30 hari terakhir" window (today included in both).
const (
	FinanceHorizonDays = 30
	ReceivedWindowDays = 30
)

// ReceivableView narrows the installment list. The empty view lists every
// installment; the four statuses of TermStatusOf can be asked for one by one,
// plus three views the page's tabs use.
type ReceivableView string

const (
	ReceivableAll      ReceivableView = ""
	ReceivableOverdue  ReceivableView = "overdue"
	ReceivableDue30    ReceivableView = "due_30d" // not paid, due from today to today+30
	ReceivableUnpaid   ReceivableView = "unpaid"  // anything left to pay, whatever the date
	ReceivablePaid     ReceivableView = "paid"
	ReceivablePartial  ReceivableView = "partial"
	ReceivableDueSoon  ReceivableView = "due_soon"
	ReceivableUpcoming ReceivableView = "upcoming"
)

// ReceivableViews lists the accepted values of ?status= on /finance/receivables.
var ReceivableViews = []ReceivableView{ReceivableOverdue, ReceivableDue30, ReceivableUnpaid, ReceivablePaid, ReceivablePartial, ReceivableDueSoon, ReceivableUpcoming}

// ReceivableFilter is everything /finance/receivables narrows by besides the scope.
type ReceivableFilter struct {
	View       ReceivableView
	Search     string
	PICStaffID *int64
	// DueFrom/DueTo bound the installment's due date, both inclusive.
	DueFrom, DueTo *time.Time
}

// ReceivableRow is one installment with its project and what the client's
// payments cover of it (allocated over the project's schedule, earliest due
// date first, as AllocateTerms does).
type ReceivableRow struct {
	Term        PaymentTerm
	ProjectName string
	BrideName   string
	GroomName   string
	PICStaffID  int64
	Paid        int64
	Remaining   int64
}

// Status is the installment's standing as of today.
func (r ReceivableRow) Status(today time.Time) TermStatus {
	return TermStatusOf(r.Paid, r.Remaining, r.Term.DueDate, today)
}

// DaysLate is how many days past its due date an unpaid amount is (0 when it is
// not late, or nothing is left).
func DaysLate(dueDate, today time.Time, remaining int64) int {
	if remaining <= 0 {
		return 0
	}
	if d := daysBetween(dueDate, today); d > 0 {
		return d
	}
	return 0
}

// PayableKind is who the money goes to.
type PayableKind string

const (
	PayableVendor PayableKind = "vendor"
	PayableVenue  PayableKind = "venue"
)

// PayableStatus is where an amount owed by the WO stands.
type PayableStatus string

const (
	PayablePaid    PayableStatus = "paid"
	PayableUnpaid  PayableStatus = "unpaid"
	PayableOverdue PayableStatus = "overdue" // a due date that has passed with something left
)

// PayableFilter is everything /finance/payables narrows by besides the scope.
// Status "open" means anything left to pay (unpaid or overdue).
type PayableFilter struct {
	Kind       PayableKind
	Status     string
	Search     string
	PICStaffID *int64
}

// PayableStatusFilters lists the accepted values of ?status= on /finance/payables.
var PayableStatusFilters = []string{"open", string(PayableUnpaid), string(PayableOverdue), string(PayablePaid)}

// PayableRow is one amount the WO owes: a vendor engagement (its contract value)
// or a project's venue (rental price + charge), with what has been paid against
// it (DP + Termin + Pelunasan - Refund over its live payments, never negative).
type PayableRow struct {
	Kind          PayableKind
	ID            int64 // the engagement's id; the project's id for a venue
	ProjectID     int64
	ProjectName   string
	BrideName     string
	GroomName     string
	PICStaffID    int64
	VendorID      int64  // vendors only
	CategoryID    int64  // vendors only
	VenueName     string // venues only: the project's venue text
	ContractValue int64
	Paid          int64
	DueDate       *time.Time // vendors only, when agreed
}

// Remaining is what is still owed (never negative), the rule of ComputeLedger.
func (r PayableRow) Remaining() int64 {
	return ComputeLedger([]PaymentLine{{Type: PaymentDP, Amount: max64(r.Paid, 0)}}, r.ContractValue).Remaining
}

// Status is the row's standing as of today.
func (r PayableRow) Status(today time.Time) PayableStatus {
	return PayableStatusOf(r.Remaining(), r.DueDate, today)
}

// PayableStatusOf: paid when nothing is left, overdue when a due date has
// passed, unpaid otherwise.
func PayableStatusOf(remaining int64, dueDate *time.Time, today time.Time) PayableStatus {
	switch {
	case remaining <= 0:
		return PayablePaid
	case dueDate != nil && daysBetween(today, *dueDate) < 0:
		return PayableOverdue
	default:
		return PayableUnpaid
	}
}

// MoneyCount is an amount with how many items (and projects) make it up.
type MoneyCount struct {
	Amount   int64
	Count    int
	Projects int
}

// PayablesSummary is what the WO still owes vendors and venues.
type PayablesSummary struct {
	Amount        int64
	Count         int
	OverdueAmount int64
	OverdueCount  int
}

// FinanceSummary is the Keuangan page's headline figures.
type FinanceSummary struct {
	Overdue             MoneyCount // installments past due: what is left on them
	DueSoon             MoneyCount // installments due from today to today+30, not yet paid
	Received            MoneyCount // client money received in the last 30 days (Count = payments)
	Payables            PayablesSummary
	PendingVerification MoneyCount // client payments with proof, not verified yet
	CashFlow            []CashFlowWeek
}

// SummarizeReceivables adds up the unpaid installments: the overdue ones and
// those due within the next 30 days. It also returns them as due amounts for
// the cash flow. Rows with nothing left count for nothing.
func SummarizeReceivables(rows []ReceivableRow, today time.Time) (overdue, dueSoon MoneyCount, due []DueAmount) {
	overdueProjects := map[int64]bool{}
	dueProjects := map[int64]bool{}
	for _, r := range rows {
		if r.Remaining <= 0 {
			continue
		}
		due = append(due, DueAmount{DueDate: r.Term.DueDate, Remaining: r.Remaining})
		days := daysBetween(today, r.Term.DueDate)
		switch {
		case days < 0:
			overdue.Amount += r.Remaining
			overdue.Count++
			overdueProjects[r.Term.ProjectID] = true
		case days <= FinanceHorizonDays:
			dueSoon.Amount += r.Remaining
			dueSoon.Count++
			dueProjects[r.Term.ProjectID] = true
		}
	}
	overdue.Projects, dueSoon.Projects = len(overdueProjects), len(dueProjects)
	return overdue, dueSoon, due
}

// SummarizePayables adds up what is still owed to vendors and venues, and
// returns the vendor balances with a due date as due amounts for the cash flow
// (the dashboard's "keluar": a venue has no due date, so it is not in it).
func SummarizePayables(rows []PayableRow, today time.Time) (sum PayablesSummary, due []DueAmount) {
	for _, r := range rows {
		left := r.Remaining()
		if left <= 0 {
			continue
		}
		sum.Amount += left
		sum.Count++
		if r.Status(today) == PayableOverdue {
			sum.OverdueAmount += left
			sum.OverdueCount++
		}
		if r.Kind == PayableVendor && r.DueDate != nil {
			due = append(due, DueAmount{DueDate: *r.DueDate, Remaining: left})
		}
	}
	return sum, due
}

// ReceivedAmount is the net client money in a set of payments: every payment
// adds (a Tambahan too: it is money received), a Refund subtracts.
func ReceivedAmount(lines []PaymentLine) int64 {
	var total int64
	for _, l := range lines {
		if l.Type == PaymentRefund {
			total -= l.Amount
		} else {
			total += l.Amount
		}
	}
	return total
}

// VerificationRow is one client payment that has proof but no verification yet.
type VerificationRow struct {
	PaymentID       int64
	ProjectID       int64
	ProjectName     string
	BrideName       string
	GroomName       string
	Type            PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	RecordedByID    int64
}
