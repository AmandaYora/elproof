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

// VendorPayment deliberately has no stored evidence-completeness field --
// unlike the direct invoice_evidence_id/proof_evidence_id columns this
// struct originally carried (dropped, migration 000029: confirmed no code
// path had ever written a non-null value to either), completeness is
// computed on demand via PaymentEvidenceStatus/IsPaymentEvidenceComplete
// below, cross-referencing the polymorphic evidence table (related_kind =
// "payment", distinguished by evidence.type Invoice vs Transfer Proof) --
// see PLAN.md "Payment evidence (Invoice/Bukti Transfer)...". A Refund only
// ever needs proof; every other type needs both invoice and proof, same
// rule as before, just finally collectible.
type VendorPayment struct {
	ID              int64
	ProjectID       int64
	ProjectVendorID int64
	Type            PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}

// PaymentEvidenceStatus cross-references a project's (or, for the
// dashboard, several projects') evidence rows into two lookup sets keyed by
// the paying entity's id (vendor_payments.id or venue_payments.id -- two
// independent AUTO_INCREMENT sequences that can collide in value, which is
// exactly why `kind` is required: it scopes the lookup to one relatedKind at
// a time so a vendor payment's evidence can never leak into a venue
// payment's completeness check, or vice versa), split by evidence.type --
// Invoice vs Transfer Proof. A pure function (no I/O) so every caller (the
// payments list endpoints, ComputeProgress, the dashboard's
// incomplete-payments widget, for both vendor and venue payments) shares one
// definition instead of separate copies of the same loop.
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

// IsPaymentEvidenceComplete applies the same Refund-only-needs-proof rule
// VendorPayment (and now VenuePayment) always had, against the lookup sets
// PaymentEvidenceStatus built. Takes the payment's type/id directly rather
// than the struct itself -- VendorPayment and VenuePayment are distinct Go
// types with no shared interface, and both only ever need these two fields
// here.
func IsPaymentEvidenceComplete(paymentType PaymentType, paymentID int64, hasInvoice, hasProof map[int64]bool) bool {
	if paymentType == PaymentRefund {
		return hasProof[paymentID]
	}
	return hasInvoice[paymentID] && hasProof[paymentID]
}

// ClientPayment tracks money coming IN from the client against the
// project's own ContractValue (PLAN.md "Uang Masuk dari Client") -- the
// opposite accounting direction from VendorPayment, and structurally
// simpler: no ProjectVendorID (it belongs to the project itself, not any
// vendor) and no evidence-completeness dual condition, since it only ever
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
	// VerifiedAt/VerifiedByStaffID: a staff member confirmed the money arrived
	// and the proof matches. A verified payment always has at least one
	// evidence file and any change to what was paid clears the verification
	// (ADR-0032), so "verified" only ever describes the payment as it stands.
	VerifiedAt        *time.Time
	VerifiedByStaffID int64
}

// ErrVerifyNeedsEvidence is returned by Verify when no proof is attached.
var ErrVerifyNeedsEvidence = errors.New("a client payment needs at least one evidence file before it can be verified")

// IsVerified reports whether the payment currently carries a verification.
func (p ClientPayment) IsVerified() bool { return p.VerifiedAt != nil }

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

// VenuePayment tracks money going OUT to the project's venue -- same
// accounting direction and same two-slot evidence rule (Invoice + Transfer
// Proof, via PaymentEvidenceStatus/IsPaymentEvidenceComplete above scoped to
// RelatedVenuePayment) as VendorPayment, but structurally simpler like
// ClientPayment: no ProjectVendorID-equivalent column, since a project has
// at most one venue (Project.VenueID) and this ties directly to ProjectID.
// See PLAN.md "Venue Payments + Pembayaran tab restructuring".
type VenuePayment struct {
	ID              int64
	ProjectID       int64
	Type            PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}

// PaymentLine is the part of a payment the overpayment rule needs, shared by
// client, vendor and venue payments (three distinct Go types, one rule).
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
// the payments of one owed amount (a client, one vendor engagement, the venue)
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
