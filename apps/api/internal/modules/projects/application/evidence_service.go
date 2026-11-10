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
	// ListAllByProject lists every file of the project, hidden ones (belonging to
	// something deleted) included; only the project hard delete needs it.
	ListAllByProject(ctx context.Context, projectID int64) ([]domain.Evidence, error)
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
	// SetClientVisible writes e.IsClientVisible (compare-and-set on updated_at when
	// expected is given; see infrastructure.patchRow).
	SetClientVisible(ctx context.Context, e *domain.Evidence, expected *time.Time) (bool, error)
	// DeleteByRelated hard-deletes every evidence row for one related_kind+
	// related_id pair — backs DeleteForRelated below.
	DeleteByRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) error
	// DeleteByID hard-deletes one evidence row of a project.
	DeleteByID(ctx context.Context, projectID, id int64) error
	// ExistsByStoragePath: has an evidence row already claimed this stored
	// object? (a signed upload is single-use)
	ExistsByStoragePath(ctx context.Context, storagePath string) (bool, error)
	// RelatedBelongsToProject: the item a document attaches to must be a row of the same project.
	RelatedBelongsToProject(ctx context.Context, projectID int64, kind domain.EvidenceRelatedKind, relatedID int64) (bool, error)
}

// ObjectStorage is the narrow slice of internal/shared/storage.Client this
// service depends on — kept as an interface so evidence_service_test (future)
// can fake it without a real bucket.
type ObjectStorage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// Presigner is what object storage offers when it can hand out direct,
// time-limited download links. Optional: storage that cannot is simply not
// asked, and callers fall back to streaming through the API.
type Presigner interface {
	CanPresign() bool
	PresignGet(ctx context.Context, key, fileName string, attachment bool, ttl time.Duration) (string, error)
}

// EvidenceLinkTTL is how long a direct link works. The browser uses it at once.
const EvidenceLinkTTL = 2 * time.Minute

// EvidenceRemovalObserver is told after a single evidence file was removed, so
// a rule that depends on "this item still has proof" can react. Registered by
// ClientPaymentService (a verified payment must keep at least one file).
type EvidenceRemovalObserver interface {
	AfterEvidenceRemoved(ctx context.Context, e domain.Evidence, actorStaffID int64)
}

type EvidenceService struct {
	repo     EvidenceRepository
	storage  ObjectStorage
	buildKey func(tenantID, projectID, category, filename string) string
	activity *ActivityService
	removed  []EvidenceRemovalObserver
}

// OnRemoved registers an observer of single-file removals (wiring time only).
func (s *EvidenceService) OnRemoved(o EvidenceRemovalObserver) {
	if s != nil {
		s.removed = append(s.removed, o)
	}
}

func NewEvidenceService(repo EvidenceRepository, storage ObjectStorage, buildKey func(string, string, string, string) string, activity *ActivityService) *EvidenceService {
	return &EvidenceService{repo: repo, storage: storage, buildKey: buildKey, activity: activity}
}

func (s *EvidenceService) List(ctx context.Context, projectID int64) ([]domain.Evidence, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// ListAll lists every file of the project including the ones hidden because the
// payment, vendor or kendala they hang off was deleted (see ListByProject's
// filter). Used only by the project hard delete, to remove all stored objects.
func (s *EvidenceService) ListAll(ctx context.Context, projectID int64) ([]domain.Evidence, error) {
	return s.repo.ListAllByProject(ctx, projectID)
}

// DeleteStorageKeys best-effort deletes stored objects by key (the purge calls
// it after its transaction committed).
func (s *EvidenceService) DeleteStorageKeys(ctx context.Context, keys []string) int {
	deleted := 0
	for _, k := range keys {
		if err := s.storage.Delete(ctx, k); err != nil {
			logger.Error("failed to delete storage object %q: %v", k, err)
			continue
		}
		deleted++
	}
	return deleted
}

// ListForRelated lists the files attached to one item (a client payment, a
// milestone, ...). The caller has already confirmed the item is in its project.
func (s *EvidenceService) ListForRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) ([]domain.Evidence, error) {
	return s.repo.ListByRelated(ctx, kind, relatedID)
}

// Delete removes one evidence file of a project: the row first, then (best
// effort) the stored object, then observers run, so a rule such as "a verified
// payment keeps at least one proof" is applied on every removal path. No HTTP
// route (DELETE /projects/{id}/evidence/{eid}, E8) reaches it, and the rule is
// applied on every removal path.
func (s *EvidenceService) Delete(ctx context.Context, projectID, id, actorStaffID int64) error {
	e, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return err
	}
	if e == nil {
		return apperror.NotFound("Evidence tidak ditemukan")
	}
	if err := s.repo.DeleteByID(ctx, projectID, id); err != nil {
		return err
	}
	s.DeleteStorageObjects(ctx, []domain.Evidence{*e})
	s.activity.Record(ctx, &projectID, domain.ActivityEvidenceDeleted, actorStaffID, "evidence", formatID(e.ID), e.Name,
		"Evidence dihapus: "+e.Name)
	for _, o := range s.removed {
		o.AfterEvidenceRemoved(ctx, *e, actorStaffID)
	}
	return nil
}

