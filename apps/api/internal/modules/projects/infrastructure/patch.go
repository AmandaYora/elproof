package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Partial PATCH writes (ADR-0036). The service tells the repository WHICH fields
// it changed; the repository writes exactly those columns from the merged row, so
// two people who edited different fields of the same item without If-Match cannot
// overwrite each other with the stale values of the fields they did not touch.

// column is how one domain field maps to storage: the column(s) it owns and how
// to read their values from the merged row.
type column[T any] struct {
	names  []string
	values func(*T) []any
}

func col[T any](name string, value func(*T) any) column[T] {
	return column[T]{names: []string{name}, values: func(r *T) []any { return []any{value(r)} }}
}

// patchRow runs ONE statement:
//
//	UPDATE table SET <changed columns> WHERE <where> [AND updated_at = ?]
//
// With expected set it is a compare-and-set: it matches only while the row still
// has the version the caller read, so it is safe under concurrency without a
// read-then-write. matched is true when the row was written (or, without
// `expected`, whenever the statement ran); false means the version no longer
// matches or the row is gone, and the caller finds out which with one read.
// MySQL counts only rows whose values really changed, so a statement that matched
// but changed nothing also reports false: the caller's read tells it apart (the
// version is still `expected`).
func patchRow[T any](ctx context.Context, db dbtx, table string, spec map[string]column[T], row *T, changed []string,
	where string, whereArgs []any, expected *time.Time) (matched bool, err error) {
	if len(changed) == 0 {
		return true, nil
	}
	var sets []string
	var args []any
	for _, key := range changed {
		c, ok := spec[key]
		if !ok {
			return false, fmt.Errorf("infrastructure: no column mapped for patched field %q of %s", key, table)
		}
		vals := c.values(row)
		for i, name := range c.names {
			sets = append(sets, name+" = ?")
			args = append(args, vals[i])
		}
	}
	query := `UPDATE ` + table + ` SET ` + strings.Join(sets, ", ") + ` WHERE ` + where
	args = append(args, whereArgs...)
	if expected != nil {
		query += ` AND updated_at = ?`
		args = append(args, *expected)
	}
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0 || expected == nil, nil
}

// readVersion loads a freshly inserted row's `updatedAt` (its ADR-0036
// version, set by MySQL) into dst, on the same executor as the INSERT. A
// create answer that carried the zero time made an edit straight from it a 409.
func readVersion(ctx context.Context, db dbtx, table string, id int64, dst *time.Time) error {
	return db.QueryRowContext(ctx, `SELECT updated_at FROM `+table+` WHERE id = ?`, id).Scan(dst)
}
