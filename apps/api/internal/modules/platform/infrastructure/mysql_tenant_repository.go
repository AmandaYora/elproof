package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

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
	subscription_status, subscription_expires_at, trial_ends_at, is_suspended, last_credential_reset_at,
	brand_color_preset, logo_storage_path, custom_domain, signup_source, onboarding_dismissed_at, created_at, updated_at`

func scanTenant(scan func(dest ...interface{}) error) (*domain.Tenant, error) {
	var t domain.Tenant
	var planID sql.NullInt64
	var status string
	var expiresAt, trialEndsAt, lastReset, onboardingDismissed sql.NullTime
	var logoStoragePath, customDomain sql.NullString
	var signupSource string

	err := scan(
		&t.ID, &t.BusinessName, &t.OwnerName, &t.Username, &t.Email, &t.Phone, &t.City, &t.JoinedAt, &planID,
		&status, &expiresAt, &trialEndsAt, &t.IsSuspended, &lastReset, &t.BrandColorPreset, &logoStoragePath, &customDomain,
		&signupSource, &onboardingDismissed, &t.CreatedAt, &t.UpdatedAt,
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
	if trialEndsAt.Valid {
		t.TrialEndsAt = &trialEndsAt.Time
	}
	if lastReset.Valid {
		t.LastCredentialResetAt = &lastReset.Time
	}
	if logoStoragePath.Valid {
		t.LogoStoragePath = &logoStoragePath.String
	}
	if customDomain.Valid {
		t.CustomDomain = &customDomain.String
	}
	t.SignupSource = domain.SignupSource(signupSource)
	if onboardingDismissed.Valid {
		t.OnboardingDismissedAt = &onboardingDismissed.Time
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

// FindByDomain resolves a tenant from an incoming request's Host header — the
// pre-auth lookup ADR-0015's public branding/logo endpoints use, distinct
// from every other lookup in this repo (all JWT/ID-scoped).
func (r *MySQLTenantRepository) FindByDomain(ctx context.Context, host string) (*domain.Tenant, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+tenantColumns+` FROM tenants WHERE custom_domain = ? LIMIT 1`, host)
	return scanTenant(row.Scan)
}

func (r *MySQLTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO tenants (business_name, owner_name, username, email, phone, city, joined_at, plan_id, subscription_status, trial_ends_at, brand_color_preset, signup_source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tenant.BusinessName, tenant.OwnerName, tenant.Username, tenant.Email, tenant.Phone, tenant.City,
		tenant.JoinedAt, tenant.PlanID, string(tenant.SubscriptionStatus), tenant.TrialEndsAt,
		brandPresetOrDefault(tenant.BrandColorPreset), signupSourceOrDefault(tenant.SignupSource),
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
		`UPDATE tenants SET business_name = ?, owner_name = ?, email = ?, phone = ?, city = ?, brand_color_preset = ?, custom_domain = ? WHERE id = ?`,
		tenant.BusinessName, tenant.OwnerName, tenant.Email, tenant.Phone, tenant.City, tenant.BrandColorPreset,
		tenant.CustomDomain, tenant.ID,
	)
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return domain.ErrDuplicateCustomDomain
	}
	return err
}

func (r *MySQLTenantRepository) UpdateLogo(ctx context.Context, id int64, logoStoragePath *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET logo_storage_path = ? WHERE id = ?`, logoStoragePath, id)
	return err
}

func (r *MySQLTenantRepository) SetSuspended(ctx context.Context, id int64, suspended bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET is_suspended = ? WHERE id = ?`, suspended, id)
	return err
}

// SetSubscriptionStatus refreshes the cached status column (the status
// sweeper; the lifecycle itself is computed from dates).
func (r *MySQLTenantRepository) SetSubscriptionStatus(ctx context.Context, id int64, status domain.SubscriptionStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET subscription_status = ? WHERE id = ?`, string(status), id)
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

