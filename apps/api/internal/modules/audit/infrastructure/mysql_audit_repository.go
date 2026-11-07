// Package infrastructure persists audit entries (table audit_log).
package infrastructure

import (
	"context"
	"database/sql"
	"elproof/internal/shared/utils"
	"strings"

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

func auditWhere(f domain.Filter) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if f.ActorID != "" {
		conds, args = append(conds, "actor_id = ?"), append(args, f.ActorID)
	}
	if f.ActorType != "" {
		conds, args = append(conds, "actor_type = ?"), append(args, f.ActorType)
	}
	if f.TenantID != nil {
		conds, args = append(conds, "tenant_id = ?"), append(args, *f.TenantID)
	}
	switch f.Outcome {
	case "ok":
		conds = append(conds, "status_code BETWEEN 200 AND 299")
	case "failed":
		conds = append(conds, "(status_code < 200 OR status_code > 299)")
	}
	if f.Query != "" {
		like := "%" + utils.EscapeLike(f.Query) + "%"
		conds, args = append(conds, "(action LIKE ? OR path LIKE ?)"), append(args, like, like)
	}
	if !f.From.IsZero() {
		conds, args = append(conds, "occurred_at >= ?"), append(args, f.From)
	}
	if !f.To.IsZero() {
		conds, args = append(conds, "occurred_at < ?"), append(args, f.To)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

const auditColumns = `id, occurred_at, actor_type, actor_id, actor_role, tenant_id, action, method, path, status_code, ip, user_agent`

func scanAuditRows(rows *sql.Rows) ([]domain.Entry, error) {
	var out []domain.Entry
	for rows.Next() {
		var e domain.Entry
		var tenant sql.NullInt64
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorType, &e.ActorID, &e.ActorRole, &tenant, &e.Action, &e.Method,
			&e.Path, &e.StatusCode, &e.IP, &e.UserAgent); err != nil {
			return nil, err
		}
		if tenant.Valid {
			e.TenantID = &tenant.Int64
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *MySQLAuditRepository) ListPaginated(ctx context.Context, params pagination.Params, f domain.Filter) ([]domain.Entry, int64, error) {
	where, args := auditWhere(f)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+auditColumns+` FROM audit_log`+where+` ORDER BY occurred_at DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out, err := scanAuditRows(rows)
	return out, total, err
}

func (r *MySQLAuditRepository) ListAll(ctx context.Context, f domain.Filter, limit int) ([]domain.Entry, error) {
	where, args := auditWhere(f)
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+auditColumns+` FROM audit_log`+where+` ORDER BY occurred_at DESC, id DESC LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditRows(rows)
}
