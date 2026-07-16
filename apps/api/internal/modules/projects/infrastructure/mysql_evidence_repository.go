package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/projects/domain"
)

type MySQLEvidenceRepository struct {
	db *sql.DB
}

func NewMySQLEvidenceRepository(db *sql.DB) *MySQLEvidenceRepository {
	return &MySQLEvidenceRepository{db: db}
}

const evidenceColumns = `id, project_id, name, type, storage_path, file_name, document_date, uploaded_at,
	description, uploaded_by_staff_id, related_kind, related_id`

func scanEvidence(scan func(dest ...interface{}) error) (*domain.Evidence, error) {
	var e domain.Evidence
	var evidenceType, relatedKind string
	var documentDate sql.NullTime
	var description sql.NullString
	err := scan(&e.ID, &e.ProjectID, &e.Name, &evidenceType, &e.StoragePath, &e.FileName, &documentDate, &e.UploadedAt,
		&description, &e.UploadedByStaffID, &relatedKind, &e.RelatedID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Type = domain.EvidenceType(evidenceType)
	e.RelatedKind = domain.EvidenceRelatedKind(relatedKind)
	e.Description = description.String
	if documentDate.Valid {
		e.DocumentDate = &documentDate.Time
	}
	return &e, nil
}

func (r *MySQLEvidenceRepository) ListByProject(ctx context.Context, projectID int64) ([]domain.Evidence, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+evidenceColumns+` FROM evidence WHERE project_id = ? ORDER BY uploaded_at DESC, id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Evidence
	for rows.Next() {
		e, err := scanEvidence(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *e)
	}
	return list, rows.Err()
}

func (r *MySQLEvidenceRepository) ListByRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) ([]domain.Evidence, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+evidenceColumns+` FROM evidence WHERE related_kind = ? AND related_id = ? ORDER BY uploaded_at DESC`, string(kind), relatedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Evidence
	for rows.Next() {
		e, err := scanEvidence(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, *e)
	}
	return list, rows.Err()
}

func (r *MySQLEvidenceRepository) FindByID(ctx context.Context, projectID, id int64) (*domain.Evidence, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+evidenceColumns+` FROM evidence WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanEvidence(row.Scan)
}

func (r *MySQLEvidenceRepository) Create(ctx context.Context, e *domain.Evidence) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO evidence (project_id, name, type, storage_path, file_name, document_date, uploaded_at,
		 description, uploaded_by_staff_id, related_kind, related_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ProjectID, e.Name, string(e.Type), e.StoragePath, e.FileName, e.DocumentDate, e.UploadedAt,
		e.Description, e.UploadedByStaffID, string(e.RelatedKind), e.RelatedID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}
