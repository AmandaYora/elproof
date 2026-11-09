package application

import (
	"context"
	"io"
	"mime"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/logger"
)

// maxProofsPerPayment is how many files one payment can be recorded with.
const maxProofsPerPayment = 4

// uploadCategory is the key segment signed uploads are stored under. The file
// is named before the thing it will prove exists, so it cannot sit under the
// related kind as the base64 route's files do; any file here that no evidence
// row ever claims is an orphan (see ADR-0033 for the cleanup stance).
const uploadCategory = "uploads"

// allowedUploadMimeTypes is what a signed upload may be: what the web's file
// pickers offer (JPEG, PNG, PDF) plus the other common phone-photo formats.
// The base64 evidence route enforces no list; a signed upload hands out a URL
// that writes to the bucket, so what it may write is narrow.
var allowedUploadMimeTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true, "image/heic": true, "image/heif": true, "application/pdf": true,
}

// ObjectStat is the optional storage capability signed uploads need: look at a
// stored object without reading it.
type ObjectStat interface {
	// Stat reports size and content type; found is false when there is no such object.
	Stat(ctx context.Context, key string) (size int64, contentType string, found bool, err error)
}

// UploadPresigner is the optional storage capability that makes uploads direct.
type UploadPresigner interface {
	CanPresign() bool
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
}

// UploadGrant is the answer to "I want to upload a file": where to PUT it and
// the token that later names it in a payment request.
type UploadGrant struct {
	Token     string
	URL       string
	MimeType  string
	ExpiresAt time.Time
	// Direct is true when URL is a presigned object-storage URL; false when it
	// is this API's own upload route.
	Direct bool
}

// UploadInput is what the browser says about the file it is about to upload.
type UploadInput struct {
	FileName string
	MimeType string
	Size     int64
}

// UploadService hands out signed uploads, accepts the bytes when storage cannot
// be reached by the browser, and checks an upload before a payment claims it.
type UploadService struct {
	signer   *UploadSigner
	storage  ObjectStorage
	files    EvidenceRepository
	buildKey func(tenantID, projectID, category, filename string) string
	now      func() time.Time
}

func NewUploadService(signer *UploadSigner, storage ObjectStorage, files EvidenceRepository, buildKey func(string, string, string, string) string) *UploadService {
	return &UploadService{signer: signer, storage: storage, files: files, buildKey: buildKey, now: time.Now}
}

// safeFileName: a name a person can read, with nothing that could pass for a
// path or a control character.
func safeFileName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || utf8.RuneCountInString(name) > 150 {
		return false
	}
	for _, r := range name {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func normalizeMime(m string) string {
	base, _, err := mime.ParseMediaType(strings.TrimSpace(m))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(m))
	}
	return base
}

// Grant validates the description of a file and returns a signed upload for it.
func (s *UploadService) Grant(ctx context.Context, tenantID, projectID int64, in UploadInput) (*UploadGrant, error) {
	fe := newFieldErrors()
	if !safeFileName(in.FileName) {
		fe.add("Nama berkas tidak valid. Pakai nama tanpa garis miring dan maksimal 150 karakter.", "fileName")
	}
	mimeType := normalizeMime(in.MimeType)
	if !allowedUploadMimeTypes[mimeType] {
		fe.add("Jenis berkas tidak didukung. Pakai foto (JPEG, PNG, WebP, HEIC) atau PDF.", "mimeType")
	}
	if in.Size <= 0 || in.Size > maxDecodedSize {
		fe.add("Ukuran berkas harus lebih dari 0 dan maksimal 15 MB.", "size")
	}
	if !fe.empty() {
		return nil, fe.err()
	}

	expires := s.now().Add(UploadTokenTTL)
	key := s.buildKey(strconv.FormatInt(tenantID, 10), strconv.FormatInt(projectID, 10), uploadCategory,
		uuid.NewString()+"-"+sanitizeFileName(strings.TrimSpace(in.FileName)))
	token := s.signer.Sign(UploadClaims{
		TenantID: tenantID, ProjectID: projectID, Key: key, MimeType: mimeType, FileName: strings.TrimSpace(in.FileName),
		MaxSize: in.Size, ExpiresAt: expires, Nonce: NewNonce(),
	})
	grant := &UploadGrant{Token: token, MimeType: mimeType, ExpiresAt: expires}
	if p, ok := s.storage.(UploadPresigner); ok && p.CanPresign() {
		url, err := p.PresignPut(ctx, key, mimeType, UploadTokenTTL)
		if err != nil {
			logger.Error("failed to presign upload %q: %v", key, err)
			return nil, apperror.Internal("Gagal menyiapkan unggahan")
		}
		grant.URL, grant.Direct = url, true
		return grant, nil
	}
	grant.URL = "/api/v1/projects/" + strconv.FormatInt(projectID, 10) + "/uploads/" + token
	return grant, nil
}

