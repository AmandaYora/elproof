package presentation

import (
	"io"
	"net/http"
	"strconv"

	"elproof/internal/shared/httpx"
	"elproof/internal/shared/response"
	"elproof/internal/shared/storage"
)

// loginSlideKeys is the fixed, ordered set of object storage keys the login
// page's photo slider plays through — uploaded by
// `api upload-login-slides <dir>` (internal/loginslides), never written by
// any request handler.
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
