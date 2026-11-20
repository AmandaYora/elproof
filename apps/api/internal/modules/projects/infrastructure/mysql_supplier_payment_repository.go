package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

// MySQLSupplierPaymentRepository stores supplier_payments: money to a
// project's vendor or venue, from the WO or straight from the client.
type MySQLSupplierPaymentRepository struct {
	db *sql.DB
}

func NewMySQLSupplierPaymentRepository(db *sql.DB) *MySQLSupplierPaymentRepository {
	return &MySQLSupplierPaymentRepository{db: db}
}

const supplierPaymentColumns = `id, project_id, project_supplier_id, IFNULL(bill_term_id, 0), payer, type, amount, payment_date, method, reference_number, notes,
	IFNULL(recorded_by_staff_id, 0), IFNULL(submitted_by_client_id, 0), verified_at, IFNULL(verified_by_staff_id, 0),
	rejected_at, IFNULL(rejected_by_staff_id, 0), IFNULL(rejection_reason, ''), deleted_at, updated_at`

func scanSupplierPayment(scan func(dest ...interface{}) error) (*domain.SupplierPayment, error) {
	var p domain.SupplierPayment
	var payer, paymentType string
	var notes sql.NullString
	var deletedAt, verifiedAt, rejectedAt sql.NullTime
	err := scan(&p.ID, &p.ProjectID, &p.ProjectSupplierID, &p.BillTermID, &payer, &paymentType, &p.Amount, &p.PaymentDate, &p.Method,
		&p.ReferenceNumber, &notes, &p.RecordedByStaffID, &p.SubmittedByClientID, &verifiedAt, &p.VerifiedByStaffID,
		&rejectedAt, &p.RejectedByStaffID, &p.RejectionReason, &deletedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Payer = domain.Payer(payer)
	p.Type = domain.PaymentType(paymentType)
	p.Notes = notes.String
	p.DeletedAt = deletedStamp(deletedAt)
	p.VerifiedAt = deletedStamp(verifiedAt)
	p.RejectedAt = deletedStamp(rejectedAt)
	return &p, nil
}

func (r *MySQLSupplierPaymentRepository) list(ctx context.Context, query string, args ...any) ([]domain.SupplierPayment, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.SupplierPayment
	for rows.Next() {
		p, err := scanSupplierPayment(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func (r *MySQLSupplierPaymentRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.SupplierPayment, error) {
	return r.list(ctx, `SELECT `+supplierPaymentColumns+` FROM supplier_payments WHERE project_id = ? AND `+liveSupplierPaymentWhere+
		` ORDER BY payment_date DESC, id DESC`, projectID)
}

func (r *MySQLSupplierPaymentRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.SupplierPayment, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	return r.list(ctx, `SELECT `+supplierPaymentColumns+` FROM supplier_payments WHERE `+liveSupplierPaymentWhere+
		` AND project_id IN (`+utils.Placeholders(len(projectIDs))+`) ORDER BY project_id, payment_date DESC, id DESC`, utils.Int64Args(projectIDs)...)
}

func (r *MySQLSupplierPaymentRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.SupplierPayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+supplierPaymentColumns+` FROM supplier_payments WHERE project_id = ? AND id = ? AND `+
		liveSupplierPaymentWhere+` LIMIT 1`, projectID, id)
	return scanSupplierPayment(row.Scan)
}

// FindDeletedByID is the restore lookup: the payment while it waits in the
// "deleted" state by its own stamp (one hidden only because its supplier was
// deleted comes back with the supplier).
func (r *MySQLSupplierPaymentRepository) FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.SupplierPayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+supplierPaymentColumns+` FROM supplier_payments WHERE project_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, projectID, id)
	return scanSupplierPayment(row.Scan)
}

func (r *MySQLSupplierPaymentRepository) SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error) {
	return softDeleteRow(ctx, r.db, "supplier_payments", projectID, id, at)
}

func (r *MySQLSupplierPaymentRepository) Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error) {
	return restoreRow(ctx, r.db, "supplier_payments", projectID, id, notBefore)
}

// Create inserts the payment with the project's tenant.
func (r *MySQLSupplierPaymentRepository) Create(ctx context.Context, p *domain.SupplierPayment) error {
	q := conn(ctx, r.db)
	result, err := q.ExecContext(ctx,
		`INSERT INTO supplier_payments (tenant_id, project_id, project_supplier_id, bill_term_id, payer, type, amount, payment_date, method,
		 reference_number, notes, recorded_by_staff_id, submitted_by_client_id)
		 SELECT tenant_id, id, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? FROM projects WHERE id = ?`,
		p.ProjectSupplierID, nullableID(p.BillTermID), string(p.Payer), string(p.Type), p.Amount, p.PaymentDate, p.Method, p.ReferenceNumber, p.Notes,
		nullableID(p.RecordedByStaffID), nullableID(p.SubmittedByClientID), p.ProjectID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return readVersion(ctx, q, "supplier_payments", id, &p.UpdatedAt)
}

var supplierPaymentPatchColumns = map[string]column[domain.SupplierPayment]{
	"type":            col("type", func(p *domain.SupplierPayment) any { return string(p.Type) }),
	"amount":          col("amount", func(p *domain.SupplierPayment) any { return p.Amount }),
	"paymentDate":     col("payment_date", func(p *domain.SupplierPayment) any { return p.PaymentDate }),
	"method":          col("method", func(p *domain.SupplierPayment) any { return p.Method }),
	"referenceNumber": col("reference_number", func(p *domain.SupplierPayment) any { return p.ReferenceNumber }),
	"notes":           col("notes", func(p *domain.SupplierPayment) any { return p.Notes }),
	// "verification" is written together with a material change: it clears both columns.
	"verification": {
		names:  []string{"verified_at", "verified_by_staff_id"},
		values: func(p *domain.SupplierPayment) []any { return []any{p.VerifiedAt, nullableID(p.VerifiedByStaffID)} },
	},
}

func (r *MySQLSupplierPaymentRepository) Patch(ctx context.Context, p *domain.SupplierPayment, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "supplier_payments", supplierPaymentPatchColumns, p, changed,
		"project_id = ? AND id = ? AND "+liveSupplierPaymentWhere, []any{p.ProjectID, p.ID}, expected)
}

// SetVerification marks the client's proof right only while it is still
// unchecked (not verified, not refused), so two staff members at once keep the
// first one's name and time. It reports whether this call set it.
func (r *MySQLSupplierPaymentRepository) SetVerification(ctx context.Context, projectID, id int64, at time.Time, byStaffID int64) (bool, error) {
	res, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE supplier_payments SET verified_at = ?, verified_by_staff_id = ?
		 WHERE project_id = ? AND id = ? AND submitted_by_client_id IS NOT NULL AND verified_at IS NULL AND rejected_at IS NULL
		 AND `+liveSupplierPaymentWhere,
		at, byStaffID, projectID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// SetRejection refuses the client's proof only while it is still unchecked.
func (r *MySQLSupplierPaymentRepository) SetRejection(ctx context.Context, projectID, id int64, at time.Time, byStaffID int64, reason string) (bool, error) {
	res, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE supplier_payments SET rejected_at = ?, rejected_by_staff_id = ?, rejection_reason = ?
		 WHERE project_id = ? AND id = ? AND submitted_by_client_id IS NOT NULL AND verified_at IS NULL AND rejected_at IS NULL
		 AND `+liveSupplierPaymentWhere,
		at, byStaffID, reason, projectID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ClearVerification removes a verification (a no-op when there is none).
func (r *MySQLSupplierPaymentRepository) ClearVerification(ctx context.Context, projectID, id int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE supplier_payments SET verified_at = NULL, verified_by_staff_id = NULL WHERE project_id = ? AND id = ?`,
		projectID, id)
	return err
}
