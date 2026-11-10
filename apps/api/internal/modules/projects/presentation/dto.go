package presentation

import (
	"time"

	"elproof/internal/modules/projects/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/etag"
)

const dateLayout = "2006-01-02"

func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

type projectResponse struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	BrideName        string `json:"brideName"`
	GroomName        string `json:"groomName"`
	EventDate        string `json:"eventDate"`
	Venue            string `json:"venue"`
	VenueID          *int64 `json:"venueId"`
	VenueRentalPrice *int64 `json:"venueRentalPrice"`
	VenueCharge      *int64 `json:"venueCharge"`
	PrepStartDate    string `json:"prepStartDate"`
	PackageName      string `json:"packageName"`
	ContractValue    int64  `json:"contractValue"`
	Status           string `json:"status"`
	PICStaffID       int64  `json:"picStaffId"`
	// PICSalesStaffID is the "PIC Sales" slot -- see
	// domain.Project.PICSalesStaffID's doc comment. 0 means "belum
	// ditugaskan", same sentinel convention as PICStaffID.
	PICSalesStaffID int64 `json:"picSalesStaffId"`
	// PICName is populated separately by getProject (PLAN.md's Client Portal
	// restructure) -- toProjectResponse itself never resolves it, same
	// pattern as Progress below. Empty string when unresolved (e.g. the
	// staff row was hard-deleted) rather than omitted, so the frontend never
	// has to distinguish "not fetched yet" from "no name available".
	PICName string `json:"picName"`
	// PICPhone is set only for a client principal (the portal's WhatsApp button).
	PICPhone    string            `json:"picPhone,omitempty"`
	Description string            `json:"description"`
	IsArchived  bool              `json:"isArchived"`
	Progress    *progressResponse `json:"progress,omitempty"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toProjectResponse(p domain.Project) projectResponse {
	return projectResponse{
		ID: p.ID, Name: p.Name, BrideName: p.BrideName, GroomName: p.GroomName,
		EventDate: p.EventDate.Format(dateLayout), Venue: p.Venue, VenueID: p.VenueID,
		VenueRentalPrice: p.VenueRentalPrice, VenueCharge: p.VenueCharge,
		PrepStartDate: p.PrepStartDate.Format(dateLayout),
		PackageName:   p.PackageName, ContractValue: p.ContractValue, Status: string(p.Status),
		PICStaffID: p.PICStaffID, PICSalesStaffID: p.PICSalesStaffID, Description: p.Description, IsArchived: p.IsArchived,
		UpdatedAt: etag.Format(p.UpdatedAt),
	}
}

// venueSummaryResponse is the public-safe subset backing GET
// /projects/{id}/venue (ADR-0016) -- shared verbatim by the WO Console
// Project Detail tab and Client Portal's Venue tab; the former additionally
// fetches GET /venues/{id} directly (staff-only) for commercial fields.
type venueSummaryResponse struct {
	ID                   int64   `json:"id"`
	Name                 string  `json:"name"`
	Address              *string `json:"address"`
	City                 *string `json:"city"`
	Capacity             *int    `json:"capacity"`
	Facilities           *string `json:"facilities"`
	SocialMedia          *string `json:"socialMedia"`
	HasVisibleAttachment bool    `json:"hasVisibleAttachment"`
}

func toVenueSummaryResponse(v vendorscontracts.VenueSummary) venueSummaryResponse {
	return venueSummaryResponse{
		ID: v.ID, Name: v.Name, Address: v.Address, City: v.City, Capacity: v.Capacity,
		Facilities: v.Facilities, SocialMedia: v.SocialMedia, HasVisibleAttachment: v.HasVisibleAttachment,
	}
}

type milestoneStatsResponse struct {
	Total      int     `json:"total"`
	Completed  int     `json:"completed"`
	InProgress int     `json:"inProgress"`
	Blocked    int     `json:"blocked"`
	NotStarted int     `json:"notStarted"`
	Cancelled  int     `json:"cancelled"`
	Overdue    int     `json:"overdue"`
	Ratio      float64 `json:"ratio"`
}

func toMilestoneStatsResponse(s domain.MilestoneStats) milestoneStatsResponse {
	return milestoneStatsResponse{
		Total: s.Total, Completed: s.Completed, InProgress: s.InProgress, Blocked: s.Blocked,
		NotStarted: s.NotStarted, Cancelled: s.Cancelled, Overdue: s.Overdue, Ratio: s.Ratio,
	}
}

type progressResponse struct {
	ProjectMilestoneStats   milestoneStatsResponse `json:"projectMilestoneStats"`
	VendorMilestoneStats    milestoneStatsResponse `json:"vendorMilestoneStats"`
	OverallPercent          int                    `json:"overallPercent"`
	Condition               string                 `json:"condition"`
	OpenIssueCount          int                    `json:"openIssueCount"`
	CriticalOrHighOpenCount int                    `json:"criticalOrHighOpenIssueCount"`
	OverdueMilestoneCount   int                    `json:"overdueMilestoneCount"`
	IncompleteEvidenceCount int                    `json:"incompleteEvidenceCount"`
}

func toProgressResponse(p domain.ProjectProgress) progressResponse {
	return progressResponse{
		ProjectMilestoneStats: toMilestoneStatsResponse(p.ProjectMilestoneStats),
		VendorMilestoneStats:  toMilestoneStatsResponse(p.VendorMilestoneStats),
		OverallPercent:        p.OverallPercent, Condition: string(p.Condition),
		OpenIssueCount: p.OpenIssueCount, CriticalOrHighOpenCount: p.CriticalOrHighOpenCount,
		OverdueMilestoneCount: p.OverdueMilestoneCount, IncompleteEvidenceCount: p.IncompleteEvidenceCount,
	}
}

type milestoneResponse struct {
	ID            int64   `json:"id"`
	SortOrder     int     `json:"order"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	TargetDate    string  `json:"targetDate"`
	CompletedDate *string `json:"completedDate"`
	// Evidence is filled by GET /projects/{id}/milestones (all attachments for
	// staff, only the ones the client may see for a client); `[]` elsewhere.
	Evidence []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toMilestoneResponse(m domain.ProjectMilestone) milestoneResponse {
	return milestoneResponse{
		ID: m.ID, SortOrder: m.SortOrder, Name: m.Name, Status: string(m.Status),
		TargetDate: m.TargetDate.Format(dateLayout), CompletedDate: formatDatePtr(m.CompletedDate),
		Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(m.UpdatedAt),
	}
}

