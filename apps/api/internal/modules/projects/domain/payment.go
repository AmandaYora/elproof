package domain

import (
	"errors"
	"time"
)

type PaymentType string

const (
	PaymentDP        PaymentType = "DP"
	PaymentTermin    PaymentType = "Termin"
	PaymentPelunasan PaymentType = "Pelunasan"
	PaymentTambahan  PaymentType = "Tambahan"
	PaymentRefund    PaymentType = "Refund"
)

// ValidPaymentTypes lists every accepted payment type (the column is an enum).
var ValidPaymentTypes = []PaymentType{PaymentDP, PaymentTermin, PaymentPelunasan, PaymentTambahan, PaymentRefund}

// IsValidPaymentType reports whether t is a known payment type.
func IsValidPaymentType(t PaymentType) bool {
	for _, v := range ValidPaymentTypes {
		if v == t {
			return true
		}
	}
	return false
}

// Ledger is where one owed amount stands: what is owed, how much of it the
// payments cover and what is left.
type Ledger struct {
	ContractValue int64
	Paid          int64
	Remaining     int64
}

// ComputeLedger summarises the payments of one owed amount (a client, one
// supplier) against what is owed. Paid is what counts
// toward the contract (AppliedToContract: DP + Termin + Pelunasan - Refund; a
// Tambahan is outside the contract) and is never negative; Remaining is never
// negative either: an overpaid account (possible only by data that predates the
// overpayment rule) simply has nothing left.
func ComputeLedger(lines []PaymentLine, contractValue int64) Ledger {
	paid := AppliedToContract(lines)
	if paid < 0 {
		paid = 0
	}
	remaining := contractValue - paid
	if remaining < 0 {
		remaining = 0
	}
	return Ledger{ContractValue: contractValue, Paid: paid, Remaining: remaining}
}

// SupplierPayment is money paid to one vendor or venue of the project. Payer
// is the payment's own (PLAN §3.2): the WO paying out of the contract, or the
// client paying the supplier directly, outside it. Cash, debt and dashboard
// queries filter on it, never on the supplier's current paid_by, so an old
// payment never changes meaning. Evidence completeness is computed from the
// evidence table (related_kind supplierPayment): Invoice + Transfer Proof, a
// Refund needs only the proof.
type SupplierPayment struct {
	ID                int64
	ProjectID         int64
	ProjectSupplierID int64
	// BillTermID is the bill (supplier_bill_terms) a client's proof pays; 0 when
	// none (every payment the WO records, a bill since deleted).
	BillTermID        int64
	Payer             Payer
	Type              PaymentType
	Amount            int64
	PaymentDate       time.Time
	Method            string
	ReferenceNumber   string
	Notes             string
	RecordedByStaffID int64
	// SubmittedByClientID is the client who sent the proof from the portal
	// (ADR-0047, Paket WO Tahap 4); 0 = staff recorded it. Such a payment counts
	// as paid only once verified (CountsAsPaid). A primitive id into clients.
	SubmittedByClientID int64
	VerifiedAt          *time.Time
	VerifiedByStaffID   int64
	// RejectedAt/RejectedByStaffID/RejectionReason: the WO refused the proof.
	RejectedAt        *time.Time
	RejectedByStaffID int64
	RejectionReason   string
	DeletedAt         *time.Time
	UpdatedAt         time.Time
}

// PaymentEvidenceStatus indexes evidence rows of one related kind by the
// payment they belong to, split by type (Invoice, Transfer Proof). The kind is
// required because ids of different payment tables collide.
func PaymentEvidenceStatus(evidences []Evidence, kind EvidenceRelatedKind) (hasInvoice, hasProof map[int64]bool) {
	hasInvoice = make(map[int64]bool, len(evidences))
	hasProof = make(map[int64]bool, len(evidences))
	for _, e := range evidences {
		if e.RelatedKind != kind {
			continue
		}
		switch e.Type {
		case EvidenceInvoice:
			hasInvoice[e.RelatedID] = true
		case EvidenceTransferProof:
			hasProof[e.RelatedID] = true
		}
	}
	return hasInvoice, hasProof
}

