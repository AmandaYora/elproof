// Hand-written database/sql queries, structured to be a drop-in swap for
// sqlc-generated code later (interface in application/ stays the same either
// way) — see knowledge/DATABASE_GUIDE.md.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"elproof/internal/modules/identity/domain"
)

type MySQLCredentialRepository struct {
	db *sql.DB
}

func NewMySQLCredentialRepository(db *sql.DB) *MySQLCredentialRepository {
	return &MySQLCredentialRepository{db: db}
}

const credentialColumns = `id, tenant_id, principal_type, principal_id, username, password_hash, role, display_name, is_active, created_at, updated_at`

func (r *MySQLCredentialRepository) FindByUsername(ctx context.Context, username string) (*domain.Credential, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+credentialColumns+` FROM credentials WHERE username = ? LIMIT 1`, username)
	return scanCredential(row)
}

func (r *MySQLCredentialRepository) FindByID(ctx context.Context, id int64) (*domain.Credential, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+credentialColumns+` FROM credentials WHERE id = ? LIMIT 1`, id)
	return scanCredential(row)
}

func (r *MySQLCredentialRepository) FindByPrincipal(ctx context.Context, principalType domain.PrincipalType, principalID string) (*domain.Credential, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+credentialColumns+` FROM credentials WHERE principal_type = ? AND principal_id = ? LIMIT 1`,
		string(principalType), principalID,
	)
	return scanCredential(row)
}

func (r *MySQLCredentialRepository) Create(ctx context.Context, cred *domain.Credential) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO credentials (tenant_id, principal_type, principal_id, username, password_hash, role, display_name, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cred.TenantID, string(cred.PrincipalType), cred.PrincipalID, cred.Username, cred.PasswordHash, cred.Role, cred.DisplayName, cred.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	cred.ID = id
	return nil
}

func (r *MySQLCredentialRepository) UpdatePasswordHash(ctx context.Context, principalType domain.PrincipalType, principalID string, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET password_hash = ? WHERE principal_type = ? AND principal_id = ?`,
		passwordHash, string(principalType), principalID,
	)
	return err
}

func (r *MySQLCredentialRepository) SetActive(ctx context.Context, principalType domain.PrincipalType, principalID string, isActive bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET is_active = ? WHERE principal_type = ? AND principal_id = ?`,
		isActive, string(principalType), principalID,
	)
	return err
}

func scanCredential(row *sql.Row) (*domain.Credential, error) {
	var c domain.Credential
	var tenantID sql.NullInt64
	var principalType string

	err := row.Scan(
		&c.ID, &tenantID, &principalType, &c.PrincipalID, &c.Username, &c.PasswordHash,
		&c.Role, &c.DisplayName, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.PrincipalType = domain.PrincipalType(principalType)
	if tenantID.Valid {
		c.TenantID = &tenantID.Int64
	}
	return &c, nil
}
