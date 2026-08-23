// Package domain holds all of projects' entities. Kept as one package with a
// handful of files (grouped by aggregate) rather than one-file-per-type,
// since this module owns 8 tightly related tables — see MODULE_MAP.md.
package domain

import "time"

type ProjectStatus string

const (
	StatusDraft       ProjectStatus = "Draft"
	StatusPreparation ProjectStatus = "Preparation"
	StatusReady       ProjectStatus = "Ready"
	StatusCompleted   ProjectStatus = "Completed"
	StatusCancelled   ProjectStatus = "Cancelled"
)

type Project struct {
	ID        int64
	TenantID  int64
	Name      string
	BrideName string
	GroomName string
	EventDate time.Time
	Venue     string
	// VenueID is a cross-module primitive reference into vendors' Venue
	// directory (ADR-0016) -- resolved via a module contract, never a SQL
	// foreign key. nil means no structured venue is attached yet; Venue
	// (the free-text field above) stays as the fallback display in that case.
	VenueID *int64
	// VenueRentalPrice/VenueCharge are a per-project SNAPSHOT of the
	// attached venue's cost, captured (and freely editable) at attach time
	// -- never a live join against venues' own current price. This mirrors
	// project_vendors.contract_value's exact lifecycle and exists so a
	// completed project's recorded Margin can never drift just because
	// venue master data changed later. Both nil whenever VenueID is nil;
	// ProjectService.Update force-clears both on detach regardless of what
	// the request body contains -- see PLAN.md "Financial Calculation
	// Correctness".
	VenueRentalPrice *int64
	VenueCharge      *int64
	PrepStartDate    time.Time
	PackageName      string
	ContractValue    int64
	Status           ProjectStatus
	PICStaffID       int64
	// PICSalesStaffID is the second, independent PIC slot ("PIC Sales") --
	// sentinel 0 means "belum ditugaskan", same convention as PICStaffID
	// itself. Set automatically to the creating Sales staff member on
	// create, immutable afterward for that role; Owner/Admin may set or
	// clear it freely. Never cleared by handover (assigning PICStaffID) --
	// see PLAN.md revisi-timeline-vendor-role-sales.
	PICSalesStaffID int64
	Description     string
	IsArchived       bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ProjectRef is a minimal {id, name} projection of a project — backs the
// hard-delete "impact" endpoints (PLAN.md's Vendor/Venue/Staff hard delete):
// naming exactly which projects would lose a data source, without exposing
// the rest of the project record cross-module.
type ProjectRef struct {
	ID   int64
	Name string
}

type MilestoneStatus string

const (
	MilestoneNotStarted MilestoneStatus = "Not Started"
	MilestoneInProgress MilestoneStatus = "In Progress"
	MilestoneCompleted  MilestoneStatus = "Completed"
	MilestoneBlocked    MilestoneStatus = "Blocked"
	MilestoneCancelled  MilestoneStatus = "Cancelled"
)

type ProjectMilestone struct {
	ID            int64
	ProjectID     int64
	SortOrder     int
	Name          string
	Status        MilestoneStatus
	TargetDate    time.Time
	CompletedDate *time.Time
}
