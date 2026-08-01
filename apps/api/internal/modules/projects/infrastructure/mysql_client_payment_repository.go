package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLClientPaymentRepository struct {
	db *sql.DB
}

func NewMySQLClientPaymentRepository(db *sql.DB) *MySQLClientPaymentRepository {
	return &MySQLClientPaymentRepository{db: db}
}

const clientPaymentColumns = `id, project_id, type, amount, payment_date, method, reference_number, notes`

func scanClientPayment(scan func(dest ...interface{}) error) (*domain.ClientPayment, error) {
	var p domain.ClientPayment
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

func (r *MySQLClientPaymentRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.ClientPayment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+clientPaymentColumns+` FROM client_payments WHERE project_id = ? ORDER BY payment_date DESC, id DESC`, projectID)
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
	row := r.db.QueryRowContext(ctx, `SELECT `+clientPaymentColumns+` FROM client_payments WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanClientPayment(row.Scan)
}

func (r *MySQLClientPaymentRepository) Create(ctx context.Context, p *domain.ClientPayment) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO client_payments (project_id, type, amount, payment_date, method, reference_number, notes)
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