// tokenMessage is the 422 message for a token that failed verification.
func tokenMessage(err error) string {
	switch err {
	case ErrTokenExpired:
		return "Unggahan ini sudah kedaluwarsa. Unggah ulang berkasnya."
	case ErrTokenScope:
		return "Berkas ini diunggah untuk project lain. Unggah ulang berkasnya di sini."
	default:
		return "Berkas ini tidak valid. Unggah ulang berkasnya."
	}
}

// Store accepts the bytes of a signed upload on this API's own route (storage
// the browser cannot reach). The same token and the same rules as the direct
// path: the declared type, the size cap, one file per token. The bytes pass
// through the same image re-compression as every other upload (ADR-0010).
func (s *UploadService) Store(ctx context.Context, tenantID, projectID int64, token, contentType string, body io.Reader) (int64, error) {
	claims, err := s.signer.Verify(token, tenantID, projectID)
	if err != nil {
		return 0, apperror.Validation(tokenMessage(err), map[string][]string{"uploadToken": {tokenMessage(err)}})
	}
	if normalizeMime(contentType) != claims.MimeType {
		msg := "Jenis berkas harus " + claims.MimeType + " sesuai yang dinyatakan saat meminta unggahan."
		return 0, apperror.Validation(msg, map[string][]string{"contentType": {msg}})
	}
	// Once a payment has claimed the object, the token is spent: writing again
	// would swap the proof of a recorded payment.
	used, err := s.files.ExistsByStoragePath(ctx, claims.Key)
	if err != nil {
		return 0, err
	}
	if used {
		msg := "Unggahan ini sudah dipakai oleh pembayaran lain."
		return 0, apperror.Conflict(msg).WithCode("upload_token_used")
	}
	data, err := io.ReadAll(io.LimitReader(body, claims.MaxSize+1))
	if err != nil {
		return 0, apperror.Validation("Gagal membaca berkas", map[string][]string{"body": {"Gagal membaca berkas"}})
	}
	if len(data) == 0 {
		msg := "Berkas tidak boleh kosong."
		return 0, apperror.Validation(msg, map[string][]string{"size": {msg}})
	}
	if int64(len(data)) > claims.MaxSize {
		return 0, errUploadTooLarge(claims.MaxSize)
	}
	compressed, err := compress.Image(data, claims.MimeType)
	if err != nil {
		return 0, apperror.Internal("Gagal memproses file")
	}
	if _, err := s.storage.Save(ctx, claims.Key, compressed, claims.MimeType); err != nil {
		logger.Error("failed to store upload %q: %v", claims.Key, err)
		return 0, apperror.Internal("Gagal mengunggah file ke object storage")
	}
	return int64(len(compressed)), nil
}

// UploadTooLargeError marks a body past the declared cap (the route answers 413).
type UploadTooLargeError struct{ Max int64 }

func (e *UploadTooLargeError) Error() string { return "upload exceeds the declared size" }

func errUploadTooLarge(max int64) error { return &UploadTooLargeError{Max: max} }

// ProofItem is one file a payment request claims: a token from Grant, what kind
// of document it is, and an optional display name.
type ProofItem struct {
	UploadToken string
	Type        domain.EvidenceType
	Name        string
}

// ResolvedProof is a ProofItem whose upload was verified.
type ResolvedProof struct {
	Claims UploadClaims
	Type   domain.EvidenceType
	Name   string
}

func proofKeys(i int) (item, token string) {
	item = "evidence[" + strconv.Itoa(i) + "]"
	return item, item + ".uploadToken"
}

