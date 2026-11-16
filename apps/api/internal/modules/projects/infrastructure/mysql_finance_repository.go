package infrastructure

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

// MySQLFinanceRepository backs the Keuangan page (/finance/*). Every query reads
// only this module's tables: projects, client_payment_terms, client_payments,
// project_vendors, vendor_payments, venue_payments and evidence. Lists are
// filtered, sorted and paged in SQL; the installment allocation is reproduced in
// SQL with the rule of domain.AllocateTerms (payments spread over the schedule,
// earliest due date first, then id), the way projectNextDueSQL already does for
// the project list. The isolation suite checks the two agree.
//
// Indexes: every query starts at projects (tenant_id …) and reaches the child
// rows by their project_id / project_vendor_id foreign-key indexes; the
// correlated sums are per project, bounded by one workspace. No new index.
type MySQLFinanceRepository struct {
	db *sql.DB
}

func NewMySQLFinanceRepository(db *sql.DB) *MySQLFinanceRepository {
	return &MySQLFinanceRepository{db: db}
}

const dateFmt = "2006-01-02"

// financeProjectWhere is the project half of every finance query: the
// workspace, not archived, not cancelled, and — for a role limited to its own
// projects — only where the person is PIC or PIC Sales.
func financeProjectWhere(scope domain.FinanceScope) (string, []interface{}) {
	where := `p.tenant_id = ? AND p.is_archived = 0 AND p.status <> 'Cancelled'`
	args := []interface{}{scope.TenantID}
	if scope.HolderStaffID != nil {
		where += ` AND (p.pic_staff_id = ? OR p.pic_sales_staff_id = ?)`
		args = append(args, *scope.HolderStaffID, *scope.HolderStaffID)
	}
	return where, args
}

// clientAppliedSQL: what the client has paid towards the contract of project
// t.project_id (DP + Termin + Pelunasan - Refund over live payments, never
// negative, only payments that domain.CountsTowardContract), as
// domain.AppliedToContract.
const clientAppliedSQL = `GREATEST(COALESCE((SELECT SUM(CASE WHEN cp.type IN ('DP','Termin','Pelunasan') THEN CAST(cp.amount AS SIGNED)
		WHEN cp.type = 'Refund' THEN -CAST(cp.amount AS SIGNED) ELSE 0 END)
	FROM client_payments cp WHERE cp.project_id = t.project_id AND cp.deleted_at IS NULL AND ` + domain.ContractCountsSQL + `), 0), 0)`

// termsBeforeSQL: the installments of the same schedule that come before t.
const termsBeforeSQL = `COALESCE((SELECT SUM(t2.amount) FROM client_payment_terms t2 WHERE t2.project_id = t.project_id
	AND (t2.due_date < t.due_date OR (t2.due_date = t.due_date AND t2.id < t.id))), 0)`

// receivableRowsSQL is one row per installment with what the payments cover of
// it: paid = min(amount, max(applied - before, 0)), remaining = amount - paid,
// and the status of domain.TermStatusOf (two placeholders: today, today+7).
func receivableRowsSQL(scope domain.FinanceScope, f domain.ReceivableFilter, today time.Time) (string, []interface{}) {
	where, args := financeProjectWhere(scope)
	if f.Search != "" {
		like := "%" + utils.EscapeLike(f.Search) + "%"
		where += ` AND (p.name LIKE ? OR p.bride_name LIKE ? OR p.groom_name LIKE ? OR t.label LIKE ?)`
		args = append(args, like, like, like, like)
	}
	if f.PICStaffID != nil {
		where += ` AND p.pic_staff_id = ?`
		args = append(args, *f.PICStaffID)
	}
	if f.DueFrom != nil {
		where += ` AND t.due_date >= ?`
		args = append(args, f.DueFrom.Format(dateFmt))
	}
	if f.DueTo != nil {
		where += ` AND t.due_date <= ?`
		args = append(args, f.DueTo.Format(dateFmt))
	}
	base := `SELECT t.id, t.project_id, t.label, CAST(t.amount AS SIGNED) AS amount, t.due_date,
			p.name AS project_name, p.bride_name, p.groom_name, p.pic_staff_id,
			CAST(LEAST(CAST(t.amount AS SIGNED), GREATEST(` + clientAppliedSQL + ` - ` + termsBeforeSQL + `, 0)) AS SIGNED) AS paid
		FROM client_payment_terms t JOIN projects p ON p.id = t.project_id
		WHERE ` + where
	rows := `SELECT r.*, r.amount - r.paid AS remaining,
			CASE WHEN r.amount - r.paid <= 0 THEN 'paid' WHEN r.due_date < ? THEN 'overdue' WHEN r.paid > 0 THEN 'partial'
			     WHEN r.due_date <= ? THEN 'due_soon' ELSE 'upcoming' END AS status
		FROM (` + base + `) r`
	day := dayOf(today)
	return rows, append([]interface{}{day.Format(dateFmt), day.AddDate(0, 0, 7).Format(dateFmt)}, args...)
}

