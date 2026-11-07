package presentation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"

	"elproof/internal/shared/compress"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
	"elproof/internal/shared/storage"
)

// loginSlideKeys is the fixed, ordered set of object storage keys the login
// page's photo slider plays through. First uploaded by
// `api upload-login-slides <dir>` (internal/loginslides); a Super Admin can
// replace a photo later from the Platform Console (Replace below).
var loginSlideKeys = []string{
	"elproof/marketing/login-slides/slide-01.jpg",
	"elproof/marketing/login-slides/slide-02.jpg",
	"elproof/marketing/login-slides/slide-03.jpg",
}

// LoginSlidesHandler serves the login page's fixed marketing photos —
// global, not tenant-scoped, so it depends only on the raw storage client
// (no service/repository layer, unlike TenantHandler's logo endpoints).
type LoginSlidesHandler struct {
	storage *storage.Client
}

func NewLoginSlidesHandler(storageClient *storage.Client) *LoginSlidesHandler {
	return &LoginSlidesHandler{storage: storageClient}
}

// List reports how many slides exist so the frontend knows which indices to
// request — no per-slide metadata, since order and keys are fixed here.
func (h *LoginSlidesHandler) List(w http.ResponseWriter, r *http.Request) {
	response.OK(w, "ok", map[string]int{"count": len(loginSlideKeys)})
}

// Slide streams one slide's JPEG bytes by 0-based index:
// GET /api/v1/public/login-slides/{index}. Unauthenticated by design, same as
// PublicLogo — this is public marketing content, not tenant data.
func (h *LoginSlidesHandler) Slide(w http.ResponseWriter, r *http.Request) {
	segments := httpx.Segments(r.URL.Path, "/api/v1/public/login-slides/")
	if len(segments) != 1 {
		response.Error(w, http.StatusNotFound, "Slide tidak ditemukan", nil)
		return
	}
	index, err := strconv.Atoi(segments[0])
	if err != nil || index < 0 || index >= len(loginSlideKeys) {
		response.Error(w, http.StatusNotFound, "Slide tidak ditemukan", nil)
		return
	}

	reader, err := h.storage.Open(r.Context(), loginSlideKeys[index])
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengambil slide dari object storage", nil)
		return
	}
	defer reader.Close()

	// Static, rarely-changing images (unlike a tenant logo, which streamLogo
	// serves with no cache headers) -- caching meaningfully cuts repeat-visit
	// bandwidth for a page every user hits before authenticating.
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

const (
	// maxSlideUploadBytes bounds the decoded photo. The browser already
	// downsizes before upload; this only stops an oversized request.
	maxSlideUploadBytes = 6 << 20
	maxSlidePixels      = 25_000_000
	slideJPEGQuality    = 92
)

type slideUploadBody struct {
	MimeType   string `json:"mimeType"`
	Base64Data string `json:"base64Data"`
}

// Replace is PUT /api/v1/console/login-slides/{index}: swaps one of the fixed
// login photos for a new JPEG or PNG. Super Admin only (the role policy in
// cmd/server denies every unlisted write to anyone else). The photo is
// flattened to JPEG and compressed like every other upload.
func (h *LoginSlidesHandler) Replace(w http.ResponseWriter, r *http.Request) {
	// The role policy in cmd/server only judges platform admins; a workspace's
	// own staff token reaches this handler too, so it is refused here.
	if claims, ok := middleware.FromContext(r.Context()); !ok || claims.PrincipalType != "platform_admin" {
		response.Error(w, http.StatusForbidden, "Hanya admin platform yang dapat mengganti foto login", nil)
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/console/login-slides/")
	if len(segments) != 1 {
		response.Error(w, http.StatusNotFound, "Slide tidak ditemukan", nil)
		return
	}
	index, err := strconv.Atoi(segments[0])
	if err != nil || index < 0 || index >= len(loginSlideKeys) {
		response.Error(w, http.StatusNotFound, "Slide tidak ditemukan", nil)
		return
	}

	// Base64 inflates by a third; leave room for the JSON wrapper.
	r.Body = http.MaxBytesReader(w, r.Body, maxSlideUploadBytes*4/3+4096)
	var body slideUploadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Foto terlalu besar atau permintaan tidak valid. Gunakan foto di bawah 6 MB.", nil)
		return
	}
	if body.MimeType != "image/jpeg" && body.MimeType != "image/png" {
		response.Error(w, http.StatusUnprocessableEntity, "Foto tidak valid", map[string][]string{"file": {"Gunakan foto JPG atau PNG"}})
		return
	}
	raw, err := base64.StdEncoding.DecodeString(body.Base64Data)
	if err != nil || len(raw) == 0 || len(raw) > maxSlideUploadBytes {
		response.Error(w, http.StatusUnprocessableEntity, "Foto tidak valid", map[string][]string{"file": {"Foto kosong atau lebih dari 6 MB"}})
		return
	}
	// Read the dimensions first: a small file can decode to a huge bitmap.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Foto tidak valid", map[string][]string{"file": {"Berkas ini bukan foto JPG atau PNG yang bisa dibaca"}})
		return
	}
	if cfg.Width*cfg.Height > maxSlidePixels {
		response.Error(w, http.StatusUnprocessableEntity, "Foto tidak valid", map[string][]string{"file": {"Resolusi foto terlalu besar. Perkecil hingga di bawah 25 megapiksel."}})
		return
	}
	// Decode what was really sent, whatever the declared type says.
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Foto tidak valid", map[string][]string{"file": {"Berkas ini bukan foto JPG atau PNG yang bisa dibaca"}})
		return
	}
	var flat bytes.Buffer
	if err := jpeg.Encode(&flat, img, &jpeg.Options{Quality: slideJPEGQuality}); err != nil {
		response.Error(w, http.StatusInternalServerError, "Foto gagal diproses", nil)
		return
	}
	compressed, err := compress.Image(flat.Bytes(), "image/jpeg")
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Foto gagal dikompres", nil)
		return
	}
	if _, err := h.storage.Save(r.Context(), loginSlideKeys[index], compressed, "image/jpeg"); err != nil {
		response.Error(w, http.StatusInternalServerError, "Foto gagal disimpan ke penyimpanan", nil)
		return
	}
	response.OK(w, "Foto login diganti", map[string]int{"index": index})
}
