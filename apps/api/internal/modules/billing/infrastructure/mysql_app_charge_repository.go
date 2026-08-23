package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/billing/domain"
)

// MySQLAppChargeRepository backs the subscription_app_charges ledger (A17)
// — a subscription-purchase intent per external App, snapshotting the
// plan's price/duration at write time.
type MySQLAppChargeRepository struct {
	db *sql.DB
}

func NewMySQLAppChargeRepository(db *sql.DB) *MySQLAppChargeRepository {
	return &MySQLAppChargeRepository{db: db}
}

const appChargeColumns = `order_ref, app_id, plan_id, plan_name, plan_price, plan_duration_months,
	status, subscription_until, created_at, paid_at, resolved_at`

func scanAppCharge(scan func(dest ...interface{}) error) (*domain.AppCharge, error) {
	var c domain.AppCharge
	var status string
	var subscriptionUntil, paidAt, resolvedAt sql.NullTime
	if err := scan(&c.OrderRef, &c.AppID, &c.PlanID, &c.PlanName, &c.PlanPrice, &c.PlanDurationMonths,
		&status, &subscriptionUntil, &c.CreatedAt, &paidAt, &resolvedAt); err != nil {
		return nil, err
	}
	c.Status = domain.AppChargeStatus(status)
	if subscriptionUntil.Valid {
		c.SubscriptionUntil = &subscriptionUntil.Time
	}
	if paidAt.Valid {
		c.PaidAt = &paidAt.Time
	}
	if resolvedAt.Valid {
		c.ResolvedAt = &resolvedAt.Time
	}
	return &c, nil
}

// Create writes the intent row BEFORE payment.Client is ever called (A20) —
// so a process death right after this leaves a ghost row the reconciler can
// find and resolve (CheckStatusForApp answers 404 for it) instead of a
// charge existing at the gateway with no local trace at all.
func (r *MySQLAppChargeRepository) Create(ctx context.Context, c *domain.AppCharge) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO subscription_app_charges (order_ref, app_id, plan_id, plan_name, plan_price, plan_duration_months, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.OrderRef, c.AppID, c.PlanID, c.PlanName, c.PlanPrice, c.PlanDurationMonths, domain.AppChargePending,
	)
	return err
}

// Delete removes the intent row — called when the payment.Client call right
// after Create fails, so the orderRef is free to be retried (A20).
func (r *MySQLAppChargeRepository) Delete(ctx context.Context, orderRef string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM subscription_app_charges WHERE order_ref = ?`, orderRef)
	return err
}

// FindUnresolved returns pending intents created before olderThan (so a
// brand-new charge isn't checked before it's had a realistic chance to be
// paid), oldest first — mirrors payment's own
// MySQLDispatchRepository.ListUnresolvedForSweep.
func (r *MySQLAppChargeRepository) FindUnresolved(ctx context.Context, olderThan time.Time, limit int) ([]domain.AppCharge, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+appChargeColumns+` FROM subscription_app_charges
		 WHERE status = ? AND resolved_at IS NULL AND created_at < ?
		 ORDER BY created_at ASC LIMIT ?`,
		domain.AppChargePending, olderThan, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.AppCharge
	for rows.Next() {
		c, err := scanAppCharge(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

// MarkPaidIfUnresolved is a SINGLE atomic UPDATE that both claims and writes
// the terminal result (A25) — there is no separate claim-then-write step.
// `until` must already be computed by the caller (SubscriptionReconciler.
// ComputeUntil, reading CurrentUntilForApp first) since it's an INPUT to
// this write, not something this method derives. Returns whether this call
// won the race (RowsAffected == 1); false means another concurrent
// reconciler tick already resolved it.
func (r *MySQLAppChargeRepository) MarkPaidIfUnresolved(ctx context.Context, orderRef string, until time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE subscription_app_charges
		 SET status = ?, paid_at = ?, subscription_until = ?, resolved_at = ?
		 WHERE order_ref = ? AND resolved_at IS NULL`,
		domain.AppChargePaid, time.Now(), until, time.Now(), orderRef,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// MarkResolvedIfUnresolved is the non-paid counterpart of
// MarkPaidIfUnresolved — same single-UPDATE, single-winner shape (A25).
func (r *MySQLAppChargeRepository) MarkResolvedIfUnresolved(ctx context.Context, orderRef string, status domain.AppChargeStatus) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE subscription_app_charges SET status = ?, resolved_at = ? WHERE order_ref = ? AND resolved_at IS NULL`,
		status, time.Now(), orderRef,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// CurrentUntilForApp returns the latest subscription_until among this App's
// paid charges, or nil if it has never paid — the basis ComputeUntil uses to
// stack a renewal on top of remaining time rather than today (so paying
// early never costs the customer time already owned).
func (r *MySQLAppChargeRepository) CurrentUntilForApp(ctx context.Context, appID string) (*time.Time, error) {
	var until sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(subscription_until) FROM subscription_app_charges WHERE app_id = ? AND status = ?`,
		appID, domain.AppChargePaid,
	).Scan(&until)
	if err != nil {
		return nil, err
	}
	if !until.Valid {
		return nil, nil
	}
	return &until.Time, nil
}

// LatestPaidPerApp backs GET /api/v1/app-subscriptions (T13/D7) — one row
// per App with its most recent paid charge. An App that has never paid
// through this endpoint simply doesn't appear (D11 — the ledger is
// intentionally empty until jwswedding switches over); the handler renders
// that as "—", not an error.
func (r *MySQLAppChargeRepository) LatestPaidPerApp(ctx context.Context) ([]domain.AppSubscriptionSummary, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.app_id, c.plan_name, c.subscription_until, c.status
		 FROM subscription_app_charges c
		 JOIN (
		   SELECT app_id, MAX(subscription_until) AS max_until
		   FROM subscription_app_charges
		   WHERE status = ?
		   GROUP BY app_id
		 ) latest ON latest.app_id = c.app_id AND latest.max_until = c.subscription_until
		 WHERE c.status = ?
		 ORDER BY c.app_id`,
		domain.AppChargePaid, domain.AppChargePaid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]domain.AppSubscriptionSummary, 0)
	for rows.Next() {
		var s domain.AppSubscriptionSummary
		var status string
		var until sql.NullTime
		if err := rows.Scan(&s.AppID, &s.PlanName, &until, &status); err != nil {
			return nil, err
		}
		if until.Valid {
			s.SubscriptionUntil = until.Time
		}
		s.Status = domain.AppChargeStatus(status)
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}
