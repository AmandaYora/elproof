package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"elproof/internal/modules/identity/domain"
)

type MySQLRefreshTokenRepository struct {
	db *sql.DB
}

func NewMySQLRefreshTokenRepository(db *sql.DB) *MySQLRefreshTokenRepository {
	return &MySQLRefreshTokenRepository{db: db}
}

func (r *MySQLRefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (credential_id, token_hash, mfa, expires_at) VALUES (?, ?, ?, ?)`,
		token.CredentialID, token.TokenHash, token.MFA, token.ExpiresAt,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	token.ID = id
	return nil
}

func (r *MySQLRefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, credential_id, token_hash, mfa, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = ? LIMIT 1`,
		tokenHash,
	)

	var t domain.RefreshToken
	var revokedAt sql.NullTime
	err := row.Scan(&t.ID, &t.CredentialID, &t.TokenHash, &t.MFA, &t.ExpiresAt, &revokedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		t.RevokedAt = &revokedAt.Time
	}
	return &t, nil
}

// Revoke marks one token used. It reports whether this call was the one that
// revoked it, so two concurrent refreshes cannot both redeem the same token.
func (r *MySQLRefreshTokenRepository) Revoke(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, time.Now(), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (r *MySQLRefreshTokenRepository) RevokeAllForCredential(ctx context.Context, credentialID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE credential_id = ? AND revoked_at IS NULL`,
		time.Now(), credentialID)
	return err
}
