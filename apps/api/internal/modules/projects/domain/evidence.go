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
	// EvidenceBookingProof is bukti booked on a project_vendors engagement --
	// see PLAN.md revisi-timeline-vendor-role-sales.
	EvidenceBookingProof EvidenceType = "Booking Proof"
)

type EvidenceRelatedKind string

const (
	RelatedVendorMilestone EvidenceRelatedKind = "vendorMilestone"
	RelatedPayment         EvidenceRelatedKind = "payment"
	RelatedProjectVendor   EvidenceRelatedKind = "projectVendor"
	RelatedIssue           EvidenceRelatedKind = "issue"
	RelatedClientPayment   EvidenceRelatedKind = "clientPayment"
	RelatedVenuePayment    EvidenceRelatedKind = "venuePayment"
	// RelatedGeneral is a project-level document with no specific
	// vendor/payment/vendor-engagement/issue to attach to (rundown, buku
	// acara, teks juru bicara, banquet order, rekap order, rekap dekor,
	// etc. -- see PLAN.md). RelatedID is meaningless for this kind (always
	// the sentinel 0, mirroring Project.VenueID's own "0 = none" convention
	// -- AUTO_INCREMENT never starts at 0), and IsClientVisible below is
	// only ever read/enforced for this one kind.
	RelatedGeneral EvidenceRelatedKind = "general"
	// RelatedProjectMilestone attaches evidence to any Timeline item
	// (project_milestones) -- general capability, not restricted to any
	// specific milestone name (PLAN.md revisi-timeline-vendor-role-sales).
	RelatedProjectMilestone EvidenceRelatedKind = "projectMilestone"
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
	// IsClientVisible controls whether a `general`-kind document appears in
	// Client Portal's own "Dokumen" tab (GET /projects/{id}/documents).
	// Meaningless for every other RelatedKind -- their evidence stays
	// unconditionally visible to a client the moment it's attached to
	// something inside their own project, same as before this field
	// existed; this is a narrower, opt-in exception, not a broader gate.
	// Defaults false (safe-by-default): a staff member must explicitly opt
	// a document in before a client can see it.
	IsClientVisible bool
}
