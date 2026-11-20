package domain

import "time"

type ActivityType string

const (
	ActivityProjectCreated       ActivityType = "project_created"
	ActivityProjectUpdated       ActivityType = "project_updated"
	ActivityProjectStatusChanged ActivityType = "project_status_changed"
	ActivityVendorAdded          ActivityType = "vendor_added"
	ActivityVendorStatusChanged  ActivityType = "vendor_status_changed"
	ActivityVendorUpdated        ActivityType = "vendor_updated"
	// Events ("Acara"): added, changed (incl. main event, order), removed.
	ActivityEventAdded       ActivityType = "event_added"
	ActivityEventUpdated     ActivityType = "event_updated"
	ActivityEventDeleted     ActivityType = "event_deleted"
	ActivityMilestoneUpdated ActivityType = "milestone_updated"
	ActivityPaymentRecorded  ActivityType = "payment_recorded"
	// ActivityPaymentUpdated/ActivityPaymentDeleted are shared generically
	// across client and supplier payments, same convention
	// as ActivityPaymentRecorded -- distinguished by entityType/entityLabel/
	// description at the call site, not by separate per-type constants.
	ActivityPaymentUpdated   ActivityType = "payment_updated"
	ActivityPaymentDeleted   ActivityType = "payment_deleted"
	ActivityEvidenceUploaded ActivityType = "evidence_uploaded"
	ActivityIssueCreated     ActivityType = "issue_created"
	ActivityIssueUpdated     ActivityType = "issue_updated"
	// ActivityRestored is written when something deleted is brought back
	// ("Urungkan"); the delete itself is logged as the existing *_deleted /
	// vendor_status_changed types (ADR-0034).
	ActivityRestored ActivityType = "item_restored"
	// ActivityEvidenceDeleted: one proof file was removed.
	ActivityEvidenceDeleted ActivityType = "evidence_deleted"
	// ActivityVendorDeleted: a vendor or venue of the project was deleted (it can
	// be restored).
	ActivityVendorDeleted ActivityType = "vendor_deleted"
	// Rincian lines (paket WO Tahap 3): every change records the old and the
	// new value in its description (PLAN §3.2 "Riwayat perubahan").
	ActivityLineAdded   ActivityType = "line_added"
	ActivityLineUpdated ActivityType = "line_updated"
	ActivityLineDeleted ActivityType = "line_deleted"
)

// ActivityFilter narrows a project's activity to one entity; an empty field
// matches everything.
type ActivityFilter struct {
	EntityType string
	EntityID   string
}

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
