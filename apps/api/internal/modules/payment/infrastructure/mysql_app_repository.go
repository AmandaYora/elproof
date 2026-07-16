package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/payment/domain"
)

type MySQLAppRepository struct {
	db *sql.DB
}

func NewMySQLAppRepository(db *sql.DB) *MySQLAppRepository {
	return &MySQLAppRepository{db: db}
}

const appColumns = `id, app_id, name, kind, secret_hash, secret_encrypted, callback_url, is_active, created_at, updated_at`

func scanApp(scan func(dest ...interface{}) error) (*domain.App, error) {
	var a domain.App
	var kind string
	var secretHash, secretEncrypted, callbackURL sql.NullString
	err := scan(&a.ID, &a.AppID, &a.Name, &kind, &secretHash, &secretEncrypted, &callbackURL, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Kind = domain.AppKind(kind)
	a.SecretHash = secretHash.String
	a.SecretEncrypted = secretEncrypted.String
	a.CallbackURL = callbackURL.String
	return &a, nil
}

func (r *MySQLAppRepository) List(ctx context.Context) ([]domain.App, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+appColumns+` FROM payment_apps ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.App
	for rows.Next() {
		a, err := scanApp(rows.Scan)
		if err != nil {
			return nil, err
		}
		apps = append(apps, *a)
	}
	return apps, rows.Err()
}

func (r *MySQLAppRepository) FindByAppID(ctx context.Context, appID string) (*domain.App, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+appColumns+` FROM payment_apps WHERE app_id = ? LIMIT 1`, appID)
	return scanApp(row.Scan)
}

func (r *MySQLAppRepository) Create(ctx context.Context, a *domain.App) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO payment_apps (app_id, name, kind, secret_hash, secret_encrypted, callback_url, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.AppID, a.Name, string(a.Kind), nullableString(a.SecretHash), nullableString(a.SecretEncrypted),
		nullableString(a.CallbackURL), a.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id
	return nil
}

// EnsureInternalApp upserts the internal App row this module bootstraps on
// every startup (see MODULE_PAYMENT.md §4: "App internal biasanya di-seed
// sekali (mis. lewat migrasi/bootstrap)") — idempotent, so repeated restarts
// (or a fresh `docker compose up`) never fail on a duplicate key.
func (r *MySQLAppRepository) EnsureInternalApp(ctx context.Context, appID, name string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO payment_apps (app_id, name, kind, is_active) VALUES (?, ?, 'internal', TRUE)
		 ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		appID, name,
	)
	return err
}

func (r *MySQLAppRepository) SetSecret(ctx context.Context, appID string, secretHash, secretEncrypted string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE payment_apps SET secret_hash = ?, secret_encrypted = ? WHERE app_id = ?`,
		secretHash, secretEncrypted, appID,
	)
	return err
}

func (r *MySQLAppRepository) SetActive(ctx context.Context, appID string, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE payment_apps SET is_active = ? WHERE app_id = ?`, isActive, appID)
	return err
}
