package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
)

// MySQLSupplierBillRepository stores supplier_bill_terms: the bills of a vendor
// or venue the client pays directly (Paket WO Tahap 4). Every query filters on
// the workspace (tenant_id) as well as the project (ADR-0004).
type MySQLSupplierBillRepository struct {
	db *sql.DB
}

func NewMySQLSupplierBillRepository(db *sql.DB) *MySQLSupplierBillRepository {
	return &MySQLSupplierBillRepository{db: db}
}

const supplierBillColumns = `id, project_id, project_supplier_id, label, amount, due_date, sort_order, deleted_at, updated_at`

// liveBillTermWhere: a bill is hidden when it is deleted itself or its supplier is.
const liveBillTermWhere = `deleted_at IS NULL AND project_supplier_id NOT IN (` + deletedSupplierIDs + `)`

func scanSupplierBill(scan func(dest ...interface{}) error) (*domain.SupplierBillTerm, error) {
	var t domain.SupplierBillTerm
	var deletedAt sql.NullTime
	err := scan(&t.ID, &t.ProjectID, &t.ProjectSupplierID, &t.Label, &t.Amount, &t.DueDate, &t.SortOrder, &deletedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.DeletedAt = deletedStamp(deletedAt)
	return &t, nil
}

// ListByProject is every live bill of the project, earliest due date first.
func (r *MySQLSupplierBillRepository) ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.SupplierBillTerm, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+supplierBillColumns+` FROM supplier_bill_terms
		WHERE tenant_id = ? AND project_id = ? AND `+liveBillTermWhere+` ORDER BY due_date, sort_order, id`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SupplierBillTerm
	for rows.Next() {
		t, err := scanSupplierBill(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// FindByID is one live bill of the project (nil when none).
func (r *MySQLSupplierBillRepository) FindByID(ctx context.Context, tenantID, projectID, id int64) (*domain.SupplierBillTerm, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+supplierBillColumns+` FROM supplier_bill_terms
		WHERE tenant_id = ? AND project_id = ? AND id = ? AND `+liveBillTermWhere+` LIMIT 1`, tenantID, projectID, id)
	return scanSupplierBill(row.Scan)
}

// FindDeletedByID is the restore lookup (nil unless the bill waits in the
// "deleted" state by its own stamp).
func (r *MySQLSupplierBillRepository) FindDeletedByID(ctx context.Context, tenantID, projectID, id int64) (*domain.SupplierBillTerm, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+supplierBillColumns+` FROM supplier_bill_terms
		WHERE tenant_id = ? AND project_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, tenantID, projectID, id)
	return scanSupplierBill(row.Scan)
}

// Create inserts the bill with the project's tenant; a project of another
// workspace inserts nothing (reported as sql.ErrNoRows).
func (r *MySQLSupplierBillRepository) Create(ctx context.Context, tenantID int64, t *domain.SupplierBillTerm) error {
	q := conn(ctx, r.db)
	res, err := q.ExecContext(ctx,
		`INSERT INTO supplier_bill_terms (tenant_id, project_id, project_supplier_id, label, amount, due_date, sort_order)
		 SELECT tenant_id, id, ?, ?, ?, ?, ? FROM projects WHERE id = ? AND tenant_id = ?`,
		t.ProjectSupplierID, t.Label, t.Amount, t.DueDate, t.SortOrder, t.ProjectID, tenantID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	return readVersion(ctx, q, "supplier_bill_terms", id, &t.UpdatedAt)
}

var supplierBillPatchColumns = map[string]column[domain.SupplierBillTerm]{
	"label":   col("label", func(t *domain.SupplierBillTerm) any { return t.Label }),
	"amount":  col("amount", func(t *domain.SupplierBillTerm) any { return t.Amount }),
	"dueDate": col("due_date", func(t *domain.SupplierBillTerm) any { return t.DueDate }),
}

// Patch writes the changed fields of a live bill (compare-and-set on
// updated_at when expected is given).
func (r *MySQLSupplierBillRepository) Patch(ctx context.Context, tenantID int64, t *domain.SupplierBillTerm, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "supplier_bill_terms", supplierBillPatchColumns, t, changed,
		"tenant_id = ? AND project_id = ? AND id = ? AND "+liveBillTermWhere, []any{tenantID, t.ProjectID, t.ID}, expected)
}

func (r *MySQLSupplierBillRepository) SoftDelete(ctx context.Context, tenantID, projectID, id int64, at time.Time) (bool, error) {
	res, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE supplier_bill_terms SET deleted_at = ? WHERE tenant_id = ? AND project_id = ? AND id = ? AND deleted_at IS NULL`,
		at, tenantID, projectID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *MySQLSupplierBillRepository) Restore(ctx context.Context, tenantID, projectID, id int64, notBefore time.Time) (bool, error) {
	res, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE supplier_bill_terms SET deleted_at = NULL
		 WHERE tenant_id = ? AND project_id = ? AND id = ? AND deleted_at IS NOT NULL AND deleted_at > ?`,
		tenantID, projectID, id, notBefore)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
