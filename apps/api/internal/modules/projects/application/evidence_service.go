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
)

// maxDecodedSize is the base64-decoded size cap enforced before any
// processing — ADR-0010.
const maxDecodedSize = 15 * 1024 * 1024

type EvidenceRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.Evidence, error)
	ListByRelated(ctx context.Context, kind domain.EvidenceRelatedKind, relatedID int64) ([]domain.Evidence, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.Evidence, error)
	Create(ctx context.Context, e *domain.Evidence) error
}

// ObjectStorage is the narrow slice of internal/shared/storage.Client this
// service depends on — kept as an interface so evidence_service_test (future)
// can fake it without a real bucket.
type ObjectStorage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
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

	e := &domain.Evidence{
		ProjectID: projectID, Name: input.Name, Type: input.Type, StoragePath: key, FileName: input.FileName,
		DocumentDate: input.DocumentDate, UploadedAt: time.Now(), Description: input.Description,
		UploadedByStaffID: actorStaffID, RelatedKind: input.RelatedKind, RelatedID: input.RelatedID,
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
