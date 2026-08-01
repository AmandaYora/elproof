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

type VendorPayment struct {
	ID                int64
	ProjectID         int64
	ProjectVendorID   int64
	Type              PaymentType
	Amount            int64
	PaymentDate       time.Time
	Method            string
	ReferenceNumber   string
	InvoiceEvidenceID *int64
	ProofEvidenceID   *int64
	Notes             string
}

// IsEvidenceComplete mirrors mock/selectors.ts's isPaymentEvidenceComplete
// exactly: a Refund only needs proof; anything else needs both invoice and
// proof.
func (p VendorPayment) IsEvidenceComplete() bool {
	if p.Type == PaymentRefund {
		return p.ProofEvidenceID != nil
	}
	return p.InvoiceEvidenceID != nil && p.ProofEvidenceID != nil
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