// ListClientDocuments backs GET /projects/{id}/documents — Client Portal's
// own "Dokumen" tab. Reachable by staff too (no principal-type branching
// here); the safety property this endpoint exists for is that it ALWAYS
// only returns general-kind, client-visible documents, not that only
// clients can call it.
func (s *EvidenceService) ListClientDocuments(ctx context.Context, projectID int64) ([]domain.Evidence, error) {
	return s.repo.ListClientVisibleGeneral(ctx, projectID)
}

// SetClientVisible sets whether a client sees a document, to exactly `visible`
// (ADR-0036). It applies to `general` documents and to
// timeline (`projectMilestone`) attachments, the two kinds staff decide on one by
// one (ADR-0032); it is rejected for any other RelatedKind, whose visibility
// follows the thing it hangs off (the field means nothing there). Setting the
// state it already has is not an error and not a write: the file comes back as it
// is. With `expected` the write is a compare-and-set on the file's updated_at (a
// stale one is a 409 carrying the current file).
func (s *EvidenceService) SetClientVisible(ctx context.Context, projectID, id int64, visible bool, expected *time.Time) (*domain.Evidence, error) {
	e, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, apperror.NotFound("Evidence tidak ditemukan")
	}
	if !e.RelatedKind.SupportsClientVisibility() {
		return nil, apperror.Validation("Visibilitas hanya berlaku untuk dokumen umum dan lampiran timeline", map[string][]string{
			"relatedKind": {"Visibilitas lampiran ini mengikuti data yang dilampirinya dan tidak dapat diubah"},
		})
	}
	if e.IsClientVisible == visible {
		return e, nil
	}
	e.IsClientVisible = visible
	wrote, err := s.repo.SetClientVisible(ctx, e, expected)
	if err != nil {
		return nil, err
	}
	return finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.Evidence, error) {
		return s.repo.FindByID(ctx, projectID, id)
	}, func(e *domain.Evidence) time.Time { return e.UpdatedAt }, apperror.NotFound("Evidence tidak ditemukan"))
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
	// IsClientVisible is only ever honored for the kinds that carry the flag
	// (general documents, timeline attachments) — see Upload's own
	// normalization below.
	IsClientVisible bool
}

func (s *EvidenceService) Upload(ctx context.Context, tenantID, projectID int64, actorStaffID int64, input UploadEvidenceInput) (*domain.Evidence, error) {
	// Checked before anything is stored: an unknown type used to reach the
	// database as a truncated enum (500) after the file was already uploaded.
	if !domain.IsValidEvidenceType(input.Type) {
		return nil, apperror.Validation("Jenis dokumen tidak valid", map[string][]string{"type": {"Pilih jenis dokumen yang tersedia"}})
	}
	// What the document attaches to must be inside this project: the project
	// in the path is the caller's own, the related id in the body is not.
	ok, err := s.repo.RelatedBelongsToProject(ctx, projectID, input.RelatedKind, input.RelatedID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Validation("Data terkait tidak ditemukan", map[string][]string{"relatedId": {"Data terkait tidak ditemukan di project ini"}})
	}
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
	// can only ever be true for a kind that carries the flag (a general
	// document or a timeline attachment) — on any other kind it would be
	// meaningless and misleading to display back.
	isClientVisible := input.IsClientVisible && input.RelatedKind.SupportsClientVisibility()

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

// Find returns one evidence row of the project (404 when it is not this
// project's). It touches no storage, so a handler can judge visibility first.
func (s *EvidenceService) Find(ctx context.Context, projectID, id int64) (*domain.Evidence, error) {
	e, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, apperror.NotFound("Evidence tidak ditemukan")
	}
	return e, nil
}

// Open streams exactly the bytes that were stored (already compressed at upload
// time — never reprocessed on read, see ADR-0010).
func (s *EvidenceService) Open(ctx context.Context, e *domain.Evidence) (io.ReadCloser, error) {
	reader, err := s.storage.Open(ctx, e.StoragePath)
	if err != nil {
		return nil, apperror.Internal("Gagal mengambil file dari object storage")
	}
	return reader, nil
}

// ErrLinkUnavailable means direct links are switched off; stream instead.
var ErrLinkUnavailable = apperror.Unsupported("Tautan langsung tidak tersedia").WithCode("presign_unavailable")

// Link returns a direct, short-lived URL to the stored file. It looks the
// evidence up inside its project first, so a link is only ever made for a
// document of the caller's own project; the handler then applies the
// client-visibility rule before handing the URL out.
func (s *EvidenceService) Link(ctx context.Context, projectID, id int64, attachment bool) (*domain.Evidence, string, time.Time, error) {
	e, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if e == nil {
		return nil, "", time.Time{}, apperror.NotFound("Evidence tidak ditemukan")
	}
	p, ok := s.storage.(Presigner)
	if !ok || !p.CanPresign() {
		return e, "", time.Time{}, ErrLinkUnavailable
	}
	url, err := p.PresignGet(ctx, e.StoragePath, e.FileName, attachment, EvidenceLinkTTL)
	if err != nil {
		return nil, "", time.Time{}, apperror.Internal("Gagal membuat tautan file")
	}
	return e, url, time.Now().Add(EvidenceLinkTTL), nil
}

var unsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFileName(name string) string {
	cleaned := unsafeFileNameChars.ReplaceAllString(name, "-")
	if cleaned == "" {
		return "file"
	}
	return cleaned
}
