package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/pagination"
)

type MySQLPlanRepository struct {
	db *sql.DB
}

func NewMySQLPlanRepository(db *sql.DB) *MySQLPlanRepository {
	return &MySQLPlanRepository{db: db}
}

const planColumns = `id, app_id, name, duration_months, price, is_active, created_at, updated_at`

// planFilter builds the WHERE clause shared by the list and its COUNT, so the
// paginated total never drifts from the rows served. appID "" = every App.
func planFilter(appID, search string) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	if appID != "" {
		conditions = append(conditions, "app_id = ?")
		args = append(args, appID)
	}
	if search != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+search+"%")
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// List returns the plans of one App (appID) or of every App (appID ""),
// ordered by id, with their features.
func (r *MySQLPlanRepository) List(ctx context.Context, appID string) ([]domain.Plan, error) {
	where, args := planFilter(appID, "")
	plans, err := r.query(ctx, `SELECT `+planColumns+` FROM subscription_plans`+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	return plans, r.attachFeatures(ctx, plans)
}

// ListPaginated backs the paginated `GET /plans` — same filter as List plus a
// name search.
func (r *MySQLPlanRepository) ListPaginated(ctx context.Context, params pagination.Params, search, appID string) ([]domain.Plan, int64, error) {
	where, args := planFilter(appID, search)

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_plans`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	plans, err := r.query(ctx, `SELECT `+planColumns+` FROM subscription_plans`+where+` ORDER BY id LIMIT ? OFFSET ?`,
		append(append([]interface{}{}, args...), params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	return plans, total, r.attachFeatures(ctx, plans)
}

func (r *MySQLPlanRepository) FindByID(ctx context.Context, id int64) (*domain.Plan, error) {
	plans, err := r.query(ctx, `SELECT `+planColumns+` FROM subscription_plans WHERE id = ? LIMIT 1`, id)
	if err != nil {
		return nil, err
	}
	if len(plans) == 0 {
		return nil, nil
	}
	p := plans[0]
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
		`INSERT INTO subscription_plans (app_id, name, duration_months, price, is_active) VALUES (?, ?, ?, ?, ?)`,
		plan.AppID, plan.Name, plan.DurationMonths, plan.Price, plan.IsActive,
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
		`UPDATE subscription_plans SET app_id = ?, name = ?, duration_months = ?, price = ? WHERE id = ?`,
		plan.AppID, plan.Name, plan.DurationMonths, plan.Price, plan.ID,
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

func (r *MySQLPlanRepository) query(ctx context.Context, q string, args ...interface{}) ([]domain.Plan, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.AppID, &p.Name, &p.DurationMonths, &p.Price, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
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

// attachFeatures loads the features of the given plans in one query (not one
// per row) and sets them in place.
func (r *MySQLPlanRepository) attachFeatures(ctx context.Context, plans []domain.Plan) error {
	if len(plans) == 0 {
		return nil
	}
	placeholders := make([]string, len(plans))
	args := make([]interface{}, len(plans))
	for i, p := range plans {
		placeholders[i] = "?"
		args[i] = p.ID
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT plan_id, label FROM plan_features WHERE plan_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY plan_id, sort_order`,
		args...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	byPlan := make(map[int64][]string)
	for rows.Next() {
		var planID int64
		var label string
		if err := rows.Scan(&planID, &label); err != nil {
			return err
		}
		byPlan[planID] = append(byPlan[planID], label)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range plans {
		plans[i].Features = byPlan[plans[i].ID]
	}
	return nil
}