type projectVendorResponse struct {
	ID         int64 `json:"id"`
	VendorID   int64 `json:"vendorId"`
	CategoryID int64 `json:"categoryId"`
	// VendorName/CategoryName are filled by GET /projects/{id}/vendors from
	// this workspace's directory, for this project's own vendors only ("" when
	// unresolved or on mutation responses).
	VendorName       string  `json:"vendorName"`
	CategoryName     string  `json:"categoryName"`
	Scope            string  `json:"scope"`
	ContractValue    int64   `json:"contractValue"`
	PricingTier      string  `json:"pricingTier"`
	EngagementStatus string  `json:"engagementStatus"`
	BookingDate      *string `json:"bookingDate"`
	EventDate        string  `json:"eventDate"`
	EventStartTime   *string `json:"eventStartTime"`
	EventEndTime     *string `json:"eventEndTime"`
	DPAmount         int64   `json:"dpAmount"`
	PaidAmount       int64   `json:"paidAmount"`
	DueDate          *string `json:"dueDate"`
	PICStaffID       int64   `json:"picStaffId"`
	// PICName is the engagement PIC's name ("" when unresolved, and always ""
	// for a client). Filled by GET /projects/{id}/vendors and the restore answer.
	PICName string `json:"picName"`
	// EvidenceCount counts every proof file of the engagement itself, of its
	// milestones and of its payments that the caller may see (staff: all).
	// Filled by GET /projects/{id}/vendors; 0 on the other answers.
	EvidenceCount int    `json:"evidenceCount"`
	Notes         string `json:"notes"`
	// Milestones is only populated by listVendorEngagements (PLAN.md
	// "Performance remediation" Phase E) — omitted (nil) from
	// create/update/cancel responses, which have no batch to draw from and
	// whose callers already re-fetch the vendor section afterward anyway.
	Milestones []vendorMilestoneResponse `json:"milestones,omitempty"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

// toProjectVendorResponse takes paidAmount explicitly rather than reading it
// off domain.ProjectVendor -- there is no such stored field anymore. The
// caller (listVendorEngagements) computes it by summing this engagement's
// own vendor_payments rows (Refund netted as a subtraction), the same
// derivation Client Payments' own "Total Diterima" already uses correctly —
// see PLAN.md "Financial Calculation Correctness".
func toProjectVendorResponse(pv domain.ProjectVendor, paidAmount int64) projectVendorResponse {
	return projectVendorResponse{
		ID: pv.ID, VendorID: pv.VendorID, CategoryID: pv.CategoryID, Scope: pv.Scope, ContractValue: pv.ContractValue,
		PricingTier: string(pv.PricingTier), EngagementStatus: string(pv.EngagementStatus), BookingDate: formatDatePtr(pv.BookingDate),
		EventDate: pv.EventDate.Format(dateLayout), EventStartTime: pv.EventStartTime, EventEndTime: pv.EventEndTime,
		DPAmount: pv.DPAmount, PaidAmount: paidAmount,
		DueDate: formatDatePtr(pv.DueDate), PICStaffID: pv.PICStaffID, Notes: pv.Notes, UpdatedAt: etag.Format(pv.UpdatedAt),
	}
}

type vendorMilestoneResponse struct {
	ID            int64   `json:"id"`
	SortOrder     int     `json:"order"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	TargetDate    string  `json:"targetDate"`
	CompletedDate *string `json:"completedDate"`
	PICStaffID    int64   `json:"picStaffId"`
	Notes         string  `json:"notes"`
	// Evidence is filled inside GET /projects/{id}/vendors (client: only what
	// portal.vendor.view allows); `[]` elsewhere.
	Evidence []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toVendorMilestoneResponse(m domain.VendorMilestone) vendorMilestoneResponse {
	return vendorMilestoneResponse{
		ID: m.ID, SortOrder: m.SortOrder, Name: m.Name, Description: m.Description, Status: string(m.Status),
		TargetDate: m.TargetDate.Format(dateLayout), CompletedDate: formatDatePtr(m.CompletedDate),
		PICStaffID: m.PICStaffID, Notes: m.Notes, Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(m.UpdatedAt),
	}
}

type paymentResponse struct {
	ID               int64  `json:"id"`
	ProjectVendorID  int64  `json:"projectVendorId"`
	Type             string `json:"type"`
	Amount           int64  `json:"amount"`
	PaymentDate      string `json:"paymentDate"`
	Method           string `json:"method"`
	ReferenceNumber  string `json:"referenceNumber"`
	Notes            string `json:"notes"`
	EvidenceComplete bool   `json:"evidenceComplete"`
	// Evidence: the payment's proof files (invoice and/or transfer proof).
	Evidence []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

// toPaymentResponse takes evidenceComplete explicitly rather than reading
// it off domain.VendorPayment -- there is no such stored field/method
// anymore. The caller (listPayments) computes it by cross-referencing this
// payment's own evidence rows (Invoice/Transfer Proof), the same
// presentation-layer derivation already used for Client Payments'
// evidenceComplete and the vendor paid-amount fix. See PLAN.md "Payment
// evidence (Invoice/Bukti Transfer)...".
func toPaymentResponse(p domain.VendorPayment, evidenceComplete bool) paymentResponse {
	return paymentResponse{
		ID: p.ID, ProjectVendorID: p.ProjectVendorID, Type: string(p.Type), Amount: p.Amount,
		PaymentDate: p.PaymentDate.Format(dateLayout), Method: p.Method, ReferenceNumber: p.ReferenceNumber,
		Notes: p.Notes, EvidenceComplete: evidenceComplete, Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(p.UpdatedAt),
	}
}

// personRef names a person without exposing an id.
type personRef struct {
	Name string `json:"name"`
}

type clientPaymentResponse struct {
	ID               int64  `json:"id"`
	Type             string `json:"type"`
	Amount           int64  `json:"amount"`
	PaymentDate      string `json:"paymentDate"`
	Method           string `json:"method"`
	ReferenceNumber  string `json:"referenceNumber"`
	Notes            string `json:"notes"`
	EvidenceComplete bool   `json:"evidenceComplete"`
	// RecordedBy is internal: always null for a client principal, and null when
	// the recorder is unknown or no longer exists.
	RecordedBy *personRef `json:"recordedBy"`
	// VerifiedAt/VerifiedBy: set only while the payment is verified (RFC 3339);
	// VerifiedBy is null when the verifier no longer exists.
	VerifiedAt *string    `json:"verifiedAt"`
	VerifiedBy *personRef `json:"verifiedBy"`
	// Evidence: the payment's proof files (client: only what it may see).
	Evidence []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toClientPaymentResponse(p domain.ClientPayment, evidenceComplete bool) clientPaymentResponse {
	return clientPaymentResponse{
		ID: p.ID, Type: string(p.Type), Amount: p.Amount, PaymentDate: p.PaymentDate.Format(dateLayout),
		Method: p.Method, ReferenceNumber: p.ReferenceNumber, Notes: p.Notes, EvidenceComplete: evidenceComplete,
		VerifiedAt: formatTimePtr(p.VerifiedAt), Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(p.UpdatedAt),
	}
}

type venuePaymentResponse struct {
	ID               int64  `json:"id"`
	Type             string `json:"type"`
	Amount           int64  `json:"amount"`
	PaymentDate      string `json:"paymentDate"`
	Method           string `json:"method"`
	ReferenceNumber  string `json:"referenceNumber"`
	Notes            string `json:"notes"`
	EvidenceComplete bool   `json:"evidenceComplete"`
	// Evidence: the payment's proof files (invoice and/or transfer proof).
	Evidence []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

// toVenuePaymentResponse takes evidenceComplete explicitly for the same
// reason toPaymentResponse does -- see PLAN.md "Venue Payments + Pembayaran
// tab restructuring".
func toVenuePaymentResponse(p domain.VenuePayment, evidenceComplete bool) venuePaymentResponse {
	return venuePaymentResponse{
		ID: p.ID, Type: string(p.Type), Amount: p.Amount, PaymentDate: p.PaymentDate.Format(dateLayout),
		Method: p.Method, ReferenceNumber: p.ReferenceNumber, Notes: p.Notes, EvidenceComplete: evidenceComplete,
		Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(p.UpdatedAt),
	}
}

// ledgerSummaryResponse is where one owed amount stands after a payment was
// recorded: what is owed, what the payments cover (DP + Termin + Pelunasan -
// Refund; Tambahan is outside the contract) and what is left (never negative).
type ledgerSummaryResponse struct {
	ContractValue int64 `json:"contractValue"`
	Paid          int64 `json:"paid"`
	Remaining     int64 `json:"remaining"`
}

func toLedgerSummary(l domain.Ledger) ledgerSummaryResponse {
	return ledgerSummaryResponse{ContractValue: l.ContractValue, Paid: l.Paid, Remaining: l.Remaining}
}

// clientLedgerSummaryResponse adds the schedule: the same objects as
// GET /payment-terms, allocated by the server.
type clientLedgerSummaryResponse struct {
	ledgerSummaryResponse
	Terms []paymentTermResponse `json:"terms"`
}

// evidenceRefs maps proof files to their short response form (never nil).
func evidenceRefs(list []domain.Evidence) []evidenceRefResponse {
	out := make([]evidenceRefResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toEvidenceRef(e))
	}
	return out
}

type issueResponse struct {
	ID int64 `json:"id"`
	// ProjectVendorID is null for a kendala about the event in general.
	ProjectVendorID *int64 `json:"projectVendorId"`
	// VendorMilestoneID is null when the kendala is general to the vendor
	// engagement, not tied to one specific deliverable (PLAN.md "Retire the
	// standalone Kendala tab").
	VendorMilestoneID *int64 `json:"vendorMilestoneId"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	Impact            string `json:"impact"`
	FoundDate         string `json:"foundDate"`
	Status            string `json:"status"`
	ResolutionPlan    string `json:"resolutionPlan"`
	PICStaffID        int64  `json:"picStaffId"`
	// PICName is the kendala PIC's name ("" when unresolved, and always "" for a
	// client). Filled by every staff answer that carries an issue.
	PICName              string  `json:"picName"`
	TargetResolutionDate *string `json:"targetResolutionDate"`
	ResolvedDate         *string `json:"resolvedDate"`
	ResolutionNotes      string  `json:"resolutionNotes"`
	// IsClientVisible: whether the couple sees this kendala in the Client Portal.
	IsClientVisible bool `json:"isClientVisible"`
	// VendorName/VendorMilestoneName/Evidence are filled by GET
	// /projects/{id}/issues (names from this project's own engagements; "" when
	// there is none; client: only the proof portal.issues.view allows).
	VendorName          string                `json:"vendorName"`
	VendorMilestoneName string                `json:"vendorMilestoneName"`
	Evidence            []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toIssueResponse(i domain.VendorIssue) issueResponse {
	var vendorID *int64
	if i.HasVendor() {
		id := i.ProjectVendorID
		vendorID = &id
	}
	return issueResponse{Evidence: []evidenceRefResponse{}, IsClientVisible: i.IsClientVisible,
		ID: i.ID, ProjectVendorID: vendorID, VendorMilestoneID: i.VendorMilestoneID, Title: i.Title, Description: i.Description,
		Impact: string(i.Impact), FoundDate: i.FoundDate.Format(dateLayout), Status: string(i.Status),
		ResolutionPlan: i.ResolutionPlan, PICStaffID: i.PICStaffID,
		TargetResolutionDate: formatDatePtr(i.TargetResolutionDate), ResolvedDate: formatDatePtr(i.ResolvedDate),
		ResolutionNotes: i.ResolutionNotes, UpdatedAt: etag.Format(i.UpdatedAt),
	}
}

type evidenceResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	FileName     string  `json:"fileName"`
	DocumentDate *string `json:"documentDate"`
	UploadedAt   string  `json:"uploadedAt"`
	Description  string  `json:"description"`
	UploadedBy   int64   `json:"uploadedByStaffId"`
	// UploadedByName is the uploader's name ("" when unresolved; always "" for a client).
	UploadedByName  string `json:"uploadedByName"`
	RelatedKind     string `json:"relatedKind"`
	RelatedID       int64  `json:"relatedId"`
	IsClientVisible bool   `json:"isClientVisible"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toEvidenceResponse(e domain.Evidence) evidenceResponse {
	return evidenceResponse{
		ID: e.ID, Name: e.Name, Type: string(e.Type), FileName: e.FileName,
		DocumentDate: formatDatePtr(e.DocumentDate), UploadedAt: e.UploadedAt.Format("2006-01-02T15:04:05Z07:00"),
		Description: e.Description, UploadedBy: e.UploadedByStaffID, RelatedKind: string(e.RelatedKind), RelatedID: e.RelatedID,
		IsClientVisible: e.IsClientVisible, UpdatedAt: etag.Format(e.UpdatedAt),
	}
}

// evidenceRefResponse is the short form of a proof file embedded in another
// item's response; the file itself is fetched through the evidence routes.
type evidenceRefResponse struct {
	ID         int64  `json:"id"`
	Type       string `json:"type"`
	FileName   string `json:"fileName"`
	UploadedAt string `json:"uploadedAt"`
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeLayout)
	return &s
}

func toEvidenceRef(e domain.Evidence) evidenceRefResponse {
	return evidenceRefResponse{ID: e.ID, Type: string(e.Type), FileName: e.FileName, UploadedAt: e.UploadedAt.Format(timeLayout)}
}

type activityResponse struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	ActorID int64  `json:"actorStaffId"`
	// ActorName is the actor's name; "" when the actor is unknown, no longer
	// exists or the entry has none (never a made-up label).
	ActorName   string `json:"actorName"`
	ProjectID   *int64 `json:"projectId"`
	EntityType  string `json:"entityType"`
	EntityID    string `json:"entityId"`
	EntityLabel string `json:"entityLabel"`
	Description string `json:"description"`
	CreatedAt   string `json:"timestamp"`
}

func toActivityResponse(a domain.ActivityLogEntry) activityResponse {
	return activityResponse{
		ID: a.ID, Type: string(a.Type), ActorID: a.ActorStaffID, ProjectID: a.ProjectID, EntityType: a.EntityType,
		EntityID: a.EntityID, EntityLabel: a.EntityLabel, Description: a.Description,
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// quotedVersion is the ETag header value of an item's `updatedAt`. Every item that
// carries one implements response.Versioned with it, so OK and Created set the
// ETag header on their own.
func quotedVersion(updatedAt string) string {
	if updatedAt == "" {
		return ""
	}
	return `"` + updatedAt + `"`
}

func (r projectResponse) ETag() string         { return quotedVersion(r.UpdatedAt) }
func (r milestoneResponse) ETag() string       { return quotedVersion(r.UpdatedAt) }
func (r projectVendorResponse) ETag() string   { return quotedVersion(r.UpdatedAt) }
func (r vendorMilestoneResponse) ETag() string { return quotedVersion(r.UpdatedAt) }
func (r paymentResponse) ETag() string         { return quotedVersion(r.UpdatedAt) }
func (r clientPaymentResponse) ETag() string   { return quotedVersion(r.UpdatedAt) }
func (r venuePaymentResponse) ETag() string    { return quotedVersion(r.UpdatedAt) }
func (r issueResponse) ETag() string           { return quotedVersion(r.UpdatedAt) }
func (r evidenceResponse) ETag() string        { return quotedVersion(r.UpdatedAt) }
