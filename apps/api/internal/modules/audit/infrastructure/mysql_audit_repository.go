// Package infrastructure persists audit entries (table audit_log).
package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/audit/domain"
	"elproof/internal/shared/pagination"
)

type MySQLAuditRepository struct {
	db *sql.DB
}

func NewMySQLAuditRepository(db *sql.DB) *MySQLAuditRepository {
	return &MySQLAuditRepository{db: db}
}

func (r *MySQLAuditRepository) Insert(ctx context.Context, e *domain.Entry) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (occurred_at, actor_type, actor_id, actor_role, tenant_id, action, method, path, status_code, ip, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.OccurredAt, e.ActorType, e.ActorID, e.ActorRole, e.TenantID, e.Action, e.Method, e.Path, e.StatusCode, e.IP, e.UserAgent,
	)
	if err != nil {
		return err
	}
	e.ID, err = res.LastInsertId()
	return err
}

func (r *MySQLAuditRepository) ListPaginated(ctx context.Context, params pagination.Params, actorID string) ([]domain.Entry, int64, error) {
	where, args := "", []interface{}{}
	if actorID != "" {
		where, args = " WHERE actor_id = ?", append(args, actorID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, actor_type, actor_id, actor_role, action, method, path, status_code, ip, user_agent
		 FROM audit_log`+where+` ORDER BY occurred_at DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorType, &e.ActorID, &e.ActorRole, &e.Action, &e.Method,
			&e.Path, &e.StatusCode, &e.IP, &e.UserAgent); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
