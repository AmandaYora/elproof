// Package infrastructure stores tenant roles (tenant_roles,
// tenant_role_permissions).
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"

	"elproof/internal/modules/access/domain"
	"elproof/internal/shared/utils"
)

type MySQLRoleRepository struct {
	db *sql.DB
}

func NewMySQLRoleRepository(db *sql.DB) *MySQLRoleRepository {
	return &MySQLRoleRepository{db: db}
}

const roleColumns = `id, tenant_id, principal_kind, name, description, IFNULL(template_key, ''), is_system, project_scope, created_at, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanRole(s rowScanner) (*domain.Role, error) {
	var r domain.Role
	var kind, scope string
	if err := s.Scan(&r.ID, &r.TenantID, &kind, &r.Name, &r.Description, &r.TemplateKey, &r.IsSystem, &scope, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Kind, r.Scope = domain.Kind(kind), domain.Scope(scope)
	return &r, nil
}

func (r *MySQLRoleRepository) loadPermissions(ctx context.Context, roles []*domain.Role) error {
	if len(roles) == 0 {
		return nil
	}
	ids := make([]int64, len(roles))
	byID := make(map[int64]*domain.Role, len(roles))
	for i, role := range roles {
		ids[i] = role.ID
		role.Permissions = []string{}
		byID[role.ID] = role
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT role_id, permission FROM tenant_role_permissions WHERE role_id IN (`+utils.Placeholders(len(ids))+`) ORDER BY permission`,
		utils.Int64Args(ids)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var perm string
		if err := rows.Scan(&id, &perm); err != nil {
			return err
		}
		if role := byID[id]; role != nil {
			role.Permissions = append(role.Permissions, perm)
		}
	}
	return rows.Err()
}

func (r *MySQLRoleRepository) List(ctx context.Context, tenantID int64, kind domain.Kind) ([]*domain.Role, error) {
	q := `SELECT ` + roleColumns + ` FROM tenant_roles WHERE tenant_id = ?`
	args := []any{tenantID}
	if kind != "" {
		q += ` AND principal_kind = ?`
		args = append(args, string(kind))
	}
	q += ` ORDER BY is_system DESC, template_key IS NULL, id`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []*domain.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, r.loadPermissions(ctx, roles)
}

func (r *MySQLRoleRepository) Find(ctx context.Context, tenantID, id int64) (*domain.Role, error) {
	role, err := scanRole(r.db.QueryRowContext(ctx, `SELECT `+roleColumns+` FROM tenant_roles WHERE tenant_id = ? AND id = ?`, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return role, r.loadPermissions(ctx, []*domain.Role{role})
}

func (r *MySQLRoleRepository) FindByTemplate(ctx context.Context, tenantID int64, templateKey string) (*domain.Role, error) {
	role, err := scanRole(r.db.QueryRowContext(ctx, `SELECT `+roleColumns+` FROM tenant_roles WHERE tenant_id = ? AND template_key = ?`, tenantID, templateKey))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return role, r.loadPermissions(ctx, []*domain.Role{role})
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func replacePermissions(ctx context.Context, tx *sql.Tx, roleID int64, perms []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM tenant_role_permissions WHERE role_id = ?`, roleID); err != nil {
		return err
	}
	if len(perms) == 0 {
		return nil
	}
	values := strings.TrimSuffix(strings.Repeat("(?, ?),", len(perms)), ",")
	args := make([]any, 0, len(perms)*2)
	for _, p := range perms {
		args = append(args, roleID, p)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO tenant_role_permissions (role_id, permission) VALUES `+values, args...)
	return err
}

// Create inserts a role with its permissions (one transaction).
func (r *MySQLRoleRepository) Create(ctx context.Context, role *domain.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var template any
	if role.TemplateKey != "" {
		template = role.TemplateKey
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		role.TenantID, string(role.Kind), role.Name, role.Description, template, role.IsSystem, string(role.Scope))
	if err != nil {
		if isDuplicate(err) {
			return domain.ErrDuplicateRoleName
		}
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if err := replacePermissions(ctx, tx, id, role.Permissions); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	role.ID = id
	return nil
}

// Update replaces a role's name, description, scope and permissions.
func (r *MySQLRoleRepository) Update(ctx context.Context, role *domain.Role) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx,
		`UPDATE tenant_roles SET name = ?, description = ?, project_scope = ? WHERE tenant_id = ? AND id = ?`,
		role.Name, role.Description, string(role.Scope), role.TenantID, role.ID); err != nil {
		if isDuplicate(err) {
			return domain.ErrDuplicateRoleName
		}
		return err
	}
	if err := replacePermissions(ctx, tx, role.ID, role.Permissions); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLRoleRepository) Delete(ctx context.Context, tenantID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tenant_roles WHERE tenant_id = ? AND id = ? AND is_system = FALSE`, tenantID, id)
	return err
}
