package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

// MySQLPurgeRepository removes what has waited in the "deleted" state for longer
// than the retention (ADR-0034): payments and vendor engagements of this module,
// with the evidence rows that hang off them. It owns no table of its own; it works
// on the tables the other repositories of this module own, inside one module.
type MySQLPurgeRepository struct {
	db *sql.DB
}

func NewMySQLPurgeRepository(db *sql.DB) *MySQLPurgeRepository {
	return &MySQLPurgeRepository{db: db}
}

// purgeLockName is the MySQL advisory lock that lets only one instance purge at
// a time (several API instances may share one database).
const purgeLockName = "elproof_projects_purge"

// TryLock takes the purge lock without waiting. ok is false when another
// instance holds it; release must be called when ok is true. The lock lives on
// a dedicated connection and disappears with it, so a crashed instance never
// leaves it behind.
func (r *MySQLPurgeRepository) TryLock(ctx context.Context) (release func(), ok bool, err error) {
	c, err := r.db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var got sql.NullInt64
	if err := c.QueryRowContext(ctx, `SELECT GET_LOCK(?, 0)`, purgeLockName).Scan(&got); err != nil {
		c.Close()
		return nil, false, err
	}
	if !got.Valid || got.Int64 != 1 {
		c.Close()
		return nil, false, nil
	}
	return func() {
		_, _ = c.ExecContext(context.Background(), `SELECT RELEASE_LOCK(?)`, purgeLockName)
		c.Close()
	}, true, nil
}

func selectIDs(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func inList(ids []int64) string { return `(` + utils.Placeholders(len(ids)) + `)` }

// deleteEvidenceOf removes the evidence rows attached to (kind, ids) and returns
// the stored object keys they pointed at.
func deleteEvidenceOf(ctx context.Context, tx *sql.Tx, kind domain.EvidenceRelatedKind, ids []int64) (keys []string, rowsDeleted int, err error) {
	if len(ids) == 0 {
		return nil, 0, nil
	}
	args := append([]any{string(kind)}, utils.Int64Args(ids)...)
	rows, err := tx.QueryContext(ctx, `SELECT storage_path FROM evidence WHERE related_kind = ? AND related_id IN `+inList(ids), args...)
	if err != nil {
		return nil, 0, err
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return nil, 0, err
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, err
	}
	rows.Close()
	res, err := tx.ExecContext(ctx, `DELETE FROM evidence WHERE related_kind = ? AND related_id IN `+inList(ids), args...)
	if err != nil {
		return nil, 0, err
	}
	n, _ := res.RowsAffected()
	return keys, int(n), nil
}

func deleteByIDs(ctx context.Context, tx *sql.Tx, table, column string, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE `+column+` IN `+inList(ids), utils.Int64Args(ids)...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// PurgeBatch hard-deletes up to `batch` expired rows of each kind (those with
// deleted_at <= cutoff) in ONE transaction and reports what went, with the
// stored object keys of the evidence rows removed so the caller can delete the
// objects once the transaction has committed (a storage failure then leaves an
// orphan object, never a row pointing at a missing file). It is idempotent: a
// second run finds nothing left to do.
func (r *MySQLPurgeRepository) PurgeBatch(ctx context.Context, cutoff time.Time, batch int) (domain.PurgeReport, []string, error) {
	var report domain.PurgeReport
	var keys []string
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return report, nil, err
	}
	defer tx.Rollback()

	// The three payment kinds: evidence first, then the row.
	simple := []struct {
		table string
		kind  domain.EvidenceRelatedKind
		count *int
	}{
		{"client_payments", domain.RelatedClientPayment, &report.ClientPayments},
		{"venue_payments", domain.RelatedVenuePayment, &report.VenuePayments},
		{"vendor_payments", domain.RelatedPayment, &report.VendorPayments},
	}
	for _, t := range simple {
		ids, err := selectIDs(ctx, tx, `SELECT id FROM `+t.table+` WHERE deleted_at IS NOT NULL AND deleted_at <= ? ORDER BY id LIMIT ?`, cutoff, batch)
		if err != nil {
			return report, nil, err
		}
		k, n, err := deleteEvidenceOf(ctx, tx, t.kind, ids)
		if err != nil {
			return report, nil, err
		}
		keys, report.Evidence = append(keys, k...), report.Evidence+n
		if *t.count, err = deleteByIDs(ctx, tx, t.table, "id", ids); err != nil {
			return report, nil, err
		}
	}

	// Engagements take their milestones, kendala and payments (all of them, live
	// or not) and the files of each with them. Order: issues reference milestones.
	engagements, err := selectIDs(ctx, tx, `SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL AND deleted_at <= ? ORDER BY id LIMIT ?`, cutoff, batch)
	if err != nil {
		return report, nil, err
	}
	if len(engagements) > 0 {
		in := inList(engagements)
		args := utils.Int64Args(engagements)
		milestones, err := selectIDs(ctx, tx, `SELECT id FROM vendor_milestones WHERE project_vendor_id IN `+in, args...)
		if err != nil {
			return report, nil, err
		}
		issues, err := selectIDs(ctx, tx, `SELECT id FROM vendor_issues WHERE project_vendor_id IN `+in, args...)
		if err != nil {
			return report, nil, err
		}
		payments, err := selectIDs(ctx, tx, `SELECT id FROM vendor_payments WHERE project_vendor_id IN `+in, args...)
		if err != nil {
			return report, nil, err
		}
		for _, e := range []struct {
			kind domain.EvidenceRelatedKind
			ids  []int64
		}{
			{domain.RelatedVendorMilestone, milestones}, {domain.RelatedIssue, issues},
			{domain.RelatedPayment, payments}, {domain.RelatedProjectVendor, engagements},
		} {
			k, n, err := deleteEvidenceOf(ctx, tx, e.kind, e.ids)
			if err != nil {
				return report, nil, err
			}
			keys, report.Evidence = append(keys, k...), report.Evidence+n
		}
		if report.EngagementIssues, err = deleteByIDs(ctx, tx, "vendor_issues", "id", issues); err != nil {
			return report, nil, err
		}
		if report.EngagementPayments, err = deleteByIDs(ctx, tx, "vendor_payments", "id", payments); err != nil {
			return report, nil, err
		}
		if report.EngagementMilestones, err = deleteByIDs(ctx, tx, "vendor_milestones", "id", milestones); err != nil {
			return report, nil, err
		}
		if report.Engagements, err = deleteByIDs(ctx, tx, "project_vendors", "id", engagements); err != nil {
			return report, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.PurgeReport{}, nil, err
	}
	return report, dedupe(keys), nil
}

func dedupe(keys []string) []string {
	seen := make(map[string]bool, len(keys))
	out := keys[:0]
	for _, k := range keys {
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}
