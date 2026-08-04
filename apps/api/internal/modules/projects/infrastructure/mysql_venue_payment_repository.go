package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLVenuePaymentRepository struct {
	db *sql.DB
}

func NewMySQLVenuePaymentRepository(db *sql.DB) *MySQLVenuePaymentRepository {
	return &MySQLVenuePaymentRepository{db: db}
}

const venuePaymentColumns = `id, project_id, type, amount, payment_date, method, reference_number, notes`

func scanVenuePayment(scan func(dest ...interface{}) error) (*domain.VenuePayment, error) {
	var p domain.VenuePayment
	var paymentType string
	var notes sql.NullString
	err := scan(&p.ID, &p.ProjectID, &paymentType, &p.Amount, &p.PaymentDate, &p.Method, &p.ReferenceNumber, &notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Type = domain.PaymentType(paymentType)
	p.Notes = notes.String
	return &p, nil
}

func (r *MySQLVenuePaymentRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.VenuePayment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+venuePaymentColumns+` FROM venue_payments WHERE project_id = ? ORDER BY payment_date DESC, id DESC`, projectID)
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
	query := `SELECT ` + venuePaymentColumns + ` FROM venue_payments WHERE project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, payment_date DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
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
	row := r.db.QueryRowContext(ctx, `SELECT `+venuePaymentColumns+` FROM venue_payments WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanVenuePayment(row.Scan)
}

func (r *MySQLVenuePaymentRepository) Create(ctx context.Context, p *domain.VenuePayment) error {
	result, err := r.db.ExecContext(ctx,
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
	return nil
}
