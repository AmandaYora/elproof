package presentation

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/response"
)

type uploadRequestBody struct {
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

// uploadGrantResponse is POST /projects/{id}/uploads: send the file with `method`
// to `url` (adding `headers`), then name `uploadToken` in the payment request.
// `direct` says whether `url` is object storage itself (a signed URL) or this
// API's own upload route; the browser does the same thing either way.
type uploadGrantResponse struct {
	UploadToken string            `json:"uploadToken"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	ExpiresAt   string            `json:"expiresAt"`
	Direct      bool              `json:"direct"`
}

// createUpload is POST /projects/{id}/uploads (staff; same permission as the
// evidence upload). It validates the file's description and returns a signed,
// 15-minute, single-purpose upload; no byte has been stored yet.
func (h *Handler) createUpload(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body uploadRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	grant, err := h.uploads.Grant(r.Context(), claims.tenantID, projectID, application.UploadInput{
		FileName: body.FileName, MimeType: body.MimeType, Size: body.Size,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Unggahan disiapkan", uploadGrantResponse{
		UploadToken: grant.Token, Method: http.MethodPut, URL: grant.URL,
		Headers:   map[string]string{"Content-Type": grant.MimeType},
		ExpiresAt: grant.ExpiresAt.In(wib).Format(time.RFC3339), Direct: grant.Direct,
	})
}

// putUpload is PUT /projects/{id}/uploads/{uploadToken}: the raw bytes of a file,
// for deployments whose object storage the browser cannot reach. Content-Type
// must be the declared one and the body at most the declared size (never above
// 15 MB); the same staff authorization as the rest of the project applies. A
// second PUT for a token a payment already used is refused (409).
func (h *Handler) putUpload(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, token string) {
	if r.ContentLength > 15*1024*1024 {
		response.Error(w, http.StatusRequestEntityTooLarge, "Ukuran berkas terlalu besar", map[string][]string{"size": {"Maksimal 15 MB"}})
		return
	}
	size, err := h.uploads.Store(r.Context(), claims.tenantID, projectID, token, r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		var tooLarge *application.UploadTooLargeError
		if errors.As(err, &tooLarge) {
			response.Error(w, http.StatusRequestEntityTooLarge, "Ukuran berkas melebihi yang dinyatakan",
				map[string][]string{"size": {"Berkas lebih besar dari ukuran yang dinyatakan saat meminta unggahan"}})
			return
		}
		writeAppError(w, err)
		return
	}
	response.OK(w, "Berkas berhasil diunggah", map[string]int64{"size": size})
}
