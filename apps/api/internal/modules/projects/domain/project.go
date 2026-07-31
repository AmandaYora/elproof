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
	VenueID       *int64
	PrepStartDate time.Time
	PackageName   string
	ContractValue int64
	Status        ProjectStatus
	PICStaffID    int64
	Description   string
	IsArchived    bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
