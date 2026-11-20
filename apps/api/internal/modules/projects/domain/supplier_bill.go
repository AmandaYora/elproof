package domain

import (
	"errors"
	"sort"
	"time"
)

// Paket WO Tahap 4 (PLAN §3.2, §4.4, ADR-0047): a vendor or venue the client
// pays directly ("dibayar klien") has its own bills, and the couple can send
// proof of a payment to it from the portal. That money is the client's: it is
// never part of the contract, the WO's cash, revenue, cash flow or payables
// (those read supplier_payments.payer = 'WO' only).

// SupplierBillTerm is one bill of a supplier the client pays directly: what
// the supplier asks for (label, amount) and by when (supplier_bill_terms).
type SupplierBillTerm struct {
	ID                int64
	ProjectID         int64
	ProjectSupplierID int64
	Label             string
	Amount            int64
	DueDate           time.Time
	SortOrder         int
	// DeletedAt is set while the bill waits in the "deleted" state (ADR-0034).
	DeletedAt *time.Time
	// UpdatedAt is the row's version (If-Match, ADR-0036).
	UpdatedAt time.Time
}

// SupplierPaymentStatus is where a supplier payment stands. Derived, never
// stored.
type SupplierPaymentStatus string

const (
	// SupplierPaymentRecorded: staff recorded it; it counts as paid.
	SupplierPaymentRecorded SupplierPaymentStatus = "recorded"
	// SupplierPaymentPending: the client sent the proof; the WO has not checked it.
	SupplierPaymentPending SupplierPaymentStatus = "pending"
	// SupplierPaymentVerified: the client sent it and the WO marked it right.
	SupplierPaymentVerified SupplierPaymentStatus = "verified"
	// SupplierPaymentRejected: the client sent it and the WO refused it.
	SupplierPaymentRejected SupplierPaymentStatus = "rejected"
)

// IsClientSubmission reports whether the client sent this payment from the portal.
func (p SupplierPayment) IsClientSubmission() bool { return p.SubmittedByClientID != 0 }

// Status is the payment's standing (see SupplierPaymentStatus).
func (p SupplierPayment) Status() SupplierPaymentStatus {
	switch {
	case !p.IsClientSubmission():
		return SupplierPaymentRecorded
	case p.VerifiedAt != nil:
		return SupplierPaymentVerified
	case p.RejectedAt != nil:
		return SupplierPaymentRejected
	default:
		return SupplierPaymentPending
	}
}

// CountsAsPaid reports whether the payment counts as paid to its supplier: one
// staff recorded always does; one the client sent only once the WO verified it.
// A pending or rejected proof never lowers what is left to pay.
func (p SupplierPayment) CountsAsPaid() bool {
	return !p.IsClientSubmission() || p.VerifiedAt != nil
}

// AwaitsReview reports whether the WO still has to check the client's proof.
func (p SupplierPayment) AwaitsReview() bool {
	return p.Status() == SupplierPaymentPending
}

// SupplierPaymentCountsSQL is CountsAsPaid for a query over supplier_payments
// aliased as sp.
const SupplierPaymentCountsSQL = "(sp.submitted_by_client_id IS NULL OR sp.verified_at IS NOT NULL)"

// SupplierPaymentPendingSQL is AwaitsReview for a query over supplier_payments
// aliased as sp.
const SupplierPaymentPendingSQL = "(sp.submitted_by_client_id IS NOT NULL AND sp.verified_at IS NULL AND sp.rejected_at IS NULL)"

// Errors of checking a client's proof to a supplier.
var (
	// ErrNotClientSubmission: only a proof the client sent is verified or refused.
	ErrNotClientSubmission = errors.New("only a payment the client sent can be verified or rejected")
	// ErrSubmissionRejected: a refused proof cannot be marked right afterwards.
	ErrSubmissionRejected = errors.New("this submission was rejected")
	// ErrSubmissionVerified: a proof marked right cannot be refused afterwards.
	ErrSubmissionVerified = errors.New("this submission was already verified")
)

// MaxRejectionReasonLen is what rejection_reason holds.
const MaxRejectionReasonLen = 500

// VerifySubmission marks the client's proof right, by `actor` at `now`. It is
// idempotent: an already verified proof is left as it is and changed is false.
// A refused proof cannot be verified (the client sends a new one), and a proof
// with no file cannot be verified (ADR-0032).
func (p *SupplierPayment) VerifySubmission(now time.Time, actor int64, evidenceCount int) (changed bool, err error) {
	switch {
	case !p.IsClientSubmission():
		return false, ErrNotClientSubmission
	case p.VerifiedAt != nil:
		return false, nil
	case p.RejectedAt != nil:
		return false, ErrSubmissionRejected
	case evidenceCount < 1:
		return false, ErrVerifyNeedsEvidence
	}
	at := now.Truncate(time.Second)
	p.VerifiedAt, p.VerifiedByStaffID = &at, actor
	return true, nil
}

