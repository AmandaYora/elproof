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
	// EventDate is the main event's date ("Acara utama", the Hari H), kept here
	// in the same transaction as the event itself so lists, the calendar and
	// the H-n schedule read one column. Where it happens comes from the events'
	// venues (LocationLine).
	EventDate     time.Time
	PrepStartDate time.Time
	// IsPKP is the workspace's PKP status copied when the project was created:
	// PPN is added to the lines only when it is set (PLAN §3.2).
	IsPKP bool
	// ContractSubtotal, ContractTax and ContractTotal are the cache of the live
	// lines (subtotal = packages + extra guests + items - discounts; PPN; total =
	// the contract value), rewritten in the same transaction as every change of
	// a line or a guest number. LinesVersion is the lines' own If-Match version.
	ContractSubtotal int64
	ContractTax      int64
	ContractTotal    int64
	LinesVersion     time.Time
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
	// QuotationID is the Accepted quotation (in `offers`) this project was
	// created from ("Jadikan proyek", PLAN §5.4); nil for a project made by
	// hand or duplicated. Unique per workspace, so a conversion is idempotent.
	// QuotationNumber and QuotationRevisionN cache the banner "Dibuat dari
	// <number> Revisi <n>" at conversion.
	QuotationID        *int64
	QuotationNumber    string
	QuotationRevisionN int
	IsArchived         bool
	// CoverPhotoPath is the object-storage key of the Client Portal hero photo
	// (ADR-0047); nil = no cover. CoverUpdatedAt is when it last changed (the
	// `?v=` of the cover URL). Neither is part of the PATCH surface: the cover
	// has its own PUT/DELETE and does not bump the project's version.
	CoverPhotoPath *string
	CoverUpdatedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

// IsValidMilestoneStatus reports whether s is a known milestone status (the
// column is an enum).
func IsValidMilestoneStatus(s MilestoneStatus) bool {
	switch s {
	case MilestoneNotStarted, MilestoneInProgress, MilestoneCompleted, MilestoneBlocked, MilestoneCancelled:
		return true
	}
	return false
}

type ProjectMilestone struct {
	ID            int64
	ProjectID     int64
	SortOrder     int
	Name          string
	Status        MilestoneStatus
	TargetDate    time.Time
	CompletedDate *time.Time
	// DaysBeforeEvent is the Jadwal bawaan offset the target follows (H-14 = 14,
	// H+3 = -3; see MilestoneTarget). nil: the date is fixed — the step was
	// added by hand or its target was set by hand.
	DaysBeforeEvent *int
	// ForClient marks the step as a task for the couple: it shows on the Client
	// Portal and a client with portal.tasks.complete may mark it done (ADR-0047).
	ForClient bool
	// CompletedByClientID is the client who marked it done (0 = staff did, or it
	// is not completed). A primitive reference into `clients`, never a foreign key.
	CompletedByClientID int64

	// UpdatedAt is the row's version (TIMESTAMP(6), bumped by MySQL on any change):
	// the ETag / `updatedAt` of ADR-0036.
	UpdatedAt time.Time
}
