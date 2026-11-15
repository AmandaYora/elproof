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
	var family sql.NullInt64
	if token.FamilyID != 0 {
		family = sql.NullInt64{Int64: token.FamilyID, Valid: true}
	}
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (credential_id, family_id, token_hash, mfa, expires_at) VALUES (?, ?, ?, ?, ?)`,
		token.CredentialID, family, token.TokenHash, token.MFA, token.ExpiresAt,
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
		`SELECT id, credential_id, family_id, token_hash, mfa, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = ? LIMIT 1`,
		tokenHash,
	)

	var t domain.RefreshToken
	var family sql.NullInt64
	var revokedAt sql.NullTime
	err := row.Scan(&t.ID, &t.CredentialID, &family, &t.TokenHash, &t.MFA, &t.ExpiresAt, &revokedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if family.Valid {
		t.FamilyID = family.Int64
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

// RevokeFamily ends one sign-in: its root and every token rotated from it.
func (r *MySQLRefreshTokenRepository) RevokeFamily(ctx context.Context, familyID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE (id = ? OR family_id = ?) AND revoked_at IS NULL`,
		time.Now(), familyID, familyID)
	return err
}

// FamilyActive reports whether one sign-in still has a usable token, i.e.
// it was neither signed out nor revoked as a whole.
func (r *MySQLRefreshTokenRepository) FamilyActive(ctx context.Context, familyID int64, now time.Time) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM refresh_tokens WHERE (id = ? OR family_id = ?) AND revoked_at IS NULL AND expires_at > ?`,
		familyID, familyID, now).Scan(&n)
	return n > 0, err
}

func (r *MySQLRefreshTokenRepository) RevokeAllForCredential(ctx context.Context, credentialID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE credential_id = ? AND revoked_at IS NULL`,
		time.Now(), credentialID)
	return err
}