// receivableViewWhere is the condition of a view on the rows above (aliased z).
func receivableViewWhere(view domain.ReceivableView, today time.Time) (string, []interface{}) {
	day := dayOf(today)
	switch view {
	case domain.ReceivableAll:
		return "", nil
	case domain.ReceivableUnpaid:
		return ` WHERE z.remaining > 0`, nil
	case domain.ReceivableDue30:
		return ` WHERE z.remaining > 0 AND z.due_date >= ? AND z.due_date <= ?`,
			[]interface{}{day.Format(dateFmt), day.AddDate(0, 0, domain.FinanceHorizonDays).Format(dateFmt)}
	default:
		return ` WHERE z.status = ?`, []interface{}{string(view)}
	}
}

var receivableSortColumns = map[string]pagination.SortColumn{
	"dueDate":     pagination.Col("z.due_date"),
	"remaining":   pagination.Col("z.remaining"),
	"amount":      pagination.Col("z.amount"),
	"projectName": pagination.Col("z.project_name"),
}

// ReceivableSortSpec: earliest due date first by default (what to chase next).
func (r *MySQLFinanceRepository) ReceivableSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return receivableSortColumns, pagination.By("dueDate", false)
}

const receivableColumns = `z.id, z.project_id, z.label, z.amount, z.due_date, z.project_name, z.bride_name, z.groom_name, z.pic_staff_id, z.paid, z.remaining`

func scanReceivable(scan func(...interface{}) error) (domain.ReceivableRow, error) {
	var row domain.ReceivableRow
	err := scan(&row.Term.ID, &row.Term.ProjectID, &row.Term.Label, &row.Term.Amount, &row.Term.DueDate,
		&row.ProjectName, &row.BrideName, &row.GroomName, &row.PICStaffID, &row.Paid, &row.Remaining)
	return row, err
}

func (r *MySQLFinanceRepository) UnpaidReceivables(ctx context.Context, scope domain.FinanceScope) ([]domain.ReceivableRow, error) {
	// The status is not read here (the caller derives it), so "today" only
	// feeds the unused CASE: any date will do.
	inner, args := receivableRowsSQL(scope, domain.ReceivableFilter{}, time.Now())
	q := `SELECT ` + receivableColumns + ` FROM (` + inner + `) z WHERE z.remaining > 0 ORDER BY z.due_date, z.id`
	return r.queryReceivables(ctx, q, args)
}

