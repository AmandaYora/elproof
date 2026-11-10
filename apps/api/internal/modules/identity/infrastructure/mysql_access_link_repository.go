package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"elproof/internal/modules/identity/domain"
)

// MySQLAccessLinkRepository stores one-time access links (auth_tokens).
type MySQLAccessLinkRepository struct {
	db *sql.DB
}

func NewMySQLAccessLinkRepository(db *sql.DB) *MySQLAccessLinkRepository {
	return &MySQLAccessLinkRepository{db: db}
}

// Issue stores a new link and revokes every still-open link of the same
// login, so only the latest link handed out works.
//
// READ COMMITTED: under REPEATABLE READ the revoking UPDATE of a login with no
// open link gap-locks idx_auth_tokens_credential, and two invitations sent at
// once (two workspaces) then deadlocked on each other's INSERT (MySQL 1213).
// The login's own row is locked first instead, so two links issued at once for
// the same person still run one after the other and only the later one works.
func (r *MySQLAccessLinkRepository) Issue(ctx context.Context, credentialID int64, purpose domain.LinkPurpose, tokenHash string, expiresAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var locked int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM credentials WHERE id = ? FOR UPDATE`, credentialID).Scan(&locked); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE auth_tokens SET revoked_at = NOW() WHERE credential_id = ? AND used_at IS NULL AND revoked_at IS NULL`,
		credentialID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO auth_tokens (token_hash, purpose, credential_id, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash, string(purpose), credentialID, expiresAt); err != nil {
		return err
	}
	return tx.Commit()
}

// RevokeAll closes every open link of a login (deactivated, replaced).
func (r *MySQLAccessLinkRepository) RevokeAll(ctx context.Context, credentialID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_tokens SET revoked_at = NOW() WHERE credential_id = ? AND used_at IS NULL AND revoked_at IS NULL`,
		credentialID)
	return err
}

func (r *MySQLAccessLinkRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.AccessLink, error) {
	var l domain.AccessLink
	var purpose string
	var used, revoked sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, credential_id, purpose, expires_at, used_at, revoked_at FROM auth_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&l.ID, &l.CredentialID, &purpose, &l.ExpiresAt, &used, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l.Purpose = domain.LinkPurpose(purpose)
	if used.Valid {
		l.UsedAt = &used.Time
	}
	if revoked.Valid {
		l.RevokedAt = &revoked.Time
	}
	return &l, nil
}

// consume marks the link used inside tx; false when it was already used,
// revoked or expired (a concurrent redemption loses).
func consume(ctx context.Context, tx *sql.Tx, linkID int64) (bool, error) {
	// Expiry is judged with the app's clock (the same one that wrote
	// expires_at), never MySQL's NOW(), so differing time zones between the
	// app container and the database cannot shorten or lengthen a link.
	now := time.Now()
	res, err := tx.ExecContext(ctx,
		`UPDATE auth_tokens SET used_at = ? WHERE id = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?`, now, linkID, now)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// ConsumeAndLinkGoogle redeems a staff invitation: the link is used and the
// login becomes Google-only for this Google account, atomically.
func (r *MySQLAccessLinkRepository) ConsumeAndLinkGoogle(ctx context.Context, linkID, credentialID int64, sub string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	ok, err := consume(ctx, tx, linkID)
	if err != nil || !ok {
		return false, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE credentials SET google_sub = ?, password_hash = NULL WHERE id = ?`, sub, credentialID); err != nil {
		if isDuplicateKey(err) {
			return false, domain.ErrGoogleAccountTaken
		}
		return false, err
	}
	return true, tx.Commit()
}

// ConsumeAndSetPassword redeems a client invitation or reset link.
func (r *MySQLAccessLinkRepository) ConsumeAndSetPassword(ctx context.Context, linkID, credentialID int64, passwordHash string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	ok, err := consume(ctx, tx, linkID)
	if err != nil || !ok {
		return false, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE credentials SET password_hash = ? WHERE id = ?`, passwordHash, credentialID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