// IsPaymentEvidenceComplete applies the supplier payment rule (a Refund needs
// only the proof, every other type an invoice and a proof) against the lookup
// sets PaymentEvidenceStatus built.
func IsPaymentEvidenceComplete(paymentType PaymentType, paymentID int64, hasInvoice, hasProof map[int64]bool) bool {
	if paymentType == PaymentRefund {
		return hasProof[paymentID]
	}
	return hasInvoice[paymentID] && hasProof[paymentID]
}

// ClientPayment tracks money coming IN from the client against the
// project's own ContractValue (PLAN.md "Uang Masuk dari Client") -- the
// opposite accounting direction from SupplierPayment, and structurally
// simpler: it belongs to the project itself, not to a supplier, and no evidence-completeness dual condition, since it only ever
// has one evidence slot (a transfer proof, checked via the polymorphic
// evidence table's related_kind/related_id, not a direct FK column).
type ClientPayment struct {
	ID              int64
	ProjectID       int64
	Type            PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
	// RecordedByStaffID is the staff member who entered the payment (0 =
	// unknown, e.g. rows from before migration 000065). A primitive reference
	// into `staff`, never a foreign key.
	RecordedByStaffID int64
	// SubmittedByClientID is the client who sent this payment's proof from the
	// Client Portal (ADR-0047); 0 = staff recorded it. A primitive reference into
	// `clients`, never a foreign key. Such a payment has no RecordedByStaffID.
	SubmittedByClientID int64
	// VerifiedAt/VerifiedByStaffID: a staff member confirmed the money arrived
	// and the proof matches. A verified payment always has at least one
	// evidence file and any change to what was paid clears the verification
	// (ADR-0032), so "verified" only ever describes the payment as it stands.
	VerifiedAt        *time.Time
	VerifiedByStaffID int64
	// DeletedAt is set while the payment waits in the "deleted" state (soft
	// delete, ADR-0034); only restore lookups ever see such a row.
	DeletedAt *time.Time

	// UpdatedAt is the row's version (TIMESTAMP(6), bumped by MySQL on any change):
	// the ETag / `updatedAt` of ADR-0036.
	UpdatedAt time.Time
}

// ErrVerifyNeedsEvidence is returned by Verify when no proof is attached.
var ErrVerifyNeedsEvidence = errors.New("a client payment needs at least one evidence file before it can be verified")

// IsVerified reports whether the payment currently carries a verification.
func (p ClientPayment) IsVerified() bool { return p.VerifiedAt != nil }

// CountsTowardContract reports whether the payment counts as paid on the
// contract. A payment staff recorded always does; one the couple sent from the
// Client Portal counts only once staff verified it (ADR-0047), so a fake or
// mistaken proof never lowers what is left to pay. The overpayment guard and the
// installment type still see every live payment (a pending submission is money
// that may still arrive), only "paid", "remaining" and the schedule wait for it.
func (p ClientPayment) CountsTowardContract() bool {
	return p.SubmittedByClientID == 0 || p.IsVerified()
}

// ContractLines is the ledger view of the payments that count toward the
// contract (CountsTowardContract).
func ContractLines(payments []ClientPayment) []PaymentLine {
	lines := make([]PaymentLine, 0, len(payments))
	for _, p := range payments {
		if p.CountsTowardContract() {
			lines = append(lines, PaymentLine{Type: p.Type, Amount: p.Amount})
		}
	}
	return lines
}

// ContractCountsSQL is the same rule as CountsTowardContract for a query over
// client_payments aliased as cp.
const ContractCountsSQL = "(cp.submitted_by_client_id IS NULL OR cp.verified_at IS NOT NULL)"

// ClearVerification removes the verification. It reports whether there was one.
func (p *ClientPayment) ClearVerification() bool {
	was := p.IsVerified()
	p.VerifiedAt, p.VerifiedByStaffID = nil, 0
	return was
}

