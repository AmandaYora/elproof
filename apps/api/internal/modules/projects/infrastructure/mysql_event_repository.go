package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

// MySQLEventRepository stores project_events and their project_event_sessions.
// Writes that touch both tables run inside the caller's transaction (the
// event service always writes under inProjectTx).
type MySQLEventRepository struct {
	db *sql.DB
}

func NewMySQLEventRepository(db *sql.DB) *MySQLEventRepository {
	return &MySQLEventRepository{db: db}
}

const eventColumns = `id, tenant_id, project_id, event_kind_id, kind_name, event_date, start_time, end_time, venue_supplier_id,
	layout, is_main, sort_order, updated_at`

func scanEvent(scan func(dest ...interface{}) error) (*domain.ProjectEvent, error) {
	var e domain.ProjectEvent
	var start, end sql.NullString
	var venue sql.NullInt64
	var layout string
	err := scan(&e.ID, &e.TenantID, &e.ProjectID, &e.EventKindID, &e.KindName, &e.EventDate, &start, &end, &venue,
		&layout, &e.IsMain, &e.SortOrder, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.StartTime = nullString(start)
	e.EndTime = nullString(end)
	if venue.Valid {
		v := venue.Int64
		e.VenueSupplierID = &v
	}
	e.Layout = domain.Layout(layout)
	return &e, nil
}

func nullString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

// list reads events, then their sessions in one more query.
func (r *MySQLEventRepository) list(ctx context.Context, query string, args ...any) ([]domain.ProjectEvent, error) {
	q := conn(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var list []domain.ProjectEvent
	for rows.Next() {
		e, err := scanEvent(rows.Scan)
		if err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, *e)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(list) == 0 {
		return list, nil
	}
	ids := make([]int64, len(list))
	at := make(map[int64]int, len(list))
	for i, e := range list {
		ids[i] = e.ID
		at[e.ID] = i
	}
	srows, err := q.QueryContext(ctx, `SELECT id, event_id, label, start_time, end_time, guests, sort_order
		FROM project_event_sessions WHERE event_id IN (`+utils.Placeholders(len(ids))+`) ORDER BY event_id, sort_order, id`,
		utils.Int64Args(ids)...)
	if err != nil {
		return nil, err
	}
	defer srows.Close()
	for srows.Next() {
		var s domain.EventSession
		var start, end sql.NullString
		if err := srows.Scan(&s.ID, &s.EventID, &s.Label, &start, &end, &s.Guests, &s.SortOrder); err != nil {
			return nil, err
		}
		s.StartTime = nullString(start)
		s.EndTime = nullString(end)
		i := at[s.EventID]
		list[i].Sessions = append(list[i].Sessions, s)
	}
	if err := srows.Err(); err != nil {
		return nil, err
	}
	domain.SortEvents(list)
	return list, nil
}

func (r *MySQLEventRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectEvent, error) {
	return r.list(ctx, `SELECT `+eventColumns+` FROM project_events WHERE project_id = ?`, projectID)
}

// ListByProjects is every event of the given projects (sorted per project by
// domain.SortEvents; callers group by ProjectID).
func (r *MySQLEventRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectEvent, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	return r.list(ctx, `SELECT `+eventColumns+` FROM project_events WHERE project_id IN (`+utils.Placeholders(len(projectIDs))+`)`,
		utils.Int64Args(projectIDs)...)
}

func (r *MySQLEventRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectEvent, error) {
	list, err := r.list(ctx, `SELECT `+eventColumns+` FROM project_events WHERE project_id = ? AND id = ?`, projectID, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

// Create inserts the event with the project's tenant, then its sessions.
func (r *MySQLEventRepository) Create(ctx context.Context, e *domain.ProjectEvent) error {
	q := conn(ctx, r.db)
	result, err := q.ExecContext(ctx,
		`INSERT INTO project_events (tenant_id, project_id, event_kind_id, kind_name, event_date, start_time, end_time,
		 venue_supplier_id, layout, is_main, sort_order)
		 SELECT tenant_id, id, ?, ?, ?, ?, ?, ?, ?, ?, ? FROM projects WHERE id = ?`,
		e.EventKindID, e.KindName, e.EventDate, e.StartTime, e.EndTime, e.VenueSupplierID, string(e.Layout), e.IsMain, e.SortOrder,
		e.ProjectID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	if err := q.QueryRowContext(ctx, `SELECT tenant_id FROM project_events WHERE id = ?`, id).Scan(&e.TenantID); err != nil {
		return err
	}
	if err := r.insertSessions(ctx, e); err != nil {
		return err
	}
	return readVersion(ctx, q, "project_events", id, &e.UpdatedAt)
}

func (r *MySQLEventRepository) insertSessions(ctx context.Context, e *domain.ProjectEvent) error {
	q := conn(ctx, r.db)
	for i := range e.Sessions {
		s := &e.Sessions[i]
		s.EventID = e.ID
		s.SortOrder = i
		res, err := q.ExecContext(ctx,
			`INSERT INTO project_event_sessions (tenant_id, event_id, label, start_time, end_time, guests, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			e.TenantID, e.ID, s.Label, s.StartTime, s.EndTime, s.Guests, s.SortOrder)
		if err != nil {
			return err
		}
		if s.ID, err = res.LastInsertId(); err != nil {
			return err
		}
	}
	return nil
}

// Replace writes every field of the event and replaces its sessions. The
// UPDATE always stamps a new version (a change only to the sessions still
// moves updated_at), so with expected it is a true compare-and-set.
func (r *MySQLEventRepository) Replace(ctx context.Context, e *domain.ProjectEvent, expected *time.Time) (bool, error) {
	q := conn(ctx, r.db)
	query := `UPDATE project_events SET event_kind_id = ?, kind_name = ?, event_date = ?, start_time = ?, end_time = ?,
		venue_supplier_id = ?, layout = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE project_id = ? AND id = ?`
	args := []any{e.EventKindID, e.KindName, e.EventDate, e.StartTime, e.EndTime, e.VenueSupplierID, string(e.Layout), e.ProjectID, e.ID}
	if expected != nil {
		query += ` AND updated_at = ?`
		args = append(args, *expected)
	}
	res, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	if e.TenantID == 0 {
		if err := q.QueryRowContext(ctx, `SELECT tenant_id FROM project_events WHERE id = ?`, e.ID).Scan(&e.TenantID); err != nil {
			return false, err
		}
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM project_event_sessions WHERE event_id = ?`, e.ID); err != nil {
		return false, err
	}
	if err := r.insertSessions(ctx, e); err != nil {
		return false, err
	}
	return true, readVersion(ctx, q, "project_events", e.ID, &e.UpdatedAt)
}

// Delete removes the event; its sessions go with it (ON DELETE CASCADE).
func (r *MySQLEventRepository) Delete(ctx context.Context, projectID, id int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM project_events WHERE project_id = ? AND id = ?`, projectID, id)
	return err
}

// SetMain makes id the one main event of the project in a single statement.
func (r *MySQLEventRepository) SetMain(ctx context.Context, projectID, id int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE project_events SET is_main = (id = ?) WHERE project_id = ?`, id, projectID)
	return err
}

func (r *MySQLEventRepository) SetSortOrder(ctx context.Context, projectID, id int64, order int) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `UPDATE project_events SET sort_order = ? WHERE project_id = ? AND id = ?`, order, projectID, id)
	return err
}

func (r *MySQLEventRepository) CountByVenueSupplier(ctx context.Context, projectID, supplierID int64) (int, error) {
	var n int
	err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*) FROM project_events WHERE project_id = ? AND venue_supplier_id = ?`,
		projectID, supplierID).Scan(&n)
	return n, err
}

func (r *MySQLEventRepository) NextSortOrder(ctx context.Context, projectID int64) (int, error) {
	var maxOrder sql.NullInt64
	if err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT MAX(sort_order) FROM project_events WHERE project_id = ?`, projectID).Scan(&maxOrder); err != nil {
		return 0, err
	}
	if !maxOrder.Valid {
		return 0, nil
	}
	return int(maxOrder.Int64) + 1, nil
}
