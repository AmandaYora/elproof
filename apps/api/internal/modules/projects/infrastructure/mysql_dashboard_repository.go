package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
)

// MySQLDashboardRepository backs the tenant-wide aggregation queries the WO
// dashboard needs. Every query here joins only within this module's own
// tables (projects + its sub-entities) to resolve tenant scoping and vendor
// IDs — no cross-module join.
type MySQLDashboardRepository struct {
	db *sql.DB
}

func NewMySQLDashboardRepository(db *sql.DB) *MySQLDashboardRepository {
	return &MySQLDashboardRepository{db: db}
}

func (r *MySQLDashboardRepository) CountActiveVendors(ctx context.Context, tenantID int64) (int, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT pv.vendor_id)
		FROM project_vendors pv
		JOIN projects p ON p.id = pv.project_id
		WHERE p.tenant_id = ? AND p.status NOT IN ('Completed','Cancelled') AND pv.engagement_status != 'Cancelled'`,
		tenantID,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *MySQLDashboardRepository) ListOpenIssues(ctx context.Context, tenantID int64) ([]domain.DashboardIssueRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT vi.id, vi.project_id, vi.project_vendor_id, vi.title, vi.description, vi.impact, vi.found_date, vi.status,
		       vi.resolution_plan, vi.pic_staff_id, vi.target_resolution_date, vi.resolved_date, vi.resolution_notes,
		       p.name, pv.vendor_id
		FROM vendor_issues vi
		JOIN projects p ON p.id = vi.project_id
		JOIN project_vendors pv ON pv.id = vi.project_vendor_id
		WHERE p.tenant_id = ? AND vi.status NOT IN ('Resolved','Closed')
		ORDER BY vi.found_date DESC, vi.id DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.DashboardIssueRow
	for rows.Next() {
		var row domain.DashboardIssueRow
		var impact, status string
		var resolutionPlan, resolutionNotes sql.NullString
		var targetResolutionDate, resolvedDate sql.NullTime
		if err := rows.Scan(&row.Issue.ID, &row.Issue.ProjectID, &row.Issue.ProjectVendorID, &row.Issue.Title, &row.Issue.Description,
			&impact, &row.Issue.FoundDate, &status, &resolutionPlan, &row.Issue.PICStaffID, &targetResolutionDate, &resolvedDate,
			&resolutionNotes, &row.ProjectName, &row.VendorID); err != nil {
			return nil, err
		}
		row.Issue.Impact = domain.IssueImpact(impact)
		row.Issue.Status = domain.IssueStatus(status)
		row.Issue.ResolutionPlan = resolutionPlan.String
		row.Issue.ResolutionNotes = resolutionNotes.String
		if targetResolutionDate.Valid {
			row.Issue.TargetResolutionDate = &targetResolutionDate.Time
		}
		if resolvedDate.Valid {
			row.Issue.ResolvedDate = &resolvedDate.Time
		}
		row.ProjectID = row.Issue.ProjectID
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *MySQLDashboardRepository) ListOverdueVendorMilestones(ctx context.Context, tenantID int64, asOf time.Time) ([]domain.DashboardMilestoneRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT vm.id, vm.project_vendor_id, vm.sort_order, vm.name, vm.description, vm.status, vm.target_date, vm.completed_date,
		       vm.pic_staff_id, vm.notes, pv.project_id, p.name, pv.vendor_id
		FROM vendor_milestones vm
		JOIN project_vendors pv ON pv.id = vm.project_vendor_id
		JOIN projects p ON p.id = pv.project_id
		WHERE p.tenant_id = ? AND vm.target_date < ? AND vm.status NOT IN ('Completed','Cancelled')
		ORDER BY vm.target_date ASC`,
		tenantID, asOf,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.DashboardMilestoneRow
	for rows.Next() {
		var row domain.DashboardMilestoneRow
		var status string
		var description, notes sql.NullString
		var completedDate sql.NullTime
		if err := rows.Scan(&row.Milestone.ID, &row.Milestone.ProjectVendorID, &row.Milestone.SortOrder, &row.Milestone.Name,
			&description, &status, &row.Milestone.TargetDate, &completedDate, &row.Milestone.PICStaffID, &notes,
			&row.ProjectID, &row.ProjectName, &row.VendorID); err != nil {
			return nil, err
		}
		row.Milestone.Status = domain.MilestoneStatus(status)
		row.Milestone.Description = description.String
		row.Milestone.Notes = notes.String
		if completedDate.Valid {
			row.Milestone.CompletedDate = &completedDate.Time
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *MySQLDashboardRepository) ListPaymentCandidates(ctx context.Context, tenantID int64) ([]domain.DashboardPaymentRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT vp.id, vp.project_id, vp.project_vendor_id, vp.type, vp.amount, vp.payment_date, vp.method, vp.reference_number,
		       vp.invoice_evidence_id, vp.proof_evidence_id, vp.notes, p.name, pv.vendor_id
		FROM vendor_payments vp
		JOIN projects p ON p.id = vp.project_id
		JOIN project_vendors pv ON pv.id = vp.project_vendor_id
		WHERE p.tenant_id = ?
		ORDER BY vp.payment_date DESC, vp.id DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.DashboardPaymentRow
	for rows.Next() {
		var row domain.DashboardPaymentRow
		var paymentType string
		var invoiceEvidenceID, proofEvidenceID sql.NullInt64
		var notes sql.NullString
		if err := rows.Scan(&row.Payment.ID, &row.Payment.ProjectID, &row.Payment.ProjectVendorID, &paymentType, &row.Payment.Amount,
			&row.Payment.PaymentDate, &row.Payment.Method, &row.Payment.ReferenceNumber, &invoiceEvidenceID, &proofEvidenceID,
			&notes, &row.ProjectName, &row.VendorID); err != nil {
			return nil, err
		}
		row.Payment.Type = domain.PaymentType(paymentType)
		row.Payment.Notes = notes.String
		if invoiceEvidenceID.Valid {
			row.Payment.InvoiceEvidenceID = &invoiceEvidenceID.Int64
		}
		if proofEvidenceID.Valid {
			row.Payment.ProofEvidenceID = &proofEvidenceID.Int64
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *MySQLDashboardRepository) ListRecentActivity(ctx context.Context, tenantID int64, limit int) ([]domain.ActivityLogEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT al.id, al.project_id, al.type, al.actor_staff_id, al.entity_type, al.entity_id, al.entity_label, al.description, al.created_at
		FROM activity_log al
		JOIN projects p ON p.id = al.project_id
		WHERE p.tenant_id = ?
		ORDER BY al.created_at DESC, al.id DESC
		LIMIT ?`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.ActivityLogEntry
	for rows.Next() {
		var e domain.ActivityLogEntry
		var projectID sql.NullInt64
		var activityType string
		if err := rows.Scan(&e.ID, &projectID, &activityType, &e.ActorStaffID, &e.EntityType, &e.EntityID, &e.EntityLabel, &e.Description, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Type = domain.ActivityType(activityType)
		if projectID.Valid {
			e.ProjectID = &projectID.Int64
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
