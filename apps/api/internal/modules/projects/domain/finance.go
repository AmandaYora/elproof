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

// PayableStatus is where an amount owed by the WO stands.
type PayableStatus string

const (
	PayablePaid    PayableStatus = "paid"
	PayableUnpaid  PayableStatus = "unpaid"
	PayableOverdue PayableStatus = "overdue" // a due date that has passed with something left
	// PayableIncomplete: the WO pays this supplier but no agreed cost is
	// recorded yet, so what is owed is unknown ("Biaya belum diisi").
	PayableIncomplete PayableStatus = "incomplete"
)

// PayableFilter is everything /finance/payables narrows by besides the scope.
// Status "open" means anything left to pay (unpaid or overdue) or not known
// yet (incomplete).
type PayableFilter struct {
	Kind       SupplierKind // "" = vendors and venues
	Status     string
	Search     string
	PICStaffID *int64
}

// PayableStatusFilters lists the accepted values of ?status= on /finance/payables.
var PayableStatusFilters = []string{"open", string(PayableUnpaid), string(PayableOverdue), string(PayablePaid), string(PayableIncomplete)}

// PayableRow is one amount the WO owes: a vendor or venue of a project the WO
// pays, with its agreed cost ("Biaya rekanan"), and what the WO has paid
// against it (DP + Termin + Pelunasan - Refund over its live WO payments, never
// negative). A supplier the client pays directly is never a payable. One the WO
// pays without an agreed cost is listed as Incomplete (AgreedCost 0), not
// dropped: the WO still owes it, the amount is just not known yet.
type PayableRow struct {
	Kind         SupplierKind
	ID           int64 // the project supplier's id
	ProjectID    int64
	ProjectName  string
	BrideName    string
	GroomName    string
	PICStaffID   int64
	SupplierID   int64 // the directory vendor or venue
	SupplierName string
	CategoryID   int64 // vendors only
	AgreedCost   int64
	Paid         int64
	DueDate      *time.Time
	Incomplete   bool // no agreed cost recorded yet
}

// Remaining is what is still owed (never negative), the rule of ComputeLedger.
func (r PayableRow) Remaining() int64 {
	return ComputeLedger([]PaymentLine{{Type: PaymentDP, Amount: max64(r.Paid, 0)}}, r.AgreedCost).Remaining
}

// Status is the row's standing as of today; a row without an agreed cost is
// incomplete whatever has been paid.
func (r PayableRow) Status(today time.Time) PayableStatus {
	if r.Incomplete {
		return PayableIncomplete
	}
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
// IncompleteCount is how many WO-paid suppliers have no agreed cost yet: what
// they are owed is not in Amount.
type PayablesSummary struct {
	Amount          int64
	Count           int
	OverdueAmount   int64
	OverdueCount    int
	IncompleteCount int
}

// FinanceSummary is the Keuangan page's headline figures.
type FinanceSummary struct {
	Overdue  MoneyCount // installments past due: what is left on them
	DueSoon  MoneyCount // installments due from today to today+30, not yet paid
	Received MoneyCount // client money received in the last 30 days (Count = payments)
	Payables PayablesSummary
	// PendingVerification: payments with proof, not verified yet. Count and
	// Projects include the client's proofs to vendors they pay directly (Paket
	// WO Tahap 4); Amount is only money to the WO (client payments).
	PendingVerification MoneyCount
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
// returns the balances with a due date as due amounts for the cash flow (the
// dashboard's "keluar"). A row without an agreed cost is only counted.
func SummarizePayables(rows []PayableRow, today time.Time) (sum PayablesSummary, due []DueAmount) {
	for _, r := range rows {
		if r.Incomplete {
			sum.IncompleteCount++
			continue
		}
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
		if r.DueDate != nil {
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

// Kinds of row of the verification queue.
const (
	// VerificationClientPayment: money the client paid the WO (client_payments).
	VerificationClientPayment = "clientPayment"
	// VerificationSupplierPayment: the client's proof of a payment to a vendor
	// they pay directly (supplier_payments, payer Client; Paket WO Tahap 4).
	// Never the WO's money: checked here, counted nowhere in the WO's figures.
	VerificationSupplierPayment = "supplierPayment"
)

// VerificationRow is one payment that has proof but no verification yet: a
// client payment, or a client's proof to a vendor they pay (Kind).
type VerificationRow struct {
	Kind            string
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
	// SubmittedByClientID: the couple sent it from the portal (0 = staff recorded it).
	SubmittedByClientID int64
	// For a supplierPayment: the vendor or venue it pays and the bill it is for
	// (0 / "" when none).
	ProjectSupplierID int64
	SupplierName      string
	BillTermID        int64
	BillTermLabel     string
}
