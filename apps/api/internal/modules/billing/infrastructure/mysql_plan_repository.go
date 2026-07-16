package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/billing/domain"
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
	for i := range plans {
		plans[i].Features = featuresByPlan[plans[i].ID]
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
	for i := range plans {
		plans[i].Features = featuresByPlan[plans[i].ID]
	}
	return plans, total, nil
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
