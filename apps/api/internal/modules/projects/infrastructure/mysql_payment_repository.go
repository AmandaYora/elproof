package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLPaymentRepository struct {
	db *sql.DB
}

func NewMySQLPaymentRepository(db *sql.DB) *MySQLPaymentRepository {
	return &MySQLPaymentRepository{db: db}
}

const paymentColumns = `id, project_id, project_vendor_id, type, amount, payment_date, method, reference_number, notes`

func scanPayment(scan func(dest ...interface{}) error) (*domain.VendorPayment, error) {
	var p domain.VendorPayment
	var paymentType string
	var notes sql.NullString
	err := scan(&p.ID, &p.ProjectID, &p.ProjectVendorID, &paymentType, &p.Amount, &p.PaymentDate, &p.Method, &p.ReferenceNumber, &notes)
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

func (r *MySQLPaymentRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.VendorPayment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+paymentColumns+` FROM vendor_payments WHERE project_id = ? ORDER BY payment_date DESC, id DESC`, projectID)
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
	query := `SELECT ` + paymentColumns + ` FROM vendor_payments WHERE project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, payment_date DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
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
	row := r.db.QueryRowContext(ctx, `SELECT `+paymentColumns+` FROM vendor_payments WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanPayment(row.Scan)
}

func (r *MySQLPaymentRepository) Create(ctx context.Context, p *domain.VendorPayment) error {
	result, err := r.db.ExecContext(ctx,
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
