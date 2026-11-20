package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

// openClientPaidSupplierWhere: a live vendor or venue the client pays directly,
// of an open project of the workspace (s = project_suppliers, p = projects).
const openClientPaidSupplierWhere = `p.tenant_id = ? AND p.is_archived = 0 AND p.status NOT IN ('Completed','Cancelled')
	  AND s.deleted_at IS NULL AND s.paid_by = 'Client' AND s.engagement_status != 'Cancelled'`

// ListClientBillTerms is every live bill of a client-paid supplier of an open
// project (Paket WO Tahap 4), for the dashboard's "tagihan vendor dibayar klien".
func (r *MySQLDashboardRepository) ListClientBillTerms(ctx context.Context, tenantID int64) ([]domain.ClientBillTermRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT bt.id, bt.project_id, bt.project_supplier_id, bt.label, bt.amount, bt.due_date, bt.sort_order, bt.updated_at,
		       p.name, s.supplier_kind, s.supplier_name
		FROM supplier_bill_terms bt
		JOIN project_suppliers s ON s.id = bt.project_supplier_id AND s.tenant_id = bt.tenant_id
		JOIN projects p ON p.id = bt.project_id AND p.tenant_id = bt.tenant_id
		WHERE bt.tenant_id = ? AND bt.deleted_at IS NULL AND `+openClientPaidSupplierWhere+`
		ORDER BY bt.due_date, bt.sort_order, bt.id`, tenantID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.ClientBillTermRow
	for rows.Next() {
		var row domain.ClientBillTermRow
		var kind string
		t := &row.Term
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ProjectSupplierID, &t.Label, &t.Amount, &t.DueDate, &t.SortOrder, &t.UpdatedAt,
			&row.ProjectName, &kind, &row.SupplierName); err != nil {
			return nil, err
		}
		row.SupplierKind = domain.SupplierKind(kind)
		list = append(list, row)
	}
	return list, rows.Err()
}

// ListClientSupplierPayments is every live payment by the client to a
// client-paid supplier of an open project, with what decides whether it counts
// (CountsAsPaid) or awaits review. Never the WO's money.
func (r *MySQLDashboardRepository) ListClientSupplierPayments(ctx context.Context, tenantID int64) ([]domain.SupplierPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sp.id, sp.project_id, sp.project_supplier_id, IFNULL(sp.bill_term_id, 0), sp.type, sp.amount, sp.payment_date,
		       IFNULL(sp.submitted_by_client_id, 0), sp.verified_at, sp.rejected_at
		FROM supplier_payments sp
		JOIN project_suppliers s ON s.id = sp.project_supplier_id AND s.tenant_id = sp.tenant_id
		JOIN projects p ON p.id = sp.project_id AND p.tenant_id = sp.tenant_id
		WHERE sp.tenant_id = ? AND sp.payer = 'Client' AND sp.deleted_at IS NULL AND `+openClientPaidSupplierWhere+`
		ORDER BY sp.id`, tenantID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.SupplierPayment
	for rows.Next() {
		var p domain.SupplierPayment
		var paymentType string
		var verified, rejected sql.NullTime
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.ProjectSupplierID, &p.BillTermID, &paymentType, &p.Amount, &p.PaymentDate,
			&p.SubmittedByClientID, &verified, &rejected); err != nil {
			return nil, err
		}
		p.Payer = domain.PayerClient
		p.Type = domain.PaymentType(paymentType)
		p.VerifiedAt, p.RejectedAt = deletedStamp(verified), deletedStamp(rejected)
		list = append(list, p)
	}
	return list, rows.Err()
}

// pendingSupplierProofFrom: the client's proofs to a vendor it pays directly
// that wait for the WO's check, with a file (the verification queue's rule).
const pendingSupplierProofFrom = `
		FROM supplier_payments sp
		JOIN projects p ON p.id = sp.project_id AND p.tenant_id = sp.tenant_id
		JOIN project_suppliers s ON s.id = sp.project_supplier_id AND s.tenant_id = sp.tenant_id AND s.deleted_at IS NULL
		LEFT JOIN supplier_bill_terms bt ON bt.id = sp.bill_term_id AND bt.tenant_id = sp.tenant_id AND bt.deleted_at IS NULL
		WHERE sp.tenant_id = ? AND sp.payer = 'Client' AND sp.deleted_at IS NULL AND ` + domain.SupplierPaymentPendingSQL + `
		  AND EXISTS (SELECT 1 FROM evidence e WHERE e.project_id = sp.project_id AND e.related_kind = 'supplierPayment' AND e.related_id = sp.id)`

// ListPendingSupplierProofs is the oldest `limit` proofs waiting for the WO's
// check, and how many wait in all.
func (r *MySQLDashboardRepository) ListPendingSupplierProofs(ctx context.Context, tenantID int64, limit int) ([]domain.SupplierProofPending, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+pendingSupplierProofFrom, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT sp.id, sp.project_id, p.name, sp.project_supplier_id, s.supplier_kind, s.supplier_name,
		       IFNULL(bt.id, 0), IFNULL(bt.label, ''), sp.type, sp.amount, sp.payment_date, sp.created_at`+pendingSupplierProofFrom+`
		ORDER BY sp.created_at, sp.id
		LIMIT ?`, tenantID, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []domain.SupplierProofPending
	for rows.Next() {
		var row domain.SupplierProofPending
		var kind, paymentType string
		if err := rows.Scan(&row.SupplierPaymentID, &row.ProjectID, &row.ProjectName, &row.ProjectSupplierID, &kind, &row.SupplierName,
			&row.BillTermID, &row.BillTermLabel, &paymentType, &row.Amount, &row.PaymentDate, &row.SubmittedAt); err != nil {
			return nil, 0, err
		}
		row.SupplierKind = domain.SupplierKind(kind)
		row.Type = domain.PaymentType(paymentType)
		list = append(list, row)
	}
	return list, total, rows.Err()
}
