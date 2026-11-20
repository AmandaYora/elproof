package domain

import (
	"errors"
	"time"
)

// ErrStoragePathTaken: an evidence row already owns this stored object. A
// signed upload names its object up front, so using the same token twice would
// attach one file to two payments; the unique index on evidence.storage_path
// (migration 66) refuses it and the repository reports it with this error.
var ErrStoragePathTaken = errors.New("an evidence row already points at this stored object")

// ErrProjectDeleteBlocked: the database still refuses to delete a project (a
// foreign key from a table the hard delete does not empty yet).
// ErrQuotationConverted is returned by ProjectRepository.Create when another
// project of the workspace already carries the same quotation_id (the unique
// key of a conversion race).
var ErrQuotationConverted = errors.New("a project already exists for this quotation")

var ErrProjectDeleteBlocked = errors.New("a foreign key still references the project")

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
	// EvidenceBookingProof is the booking proof of a project supplier.
	EvidenceBookingProof EvidenceType = "Booking Proof"
)

// ValidEvidenceTypes lists every accepted document type (the column is an enum).
var ValidEvidenceTypes = []EvidenceType{
	EvidenceQuotation, EvidenceInvoice, EvidenceContract, EvidenceTransferProof, EvidenceReceipt,
	EvidencePurchaseOrder, EvidencePhoto, EvidenceDocument, EvidenceScreenshot, EvidenceMinutesOfMeeting,
	EvidenceOther, EvidenceBookingProof,
}

// IsValidEvidenceType reports whether t is a known document type.
func IsValidEvidenceType(t EvidenceType) bool {
	for _, v := range ValidEvidenceTypes {
		if v == t {
			return true
		}
	}
	return false
}

type EvidenceRelatedKind string

const (
	RelatedVendorMilestone EvidenceRelatedKind = "vendorMilestone"
	RelatedSupplierPayment EvidenceRelatedKind = "supplierPayment"
	RelatedProjectSupplier EvidenceRelatedKind = "projectSupplier"
	RelatedIssue           EvidenceRelatedKind = "issue"
	RelatedClientPayment   EvidenceRelatedKind = "clientPayment"
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
	// UploadedByClientID is the client who sent the file from the Client Portal
	// (a primitive id into `clients`, no foreign key); 0 when staff uploaded it.
	UploadedByClientID int64
	RelatedKind        EvidenceRelatedKind
	RelatedID          int64
	// IsClientVisible controls whether a `general`-kind document appears in
	// Client Portal's own "Dokumen" tab (GET /projects/{id}/documents).
	// Meaningless for every other RelatedKind -- their evidence stays
	// unconditionally visible to a client the moment it's attached to
	// something inside their own project, same as before this field
	// existed; this is a narrower, opt-in exception, not a broader gate.
	// Defaults false (safe-by-default): a staff member must explicitly opt
	// a document in before a client can see it.
	IsClientVisible bool
	// ClientHidden is computed by the repository, never stored: the file hangs
	// off a kendala that staff did not share with the client (is_client_visible
	// = 0). Such a file is never visible to a client, whatever else the rule
	// below says.
	ClientHidden bool

	// UpdatedAt is the row's version (TIMESTAMP(6), bumped by MySQL on any change):
	// the ETag / `updatedAt` of ADR-0036.
	UpdatedAt time.Time
}

// SupportsClientVisibility: only these kinds carry the opt-in IsClientVisible
// flag ("tampilkan ke klien"); for every other kind the flag is meaningless
// and stored false. A supplier step's file needs it when the WO pays the
// supplier (PortalPolicy.Evidence).
func (k EvidenceRelatedKind) SupportsClientVisibility() bool {
	return k == RelatedGeneral || k == RelatedProjectMilestone || k == RelatedVendorMilestone
}

// IsValidRelatedKind reports whether k is a known related kind (the column is an enum).
func IsValidRelatedKind(k EvidenceRelatedKind) bool {
	switch k {
	case RelatedVendorMilestone, RelatedSupplierPayment, RelatedProjectSupplier, RelatedIssue,
		RelatedClientPayment, RelatedGeneral, RelatedProjectMilestone:
		return true
	}
	return false
}
