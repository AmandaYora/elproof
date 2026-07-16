package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

// MySQLWebhookLogRepository gives webhook processing idempotency — unique on
// (provider, event_id), so the same gateway event is never applied twice.
type MySQLWebhookLogRepository struct {
	db *sql.DB
}

func NewMySQLWebhookLogRepository(db *sql.DB) *MySQLWebhookLogRepository {
	return &MySQLWebhookLogRepository{db: db}
}

// MarkSeen records this event as processed, returning (false, nil) if it was
// already seen before (duplicate key) instead of an error — the caller uses
// this to decide whether to skip re-applying the event.
func (r *MySQLWebhookLogRepository) MarkSeen(ctx context.Context, provider, eventID string) (firstTime bool, err error) {
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO payment_webhook_events (provider, event_id) VALUES (?, ?)`,
		provider, eventID,
	)
	if err == nil {
		return true, nil
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return false, nil
	}
	return false, err
}