// RejectSubmission refuses the client's proof with a reason. Idempotent on an
// already refused proof (the first reason stays). A verified proof cannot be
// refused: the WO edits or deletes the payment instead.
func (p *SupplierPayment) RejectSubmission(now time.Time, actor int64, reason string) (changed bool, err error) {
	switch {
	case !p.IsClientSubmission():
		return false, ErrNotClientSubmission
	case p.RejectedAt != nil:
		return false, nil
	case p.VerifiedAt != nil:
		return false, ErrSubmissionVerified
	}
	at := now.Truncate(time.Second)
	p.RejectedAt, p.RejectedByStaffID, p.RejectionReason = &at, actor, reason
	return true, nil
}

// ClearVerification removes the verification. It reports whether there was one.
// A change to what was paid makes the proof unchecked again (ADR-0032).
func (p *SupplierPayment) ClearVerification() bool {
	was := p.VerifiedAt != nil
	p.VerifiedAt, p.VerifiedByStaffID = nil, 0
	return was
}

// BillTermProgress is one bill with how much of it the counted payments cover,
// how much the client sent that the WO has not checked yet, and its status.
type BillTermProgress struct {
	Term      SupplierBillTerm
	Paid      int64
	Remaining int64
	// Pending is the amount of the client's unchecked proofs sent for this bill.
	Pending int64
	Status  TermStatus
}

// SupplierBill is where one supplier the client pays directly stands: its
// bills, the payments to it and the totals.
type SupplierBill struct {
	Supplier ProjectSupplier
	Terms    []BillTermProgress
	Payments []SupplierPayment
	// Total is the sum of the bills; Paid what the counted payments cover (DP +
	// Termin + Pelunasan - Refund, never negative); Remaining what is left of
	// Total; Pending the unchecked proofs to this supplier.
	Total     int64
	Paid      int64
	Remaining int64
	Pending   int64
}

// BuildSupplierBill puts one client-paid supplier's bills and payments together
// as of `today`. Only payments by the client count (a payment the WO made to
// the supplier is not the client's), and only those that CountsAsPaid; the
// counted money is spread over the bills earliest due date first (AllocateTerms,
// the same rule as the contract's installments).
func BuildSupplierBill(sup ProjectSupplier, terms []SupplierBillTerm, payments []SupplierPayment, today time.Time) SupplierBill {
	b := SupplierBill{Supplier: sup}
	var lines []PaymentLine
	pendingByTerm := map[int64]int64{}
	for _, p := range payments {
		if p.ProjectSupplierID != sup.ID || p.Payer != PayerClient || p.DeletedAt != nil {
			continue
		}
		b.Payments = append(b.Payments, p)
		switch {
		case p.CountsAsPaid():
			lines = append(lines, PaymentLine{Type: p.Type, Amount: p.Amount})
		case p.AwaitsReview():
			b.Pending += p.Amount
			if p.BillTermID != 0 {
				pendingByTerm[p.BillTermID] += p.Amount
			}
		}
	}
	applied := AppliedToContract(lines)
	if applied < 0 {
		applied = 0
	}
	b.Paid = applied

	asTerms := make([]PaymentTerm, 0, len(terms))
	byID := make(map[int64]SupplierBillTerm, len(terms))
	for _, t := range terms {
		if t.ProjectSupplierID != sup.ID || t.DeletedAt != nil {
			continue
		}
		b.Total += t.Amount
		byID[t.ID] = t
		asTerms = append(asTerms, PaymentTerm{ID: t.ID, ProjectID: t.ProjectID, Label: t.Label, Amount: t.Amount, DueDate: t.DueDate})
	}
	for _, tp := range AllocateTerms(asTerms, applied, today) {
		b.Terms = append(b.Terms, BillTermProgress{
			Term: byID[tp.Term.ID], Paid: tp.Paid, Remaining: tp.Remaining,
			Pending: pendingByTerm[tp.Term.ID], Status: tp.Status,
		})
	}
	if r := b.Total - b.Paid; r > 0 {
		b.Remaining = r
	}
	return b
}

// SupplierBillDue is one open bill with the name of the supplier it is for
// (always the real name: the client pays this supplier, ClientSeesName).
type SupplierBillDue struct {
	SupplierName string
	Bill         BillTermProgress
}

