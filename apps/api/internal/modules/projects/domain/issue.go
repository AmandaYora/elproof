package domain

import "time"

type IssueImpact string

const (
	ImpactLow      IssueImpact = "Low"
	ImpactMedium   IssueImpact = "Medium"
	ImpactHigh     IssueImpact = "High"
	ImpactCritical IssueImpact = "Critical"
)

type IssueStatus string

const (
	IssueOpen         IssueStatus = "Open"
	IssueInReview     IssueStatus = "In Review"
	IssueInResolution IssueStatus = "In Resolution"
	IssueResolved     IssueStatus = "Resolved"
	IssueClosed       IssueStatus = "Closed"
)

type VendorIssue struct {
	ID        int64
	ProjectID int64
	// ProjectVendorID is 0 ("none") for a kendala about the event in general
	// (stored as NULL), the same sentinel convention as Project.PICStaffID.
	ProjectVendorID int64
	// VendorMilestoneID is nil when the issue is general to the vendor
	// engagement rather than about one specific deliverable.
	VendorMilestoneID    *int64
	Title                string
	Description          string
	Impact               IssueImpact
	FoundDate            time.Time
	Status               IssueStatus
	ResolutionPlan       string
	PICStaffID           int64
	TargetResolutionDate *time.Time
	ResolvedDate         *time.Time
	ResolutionNotes      string
	// IsClientVisible: staff decide whether the couple sees this kendala in the
	// Client Portal (list, home counts and actions, progress condition, proof
	// files). Rows from before this field existed are visible.
	IsClientVisible bool
}

// HasVendor reports whether the kendala is about one vendor engagement.
func (i VendorIssue) HasVendor() bool { return i.ProjectVendorID != 0 }

// ClientVisibleIssues keeps only the kendala a client may see.
func ClientVisibleIssues(list []VendorIssue) []VendorIssue {
	out := make([]VendorIssue, 0, len(list))
	for _, i := range list {
		if i.IsClientVisible {
			out = append(out, i)
		}
	}
	return out
}

func (s IssueStatus) IsOpen() bool {
	return s != IssueResolved && s != IssueClosed
}
