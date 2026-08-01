package domain

import "time"

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
