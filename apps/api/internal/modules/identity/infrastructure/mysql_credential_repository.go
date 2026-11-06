// Hand-written database/sql queries, structured to be a drop-in swap for
// sqlc-generated code later (interface in application/ stays the same either
// way) — see knowledge/DATABASE_GUIDE.md.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"

	"elproof/internal/modules/identity/domain"
)

type MySQLCredentialRepository struct {
	db *sql.DB
}

func NewMySQLCredentialRepository(db *sql.DB) *MySQLCredentialRepository {
	return &MySQLCredentialRepository{db: db}
}

const credentialColumns = `id, tenant_id, principal_type, principal_id, username, email, google_sub, password_hash, totp_secret_enc, totp_enabled, totp_last_step, role, display_name, is_active, created_at, updated_at`

func (r *MySQLCredentialRepository) FindByUsername(ctx context.Context, username string) (*domain.Credential, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+credentialColumns+` FROM credentials WHERE username = ? LIMIT 1`, username)
	return scanCredential(row)
}

// FindAllByEmail backs the login-by-email fallback (see AuthService.Login).
// Email isn't guaranteed unique across principal types/tenants the way
// Username is (no owning module enforces it), so this can return more than
// one row — the caller tries each candidate's password rather than assuming
// the first match is the right account.
func (r *MySQLCredentialRepository) FindAllByEmail(ctx context.Context, email string) ([]*domain.Credential, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+credentialColumns+` FROM credentials WHERE email = ?`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*domain.Credential
	for rows.Next() {
		c, err := scanCredentialRow(rows)
		if err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// FindByGoogleSub resolves a Google account to its ElProof login (nil when
// none is linked).
func (r *MySQLCredentialRepository) FindByGoogleSub(ctx context.Context, sub string) (*domain.Credential, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+credentialColumns+` FROM credentials WHERE google_sub = ? LIMIT 1`, sub)
	return scanCredential(row)
}

// LinkGoogle attaches a Google account to an existing login, only if that
// login has none yet — a concurrent link to a different account loses
// (false) instead of being overwritten.
func (r *MySQLCredentialRepository) LinkGoogle(ctx context.Context, credentialID int64, sub string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET google_sub = ? WHERE id = ? AND google_sub IS NULL`, sub, credentialID)
	if err != nil {
		if isDuplicateKey(err) {
			return false, nil
		}
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// UpdateEmail changes a login's email (synced from the owning module).
func (r *MySQLCredentialRepository) UpdateEmail(ctx context.Context, principalType domain.PrincipalType, principalID string, email string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET email = ? WHERE principal_type = ? AND principal_id = ?`,
		nullableEmail(email), string(principalType), principalID)
	if isDuplicateKey(err) {
		return domain.ErrEmailTaken
	}
	return err
}

