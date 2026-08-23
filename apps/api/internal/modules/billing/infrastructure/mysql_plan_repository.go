package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type MySQLPlanRepository struct {
	db *sql.DB
}

func NewMySQLPlanRepository(db *sql.DB) *MySQLPlanRepository {
	return &MySQLPlanRepository{db: db}
}

func (r *MySQLPlanRepository) List(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, duration_months, price, is_active, created_at, updated_at FROM subscription_plans ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.DurationMonths, &p.Price, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	featuresByPlan, err := r.loadFeatures(ctx)
	if err != nil {
		return nil, err
	}
	appIDsByPlan, err := r.loadAppIDs(ctx)
	if err != nil {
		return nil, err
	}
	for i := range plans {
		plans[i].Features = featuresByPlan[plans[i].ID]
		plans[i].AppIDs = appIDsByPlan[plans[i].ID]
	}
	return plans, nil
}

// ListPaginated backs the real `GET /plans` list page (Platform Console's
// plan management table) — List above stays as-is for the many "resolve my
// current plan"/"pick a plan" consumers (e.g. SubscriptionPage) that need the
// full set in one call.
func (r *MySQLPlanRepository) ListPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	countQuery := `SELECT COUNT(*) FROM subscription_plans`
	listQuery := `SELECT id, name, duration_months, price, is_active, created_at, updated_at FROM subscription_plans`
	var args []interface{}
	if search != "" {
		countQuery += ` WHERE name LIKE ?`
		listQuery += ` WHERE name LIKE ?`
		args = append(args, "%"+search+"%")
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY id LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, listQuery, append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.DurationMonths, &p.Price, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	featuresByPlan, err := r.loadFeatures(ctx)
	if err != nil {
		return nil, 0, err
	}
	appIDsByPlan, err := r.loadAppIDs(ctx)
	if err != nil {
		return nil, 0, err
	}
	for i := range plans {
		plans[i].Features = featuresByPlan[plans[i].ID]
		plans[i].AppIDs = appIDsByPlan[plans[i].ID]
	}
	return plans, total, nil
}

// ListForApp returns every plan mapped to appID via subscription_plan_apps
// (D3) — including non-active ones, with their `active` flag, since the
// consumer side already knows how to filter and needs the flag to mark a
// tenant's currently-active plan (A3). AppIDs is intentionally left nil:
// this path serves an `app` principal, which has no business knowing what
// other apps a plan is mapped to.
func (r *MySQLPlanRepository) ListForApp(ctx context.Context, appID string) ([]domain.Plan, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT sp.id, sp.name, sp.duration_months, sp.price, sp.is_active, sp.created_at, sp.updated_at
		 FROM subscription_plans sp
		 JOIN subscription_plan_apps spa ON spa.plan_id = sp.id
		 WHERE spa.app_id = ?
		 ORDER BY sp.price ASC, sp.id ASC`,
		appID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.DurationMonths, &p.Price, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	featuresByPlan, err := r.loadFeatures(ctx)
	if err != nil {
		return nil, err
	}
	for i := range plans {
		plans[i].Features = featuresByPlan[plans[i].ID]
	}
	return plans, nil
}

// ListPublic returns plans with NO row in subscription_plan_apps — ElProof's
// own catalog (demo/internal), visible to any principal that isn't
// platform_admin (D3/A12). AppIDs is left nil: for a public plan it's
// always empty anyway.
func (r *MySQLPlanRepository) ListPublic(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, duration_months, price, is_active, created_at, updated_at FROM subscription_plans sp
		 WHERE NOT EXISTS (SELECT 1 FROM subscription_plan_apps spa WHERE spa.plan_id = sp.id)
		 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.DurationMonths, &p.Price, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	featuresByPlan, err := r.loadFeatures(ctx)
	if err != nil {
		return nil, err
	}
	for i := range plans {
		plans[i].Features = featuresByPlan[plans[i].ID]
	}
	return plans, nil
}

// ListPublicPaginated is ListPaginated scoped to the public catalog (D3).
// Conditions are collected into a slice and joined with AND rather than
// string-concatenated onto an assumed `WHERE name LIKE ?` — ListPaginated's
// own conditional WHERE is exactly the trap this avoids (task 3.3). COUNT(*)
// reuses the identical condition list so the paginated total never drifts
// from what the data query actually returns.
func (r *MySQLPlanRepository) ListPublicPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	conditions := []string{"NOT EXISTS (SELECT 1 FROM subscription_plan_apps spa WHERE spa.plan_id = sp.id)"}
	var args []interface{}
	if search != "" {
		conditions = append(conditions, "sp.name LIKE ?")
		args = append(args, "%"+search+"%")
	}
	whereClause := " WHERE " + strings.Join(conditions, " AND ")

	var total int64
	countQuery := `SELECT COUNT(*) FROM subscription_plans sp` + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT sp.id, sp.name, sp.duration_months, sp.price, sp.is_active, sp.created_at, sp.updated_at FROM subscription_plans sp` +
		whereClause + ` ORDER BY sp.id LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, listQuery, append(append([]interface{}{}, args...), params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.DurationMonths, &p.Price, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	featuresByPlan, err := r.loadFeatures(ctx)
	if err != nil {
		return nil, 0, err
	}
	for i := range plans {
		plans[i].Features = featuresByPlan[plans[i].ID]
	}
	return plans, total, nil
}

// ListPlanIDsForApp returns the raw plan_id set mapped to appID — backs both
// the app-plan management GET (task 7.3) and the cross-App purchase guard
// (StartSubscriptionCharge, A18/R1).
func (r *MySQLPlanRepository) ListPlanIDsForApp(ctx context.Context, appID string) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT plan_id FROM subscription_plan_apps WHERE app_id = ? ORDER BY plan_id`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ReplaceAppPlans replaces the entire mapping set for appID in one
// transaction — the same DELETE-then-INSERT replace-set pattern Update
// already uses for plan_features (T8). planIDs may be empty (= revoke every
// mapping, a valid state).
func (r *MySQLPlanRepository) ReplaceAppPlans(ctx context.Context, appID string, planIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM subscription_plan_apps WHERE app_id = ?`, appID); err != nil {
		return err
	}
	for _, planID := range planIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO subscription_plan_apps (plan_id, app_id) VALUES (?, ?)`,
			planID, appID,
		); err != nil {
			// FK violation (1452) means a planID in the payload doesn't
			// exist — a client-side mistake, not a server failure, so it
			// maps to a validation error (422) instead of bubbling up as
			// a raw 500 (task 7.4).
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1452 {
				return apperror.Validation("Salah satu plan_id tidak ditemukan", nil)
			}
			return err
		}
	}
	return tx.Commit()
}

