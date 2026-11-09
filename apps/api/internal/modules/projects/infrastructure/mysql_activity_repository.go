package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLActivityRepository struct {
	db *sql.DB
}

func NewMySQLActivityRepository(db *sql.DB) *MySQLActivityRepository {
	return &MySQLActivityRepository{db: db}
}

func (r *MySQLActivityRepository) Create(ctx context.Context, entry *domain.ActivityLogEntry) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO activity_log (project_id, type, actor_staff_id, entity_type, entity_id, entity_label, description)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ProjectID, string(entry.Type), entry.ActorStaffID, entry.EntityType, entry.EntityID, entry.EntityLabel, entry.Description,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	entry.ID = id
	return nil
}

func (r *MySQLActivityRepository) ListByProject(ctx context.Context, projectID int64, limit int) ([]domain.ActivityLogEntry, error) {
	query := `SELECT id, project_id, type, actor_staff_id, entity_type, entity_id, entity_label, description, created_at
	          FROM activity_log WHERE project_id = ? ORDER BY created_at DESC, id DESC`
	args := []interface{}{projectID}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := conn(ctx, r.db).QueryContext(ctx, query, args...)
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

// ListByProjectPaged is the project's activity, newest first, one page at a
// time, optionally narrowed to one entity (entityType and/or entityId; an empty
// filter value matches everything). It returns the total number of matching
// entries for the page meta.
func (r *MySQLActivityRepository) ListByProjectPaged(ctx context.Context, projectID int64, f domain.ActivityFilter, limit, offset int) ([]domain.ActivityLogEntry, int64, error) {
	where := ` WHERE project_id = ?`
	args := []interface{}{projectID}
	if f.EntityType != "" {
		where += ` AND entity_type = ?`
		args = append(args, f.EntityType)
	}
	if f.EntityID != "" {
		where += ` AND entity_id = ?`
		args = append(args, f.EntityID)
	}
	var total int64
	if err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT COUNT(*) FROM activity_log`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT id, project_id, type, actor_staff_id, entity_type, entity_id, entity_label, description, created_at
		 FROM activity_log`+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := []domain.ActivityLogEntry{}
	for rows.Next() {
		var e domain.ActivityLogEntry
		var pid sql.NullInt64
		var activityType string
		if err := rows.Scan(&e.ID, &pid, &activityType, &e.ActorStaffID, &e.EntityType, &e.EntityID, &e.EntityLabel, &e.Description, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.Type = domain.ActivityType(activityType)
		if pid.Valid {
			e.ProjectID = &pid.Int64
		}
		list = append(list, e)
	}
	return list, total, rows.Err()
}
