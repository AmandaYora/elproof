package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
)

type MySQLClientPaymentRepository struct {
	db *sql.DB
}

func NewMySQLClientPaymentRepository(db *sql.DB) *MySQLClientPaymentRepository {
	return &MySQLClientPaymentRepository{db: db}
}

const clientPaymentColumns = `id, project_id, type, amount, payment_date, method, reference_number, notes,
	IFNULL(recorded_by_staff_id, 0), verified_at, IFNULL(verified_by_staff_id, 0), deleted_at, updated_at`

func scanClientPayment(scan func(dest ...interface{}) error) (*domain.ClientPayment, error) {
	var p domain.ClientPayment
	var paymentType string
	var notes sql.NullString
	var verifiedAt, deletedAt sql.NullTime
	err := scan(&p.ID, &p.ProjectID, &paymentType, &p.Amount, &p.PaymentDate, &p.Method, &p.ReferenceNumber, &notes,
		&p.RecordedByStaffID, &verifiedAt, &p.VerifiedByStaffID, &deletedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Type = domain.PaymentType(paymentType)
	p.Notes = notes.String
	if verifiedAt.Valid {
		t := verifiedAt.Time
		p.VerifiedAt = &t
	}
	p.DeletedAt = deletedStamp(deletedAt)
	return &p, nil
}

func (r *MySQLClientPaymentRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.ClientPayment, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+clientPaymentColumns+` FROM client_payments WHERE project_id = ? AND deleted_at IS NULL ORDER BY payment_date DESC, id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.ClientPayment
	for rows.Next() {
		p, err := scanClientPayment(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func (r *MySQLClientPaymentRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.ClientPayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+clientPaymentColumns+` FROM client_payments WHERE project_id = ? AND id = ? AND deleted_at IS NULL LIMIT 1`, projectID, id)
	return scanClientPayment(row.Scan)
}

// FindDeletedByID is the restore lookup (nil unless the payment waits in the
// "deleted" state).
func (r *MySQLClientPaymentRepository) FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.ClientPayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+clientPaymentColumns+` FROM client_payments WHERE project_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, projectID, id)
	return scanClientPayment(row.Scan)
}

// SoftDelete hides the payment until it is restored or purged.
func (r *MySQLClientPaymentRepository) SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error) {
	return softDeleteRow(ctx, r.db, "client_payments", projectID, id, at)
}

// Restore brings back a payment deleted after notBefore.
func (r *MySQLClientPaymentRepository) Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error) {
	return restoreRow(ctx, r.db, "client_payments", projectID, id, notBefore)
}

func (r *MySQLClientPaymentRepository) Create(ctx context.Context, p *domain.ClientPayment) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO client_payments (project_id, type, amount, payment_date, method, reference_number, notes, recorded_by_staff_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ProjectID, string(p.Type), p.Amount, p.PaymentDate, p.Method, p.ReferenceNumber, p.Notes, nullableID(p.RecordedByStaffID),
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

var clientPaymentPatchColumns = map[string]column[domain.ClientPayment]{
	"type":            col("type", func(p *domain.ClientPayment) any { return string(p.Type) }),
	"amount":          col("amount", func(p *domain.ClientPayment) any { return p.Amount }),
	"paymentDate":     col("payment_date", func(p *domain.ClientPayment) any { return p.PaymentDate }),
	"method":          col("method", func(p *domain.ClientPayment) any { return p.Method }),
	"referenceNumber": col("reference_number", func(p *domain.ClientPayment) any { return p.ReferenceNumber }),
	"notes":           col("notes", func(p *domain.ClientPayment) any { return p.Notes }),
	// "verification" is written together with a material change: it clears both columns.
	"verification": {
		names:  []string{"verified_at", "verified_by_staff_id"},
		values: func(p *domain.ClientPayment) []any { return []any{p.VerifiedAt, nullableID(p.VerifiedByStaffID)} },
	},
}

// Patch writes the changed fields of a live client payment (see patchRow).
func (r *MySQLClientPaymentRepository) Patch(ctx context.Context, p *domain.ClientPayment, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "client_payments", clientPaymentPatchColumns, p, changed,
		"project_id = ? AND id = ? AND deleted_at IS NULL", []any{p.ProjectID, p.ID}, expected)
}

// SetVerification records the verification only when the payment is not yet
// verified, so two staff members verifying at once keep the first one's name
// and time (idempotent). It reports whether this call was the one that set it.
func (r *MySQLClientPaymentRepository) SetVerification(ctx context.Context, projectID, id int64, at time.Time, byStaffID int64) (bool, error) {
	res, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE client_payments SET verified_at = ?, verified_by_staff_id = ?
		 WHERE project_id = ? AND id = ? AND verified_at IS NULL`,
		at, byStaffID, projectID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ClearVerification removes a verification (a no-op when there is none).
func (r *MySQLClientPaymentRepository) ClearVerification(ctx context.Context, projectID, id int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE client_payments SET verified_at = NULL, verified_by_staff_id = NULL WHERE project_id = ? AND id = ?`,
		projectID, id)
	return err
}

// nullableID stores the "no staff member" sentinel 0 as NULL.
func nullableID(id int64) interface{} {
	if id == 0 {
		return nil
	}
	return id
}
