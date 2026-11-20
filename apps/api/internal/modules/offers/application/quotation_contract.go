package application

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/filetype"
)

// Messages of the conversion contract and the evidence of acceptance.
const (
	MsgQuotationNotAccepted   = "Penawaran ini belum diterima. Tandai diterima dulu sebelum dijadikan proyek."
	CodeQuotationNotAccepted  = "quotation_not_accepted"
	MsgAttachmentNotAccepted  = "Lampiran hanya untuk penawaran yang sudah diterima"
	MsgAttachmentType         = "Lampiran harus foto (JPEG, PNG, WebP) atau PDF"
	MsgAttachmentTooLarge     = "Ukuran lampiran maksimal 10 MB"
	MsgAttachmentUnreadable   = "Berkas ini tidak bisa dibuka sebagai foto atau PDF"
	MsgAttachmentNotFound     = "Penawaran ini belum punya lampiran"
	MsgAttachmentSaveFailed   = "Gagal menyimpan lampiran"
	MsgAttachmentReadFailed   = "Gagal mengambil lampiran"
	MaxAcceptanceAttachmentMB = 10
)

// ---- conversion contract (PLAN §5.4; used by projects through contracts) ----

// AcceptedQuotation is an accepted quotation with the revision it was
// accepted on: what "Jadikan proyek" copies. A quotation that is not
// Accepted answers 422 quotation_not_accepted; a missing one (or another
// tenant's) 404. A quotation already converted still answers (the
// conversion endpoint is idempotent).
func (s *QuotationService) AcceptedQuotation(ctx context.Context, tenantID, quotationID int64) (*domain.Quotation, *domain.QuotationRevision, error) {
	q, err := s.get(ctx, tenantID, quotationID, Viewer{AllQuotations: true})
	if err != nil {
		return nil, nil, err
	}
	if q.Status != domain.QuotationAccepted || q.AcceptedRevisionID == nil {
		return nil, nil, apperror.Validation(MsgQuotationNotAccepted, map[string][]string{"status": {MsgQuotationNotAccepted}}).WithCode(CodeQuotationNotAccepted)
	}
	rev, err := s.repo.FindRevisionByID(ctx, tenantID, *q.AcceptedRevisionID)
	if err != nil {
		return nil, nil, err
	}
	if rev == nil || rev.QuotationID != q.ID {
		return nil, nil, apperror.NotFound(MsgRevisionNotFound)
	}
	return q, rev, nil
}

// MarkConverted records that projectID was made from the quotation: it
// becomes read-only. Idempotent; a different project id replaces the cache
// (the projects module is the source of truth).
func (s *QuotationService) MarkConverted(ctx context.Context, tenantID, quotationID, projectID int64) error {
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, quotationID, Viewer{AllQuotations: true})
		if err != nil {
			return err
		}
		if q.ConvertedProjectID != nil && *q.ConvertedProjectID == projectID && q.ProjectDeletedAt == nil {
			return nil
		}
		if q.Status != domain.QuotationAccepted {
			return apperror.Validation(MsgQuotationNotAccepted, map[string][]string{"status": {MsgQuotationNotAccepted}}).WithCode(CodeQuotationNotAccepted)
		}
		at := s.now()
		q.ConvertedProjectID, q.ConvertedAt, q.ProjectDeletedAt = &projectID, &at, nil
		if err := s.repo.SaveState(ctx, q); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, q.ID, Viewer{}, domain.ActivityConverted, "Dijadikan proyek")
	})
}

// MarkProjectDeleted records that the project made from a quotation was
// deleted permanently: the quotation stays converted ("Proyek sudah
// dihapus"). A project made without a quotation is a no-op.
func (s *QuotationService) MarkProjectDeleted(ctx context.Context, tenantID, projectID int64) error {
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		found, err := s.repo.FindByProject(ctx, tenantID, projectID)
		if err != nil || found == nil {
			return err
		}
		q, err := s.repo.Lock(ctx, tenantID, found.ID)
		if err != nil || q == nil || q.ProjectDeletedAt != nil {
			return err
		}
		at := s.now()
		q.ProjectDeletedAt = &at
		if err := s.repo.SaveState(ctx, q); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, q.ID, Viewer{}, domain.ActivityProjectDeleted, "Proyek hasil penawaran ini sudah dihapus")
	})
}

// ---- evidence of acceptance -------------------------------------------------

var unsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	cleaned := unsafeFileNameChars.ReplaceAllString(base, "-")
	if cleaned == "" || cleaned == "." || cleaned == "-" {
		return "lampiran"
	}
	if len(cleaned) > 120 {
		cleaned = cleaned[len(cleaned)-120:]
	}
	return cleaned
}

// AttachmentInput is the optional evidence of acceptance (a photo or PDF of
// the signed quotation).
type AttachmentInput struct {
	FileName string
	MimeType string
	Data     []byte
}

// SetAcceptanceAttachment stores the evidence of an accepted quotation
// (replacing an earlier one). Photos are re-encoded without their metadata;
// a PDF is checked by its signature.
func (s *QuotationService) SetAcceptanceAttachment(ctx context.Context, tenantID, id int64, v Viewer, in AttachmentInput, expected *time.Time) (*domain.Quotation, error) {
	q, err := s.get(ctx, tenantID, id, v)
	if err != nil {
		return nil, err
	}
	if q.Status != domain.QuotationAccepted {
		return nil, apperror.Validation(MsgAttachmentNotAccepted, map[string][]string{"file": {MsgAttachmentNotAccepted}}).WithCode(CodeQuotationNotAccepted)
	}
	if expected != nil && !etag.Same(*expected, q.UpdatedAt) {
		return nil, apperror.Stale(q)
	}
	if len(in.Data) == 0 || len(in.Data) > MaxAcceptanceAttachmentMB<<20 {
		msg := MsgAttachmentTooLarge
		if len(in.Data) == 0 {
			msg = MsgAttachmentUnreadable
		}
		return nil, apperror.Validation(msg, map[string][]string{"base64Data": {msg}})
	}
	data, mime := in.Data, strings.ToLower(strings.TrimSpace(in.MimeType))
	switch mime {
	case "image/jpeg", "image/jpg", "image/png", "image/webp":
		if mime == "image/jpg" {
			mime = "image/jpeg"
		}
		out, outMime, err := compress.Photo(data, mime)
		if err != nil {
			return nil, apperror.Validation(MsgAttachmentUnreadable, map[string][]string{"base64Data": {MsgAttachmentUnreadable}})
		}
		data, mime = out, outMime
	case "application/pdf":
		if !filetype.Matches(data, mime) {
			return nil, apperror.Validation(MsgAttachmentUnreadable, map[string][]string{"base64Data": {MsgAttachmentUnreadable}})
		}
	default:
		return nil, apperror.Validation(MsgAttachmentType, map[string][]string{"mimeType": {MsgAttachmentType}})
	}
	if s.files == nil {
		return nil, apperror.Internal(MsgAttachmentSaveFailed)
	}
	name := safeFileName(in.FileName)
	key := fmt.Sprintf("elproof/upload/%s/0/quotation-acceptance/%s-%s", strconv.FormatInt(tenantID, 10), uuid.NewString(), name)
	if _, err := s.files.Save(ctx, key, data, mime); err != nil {
		return nil, apperror.Internal(MsgAttachmentSaveFailed)
	}
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		locked, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		if locked.Status != domain.QuotationAccepted {
			return apperror.Validation(MsgAttachmentNotAccepted, map[string][]string{"file": {MsgAttachmentNotAccepted}}).WithCode(CodeQuotationNotAccepted)
		}
		locked.AcceptedAttachmentKey, locked.AcceptedAttachmentName, locked.AcceptedAttachmentType = &key, &name, &mime
		if err := s.repo.SaveState(ctx, locked); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, id, v, domain.ActivityAttachmentAdded, "Lampiran penerimaan ditambahkan: "+name)
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, tenantID, id, v)
}

// AcceptanceAttachment opens the evidence of acceptance.
func (s *QuotationService) AcceptanceAttachment(ctx context.Context, tenantID, id int64, v Viewer) (io.ReadCloser, string, string, error) {
	q, err := s.get(ctx, tenantID, id, v)
	if err != nil {
		return nil, "", "", err
	}
	if q.AcceptedAttachmentKey == nil || s.files == nil {
		return nil, "", "", apperror.NotFound(MsgAttachmentNotFound)
	}
	r, err := s.files.Open(ctx, *q.AcceptedAttachmentKey)
	if err != nil {
		return nil, "", "", apperror.Internal(MsgAttachmentReadFailed)
	}
	name, mime := "lampiran", "application/octet-stream"
	if q.AcceptedAttachmentName != nil {
		name = *q.AcceptedAttachmentName
	}
	if q.AcceptedAttachmentType != nil {
		mime = *q.AcceptedAttachmentType
	}
	return r, name, mime, nil
}