// UpdateUsername renames a login (a client representative was replaced).
func (r *MySQLCredentialRepository) UpdateUsername(ctx context.Context, credentialID int64, username string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE credentials SET username = ? WHERE id = ?`, username, credentialID)
	if isDuplicateKey(err) {
		return domain.ErrUsernameTaken
	}
	return err
}

// Delete removes a login with its sessions and links (the owning module
// deleted the person, or rolled back a failed invitation), freeing its
// username and email.
func (r *MySQLCredentialRepository) Delete(ctx context.Context, credentialID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, q := range []string{
		`DELETE FROM auth_tokens WHERE credential_id = ?`,
		`DELETE FROM refresh_tokens WHERE credential_id = ?`,
		`DELETE FROM credentials WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, credentialID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ClearSignIn removes every way to sign in (password and Google link):
// the login waits for a new invitation (client representative replaced).
func (r *MySQLCredentialRepository) ClearSignIn(ctx context.Context, credentialID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE credentials SET password_hash = NULL, google_sub = NULL WHERE id = ?`, credentialID)
	return err
}

// UsernameTaken reports whether any login already uses username.
func (r *MySQLCredentialRepository) UsernameTaken(ctx context.Context, username string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM credentials WHERE username = ?`, username).Scan(&n)
	return n > 0, err
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
		`INSERT INTO credentials (tenant_id, principal_type, principal_id, username, email, google_sub, password_hash, role, display_name, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cred.TenantID, string(cred.PrincipalType), cred.PrincipalID, cred.Username, nullableEmail(cred.Email),
		nullableString(cred.GoogleSub), nullableString(cred.PasswordHash), cred.Role, cred.DisplayName, cred.IsActive,
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

func (r *MySQLCredentialRepository) UpdateRole(ctx context.Context, principalType domain.PrincipalType, principalID string, role string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET role = ? WHERE principal_type = ? AND principal_id = ?`,
		role, string(principalType), principalID,
	)
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows — one scan body serves
// the single-row (FindByUsername/FindByID/...) and multi-row (FindAllByEmail)
// queries alike.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanCredential(row *sql.Row) (*domain.Credential, error) {
	c, err := scanCredentialRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func scanCredentialRow(row scanner) (*domain.Credential, error) {
	var c domain.Credential
	var tenantID sql.NullInt64
	var principalType string
	var email, googleSub, passwordHash sql.NullString
	var totpLastStep sql.NullInt64

	err := row.Scan(
		&c.ID, &tenantID, &principalType, &c.PrincipalID, &c.Username, &email, &googleSub, &passwordHash,
		&c.TOTPSecretEnc, &c.TOTPEnabled, &totpLastStep,
		&c.Role, &c.DisplayName, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	c.TOTPLastStep = totpLastStep.Int64
	if err != nil {
		return nil, err
	}
	c.GoogleSub = googleSub.String
	c.PasswordHash = passwordHash.String

	c.PrincipalType = domain.PrincipalType(principalType)
	if tenantID.Valid {
		c.TenantID = &tenantID.Int64
	}
	if email.Valid {
		c.Email = email.String
	}
	return &c, nil
}

// nullableEmail stores "no email" as SQL NULL, never as an empty string. The
// email column is UNIQUE (migration 000016): MySQL allows many NULLs there
// but only one empty string, so every second account created without an
// email used to fail to insert.
// nullableString stores "" as NULL (google_sub is UNIQUE; password_hash is
// NULL for Google-only accounts).
func nullableString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: v != ""}
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func nullableEmail(email string) sql.NullString {
	email = strings.TrimSpace(email)
	return sql.NullString{String: email, Valid: email != ""}
}

// SetTOTPSecret stores a new (not yet confirmed) encrypted secret and turns
// two-factor off until the admin confirms a first code.
// It never touches an account whose two-factor is already on (guarded in
// SQL, so a setup racing a just-completed enable cannot switch it off) and
// reports whether the secret was stored.
func (r *MySQLCredentialRepository) SetTOTPSecret(ctx context.Context, credentialID int64, enc []byte) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET totp_secret_enc = ?, totp_last_step = NULL WHERE id = ? AND totp_enabled = FALSE`,
		enc, credentialID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// EnableTOTP marks the stored secret as confirmed and records the step used.
func (r *MySQLCredentialRepository) EnableTOTP(ctx context.Context, credentialID int64, step int64) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET totp_enabled = TRUE, totp_last_step = ? WHERE id = ? AND totp_secret_enc IS NOT NULL AND totp_enabled = FALSE`,
		step, credentialID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// SetTOTPLastStep records an accepted code's time step. The WHERE clause
// makes it monotonic, so two concurrent logins can't both use one code.
func (r *MySQLCredentialRepository) SetTOTPLastStep(ctx context.Context, credentialID int64, step int64) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET totp_last_step = ? WHERE id = ? AND (totp_last_step IS NULL OR totp_last_step < ?)`,
		step, credentialID, step)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// ClearTOTP removes two-factor from one principal (lost device).
func (r *MySQLCredentialRepository) ClearTOTP(ctx context.Context, principalType domain.PrincipalType, principalID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE credentials SET totp_secret_enc = NULL, totp_enabled = FALSE, totp_last_step = NULL
		 WHERE principal_type = ? AND principal_id = ?`,
		string(principalType), principalID)
	return err
}