// Resolve verifies every claimed upload before anything is written: the token
// is ours, unexpired, for this workspace and project, not used twice in the
// request or by an earlier payment; the document type is one the payment accepts;
// the object really is in storage, within its cap and of the declared type. An
// object that is there but wrong (empty, too big, not the declared type) is
// deleted, since it can never be used. Every problem of every item is collected
// and returned as one validation error with keys "evidence[i]" (the file),
// "evidence[i].uploadToken" (same message) and "evidence[i].type"/".name".
func (s *UploadService) Resolve(ctx context.Context, tenantID, projectID int64, items []ProofItem, allowed []domain.EvidenceType) ([]ResolvedProof, error) {
	if len(items) == 0 {
		return nil, nil
	}
	fe := newFieldErrors()
	if len(items) > maxProofsPerPayment {
		fe.add("Maksimal "+strconv.Itoa(maxProofsPerPayment)+" berkas bukti per pembayaran.", "evidence")
		return nil, fe.err()
	}
	statter, canStat := s.storage.(ObjectStat)
	seen := map[string]bool{}
	out := make([]ResolvedProof, 0, len(items))
	for i, it := range items {
		itemKey, tokenKey := proofKeys(i)
		if strings.TrimSpace(it.UploadToken) == "" {
			fe.add("Pilih berkas bukti yang akan dilampirkan.", itemKey, tokenKey)
			continue
		}
		typeOK := false
		for _, a := range allowed {
			if a == it.Type {
				typeOK = true
			}
		}
		if !typeOK {
			fe.add("Jenis bukti ini tidak dapat dilampirkan pada pembayaran ini. Pilih "+joinTypes(allowed)+".", "evidence["+strconv.Itoa(i)+"].type")
		}
		if utf8.RuneCountInString(strings.TrimSpace(it.Name)) > 200 {
			fe.add("Nama bukti maksimal 200 karakter.", "evidence["+strconv.Itoa(i)+"].name")
		}
		claims, err := s.signer.Verify(it.UploadToken, tenantID, projectID)
		if err != nil {
			fe.add(tokenMessage(err), itemKey, tokenKey)
			continue
		}
		prefix := s.buildKey(strconv.FormatInt(tenantID, 10), strconv.FormatInt(projectID, 10), uploadCategory, "")
		if !strings.HasPrefix(claims.Key, prefix) {
			fe.add(tokenMessage(ErrTokenInvalid), itemKey, tokenKey)
			continue
		}
		if seen[claims.Key] {
			fe.add("Berkas yang sama dilampirkan dua kali.", itemKey, tokenKey)
			continue
		}
		seen[claims.Key] = true
		if !canStat {
			return nil, apperror.Internal("Penyimpanan tidak mendukung pemeriksaan unggahan")
		}
		if used, err := s.files.ExistsByStoragePath(ctx, claims.Key); err != nil {
			return nil, err
		} else if used {
			fe.add("Unggahan ini sudah dipakai oleh pembayaran lain. Unggah ulang berkasnya.", itemKey, tokenKey)
			continue
		}
		size, contentType, found, err := statter.Stat(ctx, claims.Key)
		if err != nil {
			logger.Error("failed to stat upload %q: %v", claims.Key, err)
			return nil, apperror.Internal("Gagal memeriksa berkas yang diunggah")
		}
		switch {
		case !found:
			fe.add("Berkas belum selesai diunggah. Unggah ulang berkasnya.", itemKey, tokenKey)
			continue
		case size <= 0 || size > claims.MaxSize || size > maxDecodedSize:
			s.discard(ctx, claims.Key)
			fe.add("Ukuran berkas tidak sesuai (maksimal 15 MB dan sesuai yang dinyatakan). Unggah ulang berkasnya.", itemKey, tokenKey)
			continue
		case normalizeMime(contentType) != claims.MimeType:
			s.discard(ctx, claims.Key)
			fe.add("Jenis berkas yang terunggah tidak sama dengan yang dinyatakan. Unggah ulang berkasnya.", itemKey, tokenKey)
			continue
		}
		out = append(out, ResolvedProof{Claims: claims, Type: it.Type, Name: strings.TrimSpace(it.Name)})
	}
	if !fe.empty() {
		return nil, fe.err()
	}
	return out, nil
}

func joinTypes(types []domain.EvidenceType) string {
	parts := make([]string, len(types))
	for i, t := range types {
		parts[i] = string(t)
	}
	return strings.Join(parts, " atau ")
}

// discard deletes an object that can never be used (best effort).
func (s *UploadService) discard(ctx context.Context, key string) {
	if err := s.storage.Delete(ctx, key); err != nil {
		logger.Error("failed to delete rejected upload %q: %v", key, err)
	}
}
