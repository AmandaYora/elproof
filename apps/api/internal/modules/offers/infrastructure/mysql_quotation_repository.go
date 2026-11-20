package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

// MySQLQuotationRepository stores quotations and everything in them. Every
// statement filters by tenant_id (ADR-0004), child tables included; joins
// stay inside the quotation tables of this module.
type MySQLQuotationRepository struct {
	db *sql.DB
}

func NewMySQLQuotationRepository(db *sql.DB) *MySQLQuotationRepository {
	return &MySQLQuotationRepository{db: db}
}

const dayLayout = "2006-01-02"

func day(t time.Time) string { return domain.CivilDate(t).Format(dayLayout) }

func dayPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return day(*t)
}

func civilPtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := domain.CivilDate(v.Time)
	return &t
}

func timePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func strPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func timeArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// ---- header ----------------------------------------------------------------

const quotationColumns = `q.id, q.tenant_id, q.number, q.doc_date, q.couple_names, q.contact_whatsapp, q.pic_staff_id, q.valid_until,
	q.notes, q.status, q.alternative_of_id, q.superseded_by_id, q.accepted_revision_id,
	COALESCE((SELECT r.n FROM quotation_revisions r WHERE r.tenant_id = q.tenant_id AND r.id = q.accepted_revision_id), 0),
	q.accepted_on, q.accepted_channel, q.accepted_attachment_key, q.accepted_attachment_name, q.accepted_attachment_type,
	q.rejected_reason, q.rejected_at, q.converted_project_id, q.converted_at, q.project_deleted_at, q.is_pkp, q.terms_text,
	q.revision_count, q.contract_total, q.main_event_date, q.created_by_staff_id, q.created_at, q.updated_at,
	q.lines_updated_at, q.terms_updated_at, q.changed_at`

func scanQuotation(scan func(dest ...any) error) (*domain.Quotation, error) {
	var q domain.Quotation
	var notes, channel, attKey, attName, attType, reason sql.NullString
	var alt, superseded, accRev, project, createdBy sql.NullInt64
	var acceptedOn, rejectedAt, convertedAt, projectDeleted, mainEvent sql.NullTime
	var status string
	err := scan(&q.ID, &q.TenantID, &q.Number, &q.DocDate, &q.CoupleNames, &q.ContactWhatsApp, &q.PICStaffID, &q.ValidUntil,
		&notes, &status, &alt, &superseded, &accRev, &q.AcceptedRevisionN,
		&acceptedOn, &channel, &attKey, &attName, &attType,
		&reason, &rejectedAt, &project, &convertedAt, &projectDeleted, &q.IsPKP, &q.TermsText,
		&q.RevisionCount, &q.ContractTotal, &mainEvent, &createdBy, &q.CreatedAt, &q.UpdatedAt,
		&q.LinesUpdatedAt, &q.TermsUpdatedAt, &q.ChangedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	q.DocDate = domain.CivilDate(q.DocDate)
	q.ValidUntil = domain.CivilDate(q.ValidUntil)
	q.Notes = notes.String
	q.Status = domain.QuotationStatus(status)
	q.AlternativeOfID = int64Ptr(alt)
	q.SupersededByID = int64Ptr(superseded)
	q.AcceptedRevisionID = int64Ptr(accRev)
	q.AcceptedOn = civilPtr(acceptedOn)
	if channel.Valid {
		c := domain.AcceptChannel(channel.String)
		q.AcceptedChannel = &c
	}
	q.AcceptedAttachmentKey, q.AcceptedAttachmentName, q.AcceptedAttachmentType = strPtr(attKey), strPtr(attName), strPtr(attType)
	q.RejectedReason = strPtr(reason)
	q.RejectedAt = timePtr(rejectedAt)
	q.ConvertedProjectID = int64Ptr(project)
	q.ConvertedAt = timePtr(convertedAt)
	q.ProjectDeletedAt = timePtr(projectDeleted)
	q.MainEventDate = civilPtr(mainEvent)
	q.CreatedByStaffID = int64Ptr(createdBy)
	return &q, nil
}

func (r *MySQLQuotationRepository) NextSequence(ctx context.Context, tenantID int64, docDate time.Time) (int, error) {
	var next int
	err := runInTx(ctx, r.db, func(x dbtx) error {
		if _, err := x.ExecContext(ctx, `INSERT INTO quotation_counters (tenant_id, doc_date, last_seq) VALUES (?, ?, 0)
			ON DUPLICATE KEY UPDATE last_seq = last_seq`, tenantID, day(docDate)); err != nil {
			return err
		}
		var last int
		if err := x.QueryRowContext(ctx, `SELECT last_seq FROM quotation_counters WHERE tenant_id = ? AND doc_date = ? FOR UPDATE`,
			tenantID, day(docDate)).Scan(&last); err != nil {
			return err
		}
		next = last + 1
		_, err := x.ExecContext(ctx, `UPDATE quotation_counters SET last_seq = ? WHERE tenant_id = ? AND doc_date = ?`, next, tenantID, day(docDate))
		return err
	})
	return next, err
}

func (r *MySQLQuotationRepository) Create(ctx context.Context, q *domain.Quotation) error {
	res, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO quotations (tenant_id, number, doc_date, couple_names, contact_whatsapp,
		pic_staff_id, valid_until, notes, status, alternative_of_id, is_pkp, terms_text, created_by_staff_id,
		created_at, updated_at, lines_updated_at, terms_updated_at, changed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(6), NOW(6), NOW(6), NOW(6), NOW(6))`,
		q.TenantID, q.Number, day(q.DocDate), q.CoupleNames, q.ContactWhatsApp, q.PICStaffID, day(q.ValidUntil),
		nullIfEmpty(q.Notes), string(q.Status), nullable(q.AlternativeOfID), q.IsPKP, q.TermsText, nullable(q.CreatedByStaffID))
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	q.ID = id
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *MySQLQuotationRepository) find(ctx context.Context, tenantID, id int64, lock bool) (*domain.Quotation, error) {
	query := `SELECT ` + quotationColumns + ` FROM quotations q WHERE q.tenant_id = ? AND q.id = ? AND q.deleted_at IS NULL`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanQuotation(conn(ctx, r.db).QueryRowContext(ctx, query, tenantID, id).Scan)
}

func (r *MySQLQuotationRepository) FindByID(ctx context.Context, tenantID, id int64) (*domain.Quotation, error) {
	return r.find(ctx, tenantID, id, false)
}

func (r *MySQLQuotationRepository) Lock(ctx context.Context, tenantID, id int64) (*domain.Quotation, error) {
	return r.find(ctx, tenantID, id, true)
}

func (r *MySQLQuotationRepository) SaveHeader(ctx context.Context, q *domain.Quotation) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE quotations SET couple_names = ?, contact_whatsapp = ?, pic_staff_id = ?,
		valid_until = ?, notes = ?, updated_at = NOW(6), changed_at = NOW(6) WHERE tenant_id = ? AND id = ?`,
		q.CoupleNames, q.ContactWhatsApp, q.PICStaffID, day(q.ValidUntil), nullIfEmpty(q.Notes), q.TenantID, q.ID)
	return err
}

