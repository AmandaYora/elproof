package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
)

// MySQLCalendarRepository backs GET /calendar. Every query joins only this
// module's own tables (projects and what hangs off a project).
type MySQLCalendarRepository struct {
	db *sql.DB
}

func NewMySQLCalendarRepository(db *sql.DB) *MySQLCalendarRepository {
	return &MySQLCalendarRepository{db: db}
}

const calendarDay = "2006-01-02"

// calendarProjectFilter is the project filter every calendar query shares (alias
// p): the workspace first (index projects (tenant_id, is_archived, event_date)),
// not archived, not cancelled, then the role's scope and the PIC filter.
func calendarProjectFilter(q application.CalendarQuery) (string, []any) {
	var b strings.Builder
	b.WriteString("p.tenant_id = ? AND p.is_archived = 0 AND p.status <> 'Cancelled'")
	args := []any{q.TenantID}
	if q.AssignedStaffID != nil {
		b.WriteString(" AND (p.pic_staff_id = ? OR p.pic_sales_staff_id = ?)")
		args = append(args, *q.AssignedStaffID, *q.AssignedStaffID)
	}
	if q.PICStaffID != nil {
		b.WriteString(" AND p.pic_staff_id = ?")
		args = append(args, *q.PICStaffID)
	}
	return b.String(), args
}

// ListEventDays is every event of the projects in scope that falls in the
// range (Akad, Resepsi, ...), not only the main one: the title is the kind and,
// when it has one, its venue ("Resepsi di Gedung Melati").
func (r *MySQLCalendarRepository) ListEventDays(ctx context.Context, q application.CalendarQuery) ([]domain.CalendarItem, error) {
	where, args := calendarProjectFilter(q)
	args = append(args, q.TenantID, q.From.Format(calendarDay), q.To.Format(calendarDay))
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.id, p.id, p.name, e.event_date, p.status, e.kind_name, COALESCE(ps.supplier_name, '')
		FROM project_events e
		JOIN projects p ON p.id = e.project_id
		LEFT JOIN project_suppliers ps ON ps.id = e.venue_supplier_id AND ps.deleted_at IS NULL
		WHERE `+where+` AND e.tenant_id = ? AND e.event_date BETWEEN ? AND ?
		ORDER BY e.event_date, e.start_time IS NULL, e.start_time, p.id, e.sort_order, e.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.CalendarItem
	for rows.Next() {
		item := domain.CalendarItem{Kind: domain.CalendarEvent}
		var venue string
		if err := rows.Scan(&item.RefID, &item.ProjectID, &item.ProjectName, &item.Date, &item.Status, &item.Title, &venue); err != nil {
			return nil, err
		}
		if venue != "" {
			item.Title += " di " + venue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *MySQLCalendarRepository) ListOpenMilestones(ctx context.Context, q application.CalendarQuery) ([]domain.CalendarItem, error) {
	where, args := calendarProjectFilter(q)
	args = append(args, q.From.Format(calendarDay), q.To.Format(calendarDay))
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.project_id, p.name, m.name, m.target_date, m.status
		FROM project_milestones m
		JOIN projects p ON p.id = m.project_id
		WHERE `+where+`
		  AND m.status NOT IN ('Completed','Cancelled') AND m.target_date BETWEEN ? AND ?
		ORDER BY m.target_date, m.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.CalendarItem
	for rows.Next() {
		item := domain.CalendarItem{Kind: domain.CalendarMilestone}
		if err := rows.Scan(&item.RefID, &item.ProjectID, &item.ProjectName, &item.Title, &item.Date, &item.Status); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *MySQLCalendarRepository) ListTermsTouching(ctx context.Context, q application.CalendarQuery) ([]domain.TermWithProject, map[int64]int64, error) {
	where, args := calendarProjectFilter(q)
	// The projects (in scope) with an installment due in the range.
	touching := `SELECT DISTINCT tt.project_id FROM client_payment_terms tt
		JOIN projects p ON p.id = tt.project_id
		WHERE ` + where + ` AND tt.due_date BETWEEN ? AND ?`
	touchArgs := append(append([]any{}, args...), q.From.Format(calendarDay), q.To.Format(calendarDay))

	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.project_id, t.label, t.amount, t.due_date, p.name
		FROM client_payment_terms t
		JOIN projects p ON p.id = t.project_id
		WHERE p.tenant_id = ? AND t.project_id IN (`+touching+`)
		ORDER BY t.project_id, t.due_date, t.id`, append([]any{q.TenantID}, touchArgs...)...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var terms []domain.TermWithProject
	for rows.Next() {
		var row domain.TermWithProject
		if err := rows.Scan(&row.Term.ID, &row.Term.ProjectID, &row.Term.Label, &row.Term.Amount, &row.Term.DueDate, &row.ProjectName); err != nil {
			return nil, nil, err
		}
		terms = append(terms, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(terms) == 0 {
		return terms, map[int64]int64{}, nil
	}

	// What each of those projects' clients has paid against the contract (same
	// rule as domain.AppliedToContract: DP + Termin + Pelunasan - Refund, live rows that
	// domain.CountsTowardContract).
	paid, err := r.db.QueryContext(ctx, `
		SELECT cp.project_id,
		       COALESCE(SUM(CASE WHEN cp.type IN ('DP','Termin','Pelunasan') THEN cp.amount
		                         WHEN cp.type = 'Refund' THEN -CAST(cp.amount AS SIGNED) ELSE 0 END), 0)
		FROM client_payments cp
		JOIN projects p ON p.id = cp.project_id
		WHERE p.tenant_id = ? AND cp.deleted_at IS NULL AND `+domain.ContractCountsSQL+` AND cp.project_id IN (`+touching+`)
		GROUP BY cp.project_id`, append([]any{q.TenantID}, touchArgs...)...)
	if err != nil {
		return nil, nil, err
	}
	defer paid.Close()
	applied := make(map[int64]int64)
	for paid.Next() {
		var projectID, total int64
		if err := paid.Scan(&projectID, &total); err != nil {
			return nil, nil, err
		}
		applied[projectID] = total
	}
	return terms, applied, paid.Err()
}
