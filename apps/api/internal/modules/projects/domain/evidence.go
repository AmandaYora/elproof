package domain

import "time"

type EvidenceType string

const (
	EvidenceQuotation        EvidenceType = "Quotation"
	EvidenceInvoice          EvidenceType = "Invoice"
	EvidenceContract         EvidenceType = "Contract"
	EvidenceTransferProof    EvidenceType = "Transfer Proof"
	EvidenceReceipt          EvidenceType = "Receipt"
	EvidencePurchaseOrder    EvidenceType = "Purchase Order"
	EvidencePhoto            EvidenceType = "Photo"
	EvidenceDocument         EvidenceType = "Document"
	EvidenceScreenshot       EvidenceType = "Screenshot"
	EvidenceMinutesOfMeeting EvidenceType = "Minutes of Meeting"
	EvidenceOther            EvidenceType = "Other"
)

type EvidenceRelatedKind string

const (
	RelatedVendorMilestone EvidenceRelatedKind = "vendorMilestone"
	RelatedPayment         EvidenceRelatedKind = "payment"
	RelatedProjectVendor   EvidenceRelatedKind = "projectVendor"
	RelatedIssue           EvidenceRelatedKind = "issue"
)

type Evidence struct {
	ID                int64
	ProjectID         int64
	Name              string
	Type              EvidenceType
	StoragePath       string
	FileName          string
	DocumentDate      *time.Time
	UploadedAt        time.Time
	Description       string
	UploadedByStaffID int64
	RelatedKind       EvidenceRelatedKind
	RelatedID         int64
}