func (r *MySQLQuotationRepository) SaveState(ctx context.Context, q *domain.Quotation) error {
	var channel any
	if q.AcceptedChannel != nil {
		channel = string(*q.AcceptedChannel)
	}
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE quotations SET status = ?, valid_until = ?, superseded_by_id = ?,
		accepted_revision_id = ?, accepted_on = ?, accepted_channel = ?, accepted_attachment_key = ?, accepted_attachment_name = ?,
		accepted_attachment_type = ?, rejected_reason = ?, rejected_at = ?, converted_project_id = ?, converted_at = ?,
		project_deleted_at = ?, updated_at = NOW(6), changed_at = NOW(6) WHERE tenant_id = ? AND id = ?`,
		string(q.Status), day(q.ValidUntil), nullable(q.SupersededByID), nullable(q.AcceptedRevisionID), dayPtr(q.AcceptedOn), channel,
		nullable(q.AcceptedAttachmentKey), nullable(q.AcceptedAttachmentName), nullable(q.AcceptedAttachmentType),
		nullable(q.RejectedReason), timeArg(q.RejectedAt), nullable(q.ConvertedProjectID), timeArg(q.ConvertedAt),
		timeArg(q.ProjectDeletedAt), q.TenantID, q.ID)
	return err
}

func (r *MySQLQuotationRepository) SaveTermsText(ctx context.Context, tenantID, id int64, text string) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE quotations SET terms_text = ?, terms_updated_at = NOW(6), changed_at = NOW(6)
		WHERE tenant_id = ? AND id = ?`, text, tenantID, id)
	return err
}

