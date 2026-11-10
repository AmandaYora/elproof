package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLVenuePaymentRepository struct {
	db *sql.DB
}

func NewMySQLVenuePaymentRepository(db *sql.DB) *MySQLVenuePaymentRepository {
	return &MySQLVenuePaymentRepository{db: db}
}

const venuePaymentColumns = `id, project_id, type, amount, payment_date, method, reference_number, notes, deleted_at, updated_at`

func scanVenuePayment(scan func(dest ...interface{}) error) (*domain.VenuePayment, error) {
	var p domain.VenuePayment
	var paymentType string
	var notes sql.NullString
	var deletedAt sql.NullTime
	err := scan(&p.ID, &p.ProjectID, &paymentType, &p.Amount, &p.PaymentDate, &p.Method, &p.ReferenceNumber, &notes, &deletedAt, &p.UpdatedAt)
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

func (r *MySQLVenuePaymentRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.VenuePayment, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+venuePaymentColumns+` FROM venue_payments WHERE project_id = ? AND deleted_at IS NULL ORDER BY payment_date DESC, id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VenuePayment
	for rows.Next() {
		p, err := scanVenuePayment(rows.Scan)
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
func (r *MySQLVenuePaymentRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.VenuePayment, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + venuePaymentColumns + ` FROM venue_payments WHERE deleted_at IS NULL AND project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, payment_date DESC, id DESC`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VenuePayment
	for rows.Next() {
		p, err := scanVenuePayment(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func (r *MySQLVenuePaymentRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.VenuePayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+venuePaymentColumns+` FROM venue_payments WHERE project_id = ? AND id = ? AND deleted_at IS NULL LIMIT 1`, projectID, id)
	return scanVenuePayment(row.Scan)
}

// FindDeletedByID is the restore lookup (nil unless the payment waits in the
// "deleted" state).
func (r *MySQLVenuePaymentRepository) FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.VenuePayment, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+venuePaymentColumns+` FROM venue_payments WHERE project_id = ? AND id = ? AND deleted_at IS NOT NULL LIMIT 1`, projectID, id)
	return scanVenuePayment(row.Scan)
}

// SoftDelete hides the payment until it is restored or purged.
func (r *MySQLVenuePaymentRepository) SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error) {
	return softDeleteRow(ctx, r.db, "venue_payments", projectID, id, at)
}

// Restore brings back a payment deleted after notBefore.
func (r *MySQLVenuePaymentRepository) Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error) {
	return restoreRow(ctx, r.db, "venue_payments", projectID, id, notBefore)
}

func (r *MySQLVenuePaymentRepository) Create(ctx context.Context, p *domain.VenuePayment) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO venue_payments (project_id, type, amount, payment_date, method, reference_number, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ProjectID, string(p.Type), p.Amount, p.PaymentDate, p.Method, p.ReferenceNumber, p.Notes,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return readVersion(ctx, conn(ctx, r.db), "venue_payments", id, &p.UpdatedAt)
}

var venuePaymentPatchColumns = map[string]column[domain.VenuePayment]{
	"type":            col("type", func(p *domain.VenuePayment) any { return string(p.Type) }),
	"amount":          col("amount", func(p *domain.VenuePayment) any { return p.Amount }),
	"paymentDate":     col("payment_date", func(p *domain.VenuePayment) any { return p.PaymentDate }),
	"method":          col("method", func(p *domain.VenuePayment) any { return p.Method }),
	"referenceNumber": col("reference_number", func(p *domain.VenuePayment) any { return p.ReferenceNumber }),
	"notes":           col("notes", func(p *domain.VenuePayment) any { return p.Notes }),
}

// Patch writes the changed fields of a live venue payment (see patchRow).
func (r *MySQLVenuePaymentRepository) Patch(ctx context.Context, p *domain.VenuePayment, changed []string, expected *time.Time) (bool, error) {
	return patchRow(ctx, conn(ctx, r.db), "venue_payments", venuePaymentPatchColumns, p, changed,
		"project_id = ? AND id = ? AND deleted_at IS NULL", []any{p.ProjectID, p.ID}, expected)
}
