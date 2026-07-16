package domain

import "time"

type EngagementStatus string

const (
	EngagementPlanned     EngagementStatus = "Planned"
	EngagementNegotiation EngagementStatus = "Negotiation"
	EngagementBooked      EngagementStatus = "Booked"
	EngagementDPPaid      EngagementStatus = "DP Paid"
	EngagementInProgress  EngagementStatus = "In Progress"
	EngagementFullyPaid   EngagementStatus = "Fully Paid"
	EngagementReady       EngagementStatus = "Ready"
	EngagementCompleted   EngagementStatus = "Completed"
	EngagementCancelled   EngagementStatus = "Cancelled"
)

type ProjectVendor struct {
	ID               int64
	ProjectID        int64
	VendorID         int64
	CategoryID       int64
	Scope            string
	ContractValue    int64
	EngagementStatus EngagementStatus
	BookingDate      *time.Time
	EventDate        time.Time
	DPAmount         int64
	PaidAmount       int64
	DueDate          *time.Time
	PICStaffID       int64
	Notes            string
}

// VendorEngagementHistoryRow is one row of a vendor's engagement history
// across every project in the tenant — backs the `vendors` module's "Lihat
// Project" feature (resolved via `contracts.Contracts.ListVendorEngagementHistory`,
// since `project_vendors` is owned by this module, not `vendors`).
type VendorEngagementHistoryRow struct {
	ProjectID        int64
	ProjectName      string
	EventDate        time.Time
	Venue            string
	EngagementStatus EngagementStatus
}

type VendorMilestone struct {
	ID              int64
	ProjectVendorID int64
	SortOrder       int
	Name            string
	Description     string
	Status          MilestoneStatus
	TargetDate      time.Time
	CompletedDate   *time.Time
	PICStaffID      int64
	Notes           string
}
