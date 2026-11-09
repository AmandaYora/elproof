package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/utils"
)

type MySQLEvidenceRepository struct {
	db *sql.DB
}

func NewMySQLEvidenceRepository(db *sql.DB) *MySQLEvidenceRepository {
	return &MySQLEvidenceRepository{db: db}
}

const evidenceColumns = `id, project_id, name, type, storage_path, file_name, document_date, uploaded_at,
	description, uploaded_by_staff_id, related_kind, related_id, is_client_visible`

func scanEvidence(scan func(dest ...interface{}) error) (*domain.Evidence, error) {
	var e domain.Evidence
	var evidenceType, relatedKind string
	var documentDate sql.NullTime
	var description sql.NullString
	err := scan(&e.ID, &e.ProjectID, &e.Name, &evidenceType, &e.StoragePath, &e.FileName, &documentDate, &e.UploadedAt,
		&description, &e.UploadedByStaffID, &relatedKind, &e.RelatedID, &e.IsClientVisible)
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
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+evidenceColumns+` FROM evidence WHERE project_id = ? ORDER BY uploaded_at DESC, id DESC`, projectID)
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

// ListByProjects backs ComputeProgressBatch — one query across every id in
// projectIDs instead of one query per project (PLAN.md "Performance
// remediation"); evidence.project_id is already denormalized on every row
// (DB_SCHEMA.md), so this applies the same batching shape here too.
func (r *MySQLEvidenceRepository) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.Evidence, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	query := `SELECT ` + evidenceColumns + ` FROM evidence WHERE project_id IN (` + utils.Placeholders(len(projectIDs)) + `) ORDER BY project_id, uploaded_at DESC, id DESC`
	rows, err := conn(ctx, r.db).QueryContext(ctx, query, utils.Int64Args(projectIDs)...)
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
	rows, err := conn(ctx, r.db).QueryContext(ctx, `SELECT `+evidenceColumns+` FROM evidence WHERE related_kind = ? AND related_id = ? ORDER BY uploaded_at DESC`, string(kind), relatedID)
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

// ListClientVisibleGeneral backs Client Portal's own "Dokumen" tab — see
// EvidenceRepository.ListClientVisibleGeneral's doc comment.
func (r *MySQLEvidenceRepository) ListClientVisibleGeneral(ctx context.Context, projectID int64) ([]domain.Evidence, error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx,
		`SELECT `+evidenceColumns+` FROM evidence WHERE project_id = ? AND related_kind = ? AND is_client_visible = TRUE ORDER BY uploaded_at DESC, id DESC`,
		projectID, string(domain.RelatedGeneral),
	)
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
	row := conn(ctx, r.db).QueryRowContext(ctx, `SELECT `+evidenceColumns+` FROM evidence WHERE project_id = ? AND id = ? LIMIT 1`, projectID, id)
	return scanEvidence(row.Scan)
}

func (r *MySQLEvidenceRepository) Create(ctx context.Context, e *domain.Evidence) error {
	result, err := conn(ctx, r.db).ExecContext(ctx,
		`INSERT INTO evidence (project_id, name, type, storage_path, file_name, document_date, uploaded_at,
		 description, uploaded_by_staff_id, related_kind, related_id, is_client_visible)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ProjectID, e.Name, string(e.Type), e.StoragePath, e.FileName, e.DocumentDate, e.UploadedAt,
		e.Description, e.UploadedByStaffID, string(e.RelatedKind), e.RelatedID, e.IsClientVisible,
	)
	if err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return domain.ErrStoragePathTaken
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

// ExistsByStoragePath reports whether an evidence row already points at the
// stored object (a signed upload is single-use: once a row owns the object, the
// token that named it is spent).
func (r *MySQLEvidenceRepository) ExistsByStoragePath(ctx context.Context, storagePath string) (bool, error) {
	var one int
	err := conn(ctx, r.db).QueryRowContext(ctx, `SELECT 1 FROM evidence WHERE storage_path = ? LIMIT 1`, storagePath).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *MySQLEvidenceRepository) SetClientVisible(ctx context.Context, projectID, id int64, visible bool) error {
	_, err := conn(ctx, r.db).ExecContext(ctx,
		`UPDATE evidence SET is_client_visible = ? WHERE project_id = ? AND id = ?`,
		visible, projectID, id,
	)
	return err
}

func (r *MySQLEvidenceRepository) DeleteByRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM evidence WHERE related_kind = ? AND related_id = ?`, string(kind), relatedID)
	return err
}

// DeleteByID hard-deletes one evidence row of a project.
func (r *MySQLEvidenceRepository) DeleteByID(ctx context.Context, projectID, id int64) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM evidence WHERE project_id = ? AND id = ?`, projectID, id)
	return err
}

// relatedOwnershipQueries: per kind, one query that is true only when the
// thing the document attaches to is a row of this very project.
var relatedOwnershipQueries = map[domain.EvidenceRelatedKind]string{
	domain.RelatedVendorMilestone:  `SELECT 1 FROM vendor_milestones vm JOIN project_vendors pv ON pv.id = vm.project_vendor_id WHERE vm.id = ? AND pv.project_id = ?`,
	domain.RelatedPayment:          `SELECT 1 FROM vendor_payments WHERE id = ? AND project_id = ?`,
	domain.RelatedProjectVendor:    `SELECT 1 FROM project_vendors WHERE id = ? AND project_id = ?`,
	domain.RelatedIssue:            `SELECT 1 FROM vendor_issues WHERE id = ? AND project_id = ?`,
	domain.RelatedClientPayment:    `SELECT 1 FROM client_payments WHERE id = ? AND project_id = ?`,
	domain.RelatedVenuePayment:     `SELECT 1 FROM venue_payments WHERE id = ? AND project_id = ?`,
	domain.RelatedProjectMilestone: `SELECT 1 FROM project_milestones WHERE id = ? AND project_id = ?`,
}

// RelatedBelongsToProject reports whether a document may attach to
// (kind, relatedID) inside projectID. A general document attaches to nothing
// in particular and is always fine; an unknown kind never is.
func (r *MySQLEvidenceRepository) RelatedBelongsToProject(ctx context.Context, projectID int64, kind domain.EvidenceRelatedKind, relatedID int64) (bool, error) {
	if kind == domain.RelatedGeneral {
		return true, nil
	}
	query, known := relatedOwnershipQueries[kind]
	if !known {
		return false, nil
	}
	var one int
	err := conn(ctx, r.db).QueryRowContext(ctx, query, relatedID, projectID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
