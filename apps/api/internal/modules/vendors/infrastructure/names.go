package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/utils"
)

// namesByIDs reads `name` for the given ids from one of this module's own
// tables, always scoped to the tenant. `table` is a package-internal constant,
// never user input.
func namesByIDs(ctx context.Context, db *sql.DB, table string, tenantID int64, ids []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := append([]interface{}{tenantID}, utils.Int64Args(ids)...)
	rows, err := db.QueryContext(ctx, `SELECT id, name FROM `+table+` WHERE tenant_id = ? AND id IN (`+utils.Placeholders(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

// qualifyColumns prefixes every column of a column list with a table alias, for
// the queries that join (the list's category name).
func qualifyColumns(alias, columns string) string {
	parts := strings.Split(columns, ",")
	for i, c := range parts {
		parts[i] = alias + "." + strings.TrimSpace(c)
	}
	return strings.Join(parts, ", ")
}

// bulkSetActive flips is_active of the given ids of one tenant in ONE
// transaction: the matching rows are locked and read, domain.PlanBulk decides
// what changes, and a single UPDATE applies it. `table` is a constant of this
// package. Ids of another tenant (or that do not exist) are never touched and
// come back as notFound, indistinguishable from each other.
func bulkSetActive(ctx context.Context, db *sql.DB, table string, tenantID int64, ids []int64, target bool) (domain.BulkResult, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.BulkResult{}, err
	}
	defer tx.Rollback()

	args := append([]interface{}{tenantID}, utils.Int64Args(ids)...)
	rows, err := tx.QueryContext(ctx,
		`SELECT id, is_active FROM `+table+` WHERE tenant_id = ? AND id IN (`+utils.Placeholders(len(ids))+`) FOR UPDATE`, args...)
	if err != nil {
		return domain.BulkResult{}, err
	}
	current := make(map[int64]bool, len(ids))
	for rows.Next() {
		var id int64
		var active bool
		if err := rows.Scan(&id, &active); err != nil {
			rows.Close()
			return domain.BulkResult{}, err
		}
		current[id] = active
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return domain.BulkResult{}, err
	}
	rows.Close()

	toUpdate, unchanged, notFound := domain.PlanBulk(ids, current, target)
	if len(toUpdate) > 0 {
		upd := append([]interface{}{target, tenantID}, utils.Int64Args(toUpdate)...)
		if _, err := tx.ExecContext(ctx,
			`UPDATE `+table+` SET is_active = ? WHERE tenant_id = ? AND id IN (`+utils.Placeholders(len(toUpdate))+`)`, upd...); err != nil {
			return domain.BulkResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.BulkResult{}, err
	}
	return domain.BulkResult{Updated: len(toUpdate), Unchanged: unchanged, NotFound: notFound}, nil
}