func (r *MySQLFinanceRepository) ListReceivables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.ReceivableFilter, params pagination.Params, sort pagination.Sort) ([]domain.ReceivableRow, int64, error) {
	inner, args := receivableRowsSQL(scope, f, today)
	viewWhere, viewArgs := receivableViewWhere(f.View, today)
	args = append(args, viewArgs...)
	from := ` FROM (` + inner + `) z` + viewWhere
	var total int64
	if err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT ` + receivableColumns + from + sort.OrderBy("z.id") + ` LIMIT ? OFFSET ?`
	list, err := r.queryReceivables(ctx, q, append(args, params.Limit, params.Offset()))
	return list, total, err
}

func (r *MySQLFinanceRepository) queryReceivables(ctx context.Context, q string, args []interface{}) ([]domain.ReceivableRow, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.ReceivableRow
	for rows.Next() {
		row, err := scanReceivable(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// payableRowsSQL is one row per amount the WO owes: each live, not cancelled
// vendor engagement with a price, and each project's venue with a cost (rental
// price + charge), with what its own live payments applied (DP + Termin +
// Pelunasan - Refund, never negative), the remaining amount and the status of
// domain.PayableStatusOf (one placeholder: today).
func payableRowsSQL(scope domain.FinanceScope, today time.Time) (string, []interface{}) {
	where, whereArgs := financeProjectWhere(scope)
	vendorPart := `SELECT 'vendor' AS kind, pv.id AS item_id, pv.project_id, p.name AS project_name, p.bride_name, p.groom_name, p.pic_staff_id,
			pv.vendor_id, pv.category_id, '' AS venue_name, CAST(pv.contract_value AS SIGNED) AS contract_value,
			CAST(GREATEST(COALESCE((SELECT SUM(CASE WHEN vp.type IN ('DP','Termin','Pelunasan') THEN CAST(vp.amount AS SIGNED)
					WHEN vp.type = 'Refund' THEN -CAST(vp.amount AS SIGNED) ELSE 0 END)
				FROM vendor_payments vp WHERE vp.project_vendor_id = pv.id AND vp.deleted_at IS NULL), 0), 0) AS SIGNED) AS paid,
			pv.due_date
		FROM project_vendors pv JOIN projects p ON p.id = pv.project_id
		WHERE ` + where + ` AND pv.deleted_at IS NULL AND pv.engagement_status <> 'Cancelled' AND pv.contract_value > 0`
	venuePart := `SELECT 'venue', p.id, p.id, p.name, p.bride_name, p.groom_name, p.pic_staff_id,
			0, 0, COALESCE(p.venue, ''), CAST(COALESCE(p.venue_rental_price, 0) + COALESCE(p.venue_charge, 0) AS SIGNED),
			CAST(GREATEST(COALESCE((SELECT SUM(CASE WHEN vp.type IN ('DP','Termin','Pelunasan') THEN CAST(vp.amount AS SIGNED)
					WHEN vp.type = 'Refund' THEN -CAST(vp.amount AS SIGNED) ELSE 0 END)
				FROM venue_payments vp WHERE vp.project_id = p.id AND vp.deleted_at IS NULL), 0), 0) AS SIGNED),
			NULL
		FROM projects p
		WHERE ` + where + ` AND p.venue_id IS NOT NULL AND COALESCE(p.venue_rental_price, 0) + COALESCE(p.venue_charge, 0) > 0`
	rows := `SELECT u.*, GREATEST(u.contract_value - u.paid, 0) AS remaining,
			CASE WHEN u.contract_value - u.paid <= 0 THEN 'paid' WHEN u.due_date IS NOT NULL AND u.due_date < ? THEN 'overdue' ELSE 'unpaid' END AS status
		FROM (` + vendorPart + ` UNION ALL ` + venuePart + `) u`
	args := []interface{}{dayOf(today).Format(dateFmt)}
	args = append(args, whereArgs...)
	args = append(args, whereArgs...)
	return rows, args
}

func payableFilterWhere(f domain.PayableFilter) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if f.Kind != "" {
		conds = append(conds, `z.kind = ?`)
		args = append(args, string(f.Kind))
	}
	switch f.Status {
	case "":
	case "open":
		conds = append(conds, `z.remaining > 0`)
	default:
		conds = append(conds, `z.status = ?`)
		args = append(args, f.Status)
	}
	if f.Search != "" {
		like := "%" + utils.EscapeLike(f.Search) + "%"
		conds = append(conds, `(z.project_name LIKE ? OR z.bride_name LIKE ? OR z.groom_name LIKE ? OR z.venue_name LIKE ?)`)
		args = append(args, like, like, like, like)
	}
	if f.PICStaffID != nil {
		conds = append(conds, `z.pic_staff_id = ?`)
		args = append(args, *f.PICStaffID)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return ` WHERE ` + strings.Join(conds, " AND "), args
}

var payableSortColumns = map[string]pagination.SortColumn{
	"dueDate":       pagination.NullableCol("z.due_date"),
	"remaining":     pagination.Col("z.remaining"),
	"contractValue": pagination.Col("z.contract_value"),
	"projectName":   pagination.Col("z.project_name"),
}

// PayableSortSpec: most still owed first by default.
func (r *MySQLFinanceRepository) PayableSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return payableSortColumns, pagination.By("remaining", true)
}

const payableColumns = `z.kind, z.item_id, z.project_id, z.project_name, z.bride_name, z.groom_name, z.pic_staff_id, z.vendor_id, z.category_id, z.venue_name, z.contract_value, z.paid, z.due_date`

func (r *MySQLFinanceRepository) OpenPayables(ctx context.Context, scope domain.FinanceScope) ([]domain.PayableRow, error) {
	inner, args := payableRowsSQL(scope, time.Now())
	return r.queryPayables(ctx, `SELECT `+payableColumns+` FROM (`+inner+`) z WHERE z.remaining > 0 ORDER BY z.kind, z.item_id`, args)
}

func (r *MySQLFinanceRepository) ListPayables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.PayableFilter, params pagination.Params, sort pagination.Sort) ([]domain.PayableRow, int64, error) {
	inner, args := payableRowsSQL(scope, today)
	where, whereArgs := payableFilterWhere(f)
	args = append(args, whereArgs...)
	from := ` FROM (` + inner + `) z` + where
	var total int64
	if err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	// Vendor and venue ids are separate sequences: the kind keeps equal ids apart.
	list, err := r.queryPayables(ctx, `SELECT `+payableColumns+from+sort.OrderBy("z.kind, z.item_id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset()))
	return list, total, err
}

func (r *MySQLFinanceRepository) queryPayables(ctx context.Context, q string, args []interface{}) ([]domain.PayableRow, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.PayableRow
	for rows.Next() {
		var row domain.PayableRow
		var kind string
		var due sql.NullTime
		if err := rows.Scan(&kind, &row.ID, &row.ProjectID, &row.ProjectName, &row.BrideName, &row.GroomName, &row.PICStaffID,
			&row.VendorID, &row.CategoryID, &row.VenueName, &row.ContractValue, &row.Paid, &due); err != nil {
			return nil, err
		}
		row.Kind = domain.PayableKind(kind)
		if due.Valid {
			d := due.Time
			row.DueDate = &d
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *MySQLFinanceRepository) ReceivedPayments(ctx context.Context, scope domain.FinanceScope, from, to time.Time) ([]domain.PaymentLine, error) {
	where, args := financeProjectWhere(scope)
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT cp.type, cp.amount FROM client_payments cp JOIN projects p ON p.id = cp.project_id
		 WHERE `+where+` AND cp.deleted_at IS NULL AND `+domain.ContractCountsSQL+` AND cp.payment_date BETWEEN ? AND ?`,
		append(args, from.Format(dateFmt), to.Format(dateFmt))...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []domain.PaymentLine
	for rows.Next() {
		var l domain.PaymentLine
		var typ string
		if err := rows.Scan(&typ, &l.Amount); err != nil {
			return nil, err
		}
		l.Type = domain.PaymentType(typ)
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// pendingVerificationWhere: live client payments not verified yet that have at
// least one proof file (only those can be verified, see ClientPaymentService.Verify).
func pendingVerificationWhere(scope domain.FinanceScope) (string, []interface{}) {
	where, args := financeProjectWhere(scope)
	return ` FROM client_payments cp JOIN projects p ON p.id = cp.project_id
		WHERE ` + where + ` AND cp.deleted_at IS NULL AND cp.verified_at IS NULL
		  AND EXISTS (SELECT 1 FROM evidence e WHERE e.project_id = cp.project_id AND e.related_kind = 'clientPayment' AND e.related_id = cp.id)`, args
}

func (r *MySQLFinanceRepository) PendingVerification(ctx context.Context, scope domain.FinanceScope) (domain.MoneyCount, error) {
	from, args := pendingVerificationWhere(scope)
	var out domain.MoneyCount
	err := conn(ctx, r.db).QueryRowContext(ctx,
		`SELECT COUNT(*), CAST(COALESCE(SUM(cp.amount), 0) AS SIGNED), COUNT(DISTINCT cp.project_id)`+from, args...).
		Scan(&out.Count, &out.Amount, &out.Projects)
	return out, err
}

var verificationSortColumns = map[string]pagination.SortColumn{
	"paymentDate": pagination.Col("cp.payment_date"),
	"amount":      pagination.Col("cp.amount"),
}

// VerificationSortSpec: the payment waiting longest first.
func (r *MySQLFinanceRepository) VerificationSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return verificationSortColumns, pagination.By("paymentDate", false)
}

func (r *MySQLFinanceRepository) ListVerifications(ctx context.Context, scope domain.FinanceScope, params pagination.Params, sort pagination.Sort) ([]domain.VerificationRow, int64, error) {
	from, args := pendingVerificationWhere(scope)
	var total int64
	if err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT cp.id, cp.project_id, p.name, p.bride_name, p.groom_name, cp.type, cp.amount, cp.payment_date,
		        COALESCE(cp.method, ''), COALESCE(cp.reference_number, ''), COALESCE(cp.recorded_by_staff_id, 0), COALESCE(cp.submitted_by_client_id, 0)`+
			from+sort.OrderBy("cp.id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []domain.VerificationRow
	for rows.Next() {
		var v domain.VerificationRow
		var typ string
		if err := rows.Scan(&v.PaymentID, &v.ProjectID, &v.ProjectName, &v.BrideName, &v.GroomName, &typ, &v.Amount, &v.PaymentDate,
			&v.Method, &v.ReferenceNumber, &v.RecordedByID, &v.SubmittedByClientID); err != nil {
			return nil, 0, err
		}
		v.Type = domain.PaymentType(typ)
		list = append(list, v)
	}
	return list, total, rows.Err()
}

// dayOf is the calendar day of t (its own zone) at midnight UTC: the dates the
// SQL compares are plain DATE columns.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
