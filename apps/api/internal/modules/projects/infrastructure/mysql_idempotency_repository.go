package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"

	"elproof/internal/modules/projects/application"
)

// MySQLIdempotencyRepository stores Idempotency-Key reservations and responses.
// Time is read from the database (NOW()) everywhere, so retention and staleness
// never depend on the clock or time zone of the API process.
type MySQLIdempotencyRepository struct {
	db *sql.DB
}

func NewMySQLIdempotencyRepository(db *sql.DB) *MySQLIdempotencyRepository {
	return &MySQLIdempotencyRepository{db: db}
}

// Reserve claims the key. A request still "running" after 5 minutes died (the
// server's own write timeout is 30 seconds): the record comes back Stale and
// the key can be re-claimed with TakeOver.
func (r *MySQLIdempotencyRepository) Reserve(ctx context.Context, tenantID, staffID int64, route, key, requestHash string) (*application.IdempotencyRecord, error) {
	// Retention is 24 hours. Expired rows are removed here, opportunistically:
	// first any expired row of this very key (so it can never be replayed late),
	// then a bounded batch of others.
	_, _ = r.db.ExecContext(ctx,
		`DELETE FROM idempotency_keys WHERE tenant_id = ? AND idempotency_key = ? AND route = ? AND created_at < NOW() - INTERVAL 24 HOUR`,
		tenantID, key, route)
	_, _ = r.db.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE created_at < NOW() - INTERVAL 24 HOUR LIMIT 100`)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO idempotency_keys (tenant_id, staff_id, route, idempotency_key, request_hash) VALUES (?, ?, ?, ?, ?)`,
		tenantID, staffID, route, key, requestHash)
	if err == nil {
		return nil, nil
	}
	var me *mysql.MySQLError
	if !errors.As(err, &me) || me.Number != 1062 {
		return nil, err
	}
	var rec application.IdempotencyRecord
	var status sql.NullInt64
	var body sql.NullString
	err = r.db.QueryRowContext(ctx,
		`SELECT staff_id, request_hash, status_code, response_body, created_at < NOW() - INTERVAL 5 MINUTE
		 FROM idempotency_keys WHERE tenant_id = ? AND idempotency_key = ? AND route = ?`,
		tenantID, key, route).Scan(&rec.StaffID, &rec.RequestHash, &status, &body, &rec.Stale)
	if err == sql.ErrNoRows {
		// Removed between the insert and the read: claim it again.
		return r.Reserve(ctx, tenantID, staffID, route, key, requestHash)
	}
	if err != nil {
		return nil, err
	}
	if status.Valid {
		code := int(status.Int64)
		rec.StatusCode = &code
		rec.Body = []byte(body.String)
	}
	return &rec, nil
}

func (r *MySQLIdempotencyRepository) TakeOver(ctx context.Context, tenantID, staffID int64, route, key, requestHash string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE idempotency_keys SET staff_id = ?, request_hash = ?, created_at = NOW()
		 WHERE tenant_id = ? AND idempotency_key = ? AND route = ? AND status_code IS NULL AND created_at < NOW() - INTERVAL 5 MINUTE`,
		staffID, requestHash, tenantID, key, route)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *MySQLIdempotencyRepository) Complete(ctx context.Context, tenantID int64, route, key string, statusCode int, body []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE idempotency_keys SET status_code = ?, response_body = ? WHERE tenant_id = ? AND idempotency_key = ? AND route = ?`,
		statusCode, string(body), tenantID, key, route)
	return err
}

func (r *MySQLIdempotencyRepository) Release(ctx context.Context, tenantID int64, route, key string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM idempotency_keys WHERE tenant_id = ? AND idempotency_key = ? AND route = ? AND status_code IS NULL`,
		tenantID, key, route)
	return err
}
