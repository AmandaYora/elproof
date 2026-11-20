package presentation

import (
	"strconv"
	"time"

	"elproof/internal/modules/projects/domain"
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
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	BrideName string `json:"brideName"`
	GroomName string `json:"groomName"`
	// EventDate is the main event's date (Hari H).
	EventDate     string `json:"eventDate"`
	PrepStartDate string `json:"prepStartDate"`
	// The contract (paket WO Tahap 3): the cache of the live Rincian lines,
	// read-only. ContractValue is the total (subtotal + PPN). A client without
	// portal.finance.view gets none of the three (PortalPolicy.Project).
	ContractValue    *int64 `json:"contractValue,omitempty"`
	ContractSubtotal *int64 `json:"contractSubtotal,omitempty"`
	ContractTax      *int64 `json:"contractTax,omitempty"`
	// IsPKP: the workspace was PKP when the project was created, so its
	// contract carries PPN.
	IsPKP bool `json:"isPkp"`
	// LinesVersion is the Rincian's own version (If-Match of line writes);
	// staff only.
	LinesVersion string `json:"linesVersion,omitempty"`
	// PackageNames are the names of the project's packages (Rincian), in
	// screen order. A client needs portal.vendor.view to see them.
	PackageNames []string `json:"packageNames"`
	Status       string   `json:"status"`
	PICStaffID   int64    `json:"picStaffId"`
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
	// CoverURL is where the Client Portal hero photo streams from
	// (/api/v1/projects/{id}/cover?v=<unix of the last change>); null when the
	// project has no cover (ADR-0047).
	CoverURL *string `json:"coverUrl"`
	// Location is where the events happen ("Masjid Al-Azhar · Gedung Graha"),
	// "" before a venue is chosen. Filled by the project read and list.
	Location string `json:"location"`
	// EventsLine names the events ("Akad, Resepsi"). Filled with Location.
	EventsLine string `json:"eventsLine"`
	// Quotation is the Accepted quotation the project was made from ("Dibuat
	// dari MO-PNW-261119-001 Revisi 2", the "Kesepakatan awal" link: GET
	// /projects/{id}/agreement); null for a project made by hand or
	// duplicated. Staff only.
	Quotation *projectQuotationResponse `json:"quotation"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

// projectQuotationResponse is the quotation a project was made from.
type projectQuotationResponse struct {
	ID        int64  `json:"id"`
	Number    string `json:"number"`
	RevisionN int    `json:"revisionN"`
}

func toProjectQuotation(p domain.Project) *projectQuotationResponse {
	if p.QuotationID == nil {
		return nil
	}
	return &projectQuotationResponse{ID: *p.QuotationID, Number: p.QuotationNumber, RevisionN: p.QuotationRevisionN}
}

func toProjectResponse(p domain.Project) projectResponse {
	return projectResponse{
		ID: p.ID, Name: p.Name, BrideName: p.BrideName, GroomName: p.GroomName,
		EventDate: p.EventDate.Format(dateLayout), PrepStartDate: p.PrepStartDate.Format(dateLayout),
		ContractValue: money(p.ContractTotal), ContractSubtotal: money(p.ContractSubtotal), ContractTax: money(p.ContractTax),
		IsPKP: p.IsPKP, LinesVersion: etag.Format(p.LinesVersion), Status: string(p.Status),
		PICStaffID: p.PICStaffID, PICSalesStaffID: p.PICSalesStaffID, Description: p.Description, IsArchived: p.IsArchived,
		CoverURL: coverURL(p), Quotation: toProjectQuotation(p), UpdatedAt: etag.Format(p.UpdatedAt),
	}
}

func money(v int64) *int64 { return &v }

// toPortalProjectResponse is a project as the Client Portal shows it
// (PortalPolicy.Project): no preparation start, no Sales PIC, no lines'
// version, and no money without portal.finance.view.
func toPortalProjectResponse(pp domain.PortalProject) projectResponse {
	resp := toProjectResponse(pp.Project)
	resp.PrepStartDate = ""
	resp.LinesVersion = ""
	resp.Quotation = nil
	if !pp.ShowMoney {
		resp.ContractValue, resp.ContractSubtotal, resp.ContractTax = nil, nil, nil
	}
	return resp
}

// coverURL is the cover photo's address, versioned by when it last changed so a
// browser never shows a cached old photo after a replace; nil without a cover.
func coverURL(p domain.Project) *string {
	if p.CoverPhotoPath == nil {
		return nil
	}
	var v int64
	if p.CoverUpdatedAt != nil {
		v = p.CoverUpdatedAt.Unix()
	}
	u := "/api/v1/projects/" + strconv.FormatInt(p.ID, 10) + "/cover?v=" + strconv.FormatInt(v, 10)
	return &u
}

// eventSessionResponse is one session of an event.
type eventSessionResponse struct {
	ID        int64   `json:"id"`
	Label     string  `json:"label"`
	StartTime *string `json:"startTime"`
	EndTime   *string `json:"endTime"`
	Guests    int     `json:"guests"`
}

// eventVenueResponse is the venue an event takes place at. Staff get the
// project supplier, the directory venue and the capacities copied when it was
// chosen; a client gets the name, address and city only.
type eventVenueResponse struct {
	SupplierID       int64   `json:"supplierId,omitempty"`
	VenueID          int64   `json:"venueId,omitempty"`
	Name             string  `json:"name"`
	CapacityStanding *int    `json:"capacityStanding,omitempty"`
	CapacitySeated   *int    `json:"capacitySeated,omitempty"`
	Address          *string `json:"address"`
	City             *string `json:"city"`
}

// eventResponse is one event of a project ("Acara & rincian", portal Jadwal).
type eventResponse struct {
	ID          int64                  `json:"id"`
	EventKindID int64                  `json:"eventKindId"`
	KindName    string                 `json:"kindName"`
	EventDate   string                 `json:"eventDate"`
	StartTime   *string                `json:"startTime"`
	EndTime     *string                `json:"endTime"`
	Venue       *eventVenueResponse    `json:"venue"`
	Layout      string                 `json:"layout"`
	IsMain      bool                   `json:"isMain"`
	SortOrder   int                    `json:"order"`
	Sessions    []eventSessionResponse `json:"sessions"`
	TotalGuests int                    `json:"totalGuests"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PUT sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toEventResponse(e domain.ProjectEvent) eventResponse {
	sessions := make([]eventSessionResponse, 0, len(e.Sessions))
	for _, ss := range e.Sessions {
		sessions = append(sessions, eventSessionResponse{ID: ss.ID, Label: ss.Label, StartTime: ss.StartTime, EndTime: ss.EndTime, Guests: ss.Guests})
	}
	return eventResponse{
		ID: e.ID, EventKindID: e.EventKindID, KindName: e.KindName, EventDate: e.EventDate.Format(dateLayout),
		StartTime: e.StartTime, EndTime: e.EndTime, Layout: string(e.Layout), IsMain: e.IsMain, SortOrder: e.SortOrder,
		Sessions: sessions, TotalGuests: e.TotalGuests(), UpdatedAt: etag.Format(e.UpdatedAt),
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
	// DaysBeforeEvent is the Jadwal bawaan offset the target follows (H-14 = 14,
	// H+3 = -3); null when the date is fixed (added or dated by hand). EP-06.
	DaysBeforeEvent *int `json:"daysBeforeEvent"`
	// ForClient: a task for the couple, shown on the Client Portal (ADR-0047).
	ForClient bool `json:"forClient"`
	// CompletedByClient: the couple marked it done from the portal.
	CompletedByClient bool `json:"completedByClient"`
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
		TargetDate: m.TargetDate.Format(dateLayout), CompletedDate: formatDatePtr(m.CompletedDate), DaysBeforeEvent: m.DaysBeforeEvent,
		ForClient: m.ForClient, CompletedByClient: m.CompletedByClientID != 0,
		Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(m.UpdatedAt),
	}
}

// supplierResponse is one vendor or venue of a project ("Vendor & venue").
type supplierResponse struct {
	ID   int64  `json:"id"`
	Kind string `json:"kind"`
	// SupplierID is the directory vendor or venue it was chosen from.
	SupplierID   int64  `json:"supplierId,omitempty"`
	Name         string `json:"name"`
	CategoryID   int64  `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	PaidBy       string `json:"paidBy"`
	ShowName     bool   `json:"showName"`
	// CapacityStanding/CapacitySeated: a venue's capacities, copied when chosen.
	CapacityStanding *int `json:"capacityStanding,omitempty"`
	CapacitySeated   *int `json:"capacitySeated,omitempty"`
	// AgreedCost ("Biaya rekanan") and PaidWO (what the WO paid it) are only
	// sent with finance.margin.view; null otherwise.
	AgreedCost *int64 `json:"agreedCost"`
	PaidWO     *int64 `json:"paidWo"`
	// PaidClient is what the client paid this supplier directly.
	PaidClient       *int64  `json:"paidClient"`
	EngagementStatus string  `json:"engagementStatus"`
	BookingDate      *string `json:"bookingDate"`
	DueDate          *string `json:"dueDate"`
	PICStaffID       int64   `json:"picStaffId"`
	PICName          string  `json:"picName"`
	Scope            string  `json:"scope"`
	Notes            string  `json:"notes"`
	// PaidByLocked explains why "Dibayar oleh" can no longer change ("" when it can).
	PaidByLocked string `json:"paidByLocked"`
	// UsedByEvents is how many events take place at this venue (0 for a vendor;
	// a venue at 0 shows "Tidak dipakai acara mana pun").
	UsedByEvents int `json:"usedByEvents"`
	// EvidenceCount counts the files of the supplier, its steps and its payments
	// that the caller may see.
	EvidenceCount int                       `json:"evidenceCount"`
	Milestones    []vendorMilestoneResponse `json:"milestones"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toSupplierResponse(s domain.ProjectSupplier) supplierResponse {
	return supplierResponse{
		ID: s.ID, Kind: string(s.Kind), SupplierID: s.SupplierID, Name: s.Name, CategoryID: s.CategoryID,
		PaidBy: string(s.PaidBy), ShowName: s.ShowName, CapacityStanding: s.CapacityStanding, CapacitySeated: s.CapacitySeated,
		AgreedCost: s.AgreedCost, EngagementStatus: string(s.EngagementStatus), BookingDate: formatDatePtr(s.BookingDate),
		DueDate: formatDatePtr(s.DueDate), PICStaffID: s.PICStaffID, Scope: s.Scope, Notes: s.Notes,
		Milestones: []vendorMilestoneResponse{}, UpdatedAt: etag.Format(s.UpdatedAt),
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

// supplierPaymentResponse is one payment to a vendor or venue of the project,
// by the WO or straight from the client (payer).
type supplierPaymentResponse struct {
	ID                int64  `json:"id"`
	ProjectSupplierID int64  `json:"projectSupplierId"`
	Payer             string `json:"payer"`
	Type              string `json:"type"`
	Amount            int64  `json:"amount"`
	PaymentDate       string `json:"paymentDate"`
	Method            string `json:"method"`
	ReferenceNumber   string `json:"referenceNumber"`
	Notes             string `json:"notes"`
	EvidenceComplete  bool   `json:"evidenceComplete"`
	// BillTermID is the bill of a vendor the client pays that a client's proof
	// pays (Paket WO Tahap 4); null when none.
	BillTermID *int64 `json:"billTermId"`
	// Source is "client" when the couple sent it from the portal, "staff" when
	// the WO recorded it.
	Source string `json:"source"`
	// Status: "recorded" (staff recorded it, counts as paid), "pending" (the
	// client's proof, not checked yet: not counted), "verified" (checked,
	// counted), "rejected" (refused, not counted; see rejectionReason).
	Status          string     `json:"status"`
	VerifiedAt      *string    `json:"verifiedAt"`
	VerifiedBy      *personRef `json:"verifiedBy"`
	RejectedAt      *string    `json:"rejectedAt"`
	RejectedBy      *personRef `json:"rejectedBy"`
	RejectionReason string     `json:"rejectionReason"`
	// Evidence: the payment's proof files (invoice and/or transfer proof).
	Evidence []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toSupplierPaymentResponse(p domain.SupplierPayment, evidenceComplete bool) supplierPaymentResponse {
	return supplierPaymentResponse{
		ID: p.ID, ProjectSupplierID: p.ProjectSupplierID, Payer: string(p.Payer), Type: string(p.Type), Amount: p.Amount,
		PaymentDate: p.PaymentDate.Format(dateLayout), Method: p.Method, ReferenceNumber: p.ReferenceNumber,
		Notes: p.Notes, EvidenceComplete: evidenceComplete, Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(p.UpdatedAt),
		BillTermID: optionalID(p.BillTermID), Source: paymentSource(p.IsClientSubmission()), Status: string(p.Status()),
		VerifiedAt: formatTimePtr(p.VerifiedAt), RejectedAt: formatTimePtr(p.RejectedAt), RejectionReason: p.RejectionReason,
	}
}

// optionalID is null for the "none" sentinel 0.
func optionalID(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

// paymentSource is "client" for a proof the couple sent, "staff" otherwise.
func paymentSource(fromClient bool) string {
	if fromClient {
		return "client"
	}
	return "staff"
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
	// Source is "client" when the couple sent it from the portal, "staff" when
	// the WO recorded it (ADR-0047).
	Source string `json:"source"`
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
		Source:     string(p.Source()),
		VerifiedAt: formatTimePtr(p.VerifiedAt), Evidence: []evidenceRefResponse{}, UpdatedAt: etag.Format(p.UpdatedAt),
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
	// ProjectSupplierID is null for a kendala about the event in general.
	ProjectSupplierID *int64 `json:"projectSupplierId"`
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
	// SupplierName/VendorMilestoneName/Evidence are filled by GET
	// /projects/{id}/issues ("" when there is none). A client gets the name
	// PortalPolicy.SupplierName allows (the service for a hidden vendor) and
	// only the proof portal.issues.view allows.
	SupplierName        string                `json:"supplierName"`
	VendorMilestoneName string                `json:"vendorMilestoneName"`
	Evidence            []evidenceRefResponse `json:"evidence"`
	// UpdatedAt is the row's version: RFC 3339, microseconds, UTC. It is also the
	// ETag of the item and what a PATCH sends back as If-Match (ADR-0036).
	UpdatedAt string `json:"updatedAt"`
}

func toIssueResponse(i domain.VendorIssue) issueResponse {
	var supplierID *int64
	if i.ProjectSupplierID != 0 {
		id := i.ProjectSupplierID
		supplierID = &id
	}
	return issueResponse{Evidence: []evidenceRefResponse{}, IsClientVisible: i.IsClientVisible,
		ID: i.ID, ProjectSupplierID: supplierID, VendorMilestoneID: i.VendorMilestoneID, Title: i.Title, Description: i.Description,
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
	// UploadedByName is the uploader's name: the staff member, or for a file the
	// couple sent from the Client Portal the client (RT-07). "" when unresolved,
	// and always "" when a client reads the list.
	UploadedByName string `json:"uploadedByName"`
	// UploadedBySource is who uploaded the file: "staff" or "client" (sent from
	// the Client Portal; uploadedByStaffId is then 0).
	UploadedBySource string `json:"uploadedBySource"`
	RelatedKind      string `json:"relatedKind"`
	RelatedID        int64  `json:"relatedId"`
	IsClientVisible  bool   `json:"isClientVisible"`
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
		UploadedBySource: uploaderSource(e),
	}
}

// uploaderSource is evidenceResponse.UploadedBySource.
func uploaderSource(e domain.Evidence) string {
	if e.UploadedByClientID != 0 {
		return "client"
	}
	return "staff"
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
func (r supplierResponse) ETag() string        { return quotedVersion(r.UpdatedAt) }
func (r eventResponse) ETag() string           { return quotedVersion(r.UpdatedAt) }
func (r vendorMilestoneResponse) ETag() string { return quotedVersion(r.UpdatedAt) }
func (r supplierPaymentResponse) ETag() string { return quotedVersion(r.UpdatedAt) }
func (r clientPaymentResponse) ETag() string   { return quotedVersion(r.UpdatedAt) }
func (r issueResponse) ETag() string           { return quotedVersion(r.UpdatedAt) }
func (r evidenceResponse) ETag() string        { return quotedVersion(r.UpdatedAt) }
