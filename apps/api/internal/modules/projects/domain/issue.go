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
	ID                   int64
	ProjectID            int64
	ProjectVendorID      int64
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
}

func (s IssueStatus) IsOpen() bool {
	return s != IssueResolved && s != IssueClosed
}
