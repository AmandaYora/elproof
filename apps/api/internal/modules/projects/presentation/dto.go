package presentation

import (
	"time"

	"elproof/internal/modules/projects/domain"
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
	ID            int64             `json:"id"`
	Name          string            `json:"name"`
	BrideName     string            `json:"brideName"`
	GroomName     string            `json:"groomName"`
	EventDate     string            `json:"eventDate"`
	Venue         string            `json:"venue"`
	PrepStartDate string            `json:"prepStartDate"`
	PackageName   string            `json:"packageName"`
	ContractValue int64             `json:"contractValue"`
	Status        string            `json:"status"`
	PICStaffID    int64             `json:"picStaffId"`
	Description   string            `json:"description"`
	Progress      *progressResponse `json:"progress,omitempty"`
}

func toProjectResponse(p domain.Project) projectResponse {
	return projectResponse{
		ID: p.ID, Name: p.Name, BrideName: p.BrideName, GroomName: p.GroomName,
		EventDate: p.EventDate.Format(dateLayout), Venue: p.Venue, PrepStartDate: p.PrepStartDate.Format(dateLayout),
		PackageName: p.PackageName, ContractValue: p.ContractValue, Status: string(p.Status),
		PICStaffID: p.PICStaffID, Description: p.Description,
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
}

func toMilestoneResponse(m domain.ProjectMilestone) milestoneResponse {
	return milestoneResponse{
		ID: m.ID, SortOrder: m.SortOrder, Name: m.Name, Status: string(m.Status),
		TargetDate: m.TargetDate.Format(dateLayout), CompletedDate: formatDatePtr(m.CompletedDate),
	}
}

type projectVendorResponse struct {
	ID               int64   `json:"id"`
	VendorID         int64   `json:"vendorId"`
	CategoryID       int64   `json:"categoryId"`
	Scope            string  `json:"scope"`
	ContractValue    int64   `json:"contractValue"`
	EngagementStatus string  `json:"engagementStatus"`
	BookingDate      *string `json:"bookingDate"`
	EventDate        string  `json:"eventDate"`
	DPAmount         int64   `json:"dpAmount"`
	PaidAmount       int64   `json:"paidAmount"`
	DueDate          *string `json:"dueDate"`
	PICStaffID       int64   `json:"picStaffId"`
	Notes            string  `json:"notes"`
}

func toProjectVendorResponse(pv domain.ProjectVendor) projectVendorResponse {
	return projectVendorResponse{
		ID: pv.ID, VendorID: pv.VendorID, CategoryID: pv.CategoryID, Scope: pv.Scope, ContractValue: pv.ContractValue,
		EngagementStatus: string(pv.EngagementStatus), BookingDate: formatDatePtr(pv.BookingDate),
		EventDate: pv.EventDate.Format(dateLayout), DPAmount: pv.DPAmount, PaidAmount: pv.PaidAmount,
		DueDate: formatDatePtr(pv.DueDate), PICStaffID: pv.PICStaffID, Notes: pv.Notes,
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
}

func toVendorMilestoneResponse(m domain.VendorMilestone) vendorMilestoneResponse {
	return vendorMilestoneResponse{
		ID: m.ID, SortOrder: m.SortOrder, Name: m.Name, Description: m.Description, Status: string(m.Status),
		TargetDate: m.TargetDate.Format(dateLayout), CompletedDate: formatDatePtr(m.CompletedDate),
		PICStaffID: m.PICStaffID, Notes: m.Notes,
	}
}

type paymentResponse struct {
	ID                int64  `json:"id"`
	ProjectVendorID   int64  `json:"projectVendorId"`
	Type              string `json:"type"`
	Amount            int64  `json:"amount"`
	PaymentDate       string `json:"paymentDate"`
	Method            string `json:"method"`
	ReferenceNumber   string `json:"referenceNumber"`
	InvoiceEvidenceID *int64 `json:"invoiceEvidenceId"`
	ProofEvidenceID   *int64 `json:"proofEvidenceId"`
	Notes             string `json:"notes"`
	EvidenceComplete  bool   `json:"evidenceComplete"`
}

func toPaymentResponse(p domain.VendorPayment) paymentResponse {
	return paymentResponse{
		ID: p.ID, ProjectVendorID: p.ProjectVendorID, Type: string(p.Type), Amount: p.Amount,
		PaymentDate: p.PaymentDate.Format(dateLayout), Method: p.Method, ReferenceNumber: p.ReferenceNumber,
		InvoiceEvidenceID: p.InvoiceEvidenceID, ProofEvidenceID: p.ProofEvidenceID, Notes: p.Notes,
		EvidenceComplete: p.IsEvidenceComplete(),
	}
}

type issueResponse struct {
	ID                   int64   `json:"id"`
	ProjectVendorID      int64   `json:"projectVendorId"`
	Title                string  `json:"title"`
	Description          string  `json:"description"`
	Impact               string  `json:"impact"`
	FoundDate            string  `json:"foundDate"`
	Status               string  `json:"status"`
	ResolutionPlan       string  `json:"resolutionPlan"`
	PICStaffID           int64   `json:"picStaffId"`
	TargetResolutionDate *string `json:"targetResolutionDate"`
	ResolvedDate         *string `json:"resolvedDate"`
	ResolutionNotes      string  `json:"resolutionNotes"`
}

func toIssueResponse(i domain.VendorIssue) issueResponse {
	return issueResponse{
		ID: i.ID, ProjectVendorID: i.ProjectVendorID, Title: i.Title, Description: i.Description,
		Impact: string(i.Impact), FoundDate: i.FoundDate.Format(dateLayout), Status: string(i.Status),
		ResolutionPlan: i.ResolutionPlan, PICStaffID: i.PICStaffID,
		TargetResolutionDate: formatDatePtr(i.TargetResolutionDate), ResolvedDate: formatDatePtr(i.ResolvedDate),
		ResolutionNotes: i.ResolutionNotes,
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
	RelatedKind  string  `json:"relatedKind"`
	RelatedID    int64   `json:"relatedId"`
}

func toEvidenceResponse(e domain.Evidence) evidenceResponse {
	return evidenceResponse{
		ID: e.ID, Name: e.Name, Type: string(e.Type), FileName: e.FileName,
		DocumentDate: formatDatePtr(e.DocumentDate), UploadedAt: e.UploadedAt.Format("2006-01-02T15:04:05Z07:00"),
		Description: e.Description, UploadedBy: e.UploadedByStaffID, RelatedKind: string(e.RelatedKind), RelatedID: e.RelatedID,
	}
}

type activityResponse struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	ActorID     int64  `json:"actorStaffId"`
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
