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
		`INSERT INTO refresh_tokens (credential_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		token.CredentialID, token.TokenHash, token.ExpiresAt,
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
		`SELECT id, credential_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = ? LIMIT 1`,
		tokenHash,
	)

	var t domain.RefreshToken
	var revokedAt sql.NullTime
	err := row.Scan(&t.ID, &t.CredentialID, &t.TokenHash, &t.ExpiresAt, &revokedAt, &t.CreatedAt)
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

func (r *MySQLRefreshTokenRepository) Revoke(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = ? WHERE id = ?`, time.Now(), id)
	return err
}
