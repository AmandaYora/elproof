package infrastructure

import (
	"context"
	"database/sql"

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