// OpenSupplierBills lists every bill not fully paid, earliest due date first.
func OpenSupplierBills(bills []SupplierBill) []SupplierBillDue {
	var out []SupplierBillDue
	for _, b := range bills {
		for _, t := range b.Terms {
			if t.Status != TermPaid {
				out = append(out, SupplierBillDue{SupplierName: b.Supplier.Name, Bill: t})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, c := out[i].Bill.Term, out[j].Bill.Term
		if !a.DueDate.Equal(c.DueDate) {
			return a.DueDate.Before(c.DueDate)
		}
		return a.ID < c.ID
	})
	return out
}

// NextSupplierBill is the earliest bill not fully paid ("tagihan berikutnya"),
// nil when none.
func NextSupplierBill(bills []SupplierBill) *SupplierBillDue {
	open := OpenSupplierBills(bills)
	if len(open) == 0 {
		return nil
	}
	return &open[0]
}

// NeedsClientAction reports whether the client still has to pay this bill: it
// is not fully paid, and the proofs the WO has not checked yet do not cover
// what is left.
func (d SupplierBillDue) NeedsClientAction() bool {
	return d.Bill.Remaining > 0 && d.Bill.Pending < d.Bill.Remaining
}

// PaidByLock is PaidByLockMessage with the supplier's bills: a
// supplier with bills is the client's to pay, so its payer cannot change while
// any live bill hangs off it.
func (a SupplierAttachments) PaidByLock(paidBy Payer) string {
	if msg := PaidByLockMessage(paidBy, a.Payments, a.Files); msg != "" {
		return msg
	}
	if a.BillTerms > 0 {
		return "Klien langsung. Terkunci karena sudah ada tagihan vendor. Hapus tagihan itu untuk mengubah."
	}
	return ""
}

// ClientBillTermRow is one live bill of a client-paid supplier of an open
// project, as the dashboard reads them across the workspace.
type ClientBillTermRow struct {
	Term         SupplierBillTerm
	ProjectName  string
	SupplierKind SupplierKind
	SupplierName string
}

// ClientBillDue is one dashboard attention item "tagihan vendor dibayar klien":
// a bill not fully paid that is overdue or due within dueSoonDays. Pending is
// what the client sent for it that the WO has not checked yet.
type ClientBillDue struct {
	BillTermID        int64
	ProjectSupplierID int64
	ProjectID         int64
	ProjectName       string
	SupplierKind      SupplierKind
	SupplierName      string
	Label             string
	DueDate           time.Time
	Remaining         int64
	Pending           int64
	Status            TermStatus
}

// SelectClientBillsDue spreads each client-paid supplier's counted client
// payments over its bills (BuildSupplierBill) and keeps the bills with
// something left that are overdue or due within dueSoonDays, earliest due date
// first, at most MaxDashboardRows. These are the client's debts to its own
// vendors: never part of the WO's cash flow or payables.
func SelectClientBillsDue(today time.Time, rows []ClientBillTermRow, payments []SupplierPayment) []ClientBillDue {
	type group struct {
		row   ClientBillTermRow
		terms []SupplierBillTerm
	}
	bySupplier := map[int64]*group{}
	var order []int64
	for _, r := range rows {
		g, ok := bySupplier[r.Term.ProjectSupplierID]
		if !ok {
			g = &group{row: r}
			bySupplier[r.Term.ProjectSupplierID] = g
			order = append(order, r.Term.ProjectSupplierID)
		}
		g.terms = append(g.terms, r.Term)
	}
	paid := map[int64][]SupplierPayment{}
	for _, p := range payments {
		paid[p.ProjectSupplierID] = append(paid[p.ProjectSupplierID], p)
	}
	var out []ClientBillDue
	for _, id := range order {
		g := bySupplier[id]
		sup := ProjectSupplier{ID: id, ProjectID: g.row.Term.ProjectID, Kind: g.row.SupplierKind, Name: g.row.SupplierName, PaidBy: PayerClient}
		for _, t := range BuildSupplierBill(sup, g.terms, paid[id], today).Terms {
			if t.Remaining <= 0 || daysBetween(today, t.Term.DueDate) > dueSoonDays {
				continue
			}
			out = append(out, ClientBillDue{
				BillTermID: t.Term.ID, ProjectSupplierID: id, ProjectID: t.Term.ProjectID,
				ProjectName: g.row.ProjectName, SupplierKind: g.row.SupplierKind, SupplierName: g.row.SupplierName,
				Label: t.Term.Label, DueDate: t.Term.DueDate, Remaining: t.Remaining, Pending: t.Pending, Status: t.Status,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].DueDate.Equal(out[j].DueDate) {
			return out[i].DueDate.Before(out[j].DueDate)
		}
		return out[i].BillTermID < out[j].BillTermID
	})
	if len(out) > MaxDashboardRows {
		out = out[:MaxDashboardRows]
	}
	return out
}

// SupplierProofPending is one dashboard attention item "bukti menunggu
// diperiksa": a proof the client sent to a vendor it pays directly that the WO
// has not verified or rejected yet.
type SupplierProofPending struct {
	SupplierPaymentID int64
	ProjectID         int64
	ProjectName       string
	ProjectSupplierID int64
	SupplierKind      SupplierKind
	SupplierName      string
	BillTermID        int64 // 0 = the bill was deleted since
	BillTermLabel     string
	Type              PaymentType
	Amount            int64
	PaymentDate       time.Time
	SubmittedAt       time.Time
}
