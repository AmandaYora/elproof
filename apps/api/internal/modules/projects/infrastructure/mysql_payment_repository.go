package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLPaymentRepository struct {
	db *sql.DB
}

func NewMySQLPaymentRepository(db *sql.DB) *MySQLPaymentRepository {
	return &MySQLPaymentRepository{db: db}
}

const paymentColumns = `id, project_id, project_vendor_id, type, amount, payment_date, method, reference_number, notes, deleted_at, updated_at`

func scanPayment(scan func(dest ...interface{}) error) (*domain.VendorPayment, error) {
	var p domain.VendorPayment
	var paymentType string
	var notes sql.NullString
	var deletedAt sql.NullTime
	err := scan(&p.ID, &p.ProjectID, &p.ProjectVendorID, &paymentType, &p.Amount, &p.PaymentDate, &p.Method, &p.ReferenceNumber, &notes, &deletedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Type = domain.PaymentType(paymentType)
	p.Notes = notes.String
	p.DeletedAt = deletedStamp(deletedAt)
	return &p, nil
}

func (r *MySQLPaymentRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.VendorPayment, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+paymentColumns+` FROM vendor_payments WHERE project_id = ? AND `+liveVendorPaymentWhere+` ORDER BY payment_date DESC, id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VendorPayment
	for rows.Next() {
		p, err := scanPayment(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

// ListByProjects backs ComputeProgressBatch — one query across every id in
// projectIDs instead of one query per project (PLAN.md "Performance
// remediation").
func (r *MySQLPaymentRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.VendorPayment, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + paymentColumns + ` FROM vendor_payments WHERE ` + liveVendorPaymentWhere + ` AND project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, payment_date DESC, id DESC`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VendorPayment
	for rows.Next() {
		p, err := scanPayment(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func (r *MySQLPaymentRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.VendorPayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+paymentColumns+` FROM vendor_payments WHERE project_id = ? AND id = ? AND `+liveVendorPaymentWhere+` LIMIT 1`, projectID, id)
	return scanPayment(row.Scan)
}

// FindDeletedByID is the restore lookup: the payment while it waits in the
// "deleted" state (its own stamp; a payment hidden only because its engagement
// was deleted is restored with the engagement).
func (r *MySQLPaymentRepository) FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.VendorPayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+paymentColumns+` FROM vendor_payments WHERE project_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, projectID, id)
	return scanPayment(row.Scan)
}

// SoftDelete hides the payment until it is restored or purged.
func (r *MySQLPaymentRepository) SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error) {
	return softDeleteRow(ctx, r.db, "vendor_payments", projectID, id, at)
}

// Restore brings back a payment deleted after notBefore.
func (r *MySQLPaymentRepository) Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error) {
	return restoreRow(ctx, r.db, "vendor_payments", projectID, id, notBefore)
}

func (r *MySQLPaymentRepository) Create(ctx context.Context, p *domain.VendorPayment) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO vendor_payments (project_id, project_vendor_id, type, amount, payment_date, method, reference_number, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ProjectID, p.ProjectVendorID, string(p.Type), p.Amount, p.PaymentDate, p.Method, p.ReferenceNumber, p.Notes,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

var vendorPaymentPatchColumns = map[string]column[domain.VendorPayment]{
	"type":            col("type", func(p *domain.VendorPayment) any { return string(p.Type) }),
	"amount":          col("amount", func(p *domain.VendorPayment) any { return p.Amount }),
	"paymentDate":     col("payment_date", func(p *domain.VendorPayment) any { return p.PaymentDate }),
	"method":          col("method", func(p *domain.VendorPayment) any { return p.Method }),
	"referenceNumber": col("reference_number", func(p *domain.VendorPayment) any { return p.ReferenceNumber }),
	"notes":           col("notes", func(p *domain.VendorPayment) any { return p.Notes }),
}

// Patch writes the changed fields of a live vendor payment (see patchRow).
func (r *MySQLPaymentRepository) Patch(ctx context.Context, p *domain.VendorPayment, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "vendor_payments", vendorPaymentPatchColumns, p, changed,
		"project_id = ? AND id = ? AND "+liveVendorPaymentWhere, []any{p.ProjectID, p.ID}, expected)
}
