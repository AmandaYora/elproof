package infrastructure

import (
	"context"
	"database/sql"
	"elproof/internal/shared/utils"
	"time"

	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/pagination"
)

type MySQLClientRepository struct {
	db *sql.DB
}

func NewMySQLClientRepository(db *sql.DB) *MySQLClientRepository {
	return &MySQLClientRepository{db: db}
}

const clientColumns = `id, tenant_id, project_id, role, IFNULL(access_role_id, 0), username, relation_note, name, phone, email, is_active,
	last_credential_reset_at, created_at, updated_at, deleted_at`

func scanClient(scan func(dest ...interface{}) error) (*domain.Client, error) {
	var c domain.Client
	var role string
	var lastReset, deletedAt sql.NullTime
	err := scan(&c.ID, &c.TenantID, &c.ProjectID, &role, &c.AccessRoleID, &c.Username, &c.RelationNote, &c.Name, &c.Phone, &c.Email, &c.IsActive,
		&lastReset, &c.CreatedAt, &c.UpdatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Role = domain.ClientRole(role)
	if lastReset.Valid {
		c.LastCredentialResetAt = &lastReset.Time
	}
	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}
	return &c, nil
}

func (r *MySQLClientRepository) ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? AND project_id = ? AND deleted_at IS NULL ORDER BY id`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

func (r *MySQLClientRepository) ListByTenant(ctx context.Context, tenantID int64) ([]domain.Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? AND deleted_at IS NULL ORDER BY id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

// clientSortColumns is the whitelist of the columns this module can order by in
// SQL. projectName, projectEventDate and loginStatus are owned by other modules
// and ordered in memory (domain.NeedsMemoryPaging); they are listed by
// ClientSortFields for the whitelist the endpoint accepts.
var clientSortColumns = map[string]pagination.SortColumn{
	"name":      pagination.Col("name"),
	"role":      pagination.Col("role"),
	"createdAt": pagination.Col("created_at"),
}

// ListSortSpec is the whitelist (SQL columns) of GET /clients' own columns; the
// application layer adds the three cross-module fields.
func (r *MySQLClientRepository) ListSortSpec() map[string]pagination.SortColumn {
	return clientSortColumns
}

// clientWhere builds the shared WHERE of the paginated list (the loginStatus
// filter is not SQL: it is applied by the service).
func clientWhere(tenantID int64, f domain.ListFilter) (string, []interface{}) {
	where := ` WHERE tenant_id = ? AND deleted_at IS NULL`
	args := []interface{}{tenantID}
	if f.Search != "" {
		where += ` AND (name LIKE ? OR email LIKE ?)`
		like := "%" + utils.EscapeLike(f.Search) + "%"
		args = append(args, like, like)
	}
	if f.Role != "" {
		where += ` AND role = ?`
		args = append(args, f.Role)
	}
	if f.ProjectID != nil {
		where += ` AND project_id = ?`
		args = append(args, *f.ProjectID)
	}
	return where, args
}

// ListByTenantPaginated backs the real `GET /clients` list page — ListByTenant
// above stays as-is for global search. Filtering and ordering are SQL; sort is
// one of the columns of clientSortColumns.
func (r *MySQLClientRepository) ListByTenantPaginated(ctx context.Context, tenantID int64, params pagination.Params, f domain.ListFilter, sort pagination.Sort) ([]domain.Client, int64, error) {
	where, args := clientWhere(tenantID, f)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+clientColumns+` FROM clients`+where+sort.OrderBy("id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *c)
	}
	return list, total, rows.Err()
}

// ListAllFiltered returns every live client of the tenant matching the SQL
// filters, in id order — the base of the in-memory paging used when the order or
// a filter depends on another module's data.
func (r *MySQLClientRepository) ListAllFiltered(ctx context.Context, tenantID int64, f domain.ListFilter) ([]domain.Client, error) {
	where, args := clientWhere(tenantID, f)
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientColumns+` FROM clients`+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

func (r *MySQLClientRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? AND id = ? AND deleted_at IS NULL LIMIT 1`, tenantID, id)
	return scanClient(row.Scan)
}

// FindDeletedByID is the restore lookup: the client while it waits in the
// "deleted" state (nil when it is live or never existed).
func (r *MySQLClientRepository) FindDeletedByID(ctx context.Context, tenantID, id int64) (*domain.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, tenantID, id)
	return scanClient(row.Scan)
}

