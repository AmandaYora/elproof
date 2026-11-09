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

// ClientCanSeeEvidence is the per-document visibility rule for a `client`
// principal (Client Portal). Only evidence that backs something the portal
// already shows is visible: proof of the client's own payments, documents
// attached to a kendala or a vendor deliverable, and project-level documents or
// timeline attachments a staff member explicitly opted in. Vendor/venue payment
// proofs and booking proofs stay internal.
//
// can is the client's portal permission check (ADR-0026): a payment receipt is
// finance data, so a role without portal.finance.view (Keluarga by default)
// never sees it; issue, vendor and timeline attachments follow their own tabs.
// A timeline attachment additionally needs IsClientVisible (ADR-0032): staff
// decide which photos of a step the couple sees.
func ClientCanSeeEvidence(e Evidence, can func(string) bool) bool {
	switch e.RelatedKind {
	case RelatedClientPayment:
		return can("portal.finance.view")
	case RelatedIssue:
		return can("portal.issues.view")
	case RelatedVendorMilestone:
		return can("portal.vendor.view")
	case RelatedProjectMilestone:
		return can("portal.timeline.view") && e.IsClientVisible
	case RelatedGeneral:
		// The Dokumen tab: shared documents also need portal.documents.view
		// (the per-file routes no longer ask for it up front, see
		// clientReadPermission, so this rule is the only gate for a file).
		return e.IsClientVisible && can("portal.documents.view")
	default:
		return false
	}
}

// SupportsClientVisibility: only these kinds carry the opt-in IsClientVisible
// flag; for every other kind the flag is meaningless and stored false.
func (k EvidenceRelatedKind) SupportsClientVisibility() bool {
	return k == RelatedGeneral || k == RelatedProjectMilestone
}
