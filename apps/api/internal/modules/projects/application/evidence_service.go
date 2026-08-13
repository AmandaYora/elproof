package application

import (
	"context"
	"encoding/base64"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/logger"
)

// maxDecodedSize is the base64-decoded size cap enforced before any
// processing — ADR-0010.
const maxDecodedSize = 15 * 1024 * 1024

type EvidenceRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.Evidence, error)
	// ListByProjects backs ComputeProgressBatch: every matching row across
	// the given projects in one query (WHERE project_id IN (...)).
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.Evidence, error)
	ListByRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) ([]domain.Evidence, error)
	// ListClientVisibleGeneral backs Client Portal's own "Dokumen" tab
	// (EvidenceService.ListClientDocuments) — always filters
	// related_kind='general' AND is_client_visible=true, unconditionally,
	// regardless of caller. Deliberately its own query rather than a filter
	// bolted onto ListByProject, so this is trivial to audit as "only ever
	// returns documents safe to show a client."
	ListClientVisibleGeneral(ctx context.Context, projectID int64) ([]domain.Evidence, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.Evidence, error)
	Create(ctx context.Context, e *domain.Evidence) error
	SetClientVisible(ctx context.Context, projectID, id int64, visible bool) error
	// DeleteByRelated hard-deletes every evidence row for one related_kind+
	// related_id pair — backs DeleteForRelated below.
	DeleteByRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) error
}

// ObjectStorage is the narrow slice of internal/shared/storage.Client this
// service depends on — kept as an interface so evidence_service_test (future)
// can fake it without a real bucket.
type ObjectStorage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type EvidenceService struct {
	repo     EvidenceRepository
	storage  ObjectStorage
	buildKey func(tenantID, projectID, category, filename string) string
	activity *ActivityService
}

func NewEvidenceService(repo EvidenceRepository, storage ObjectStorage, buildKey func(string, string, string, string) string, activity *ActivityService) *EvidenceService {
	return &EvidenceService{repo: repo, storage: storage, buildKey: buildKey, activity: activity}
}

func (s *EvidenceService) List(ctx context.Context, projectID int64) ([]domain.Evidence, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// ListClientDocuments backs GET /projects/{id}/documents — Client Portal's
// own "Dokumen" tab. Reachable by staff too (no principal-type branching
// here); the safety property this endpoint exists for is that it ALWAYS
// only returns general-kind, client-visible documents, not that only
// clients can call it.
func (s *EvidenceService) ListClientDocuments(ctx context.Context, projectID int64) ([]domain.Evidence, error) {
	return s.repo.ListClientVisibleGeneral(ctx, projectID)
}

// ToggleClientVisible flips a `general`-kind document's client visibility
// without needing to re-upload the file — rejected for any other
// RelatedKind, since visibility only ever means anything for a document
// with no other, already-visible-to-the-client context to inherit from
// (every other kind stays unconditionally client-visible, unaffected by
// this field entirely).
func (s *EvidenceService) ToggleClientVisible(ctx context.Context, projectID, id int64) (*domain.Evidence, error) {
	e, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, apperror.NotFound("Evidence tidak ditemukan")
	}
	if e.RelatedKind != domain.RelatedGeneral {
		return nil, apperror.Validation("Visibilitas hanya berlaku untuk dokumen umum", map[string][]string{
			"relatedKind": {"Evidence ini bukan dokumen umum, visibilitasnya tidak dapat diubah"},
		})
	}
	newVisible := !e.IsClientVisible
	if err := s.repo.SetClientVisible(ctx, projectID, id, newVisible); err != nil {
		return nil, err
	}
	e.IsClientVisible = newVisible
	return e, nil
}

// ListByProjects backs ComputeProgressBatch — see EvidenceRepository.
func (s *EvidenceService) ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.Evidence, error) {
	return s.repo.ListByProjects(ctx, projectIDs)
}

// DeleteStorageObjects best-effort deletes each evidence's file from object
// storage — used only by ProjectService.Delete (hard delete, ADR-0013),
// always called AFTER that project's DB rows are already gone. A failure
// here is logged, never returned/blocking: the worst case is an orphaned S3
// object, which is strictly safer than the reverse (a DB row surviving with
// a reference to a file that no longer exists).
func (s *EvidenceService) DeleteStorageObjects(ctx context.Context, evidences []domain.Evidence) {
	for _, e := range evidences {
		if err := s.storage.Delete(ctx, e.StoragePath); err != nil {
			logger.Error("failed to delete storage object %q for evidence %d: %v", e.StoragePath, e.ID, err)
		}
	}
}