// Verify marks the payment verified by `actor` at `now`. It is idempotent: an
// already verified payment is left exactly as it is (same person, same time)
// and changed is false. A payment with no evidence file cannot be verified.
func (p *ClientPayment) Verify(now time.Time, actor int64, evidenceCount int) (changed bool, err error) {
	if p.IsVerified() {
		return false, nil
	}
	if evidenceCount < 1 {
		return false, ErrVerifyNeedsEvidence
	}
	// Whole seconds: the column is a DATETIME, which would round a fractional
	// second, and the verify response must say exactly what a later read says.
	at := now.Truncate(time.Second)
	p.VerifiedAt, p.VerifiedByStaffID = &at, actor
	return true, nil
}

// ApplyEdit overwrites the payment's fields with an edit. Changing what was
// paid (type, amount, date, method or reference number) invalidates a
// verification, so it is cleared and cleared is true; changing only the note
// keeps it.
func (p *ClientPayment) ApplyEdit(t PaymentType, amount int64, date time.Time, method, reference, notes string) (cleared bool) {
	material := p.Type != t || p.Amount != amount || p.PaymentDate.Format("2006-01-02") != date.Format("2006-01-02") ||
		p.Method != method || p.ReferenceNumber != reference
	p.Type, p.Amount, p.PaymentDate, p.Method, p.ReferenceNumber, p.Notes = t, amount, date, method, reference, notes
	if material {
		return p.ClearVerification()
	}
	return false
}

// PaymentLine is the part of a payment the overpayment rule needs, shared by
// client and supplier payments (two distinct Go types, one rule).
type PaymentLine struct {
	Type   PaymentType
	Amount int64
}

// countsAgainstContract: a Tambahan is money for work outside the agreed
// contract, so it never uses up the contract value (and never trips the
// limit). Refund is handled separately: it gives room back.
func countsAgainstContract(t PaymentType) bool {
	return t == PaymentDP || t == PaymentTermin || t == PaymentPelunasan
}

// AppliedToContract is how much of the contract value has been paid:
// DP + Termin + Pelunasan, minus Refund. Tambahan is left out.
func AppliedToContract(lines []PaymentLine) int64 {
	var applied int64
	for _, l := range lines {
		switch {
		case countsAgainstContract(l.Type):
			applied += l.Amount
		case l.Type == PaymentRefund:
			applied -= l.Amount
		}
	}
	return applied
}

// MaxMoney caps any single amount (Rp 1 quadrillion). Far above any real
// wedding, far below int64's limit, so sums of many amounts can never wrap.
const MaxMoney int64 = 1_000_000_000_000_000

// ExcessAfterChange is how far a change to the payment list pushes the paid
// total past what is owed: 0 when the change is fine. `before` and `after` are
// the payments of one owed amount (a client, one supplier)
// before and after the change (add, edit or delete).
//
// A change is refused only when it BOTH makes the applied total exceed
// contractValue AND raises it. So lowering or removing a limited payment is
// always allowed, editing a reference number or date of a payment on an
// already-overpaid project is allowed (the total does not move), and removing
// a Refund (which raises the total) is checked like adding a payment.
// A contractValue <= 0 means no amount is set yet: nothing is limited.
func ExcessAfterChange(before, after []PaymentLine, contractValue int64) int64 {
	if contractValue <= 0 {
		return 0
	}
	was, now := AppliedToContract(before), AppliedToContract(after)
	if now <= was || now <= contractValue {
		return 0
	}
	return now - contractValue
}

// OverpaymentExcess is ExcessAfterChange for recording one new payment `next`
// on top of `existing`.
func OverpaymentExcess(existing []PaymentLine, next PaymentLine, contractValue int64) int64 {
	after := append(append([]PaymentLine(nil), existing...), next)
	return ExcessAfterChange(existing, after, contractValue)
}
