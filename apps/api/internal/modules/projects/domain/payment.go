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
