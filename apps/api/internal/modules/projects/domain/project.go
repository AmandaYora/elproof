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
	ID             int64
	TenantID       int64
	Name           string
	BrideName      string
	GroomName      string
	EventDate      time.Time
	Venue          string
	PrepStartDate  time.Time
	PackageName    string
	ContractValue  int64
	Status         ProjectStatus
	PICStaffID     int64
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