func (r *MySQLQuotationRepository) SetCaches(ctx context.Context, tenantID, id int64, total int64, mainEventDate *time.Time, bumpLines, bumpTerms bool) error {
	set := `contract_total = ?, main_event_date = ?, changed_at = NOW(6)`
	if bumpLines {
		set += `, lines_updated_at = NOW(6)`
	}
	if bumpTerms {
		set += `, terms_updated_at = NOW(6)`
	}
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE quotations SET `+set+` WHERE tenant_id = ? AND id = ?`,
		total, dayPtr(mainEventDate), tenantID, id)
	return err
}

func (r *MySQLQuotationRepository) SoftDelete(ctx context.Context, tenantID, id int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE quotations SET deleted_at = NOW(), changed_at = NOW(6) WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return err
}

func (r *MySQLQuotationRepository) FindByProject(ctx context.Context, tenantID, projectID int64) (*domain.Quotation, error) {
	return scanQuotation(conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+quotationColumns+` FROM quotations q
		WHERE q.tenant_id = ? AND q.converted_project_id = ? AND q.deleted_at IS NULL ORDER BY q.id LIMIT 1`, tenantID, projectID).Scan)
}

// ---- list -------------------------------------------------------------------

// quotationSortColumns is the ?sort= whitelist of GET /quotations.
var quotationSortColumns = map[string]pagination.SortColumn{
	"number":        pagination.Col("q.number"),
	"coupleNames":   pagination.Col("q.couple_names"),
	"validUntil":    pagination.Col("q.valid_until"),
	"contractTotal": pagination.Col("q.contract_total"),
	"mainEventDate": pagination.NullableCol("q.main_event_date"),
	"createdAt":     pagination.Col("q.created_at"),
	"changedAt":     pagination.Col("q.changed_at"),
}

func (r *MySQLQuotationRepository) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return quotationSortColumns, pagination.By("changedAt", true)
}

// memberConditions is the filter on one member x of a group.
func memberConditions(filter application.QuotationListFilter, tab domain.QuotationStatus, expiringSoon bool) (string, []any) {
	var conds []string
	var args []any
	today := day(filter.Today)
	switch tab {
	case "":
	case domain.QuotationExpired:
		conds = append(conds, `x.status = 'Draft' AND x.valid_until < ?`)
		args = append(args, today)
	case domain.QuotationDraft:
		conds = append(conds, `x.status = 'Draft' AND x.valid_until >= ?`)
		args = append(args, today)
	default:
		conds = append(conds, `x.status = ?`)
		args = append(args, string(tab))
	}
	if expiringSoon {
		conds = append(conds, `x.status = 'Draft' AND x.valid_until BETWEEN ? AND ?`)
		args = append(args, today, day(filter.Today.AddDate(0, 0, 7)))
	}
	if filter.PICStaffID != nil {
		conds = append(conds, `x.pic_staff_id = ?`)
		args = append(args, *filter.PICStaffID)
	}
	if filter.OnlyPICStaffID != nil {
		conds = append(conds, `x.pic_staff_id = ?`)
		args = append(args, *filter.OnlyPICStaffID)
	}
	if s := strings.TrimSpace(filter.Search); s != "" {
		like := "%" + s + "%"
		conds = append(conds, `(x.number LIKE ? OR x.couple_names LIKE ? OR x.contact_whatsapp LIKE ?)`)
		args = append(args, like, like, like)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return ` AND ` + strings.Join(conds, ` AND `), args
}

// groupWhere keeps the first quotation of every group (alternatives come as
// children) when any member of the group matches. Under the `assigned` scope
// the first quotation itself must be the person's.
func groupWhere(tenantID int64, filter application.QuotationListFilter, tab domain.QuotationStatus, expiringSoon bool) (string, []any) {
	where := ` WHERE q.tenant_id = ? AND q.deleted_at IS NULL AND q.alternative_of_id IS NULL`
	args := []any{tenantID}
	if filter.OnlyPICStaffID != nil {
		where += ` AND q.pic_staff_id = ?`
		args = append(args, *filter.OnlyPICStaffID)
	}
	cond, condArgs := memberConditions(filter, tab, expiringSoon)
	if cond != "" {
		where += ` AND EXISTS (SELECT 1 FROM quotations x WHERE x.tenant_id = q.tenant_id AND x.deleted_at IS NULL
			AND (x.id = q.id OR x.alternative_of_id = q.id)` + cond + `)`
		args = append(args, condArgs...)
	}
	return where, args
}

func (r *MySQLQuotationRepository) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, filter application.QuotationListFilter, sort pagination.Sort) ([]domain.Quotation, int64, error) {
	where, args := groupWhere(tenantID, filter, filter.Tab, filter.ExpiringSoon)
	db := conn(ctx, r.db)
	var total int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quotations q`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.QueryContext(ctx, `SELECT `+quotationColumns+` FROM quotations q`+where+sort.OrderBy("q.id")+` LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	out, err := scanQuotations(rows)
	return out, total, err
}

func scanQuotations(rows *sql.Rows) ([]domain.Quotation, error) {
	defer rows.Close()
	var out []domain.Quotation
	for rows.Next() {
		q, err := scanQuotation(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *q)
	}
	return out, rows.Err()
}

// StatusCounts keys: every stored status, Expired, "" (all) and
// "ExpiringSoon".
func (r *MySQLQuotationRepository) StatusCounts(ctx context.Context, tenantID int64, filter application.QuotationListFilter) (map[domain.QuotationStatus]int, error) {
	type tab struct {
		key      domain.QuotationStatus
		status   domain.QuotationStatus
		expiring bool
	}
	tabs := []tab{{"", "", false}, {domain.QuotationDraft, domain.QuotationDraft, false}, {domain.QuotationExpired, domain.QuotationExpired, false},
		{domain.QuotationAccepted, domain.QuotationAccepted, false}, {domain.QuotationRejected, domain.QuotationRejected, false},
		{domain.QuotationSuperseded, domain.QuotationSuperseded, false}, {"ExpiringSoon", "", true}}
	out := make(map[domain.QuotationStatus]int, len(tabs))
	db := conn(ctx, r.db)
	for _, t := range tabs {
		where, args := groupWhere(tenantID, filter, t.status, t.expiring)
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quotations q`+where, args...).Scan(&n); err != nil {
			return nil, err
		}
		out[t.key] = n
	}
	return out, nil
}

