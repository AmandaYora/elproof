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
	result, err := r.db.ExecContext(ctx,
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

	rows, err := r.db.QueryContext(ctx, query, args...)
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
