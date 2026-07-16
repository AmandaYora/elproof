package infrastructure

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/pagination"
)

type MySQLTenantRepository struct {
	db *sql.DB
}

func NewMySQLTenantRepository(db *sql.DB) *MySQLTenantRepository {
	return &MySQLTenantRepository{db: db}
}

const tenantColumns = `id, business_name, owner_name, username, email, phone, city, joined_at, plan_id,
	subscription_status, subscription_expires_at, is_suspended, last_credential_reset_at, created_at, updated_at`

func scanTenant(scan func(dest ...interface{}) error) (*domain.Tenant, error) {
	var t domain.Tenant
	var planID sql.NullInt64
	var status string
	var expiresAt, lastReset sql.NullTime

	err := scan(
		&t.ID, &t.BusinessName, &t.OwnerName, &t.Username, &t.Email, &t.Phone, &t.City, &t.JoinedAt, &planID,
		&status, &expiresAt, &t.IsSuspended, &lastReset, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t.SubscriptionStatus = domain.SubscriptionStatus(status)
	if planID.Valid {
		t.PlanID = &planID.Int64
	}
	if expiresAt.Valid {
		t.SubscriptionExpiresAt = &expiresAt.Time
	}
	if lastReset.Valid {
		t.LastCredentialResetAt = &lastReset.Time
	}
	return &t, nil
}

func (r *MySQLTenantRepository) List(ctx context.Context) ([]domain.Tenant, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+tenantColumns+` FROM tenants ORDER BY joined_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		t, err := scanTenant(rows.Scan)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, *t)
	}
	return tenants, rows.Err()
}

// ListPaginated backs the real `GET /tenants` list page — List above stays
// as-is for any full-roster consumer.
func (r *MySQLTenantRepository) ListPaginated(ctx context.Context, params pagination.Params, search, status string) ([]domain.Tenant, int64, error) {
	countQuery := `SELECT COUNT(*) FROM tenants`
	listQuery := `SELECT ` + tenantColumns + ` FROM tenants`
	var args []interface{}
	var conditions []string
	if search != "" {
		conditions = append(conditions, `(business_name LIKE ? OR owner_name LIKE ? OR email LIKE ?)`)
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if status != "" {
		conditions = append(conditions, `subscription_status = ?`)
		args = append(args, status)
	}
	if len(conditions) > 0 {
		where := ` WHERE ` + strings.Join(conditions, " AND ")
		countQuery += where
		listQuery += where
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY joined_at DESC, id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, listQuery, append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		t, err := scanTenant(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, *t)
	}
	return tenants, total, rows.Err()
}

func (r *MySQLTenantRepository) FindByID(ctx context.Context, id int64) (*domain.Tenant, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+tenantColumns+` FROM tenants WHERE id = ? LIMIT 1`, id)
	return scanTenant(row.Scan)
}

func (r *MySQLTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO tenants (business_name, owner_name, username, email, phone, city, joined_at, plan_id, subscription_status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tenant.BusinessName, tenant.OwnerName, tenant.Username, tenant.Email, tenant.Phone, tenant.City,
		tenant.JoinedAt, tenant.PlanID, string(tenant.SubscriptionStatus),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	tenant.ID = id
	return nil
}

func (r *MySQLTenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET business_name = ?, owner_name = ?, email = ?, phone = ?, city = ? WHERE id = ?`,
		tenant.BusinessName, tenant.OwnerName, tenant.Email, tenant.Phone, tenant.City, tenant.ID,
	)
	return err
}

func (r *MySQLTenantRepository) SetSuspended(ctx context.Context, id int64, suspended bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET is_suspended = ? WHERE id = ?`, suspended, id)
	return err
}

func (r *MySQLTenantRepository) SetCredentialResetAt(ctx context.Context, id int64, when time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET last_credential_reset_at = ? WHERE id = ?`, when, id)
	return err
}

func (r *MySQLTenantRepository) UpdateSubscription(ctx context.Context, id int64, planID int64, status domain.SubscriptionStatus, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET plan_id = ?, subscription_status = ?, subscription_expires_at = ? WHERE id = ?`,
		planID, string(status), expiresAt, id,
	)
	return err
}
