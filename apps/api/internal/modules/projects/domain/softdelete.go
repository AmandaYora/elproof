package domain

import "time"

// SoftDeleteRetention is how long a deleted payment, vendor engagement or client
// can still be restored ("Urungkan"). After it the purge routine removes the row
// for good, together with its evidence rows and stored files (ADR-0034).
const SoftDeleteRetention = 24 * time.Hour

// RestorableUntil is the instant after which a row deleted at deletedAt can no
// longer be restored. It is what DELETE answers as `restorableUntil`.
func RestorableUntil(deletedAt time.Time) time.Time { return deletedAt.Add(SoftDeleteRetention) }

// CanRestore reports whether a row deleted at deletedAt may still be restored at
// now. A row past its retention that the purge has not reached yet counts as
// already purged: restoring it is a 404, whatever the purge schedule is.
func CanRestore(deletedAt, now time.Time) bool { return now.Before(RestorableUntil(deletedAt)) }

// PurgeCutoff is the deleted_at value at or before which a row is due for the
// purge at now.
func PurgeCutoff(now time.Time) time.Time { return now.Add(-SoftDeleteRetention) }

// IsPurgeDue reports whether the purge removes a row deleted at deletedAt when
// it runs at now. It is exactly the complement of CanRestore, so a row is always
// in exactly one of the two states: restorable or due for purge.
func IsPurgeDue(deletedAt, now time.Time) bool { return !CanRestore(deletedAt, now) }

// PurgeReport is what one purge run removed.
type PurgeReport struct {
	ClientPayments int
	VendorPayments int
	VenuePayments  int
	Engagements    int
	// Milestones, Issues and Payments of purged engagements.
	EngagementMilestones int
	EngagementIssues     int
	EngagementPayments   int
	Evidence             int
	// Objects is how many stored files were deleted from object storage.
	Objects int
}

// Total is how many rows (not counting stored files) the run removed.
func (r PurgeReport) Total() int {
	return r.ClientPayments + r.VendorPayments + r.VenuePayments + r.Engagements +
		r.EngagementMilestones + r.EngagementIssues + r.EngagementPayments + r.Evidence
}

// Add folds another report into this one.
func (r *PurgeReport) Add(o PurgeReport) {
	r.ClientPayments += o.ClientPayments
	r.VendorPayments += o.VendorPayments
	r.VenuePayments += o.VenuePayments
	r.Engagements += o.Engagements
	r.EngagementMilestones += o.EngagementMilestones
	r.EngagementIssues += o.EngagementIssues
	r.EngagementPayments += o.EngagementPayments
	r.Evidence += o.Evidence
	r.Objects += o.Objects
}
