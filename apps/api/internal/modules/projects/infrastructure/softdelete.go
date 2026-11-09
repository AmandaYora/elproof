package infrastructure

import (
	"context"
	"database/sql"
	"time"
)

// Soft delete (ADR-0034): a payment or a vendor engagement is "deleted" by
// stamping deleted_at; every read of a live row filters `deleted_at IS NULL`.
// These helpers are the three writes every table of this module shares. The
// table name is always a constant of this package, never request input.

// softDeleteRow stamps a live row of the project; false when there was none.
func softDeleteRow(ctx context.Context, db *sql.DB, table string, projectID, id int64, at time.Time) (bool, error) {
	res, err := conn(ctx, db).ExecContext(ctx,
		`UPDATE `+table+` SET deleted_at = ? WHERE project_id = ? AND id = ? AND deleted_at IS NULL`, at, projectID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// restoreRow clears the stamp of a row deleted after notBefore (the purge
// cutoff: rows deleted at or before it count as already purged, the exact
// complement of the purge's `deleted_at <= cutoff`); false when there was none.
func restoreRow(ctx context.Context, db *sql.DB, table string, projectID, id int64, notBefore time.Time) (bool, error) {
	res, err := conn(ctx, db).ExecContext(ctx,
		`UPDATE `+table+` SET deleted_at = NULL WHERE project_id = ? AND id = ? AND deleted_at IS NOT NULL AND deleted_at > ?`,
		projectID, id, notBefore)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// deletedStamp turns a nullable column into the *time.Time the domain carries.
func deletedStamp(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

// SQL fragments saying "this row is visible" for the tables a deleted engagement
// takes its children down with. Written against unaliased table names.
const (
	// liveEngagementIDs: the engagements that are not deleted.
	deletedEngagementIDs = `SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL`

	// A vendor payment is hidden when it is deleted itself or its engagement is.
	liveVendorPaymentWhere = `deleted_at IS NULL AND project_vendor_id NOT IN (` + deletedEngagementIDs + `)`

	// A kendala is hidden when its vendor engagement is deleted (a kendala with
	// no vendor is never hidden that way).
	liveIssueWhere = `(project_vendor_id IS NULL OR project_vendor_id NOT IN (` + deletedEngagementIDs + `))`

	// evidenceLiveWhere hides the proof files of everything deleted: a deleted
	// payment's (or engagement's, or its milestones'/kendala's) files are not
	// listed, opened or linked by anyone until it is restored.
	evidenceLiveWhere = `
		AND NOT (related_kind = 'clientPayment' AND related_id IN (SELECT id FROM client_payments WHERE deleted_at IS NOT NULL))
		AND NOT (related_kind = 'venuePayment' AND related_id IN (SELECT id FROM venue_payments WHERE deleted_at IS NOT NULL))
		AND NOT (related_kind = 'payment' AND related_id IN (
			SELECT id FROM vendor_payments WHERE deleted_at IS NOT NULL OR project_vendor_id IN (` + deletedEngagementIDs + `)))
		AND NOT (related_kind = 'projectVendor' AND related_id IN (` + deletedEngagementIDs + `))
		AND NOT (related_kind = 'vendorMilestone' AND related_id IN (
			SELECT vm.id FROM vendor_milestones vm JOIN project_vendors pv ON pv.id = vm.project_vendor_id WHERE pv.deleted_at IS NOT NULL))
		AND NOT (related_kind = 'issue' AND related_id IN (
			SELECT id FROM vendor_issues WHERE project_vendor_id IN (` + deletedEngagementIDs + `)))`
)