func (r *MySQLPlanRepository) FindByID(ctx context.Context, id int64) (*domain.Plan, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, duration_months, price, is_active, created_at, updated_at FROM subscription_plans WHERE id = ? LIMIT 1`,
		id,
	)
	var p domain.Plan
	err := row.Scan(&p.ID, &p.Name, &p.DurationMonths, &p.Price, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	features, err := r.loadFeaturesForPlan(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Features = features
	return &p, nil
}

func (r *MySQLPlanRepository) Create(ctx context.Context, plan *domain.Plan) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO subscription_plans (name, duration_months, price, is_active) VALUES (?, ?, ?, ?)`,
		plan.Name, plan.DurationMonths, plan.Price, plan.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	plan.ID = id

	if err := insertFeatures(ctx, tx, id, plan.Features); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLPlanRepository) Update(ctx context.Context, plan *domain.Plan) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE subscription_plans SET name = ?, duration_months = ?, price = ? WHERE id = ?`,
		plan.Name, plan.DurationMonths, plan.Price, plan.ID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM plan_features WHERE plan_id = ?`, plan.ID); err != nil {
		return err
	}
	if err := insertFeatures(ctx, tx, plan.ID, plan.Features); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLPlanRepository) SetActive(ctx context.Context, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE subscription_plans SET is_active = ? WHERE id = ?`, isActive, id)
	return err
}

func insertFeatures(ctx context.Context, tx *sql.Tx, planID int64, features []string) error {
	for i, label := range features {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO plan_features (plan_id, label, sort_order) VALUES (?, ?, ?)`,
			planID, label, i,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *MySQLPlanRepository) loadFeaturesForPlan(ctx context.Context, planID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT label FROM plan_features WHERE plan_id = ? ORDER BY sort_order`, planID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []string
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return nil, err
		}
		features = append(features, label)
	}
	return features, rows.Err()
}

func (r *MySQLPlanRepository) loadFeatures(ctx context.Context) (map[int64][]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT plan_id, label FROM plan_features ORDER BY plan_id, sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]string)
	for rows.Next() {
		var planID int64
		var label string
		if err := rows.Scan(&planID, &label); err != nil {
			return nil, err
		}
		result[planID] = append(result[planID], label)
	}
	return result, rows.Err()
}

// loadAppIDs mirrors loadFeatures — one query for every plan's app mapping,
// so List/ListPaginated don't run a per-row query. Called only from those
// two (the platform_admin path, D12/A23); ListForApp/ListPublic*
// deliberately never call it.
func (r *MySQLPlanRepository) loadAppIDs(ctx context.Context) (map[int64][]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT plan_id, app_id FROM subscription_plan_apps ORDER BY plan_id, app_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]string)
	for rows.Next() {
		var planID int64
		var appID string
		if err := rows.Scan(&planID, &appID); err != nil {
			return nil, err
		}
		result[planID] = append(result[planID], appID)
	}
	return result, rows.Err()
}
