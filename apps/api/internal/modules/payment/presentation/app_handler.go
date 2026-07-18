package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/payment/infrastructure"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/response"
)

// AppToken exchanges an external App's {appId, secret} for a bearer access
// token — see knowledge/MODULE_PAYMENT.md §7.1. Deliberately unauthenticated
// (this IS the login step for Apps) but rate-limited by the caller
// (payment.module.go wraps this route with middleware.RateLimit).
func (h *Handler) AppToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AppID  string `json:"appId"`
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AppID == "" || body.Secret == "" {
		writeExternalError(w, http.StatusBadRequest, "bad_request", "Body permintaan tidak valid")
		return
	}

	token, expiresAt, err := h.service.IssueAppToken(r.Context(), body.AppID, body.Secret)
	if err != nil {
		writeExternalAppError(w, err)
		return
	}
	response.OK(w, "ok", map[string]interface{}{
		"accessToken": token,
		"tokenType":   "Bearer",
		"expiresIn":   int64(time.Until(expiresAt).Seconds()),
	})
}

type appResponse struct {
	AppID       string `json:"appId"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	CallbackURL string `json:"callbackUrl"`
	IsActive    bool   `json:"isActive"`
	CreatedAt   string `json:"createdAt"`
}

func toAppResponse(a infrastructure.AppView) appResponse {
	return appResponse{
		AppID: a.AppID, Name: a.Name, Kind: a.Kind,
		CallbackURL: a.CallbackURL, IsActive: a.IsActive, CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}
}

// Apps handles the App registry collection — GET lists every App (internal +
// external), POST registers a new external App. platform_admin only.
func (h *Handler) Apps(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		apps, err := h.service.ListApps(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]appResponse, 0, len(apps))
		for _, a := range apps {
			result = append(result, toAppResponse(a))
		}
		response.OK(w, "ok", result)
	case http.MethodPost:
		h.createApp(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

func (h *Handler) createApp(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		CallbackURL string `json:"callbackUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	appID, secret, err := h.service.CreateExternalApp(r.Context(), body.Name, body.CallbackURL)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Aplikasi berhasil didaftarkan", map[string]string{
		"appId": appID, "name": body.Name, "secret": secret,
	})
}

// AppItem handles the two admin actions on one App — reset-secret and
// toggle-active — mirroring the single-toggle convention used everywhere
// else in this codebase (tenants/staff/vendors' own toggle-active routes).
func (h *Handler) AppItem(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/payment/apps/")
	if len(segments) != 2 || r.Method != http.MethodPost {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	appID := segments[0]

	switch segments[1] {
	case "reset-secret":
		secret, err := h.service.ResetAppSecret(r.Context(), appID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Secret aplikasi berhasil direset", map[string]string{"appId": appID, "secret": secret})
	case "toggle-active":
		current, err := h.service.GetApp(r.Context(), appID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		if err := h.service.SetAppStatus(r.Context(), appID, !current.IsActive); err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Status aplikasi diperbarui", nil)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}