func (r *MySQLQuotationRepository) Alternatives(ctx context.Context, tenantID int64, rootIDs []int64) ([]domain.Quotation, error) {
	if len(rootIDs) == 0 {
		return nil, nil
	}
	args := append([]any{tenantID}, utils.Int64Args(rootIDs)...)
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+quotationColumns+` FROM quotations q
		WHERE q.tenant_id = ? AND q.deleted_at IS NULL AND q.alternative_of_id IN (`+utils.Placeholders(len(rootIDs))+`) ORDER BY q.id`, args...)
	if err != nil {
		return nil, err
	}
	return scanQuotations(rows)
}

// ---- content ----------------------------------------------------------------

func (r *MySQLQuotationRepository) LoadContent(ctx context.Context, tenantID, quotationID int64) (*domain.WorkingCopy, error) {
	db := conn(ctx, r.db)
	w := &domain.WorkingCopy{Events: []domain.QuotationEvent{}, Suppliers: []domain.QuotationSupplier{}, Lines: []domain.QuotationLine{}, Terms: []domain.QuotationTerm{}}

	rows, err := db.QueryContext(ctx, `SELECT id, quotation_id, event_kind_id, kind_name, event_date, start_time, end_time,
		venue_supplier_id, layout, sort_order, updated_at FROM quotation_events WHERE tenant_id = ? AND quotation_id = ? ORDER BY sort_order, id`,
		tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	evIndex := map[int64]int{}
	for rows.Next() {
		var e domain.QuotationEvent
		var date sql.NullTime
		var start, end sql.NullString
		var venue sql.NullInt64
		var layout string
		if err := rows.Scan(&e.ID, &e.QuotationID, &e.EventKindID, &e.KindName, &date, &start, &end, &venue, &layout, &e.SortOrder, &e.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		e.EventDate, e.StartTime, e.EndTime, e.VenueSupplierID, e.Layout = civilPtr(date), strPtr(start), strPtr(end), int64Ptr(venue), domain.Layout(layout)
		e.Sessions = []domain.QuotationSession{}
		evIndex[e.ID] = len(w.Events)
		w.Events = append(w.Events, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(w.Events) > 0 {
		rows, err = db.QueryContext(ctx, `SELECT s.id, s.event_id, s.label, s.start_time, s.end_time, s.guests, s.sort_order
			FROM quotation_event_sessions s JOIN quotation_events e ON e.id = s.event_id AND e.tenant_id = s.tenant_id
			WHERE s.tenant_id = ? AND e.quotation_id = ? ORDER BY s.sort_order, s.id`, tenantID, quotationID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var s domain.QuotationSession
			var eventID int64
			var start, end sql.NullString
			if err := rows.Scan(&s.ID, &eventID, &s.Label, &start, &end, &s.Guests, &s.SortOrder); err != nil {
				rows.Close()
				return nil, err
			}
			s.StartTime, s.EndTime = strPtr(start), strPtr(end)
			if i, ok := evIndex[eventID]; ok {
				w.Events[i].Sessions = append(w.Events[i].Sessions, s)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	rows, err = db.QueryContext(ctx, `SELECT id, quotation_id, supplier_kind, supplier_id, supplier_name, category_id, paid_by, show_name,
		capacity_standing, capacity_seated, agreed_cost, notes, sort_order, updated_at
		FROM quotation_suppliers WHERE tenant_id = ? AND quotation_id = ? ORDER BY sort_order, id`, tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s domain.QuotationSupplier
		var kind, paidBy string
		var category, cost sql.NullInt64
		var standing, seated sql.NullInt32
		var notes sql.NullString
		if err := rows.Scan(&s.ID, &s.QuotationID, &kind, &s.SupplierID, &s.Name, &category, &paidBy, &s.ShowName,
			&standing, &seated, &cost, &notes, &s.SortOrder, &s.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		s.Kind, s.PaidBy, s.CategoryID, s.AgreedCost, s.Notes = domain.SupplierKind(kind), domain.Payer(paidBy), int64Ptr(category), int64Ptr(cost), notes.String
		s.CapacityStanding, s.CapacitySeated = intPtr(standing), intPtr(seated)
		w.Suppliers = append(w.Suppliers, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.QueryContext(ctx, `SELECT id, quotation_id, kind, parent_line_id, package_id, package_rule, copied_at, description,
		details, category_id, event_id, quantity, unit, unit_price, amount, is_bonus, supplier_ref, sort_order
		FROM quotation_lines WHERE tenant_id = ? AND quotation_id = ? ORDER BY sort_order, id`, tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l domain.QuotationLine
		var kind string
		var parent, pkg, category, event, qty, price, supplier sql.NullInt64
		var rule []byte
		var copied sql.NullTime
		var details, unit sql.NullString
		if err := rows.Scan(&l.ID, &l.QuotationID, &kind, &parent, &pkg, &rule, &copied, &l.Description, &details, &category, &event,
			&qty, &unit, &price, &l.Amount, &l.IsBonus, &supplier, &l.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		l.Kind, l.ParentLineID, l.PackageID, l.CopiedAt = domain.LineKind(kind), int64Ptr(parent), int64Ptr(pkg), timePtr(copied)
		l.Details, l.CategoryID, l.EventID, l.Quantity, l.Unit = details.String, int64Ptr(category), int64Ptr(event), int64Ptr(qty), strPtr(unit)
		l.UnitPrice, l.SupplierRef = int64Ptr(price), int64Ptr(supplier)
		if len(rule) > 0 {
			var pr domain.PackageRule
			if err := json.Unmarshal(rule, &pr); err != nil {
				rows.Close()
				return nil, err
			}
			l.PackageRule = &pr
		}
		w.Lines = append(w.Lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.QueryContext(ctx, `SELECT id, quotation_id, label, percent, amount, due_basis, due_offset_days, due_event_id, sort_order
		FROM quotation_terms WHERE tenant_id = ? AND quotation_id = ? ORDER BY sort_order, id`, tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t domain.QuotationTerm
		var percent sql.NullInt32
		var amount, event sql.NullInt64
		var basis string
		if err := rows.Scan(&t.ID, &t.QuotationID, &t.Label, &percent, &amount, &basis, &t.DueOffsetDays, &event, &t.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		t.Percent, t.Amount, t.DueBasis, t.DueEventID = intPtr(percent), int64Ptr(amount), domain.DueBasis(basis), int64Ptr(event)
		w.Terms = append(w.Terms, t)
	}
	rows.Close()
	return w, rows.Err()
}

func insertSessions(ctx context.Context, x dbtx, tenantID int64, e *domain.QuotationEvent) error {
	for i := range e.Sessions {
		s := &e.Sessions[i]
		res, err := x.ExecContext(ctx, `INSERT INTO quotation_event_sessions (tenant_id, event_id, label, start_time, end_time, guests, sort_order)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, tenantID, e.ID, s.Label, nullable(s.StartTime), nullable(s.EndTime), s.Guests, s.SortOrder)
		if err != nil {
			return err
		}
		if s.ID, err = res.LastInsertId(); err != nil {
			return err
		}
	}
	return nil
}

func (r *MySQLQuotationRepository) InsertEvent(ctx context.Context, tenantID int64, e *domain.QuotationEvent) error {
	return runInTx(ctx, r.db, func(x dbtx) error {
		res, err := x.ExecContext(ctx, `INSERT INTO quotation_events (tenant_id, quotation_id, event_kind_id, kind_name, event_date,
			start_time, end_time, venue_supplier_id, layout, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			tenantID, e.QuotationID, e.EventKindID, e.KindName, dayPtr(e.EventDate), nullable(e.StartTime), nullable(e.EndTime),
			nullable(e.VenueSupplierID), string(e.Layout), e.SortOrder)
		if err != nil {
			return err
		}
		if e.ID, err = res.LastInsertId(); err != nil {
			return err
		}
		return insertSessions(ctx, x, tenantID, e)
	})
}

func (r *MySQLQuotationRepository) UpdateEvent(ctx context.Context, tenantID int64, e *domain.QuotationEvent) error {
	return runInTx(ctx, r.db, func(x dbtx) error {
		if _, err := x.ExecContext(ctx, `UPDATE quotation_events SET event_kind_id = ?, kind_name = ?, event_date = ?, start_time = ?,
			end_time = ?, venue_supplier_id = ?, layout = ?, updated_at = NOW(6) WHERE tenant_id = ? AND quotation_id = ? AND id = ?`,
			e.EventKindID, e.KindName, dayPtr(e.EventDate), nullable(e.StartTime), nullable(e.EndTime), nullable(e.VenueSupplierID),
			string(e.Layout), tenantID, e.QuotationID, e.ID); err != nil {
			return err
		}
		if _, err := x.ExecContext(ctx, `DELETE FROM quotation_event_sessions WHERE tenant_id = ? AND event_id = ?`, tenantID, e.ID); err != nil {
			return err
		}
		return insertSessions(ctx, x, tenantID, e)
	})
}

func (r *MySQLQuotationRepository) DeleteEvent(ctx context.Context, tenantID, quotationID, eventID int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM quotation_events WHERE tenant_id = ? AND quotation_id = ? AND id = ?`,
		tenantID, quotationID, eventID)
	return err
}

// SetEventOrder keeps each event's version: reordering is not an edit of the
// event an open form holds.
func (r *MySQLQuotationRepository) SetEventOrder(ctx context.Context, tenantID, quotationID int64, ids []int64) error {
	return runInTx(ctx, r.db, func(x dbtx) error {
		for i, id := range ids {
			if _, err := x.ExecContext(ctx, `UPDATE quotation_events SET sort_order = ?, updated_at = updated_at
				WHERE tenant_id = ? AND quotation_id = ? AND id = ?`, i, tenantID, quotationID, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MySQLQuotationRepository) InsertSupplier(ctx context.Context, tenantID int64, s *domain.QuotationSupplier) error {
	res, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO quotation_suppliers (tenant_id, quotation_id, supplier_kind, supplier_id,
		supplier_name, category_id, paid_by, show_name, capacity_standing, capacity_seated, agreed_cost, notes, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tenantID, s.QuotationID, string(s.Kind), s.SupplierID, s.Name, nullable(s.CategoryID), string(s.PaidBy), s.ShowName,
		nullable(s.CapacityStanding), nullable(s.CapacitySeated), nullable(s.AgreedCost), nullIfEmpty(s.Notes), s.SortOrder)
	if err != nil {
		return err
	}
	s.ID, err = res.LastInsertId()
	return err
}

func (r *MySQLQuotationRepository) UpdateSupplier(ctx context.Context, tenantID int64, s *domain.QuotationSupplier) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE quotation_suppliers SET supplier_name = ?, category_id = ?, paid_by = ?, show_name = ?,
		capacity_standing = ?, capacity_seated = ?, agreed_cost = ?, notes = ?, sort_order = ?, updated_at = NOW(6)
		WHERE tenant_id = ? AND quotation_id = ? AND id = ?`,
		s.Name, nullable(s.CategoryID), string(s.PaidBy), s.ShowName, nullable(s.CapacityStanding), nullable(s.CapacitySeated),
		nullable(s.AgreedCost), nullIfEmpty(s.Notes), s.SortOrder, tenantID, s.QuotationID, s.ID)
	return err
}

func (r *MySQLQuotationRepository) DeleteSupplier(ctx context.Context, tenantID, quotationID, supplierID int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM quotation_suppliers WHERE tenant_id = ? AND quotation_id = ? AND id = ?`,
		tenantID, quotationID, supplierID)
	return err
}

func (r *MySQLQuotationRepository) ReplaceLines(ctx context.Context, tenantID, quotationID int64, rows []application.LineRow) error {
	return runInTx(ctx, r.db, func(x dbtx) error {
		// Children first, so the self-referencing key never blocks the delete.
		if _, err := x.ExecContext(ctx, `DELETE FROM quotation_lines WHERE tenant_id = ? AND quotation_id = ? AND parent_line_id IS NOT NULL`,
			tenantID, quotationID); err != nil {
			return err
		}
		if _, err := x.ExecContext(ctx, `DELETE FROM quotation_lines WHERE tenant_id = ? AND quotation_id = ?`, tenantID, quotationID); err != nil {
			return err
		}
		ids := make([]int64, len(rows))
		for i, row := range rows {
			l := row.Line
			var parent any
			if row.Parent >= 0 && row.Parent < i {
				parent = ids[row.Parent]
			}
			var rule any
			if l.PackageRule != nil {
				raw, err := json.Marshal(l.PackageRule)
				if err != nil {
					return err
				}
				rule = string(raw)
			}
			res, err := x.ExecContext(ctx, `INSERT INTO quotation_lines (tenant_id, quotation_id, kind, parent_line_id, package_id, package_rule,
				copied_at, description, details, category_id, event_id, quantity, unit, unit_price, amount, is_bonus, supplier_ref, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				tenantID, quotationID, string(l.Kind), parent, nullable(l.PackageID), rule, timeArg(l.CopiedAt), l.Description,
				nullIfEmpty(l.Details), nullable(l.CategoryID), nullable(l.EventID), nullable(l.Quantity), nullable(l.Unit),
				nullable(l.UnitPrice), l.Amount, l.IsBonus, nullable(l.SupplierRef), l.SortOrder)
			if err != nil {
				return err
			}
			if ids[i], err = res.LastInsertId(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MySQLQuotationRepository) SetLineAmounts(ctx context.Context, tenantID, quotationID int64, amounts map[int64]int64) error {
	return runInTx(ctx, r.db, func(x dbtx) error {
		for id, amount := range amounts {
			if _, err := x.ExecContext(ctx, `UPDATE quotation_lines SET amount = ? WHERE tenant_id = ? AND quotation_id = ? AND id = ?`,
				amount, tenantID, quotationID, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MySQLQuotationRepository) ReplaceTerms(ctx context.Context, tenantID, quotationID int64, terms []domain.QuotationTerm) error {
	return runInTx(ctx, r.db, func(x dbtx) error {
		if _, err := x.ExecContext(ctx, `DELETE FROM quotation_terms WHERE tenant_id = ? AND quotation_id = ?`, tenantID, quotationID); err != nil {
			return err
		}
		for i, t := range terms {
			if _, err := x.ExecContext(ctx, `INSERT INTO quotation_terms (tenant_id, quotation_id, label, percent, amount, due_basis,
				due_offset_days, due_event_id, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				tenantID, quotationID, t.Label, nullable(t.Percent), nullable(t.Amount), string(t.DueBasis), t.DueOffsetDays,
				nullable(t.DueEventID), i); err != nil {
				return err
			}
		}
		return nil
	})
}

// ---- revisions ----------------------------------------------------------------

const revisionColumns = `id, quotation_id, n, snapshot, content_hash, contract_total, created_by_staff_id, created_at`

func scanRevision(scan func(dest ...any) error) (*domain.QuotationRevision, error) {
	var rev domain.QuotationRevision
	var raw []byte
	var createdBy sql.NullInt64
	err := scan(&rev.ID, &rev.QuotationID, &rev.N, &raw, &rev.ContentHash, &rev.ContractTotal, &createdBy, &rev.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &rev.Snapshot); err != nil {
		return nil, err
	}
	rev.CreatedByStaffID = int64Ptr(createdBy)
	return &rev, nil
}

func (r *MySQLQuotationRepository) FindRevisionByHash(ctx context.Context, tenantID, quotationID int64, hash string) (*domain.QuotationRevision, error) {
	return scanRevision(conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+revisionColumns+` FROM quotation_revisions
		WHERE tenant_id = ? AND quotation_id = ? AND content_hash = ?`, tenantID, quotationID, hash).Scan)
}

func (r *MySQLQuotationRepository) FindRevision(ctx context.Context, tenantID, quotationID int64, n int) (*domain.QuotationRevision, error) {
	return scanRevision(conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+revisionColumns+` FROM quotation_revisions
		WHERE tenant_id = ? AND quotation_id = ? AND n = ?`, tenantID, quotationID, n).Scan)
}

func (r *MySQLQuotationRepository) FindRevisionByID(ctx context.Context, tenantID, revisionID int64) (*domain.QuotationRevision, error) {
	return scanRevision(conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+revisionColumns+` FROM quotation_revisions
		WHERE tenant_id = ? AND id = ?`, tenantID, revisionID).Scan)
}

func (r *MySQLQuotationRepository) ListRevisions(ctx context.Context, tenantID, quotationID int64) ([]domain.QuotationRevision, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+revisionColumns+` FROM quotation_revisions
		WHERE tenant_id = ? AND quotation_id = ? ORDER BY n DESC`, tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.QuotationRevision{}
	for rows.Next() {
		rev, err := scanRevision(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *rev)
	}
	return out, rows.Err()
}

func (r *MySQLQuotationRepository) InsertRevision(ctx context.Context, tenantID int64, rev *domain.QuotationRevision) error {
	raw, err := json.Marshal(rev.Snapshot)
	if err != nil {
		return err
	}
	return runInTx(ctx, r.db, func(x dbtx) error {
		res, err := x.ExecContext(ctx, `INSERT INTO quotation_revisions (tenant_id, quotation_id, n, snapshot, content_hash, contract_total,
			created_by_staff_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			tenantID, rev.QuotationID, rev.N, string(raw), rev.ContentHash, rev.ContractTotal, nullable(rev.CreatedByStaffID), rev.CreatedAt)
		if isDuplicate(err) {
			return application.ErrDuplicateRevision
		}
		if err != nil {
			return err
		}
		if rev.ID, err = res.LastInsertId(); err != nil {
			return err
		}
		_, err = x.ExecContext(ctx, `UPDATE quotations SET revision_count = GREATEST(revision_count, ?), changed_at = NOW(6)
			WHERE tenant_id = ? AND id = ?`, rev.N, tenantID, rev.QuotationID)
		return err
	})
}

// ---- activity -----------------------------------------------------------------

func (r *MySQLQuotationRepository) AddActivity(ctx context.Context, tenantID int64, a domain.Activity) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO quotation_activity (tenant_id, quotation_id, type, actor_staff_id, summary, created_at)
		VALUES (?, ?, ?, ?, ?, NOW(6))`, tenantID, a.QuotationID, string(a.Type), nullable(a.ActorStaffID), a.Summary)
	return err
}

func (r *MySQLQuotationRepository) ListActivity(ctx context.Context, tenantID, quotationID int64) ([]domain.Activity, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT id, quotation_id, type, actor_staff_id, summary, created_at
		FROM quotation_activity WHERE tenant_id = ? AND quotation_id = ? ORDER BY created_at DESC, id DESC LIMIT 500`, tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Activity{}
	for rows.Next() {
		var a domain.Activity
		var t string
		var actor sql.NullInt64
		if err := rows.Scan(&a.ID, &a.QuotationID, &t, &actor, &a.Summary, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Type, a.ActorStaffID = domain.ActivityType(t), int64Ptr(actor)
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---- references (catalog and directory) --------------------------------------

func (r *MySQLQuotationRepository) exists(ctx context.Context, query string, args ...any) (bool, error) {
	var one int
	err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT 1 FROM DUAL WHERE EXISTS (`+query+`)`, args...).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

const liveQuotation = ` JOIN quotations q ON q.id = c.quotation_id AND q.tenant_id = c.tenant_id AND q.deleted_at IS NULL`

func (r *MySQLQuotationRepository) PackageReferenced(ctx context.Context, tenantID, packageID int64) (bool, error) {
	return r.exists(ctx, `SELECT 1 FROM quotation_lines c`+liveQuotation+` WHERE c.tenant_id = ? AND c.package_id = ?`, tenantID, packageID)
}

func (r *MySQLQuotationRepository) VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error) {
	return r.exists(ctx, `SELECT 1 FROM quotation_suppliers c`+liveQuotation+`
		WHERE c.tenant_id = ? AND c.supplier_kind = 'vendor' AND c.supplier_id = ?`, tenantID, vendorID)
}

func (r *MySQLQuotationRepository) VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error) {
	return r.exists(ctx, `SELECT 1 FROM quotation_suppliers c`+liveQuotation+`
		WHERE c.tenant_id = ? AND c.supplier_kind = 'venue' AND c.supplier_id = ?`, tenantID, venueID)
}

func (r *MySQLQuotationRepository) CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error) {
	used, err := r.exists(ctx, `SELECT 1 FROM quotation_suppliers c`+liveQuotation+` WHERE c.tenant_id = ? AND c.category_id = ?`, tenantID, categoryID)
	if err != nil || used {
		return used, err
	}
	return r.exists(ctx, `SELECT 1 FROM quotation_lines c`+liveQuotation+` WHERE c.tenant_id = ? AND c.category_id = ?`, tenantID, categoryID)
}

func (r *MySQLQuotationRepository) EventKindQuotationCounts(ctx context.Context, tenantID int64) (map[int64]int, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT c.event_kind_id, COUNT(DISTINCT c.quotation_id) FROM quotation_events c`+liveQuotation+`
		WHERE c.tenant_id = ? GROUP BY c.event_kind_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var kind int64
		var n int
		if err := rows.Scan(&kind, &n); err != nil {
			return nil, err
		}
		out[kind] = n
	}
	return out, rows.Err()
}

var _ application.QuotationRepository = (*MySQLQuotationRepository)(nil)