// ListAllByProject lists every client of the project, deleted ones included. Only
// the project hard delete needs it, so no row of a purged project is left behind.
func (r *MySQLClientRepository) ListAllByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE tenant_id = ? AND project_id = ? ORDER BY id`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

// SoftDelete hides a live client; false when there was none.
func (r *MySQLClientRepository) SoftDelete(ctx context.Context, tenantID, id int64, at time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE clients SET deleted_at = ? WHERE tenant_id = ? AND id = ? AND deleted_at IS NULL`, at, tenantID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// Restore clears the stamp of a client deleted after notBefore (the purge
// cutoff, the exact complement of the purge's `deleted_at <= cutoff`).
func (r *MySQLClientRepository) Restore(ctx context.Context, tenantID, id int64, notBefore time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE clients SET deleted_at = NULL WHERE tenant_id = ? AND id = ? AND deleted_at IS NOT NULL AND deleted_at > ?`, tenantID, id, notBefore)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ListExpired returns up to limit clients deleted at or before cutoff.
func (r *MySQLClientRepository) ListExpired(ctx context.Context, cutoff time.Time, limit int) ([]domain.Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientColumns+` FROM clients WHERE deleted_at IS NOT NULL AND deleted_at <= ? ORDER BY id LIMIT ?`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Client
	for rows.Next() {
		c, err := scanClient(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

// PurgeDelete hard-deletes a client that is still due for the purge (conditional,
// so a row restored in the meantime is never removed); false when it was not.
func (r *MySQLClientRepository) PurgeDelete(ctx context.Context, tenantID, id int64, cutoff time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM clients WHERE tenant_id = ? AND id = ? AND deleted_at IS NOT NULL AND deleted_at <= ?`, tenantID, id, cutoff)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// purgeLockName makes only one API instance purge clients at a time.
const purgeLockName = "elproof_clients_purge"

// TryLock takes the purge lock without waiting (ok false: another instance holds
// it). The lock lives on a dedicated connection and goes away with it.
func (r *MySQLClientRepository) TryLock(ctx context.Context) (release func(), ok bool, err error) {
	c, err := r.db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var got sql.NullInt64
	if err := c.QueryRowContext(ctx, `SELECT GET_LOCK(?, 0)`, purgeLockName).Scan(&got); err != nil {
		c.Close()
		return nil, false, err
	}
	if !got.Valid || got.Int64 != 1 {
		c.Close()
		return nil, false, nil
	}
	return func() {
		_, _ = c.ExecContext(context.Background(), `SELECT RELEASE_LOCK(?)`, purgeLockName)
		c.Close()
	}, true, nil
}

func (r *MySQLClientRepository) Create(ctx context.Context, c *domain.Client) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO clients (tenant_id, project_id, role, access_role_id, username, relation_note, name, phone, email, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.TenantID, c.ProjectID, string(c.Role), sql.NullInt64{Int64: c.AccessRoleID, Valid: c.AccessRoleID > 0}, c.Username, c.RelationNote, c.Name, c.Phone, c.Email, c.IsActive,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

func (r *MySQLClientRepository) Update(ctx context.Context, c *domain.Client) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clients SET name = ?, phone = ?, email = ?, relation_note = ? WHERE tenant_id = ? AND id = ?`,
		c.Name, c.Phone, c.Email, c.RelationNote, c.TenantID, c.ID,
	)
	return err
}

// UpdateAccessRole changes the client's portal role.
func (r *MySQLClientRepository) UpdateAccessRole(ctx context.Context, tenantID, id, roleID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE clients SET access_role_id = ? WHERE tenant_id = ? AND id = ?`, roleID, tenantID, id)
	return err
}

// UpdateUsername records the login name (renamed with a new representative).
func (r *MySQLClientRepository) UpdateUsername(ctx context.Context, tenantID, id int64, username string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE clients SET username = ? WHERE tenant_id = ? AND id = ?`, username, tenantID, id)
	return err
}

func (r *MySQLClientRepository) SetActive(ctx context.Context, tenantID, id int64, isActive bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE clients SET is_active = ? WHERE tenant_id = ? AND id = ?`, isActive, tenantID, id)
	return err
}

func (r *MySQLClientRepository) SetCredentialResetAt(ctx context.Context, tenantID, id int64, when time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE clients SET last_credential_reset_at = ? WHERE tenant_id = ? AND id = ?`, when, tenantID, id)
	return err
}

// Delete is only ever called by ClientService.Create as a compensating
// rollback when identity.CreateCredential fails after this row already
// committed — clients are otherwise never hard-deleted (SetActive is the
// normal deactivation path).
func (r *MySQLClientRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM clients WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

// CountActive counts active client accounts (trial cap, PLAN.md K6).
func (r *MySQLClientRepository) CountActive(ctx context.Context, tenantID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients WHERE tenant_id = ? AND is_active = TRUE AND deleted_at IS NULL`, tenantID).Scan(&n)
	return n, err
}
