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

	// The two payment kinds: evidence first, then the row.
	simple := []struct {
		table string
		kind  domain.EvidenceRelatedKind
		count *int
	}{
		{"client_payments", domain.RelatedClientPayment, &report.ClientPayments},
		{"supplier_payments", domain.RelatedSupplierPayment, &report.SupplierPayments},
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

	// Bills of client-paid suppliers deleted longer ago than the retention (a
	// proof sent for one keeps its payment; the foreign key clears its link).
	billIDs, err := selectIDs(ctx, tx, `SELECT id FROM supplier_bill_terms WHERE deleted_at IS NOT NULL AND deleted_at <= ? ORDER BY id LIMIT ?`, cutoff, batch)
	if err != nil {
		return report, nil, err
	}
	if report.BillTerms, err = deleteByIDs(ctx, tx, "supplier_bill_terms", "id", billIDs); err != nil {
		return report, nil, err
	}

	// Suppliers take their steps, kendala, payments and bills (all of them, live
	// or not) and the files of each with them. Order: issues reference steps,
	// payments reference bills.
	suppliers, err := selectIDs(ctx, tx, `SELECT id FROM project_suppliers WHERE deleted_at IS NOT NULL AND deleted_at <= ? ORDER BY id LIMIT ?`, cutoff, batch)
	if err != nil {
		return report, nil, err
	}
	if len(suppliers) > 0 {
		in := inList(suppliers)
		args := utils.Int64Args(suppliers)
		milestones, err := selectIDs(ctx, tx, `SELECT id FROM vendor_milestones WHERE project_supplier_id IN `+in, args...)
		if err != nil {
			return report, nil, err
		}
		issues, err := selectIDs(ctx, tx, `SELECT id FROM vendor_issues WHERE project_supplier_id IN `+in, args...)
		if err != nil {
			return report, nil, err
		}
		payments, err := selectIDs(ctx, tx, `SELECT id FROM supplier_payments WHERE project_supplier_id IN `+in, args...)
		if err != nil {
			return report, nil, err
		}
		for _, e := range []struct {
			kind domain.EvidenceRelatedKind
			ids  []int64
		}{
			{domain.RelatedVendorMilestone, milestones}, {domain.RelatedIssue, issues},
			{domain.RelatedSupplierPayment, payments}, {domain.RelatedProjectSupplier, suppliers},
		} {
			k, n, err := deleteEvidenceOf(ctx, tx, e.kind, e.ids)
			if err != nil {
				return report, nil, err
			}
			keys, report.Evidence = append(keys, k...), report.Evidence+n
		}
		if report.SupplierIssues, err = deleteByIDs(ctx, tx, "vendor_issues", "id", issues); err != nil {
			return report, nil, err
		}
		if report.SupplierOwnedPayments, err = deleteByIDs(ctx, tx, "supplier_payments", "id", payments); err != nil {
			return report, nil, err
		}
		if report.SupplierMilestones, err = deleteByIDs(ctx, tx, "vendor_milestones", "id", milestones); err != nil {
			return report, nil, err
		}
		bills, err := deleteByIDs(ctx, tx, "supplier_bill_terms", "project_supplier_id", suppliers)
		if err != nil {
			return report, nil, err
		}
		report.BillTerms += bills
		if report.Suppliers, err = deleteByIDs(ctx, tx, "project_suppliers", "id", suppliers); err != nil {
			return report, nil, err
		}
	}
	// Rincian lines: a package takes its children with it (they share its
	// deleted_at; the foreign key cascades the rest). The contract cache never
	// counted them, so it does not move.
	lines, err := selectIDs(ctx, tx, `SELECT id FROM project_lines WHERE deleted_at IS NOT NULL AND deleted_at <= ? ORDER BY id LIMIT ?`, cutoff, batch)
	if err != nil {
		return report, nil, err
	}
	if len(lines) > 0 {
		args := utils.Int64Args(lines)
		res, err := tx.ExecContext(ctx, `DELETE FROM project_lines WHERE parent_line_id IN `+inList(lines), args...)
		if err != nil {
			return report, nil, err
		}
		children, _ := res.RowsAffected()
		n, err := deleteByIDs(ctx, tx, "project_lines", "id", lines)
		if err != nil {
			return report, nil, err
		}
		report.Lines = n + int(children)
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