func brandPresetOrDefault(p string) string {
	if p == "" {
		return domain.DefaultBrandColorPreset
	}
	return p
}

func signupSourceOrDefault(s domain.SignupSource) string {
	if s == "" {
		return string(domain.SignupSourcePlatform)
	}
	return string(s)
}

// CreateForSignup inserts a self-service tenant and records it on its signup
// row in one transaction, so a crash can never leave a tenant no signup
// owns. ok=false (nothing written) when the signup already has a tenant.
func (r *MySQLTenantRepository) CreateForSignup(ctx context.Context, tenant *domain.Tenant, signupID int64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO tenants (business_name, owner_name, username, email, phone, city, joined_at, plan_id, subscription_status, trial_ends_at, brand_color_preset, signup_source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tenant.BusinessName, tenant.OwnerName, tenant.Username, tenant.Email, tenant.Phone, tenant.City,
		tenant.JoinedAt, tenant.PlanID, string(tenant.SubscriptionStatus), tenant.TrialEndsAt,
		brandPresetOrDefault(tenant.BrandColorPreset), signupSourceOrDefault(tenant.SignupSource),
	)
	if err != nil {
		return false, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return false, err
	}
	upd, err := tx.ExecContext(ctx,
		`UPDATE signup_requests SET tenant_id = ? WHERE id = ? AND tenant_id IS NULL`, id, signupID)
	if err != nil {
		return false, err
	}
	if n, err := upd.RowsAffected(); err != nil || n != 1 {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	tenant.ID = id
	return true, nil
}

// UpdateUsername replaces the Owner username recorded on a tenant.
func (r *MySQLTenantRepository) UpdateUsername(ctx context.Context, id int64, username string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET username = ? WHERE id = ?`, username, id)
	return err
}

// UpdateProfile is the WO Owner's own edit (Lengkapi Profil): business
// name, WhatsApp number and city only.
func (r *MySQLTenantRepository) UpdateProfile(ctx context.Context, id int64, businessName, phone, city string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET business_name = ?, phone = ?, city = ? WHERE id = ?`, businessName, phone, city, id)
	return err
}

// UpdateBrandColor sets only the workspace's brand color preset (the Owner's
// own choice on Profil Usaha).
func (r *MySQLTenantRepository) UpdateBrandColor(ctx context.Context, id int64, preset string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET brand_color_preset = ? WHERE id = ?`, preset, id)
	return err
}

// GetBusinessDetails returns the tenant's details; an unset tenant has none
// (the zero value), not an error.
func (r *MySQLTenantRepository) GetBusinessDetails(ctx context.Context, tenantID int64) (domain.BusinessDetails, error) {
	var d domain.BusinessDetails
	err := r.db.QueryRowContext(ctx,
		`SELECT address, bank_name, bank_account_number, bank_account_holder FROM tenant_business_details WHERE tenant_id = ?`, tenantID,
	).Scan(&d.Address, &d.BankName, &d.BankAccountNumber, &d.BankAccountHolder)
	if err == sql.ErrNoRows {
		return domain.BusinessDetails{}, nil
	}
	return d, err
}

func (r *MySQLTenantRepository) SaveBusinessDetails(ctx context.Context, tenantID int64, d domain.BusinessDetails) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenant_business_details (tenant_id, address, bank_name, bank_account_number, bank_account_holder)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE address = VALUES(address), bank_name = VALUES(bank_name),
		   bank_account_number = VALUES(bank_account_number), bank_account_holder = VALUES(bank_account_holder)`,
		tenantID, d.Address, d.BankName, d.BankAccountNumber, d.BankAccountHolder)
	return err
}

// SetOnboardingDismissed records that the Owner closed the checklist.
func (r *MySQLTenantRepository) SetOnboardingDismissed(ctx context.Context, id int64, when time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET onboarding_dismissed_at = ? WHERE id = ?`, when, id)
	return err
}