// DeleteForRelated hard-deletes every evidence row (and, best-effort, its S3
// object) attached to one specific related_kind+related_id pair -- the
// cascade a payment Delete needs (PLAN.md "Cascade evidence: hapus evidence
// DULU, baru payment") so a hard-deleted payment never leaves behind an
// evidence row that can never be deleted through any other path (evidence
// itself has no standalone delete). Deletes the DB rows first and only then
// best-effort deletes their S3 objects, so a failure here always aborts
// before the caller's own row is touched -- the worst case is an orphaned S3
// object, never an orphaned DB row.
func (s *EvidenceService) DeleteForRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) error {
	list, err := s.repo.ListByRelated(ctx, kind, relatedID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteByRelated(ctx, kind, relatedID); err != nil {
		return err
	}
	s.DeleteStorageObjects(ctx, list)
	return nil
}

type UploadEvidenceInput struct {
	Name         string
	Type         domain.EvidenceType
	FileName     string
	MimeType     string
	Base64Data   string
	DocumentDate *time.Time
	Description  string
	RelatedKind  domain.EvidenceRelatedKind
	RelatedID    int64
	// IsClientVisible is only ever honored when RelatedKind is
	// RelatedGeneral — see Upload's own normalization below.
	IsClientVisible bool
}

func (s *EvidenceService) Upload(ctx context.Context, tenantID, projectID int64, actorStaffID int64, input UploadEvidenceInput) (*domain.Evidence, error) {
	decoded, err := base64.StdEncoding.DecodeString(input.Base64Data)
	if err != nil {
		return nil, apperror.Validation("Data file tidak valid", map[string][]string{"base64Data": {"Gagal membaca data file"}})
	}
	if len(decoded) == 0 {
		return nil, apperror.Validation("File kosong", map[string][]string{"base64Data": {"File tidak boleh kosong"}})
	}
	if len(decoded) > maxDecodedSize {
		return nil, apperror.Validation("Ukuran file terlalu besar", map[string][]string{"base64Data": {"Maksimal 15 MB"}})
	}

	// Backend re-compression is the authoritative pass — see ADR-0010. Runs
	// regardless of whether the frontend already compressed the image.
	compressed, err := compress.Image(decoded, input.MimeType)
	if err != nil {
		return nil, apperror.Internal("Gagal memproses file")
	}

	key := s.buildKey(
		strconv.FormatInt(tenantID, 10),
		strconv.FormatInt(projectID, 10),
		strings.ToLower(string(input.RelatedKind)),
		uuid.NewString()+"-"+sanitizeFileName(input.FileName),
	)
	if _, err := s.storage.Save(ctx, key, compressed, input.MimeType); err != nil {
		return nil, apperror.Internal("Gagal mengunggah file ke object storage")
	}

	// Server-side, not just trusted from the request body: IsClientVisible
	// can only ever be true for a general-kind document — a client-visible
	// flag on any other kind would be meaningless (those stay unconditionally
	// visible regardless) and misleading to display back.
	isClientVisible := input.IsClientVisible && input.RelatedKind == domain.RelatedGeneral

	e := &domain.Evidence{
		ProjectID: projectID, Name: input.Name, Type: input.Type, StoragePath: key, FileName: input.FileName,
		DocumentDate: input.DocumentDate, UploadedAt: time.Now(), Description: input.Description,
		UploadedByStaffID: actorStaffID, RelatedKind: input.RelatedKind, RelatedID: input.RelatedID,
		IsClientVisible: isClientVisible,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityEvidenceUploaded, actorStaffID, "evidence", formatID(e.ID), e.Name,
		"Evidence diunggah: "+e.Name)
	return e, nil
}

// Download returns the evidence metadata plus a stream of exactly the bytes
// that were stored (already compressed at upload time — never reprocessed on
// read, see ADR-0010).
func (s *EvidenceService) Download(ctx context.Context, projectID, id int64) (*domain.Evidence, io.ReadCloser, error) {
	e, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, nil, err
	}
	if e == nil {
		return nil, nil, apperror.NotFound("Evidence tidak ditemukan")
	}
	reader, err := s.storage.Open(ctx, e.StoragePath)
	if err != nil {
		return nil, nil, apperror.Internal("Gagal mengambil file dari object storage")
	}
	return e, reader, nil
}

var unsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFileName(name string) string {
	cleaned := unsafeFileNameChars.ReplaceAllString(name, "-")
	if cleaned == "" {
		return "file"
	}
	return cleaned
}
