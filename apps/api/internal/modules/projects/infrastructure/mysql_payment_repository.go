package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLPaymentRepository struct {
	db *sql.DB
}

func NewMySQLPaymentRepository(db *sql.DB) *MySQLPaymentRepository {
	return &MySQLPaymentRepository{db: db}
}

const paymentColumns = `id, project_id, project_vendor_id, type, amount, payment_date, method, reference_number,
	invoice_evidence_id, proof_evidence_id, notes`

func scanPayment(scan func(dest ...interface{}) error) (*domain.VendorPayment, error) {
	var p domain.VendorPayment
	var paymentType string
	var invoiceEvidenceID, proofEvidenceID sql.NullInt64
	var notes sql.NullString
	err := scan(&p.ID, &p.ProjectID, &p.ProjectVendorID, &paymentType, &p.Amount, &p.PaymentDate, &p.Method, &p.ReferenceNumber,
		&invoiceEvidenceID, &proofEvidenceID, &notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Type = domain.PaymentType(paymentType)
	p.Notes = notes.String
	if invoiceEvidenceID.Valid {
		p.InvoiceEvidenceID = &invoiceEvidenceID.Int64
	}
	if proofEvidenceID.Valid {
		p.ProofEvidenceID = &proofEvidenceID.Int64
	}
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
