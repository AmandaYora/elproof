package application

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/filetype"
	"elproof/internal/shared/logger"
)

// maxCoverDecodedSize caps a cover photo before it is processed: a phone photo
// the browser already compressed is far smaller; the server re-encodes it anyway.
const maxCoverDecodedSize = 6 * 1024 * 1024

// coverCategory is the key segment cover photos are stored under.
const coverCategory = "cover"

// allowedCoverMimeTypes: the photo formats a browser can show as a background.
var allowedCoverMimeTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// CoverRepository is the narrow storage of a project's cover key.
type CoverRepository interface {
	// LockCover reads the current key under a row lock (inside a transaction);
	// found is false when the project is not the tenant's.
	LockCover(ctx context.Context, tenantID, id int64) (path *string, found bool, err error)
	// SetCover stores the key (nil clears it) without bumping the project's version.
	SetCover(ctx context.Context, tenantID, id int64, path *string, at time.Time) error
}

// CoverInput is PUT /projects/{id}/cover's body.
type CoverInput struct {
	FileName   string
	MimeType   string
	Base64Data string
}

// CoverService owns a project's Client Portal cover photo (ADR-0047): upload or
// replace it, remove it, stream it.
type CoverService struct {
	projects *ProjectService
	repo     CoverRepository
	tx       Transactor
	storage  ObjectStorage
	buildKey func(tenantID, projectID, category, filename string) string
	activity *ActivityService
	now      func() time.Time
}

func NewCoverService(projects *ProjectService, repo CoverRepository, tx Transactor, storage ObjectStorage,
	buildKey func(string, string, string, string) string, activity *ActivityService) *CoverService {
	if tx == nil {
		tx = noTransactor{}
	}
	return &CoverService{projects: projects, repo: repo, tx: tx, storage: storage, buildKey: buildKey, activity: activity, now: time.Now}
}

func coverError(msg string) error {
	return apperror.Validation(msg, map[string][]string{"base64Data": {msg}})
}

// prepareCover checks the uploaded photo and re-encodes it (downscaled to
// compress.MaxDimension, metadata stripped). Every refusal is a 422 on
// "mimeType" or "base64Data".
func prepareCover(in CoverInput) ([]byte, string, error) {
	mimeType := normalizeMime(in.MimeType)
	if !allowedCoverMimeTypes[mimeType] {
		msg := "Foto sampul harus JPG, PNG, atau WebP."
		return nil, "", apperror.Validation(msg, map[string][]string{"mimeType": {msg}})
	}
	decoded, err := base64.StdEncoding.DecodeString(in.Base64Data)
	if err != nil {
		return nil, "", coverError("Data foto tidak terbaca. Pilih fotonya lagi.")
	}
	if len(decoded) == 0 {
		return nil, "", coverError("Foto tidak boleh kosong.")
	}
	if len(decoded) > maxCoverDecodedSize {
		return nil, "", coverError("Ukuran foto maksimal 6 MB.")
	}
	if !filetype.Matches(decoded, mimeType) {
		return nil, "", coverError(notTheDeclaredType(mimeType))
	}
	out, outMime, err := compress.Photo(decoded, mimeType)
	switch {
	case errors.Is(err, compress.ErrPhotoTooLarge):
		return nil, "", coverError("Resolusi foto terlalu besar (maksimal 40 megapiksel). Pilih foto yang lebih kecil.")
	case errors.Is(err, compress.ErrNotAPhoto):
		return nil, "", coverError(notTheDeclaredType(mimeType))
	case err != nil:
		return nil, "", apperror.Internal("Gagal memproses foto")
	}
	return out, outMime, nil
}

func coverExtension(mimeType string) string {
	if mimeType == "image/png" {
		return "png"
	}
	return "jpg"
}

// coverContentType is the MIME type of a stored cover, from its key.
func coverContentType(key string) string {
	if strings.EqualFold(path.Ext(key), ".png") {
		return "image/png"
	}
	return "image/jpeg"
}

// Set stores a new cover photo and points the project at it; the previous
// photo's object is deleted afterwards (best effort). It returns the project as
// it now stands.
func (s *CoverService) Set(ctx context.Context, tenantID, projectID, actorStaffID int64, in CoverInput) (*domain.Project, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	data, mimeType, err := prepareCover(in)
	if err != nil {
		return nil, err
	}
	now := s.now()
	key := s.buildKey(strconv.FormatInt(tenantID, 10), strconv.FormatInt(projectID, 10), coverCategory,
		strconv.FormatInt(now.Unix(), 10)+"-"+uuid.NewString()+"."+coverExtension(mimeType))
	if _, err := s.storage.Save(ctx, key, data, mimeType); err != nil {
		logger.Error("failed to store cover %q: %v", key, err)
		return nil, apperror.Internal("Gagal mengunggah foto ke object storage")
	}
	var previous *string
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		old, found, err := s.repo.LockCover(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		if !found {
			return apperror.NotFound("Proyek tidak ditemukan")
		}
		previous = old
		return s.repo.SetCover(ctx, tenantID, projectID, &key, now)
	})
	if err != nil {
		s.deleteObject(ctx, key)
		return nil, err
	}
	if previous != nil && *previous != key {
		s.deleteObject(ctx, *previous)
	}
	desc := "Foto sampul portal diunggah"
	if previous != nil {
		desc = "Foto sampul portal diganti"
	}
	s.activity.Record(ctx, &projectID, domain.ActivityProjectUpdated, actorStaffID, "project", formatID(projectID), "", desc)
	return s.projects.Get(ctx, tenantID, projectID)
}

// Remove clears the cover photo (a no-op when there is none) and deletes its
// object afterwards (best effort).
func (s *CoverService) Remove(ctx context.Context, tenantID, projectID, actorStaffID int64) error {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return err
	}
	var previous *string
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		old, found, err := s.repo.LockCover(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		if !found {
			return apperror.NotFound("Proyek tidak ditemukan")
		}
		if old == nil {
			return nil
		}
		previous = old
		return s.repo.SetCover(ctx, tenantID, projectID, nil, s.now())
	})
	if err != nil {
		return err
	}
	if previous != nil {
		s.deleteObject(ctx, *previous)
		s.activity.Record(ctx, &projectID, domain.ActivityProjectUpdated, actorStaffID, "project", formatID(projectID), "",
			"Foto sampul portal dihapus")
	}
	return nil
}

// Open streams the project's cover photo. 404 when the project has none.
func (s *CoverService) Open(ctx context.Context, tenantID, projectID int64) (io.ReadCloser, string, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, "", err
	}
	if p.CoverPhotoPath == nil {
		return nil, "", apperror.NotFound("Proyek ini belum punya foto sampul")
	}
	reader, err := s.storage.Open(ctx, *p.CoverPhotoPath)
	if err != nil {
		logger.Error("failed to open cover %q: %v", *p.CoverPhotoPath, err)
		return nil, "", apperror.NotFound("Foto sampul tidak ditemukan")
	}
	return reader, coverContentType(*p.CoverPhotoPath), nil
}

func (s *CoverService) deleteObject(ctx context.Context, key string) {
	if err := s.storage.Delete(ctx, key); err != nil {
		logger.Error("failed to delete cover object %q: %v", key, err)
	}
}
