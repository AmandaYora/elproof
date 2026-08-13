package domain

import "time"

type ActivityType string

const (
	ActivityProjectCreated       ActivityType = "project_created"
	ActivityProjectUpdated       ActivityType = "project_updated"
	ActivityProjectStatusChanged ActivityType = "project_status_changed"
	ActivityVendorAdded          ActivityType = "vendor_added"
	ActivityVendorStatusChanged  ActivityType = "vendor_status_changed"
	ActivityMilestoneUpdated     ActivityType = "milestone_updated"
	ActivityPaymentRecorded      ActivityType = "payment_recorded"
	// ActivityPaymentUpdated/ActivityPaymentDeleted are shared generically
	// across all three payment types (vendor/client/venue), same convention
	// as ActivityPaymentRecorded -- distinguished by entityType/entityLabel/
	// description at the call site, not by separate per-type constants.
	ActivityPaymentUpdated   ActivityType = "payment_updated"
	ActivityPaymentDeleted   ActivityType = "payment_deleted"
	ActivityEvidenceUploaded ActivityType = "evidence_uploaded"
	ActivityIssueCreated     ActivityType = "issue_created"
	ActivityIssueUpdated     ActivityType = "issue_updated"
)

// ActivityLogEntry is append-only — see ADR-0007. Every mutating use case in
// this module appends one row; nothing ever updates or deletes an entry.
type ActivityLogEntry struct {
	ID           int64
	ProjectID    *int64
	Type         ActivityType
	ActorStaffID int64
	EntityType   string
	EntityID     string
	EntityLabel  string
	Description  string
	CreatedAt    time.Time
}
