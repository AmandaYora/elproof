// Package infrastructure holds the offers module's MySQL repositories. They
// read and write only offers' own tables (ADR-0049), always scoped to a
// tenant (ADR-0004).
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"

	"github.com/go-sql-driver/mysql"
)

type MySQLEventKindRepository struct {
	db *sql.DB
}

func NewMySQLEventKindRepository(db *sql.DB) *MySQLEventKindRepository {
	return &MySQLEventKindRepository{db: db}
}

const eventKindColumns = `id, tenant_id, name, default_key, sort_order, is_active, created_at, updated_at`

func scanEventKind(scan func(dest ...any) error) (*domain.EventKind, error) {
	var k domain.EventKind
	var key sql.NullString
	err := scan(&k.ID, &k.TenantID, &k.Name, &key, &k.SortOrder, &k.IsActive, &k.CreatedAt, &k.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	k.DefaultKey = key.String
	return &k, nil
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// errNameTaken is the 422 a duplicate name answers when two writes race past
// the service's own check and the unique key catches the second.
func errNameTaken(msg string) error {
	return apperror.Validation(msg, map[string][]string{"name": {msg}})
}

func (r *MySQLEventKindRepository) List(ctx context.Context, tenantID int64) ([]domain.EventKind, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+eventKindColumns+` FROM event_kinds WHERE tenant_id = ? ORDER BY sort_order, id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EventKind
	for rows.Next() {
		k, err := scanEventKind(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *k)
	}
	return out, rows.Err()
}

func (r *MySQLEventKindRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.EventKind, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+eventKindColumns+` FROM event_kinds WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return scanEventKind(row.Scan)
}

func (r *MySQLEventKindRepository) Create(ctx context.Context, k *domain.EventKind) error {
	var key any
	if k.DefaultKey != "" {
		key = k.DefaultKey
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO event_kinds (tenant_id, name, default_key, sort_order, is_active) VALUES (?, ?, ?, ?, ?)`,
		k.TenantID, k.Name, key, k.SortOrder, k.IsActive)
	if isDuplicate(err) {
		return errNameTaken("Jenis acara dengan nama ini sudah ada")
	}
	if err != nil {
		return err
	}
	k.ID, err = res.LastInsertId()
	return err
}

func (r *MySQLEventKindRepository) Rename(ctx context.Context, tenantID, id int64, name string, expected *time.Time) (bool, error) {
	query := `UPDATE event_kinds SET name = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE tenant_id = ? AND id = ?`
	args := []any{name, tenantID, id}
	if expected != nil {
		query += ` AND updated_at = ?`
		args = append(args, *expected)
	}
	res, err := r.db.ExecContext(ctx, query, args...)
	if isDuplicate(err) {
		return false, errNameTaken("Jenis acara dengan nama ini sudah ada")
	}
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (r *MySQLEventKindRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE event_kinds SET is_active = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}

func (r *MySQLEventKindRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM event_kinds WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

func (r *MySQLEventKindRepository) PackageCounts(ctx context.Context, tenantID int64) (map[int64]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_kind_id, COUNT(*) FROM package_event_kinds WHERE tenant_id = ? GROUP BY event_kind_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
