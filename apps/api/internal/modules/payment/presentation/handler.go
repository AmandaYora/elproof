package presentation

import (
	"encoding/json"
	"io"
	"net/http"

	"elproof/internal/modules/payment/infrastructure"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

type Handler struct {
	service *infrastructure.PaymentService
}

func NewHandler(service *infrastructure.PaymentService) *Handler {
	return &Handler{service: service}
}

// Webhook is the single, permanent callback route registered with the
// gateway — see MODULE_PAYMENT.md §6 step 4. Deliberately unauthenticated
// (no JWT): the gateway itself can't carry a bearer token, so trust comes
// entirely from the signature check inside HandleWebhook.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Body tidak terbaca", nil)
		return
	}
	signature := r.Header.Get("X-Callback-Signature")

	if err := h.service.HandleWebhook(r.Context(), signature, body); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", nil)
}

type gatewayConfigResponse struct {
	ActiveProvider      string `json:"activeProvider"`
	IsSandbox           bool   `json:"isSandbox"`
	TripayMerchantCode  string `json:"tripayMerchantCode"`
	HasTripayAPIKey     bool   `json:"hasTripayApiKey"`
	HasTripayPrivateKey bool   `json:"hasTripayPrivateKey"`
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	cfg, err := h.service.GetConfig(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", gatewayConfigResponse{
		ActiveProvider: cfg.ActiveProvider, IsSandbox: cfg.IsSandbox, TripayMerchantCode: cfg.TripayMerchantCode,
		HasTripayAPIKey: cfg.HasTripayAPIKey, HasTripayPrivateKey: cfg.HasTripayPrivateKey,
	})
}

type updateConfigBody struct {
	ActiveProvider     string `json:"activeProvider"`
	IsSandbox          bool   `json:"isSandbox"`
	TripayMerchantCode string `json:"tripayMerchantCode"`
	TripayAPIKey       string `json:"tripayApiKey"`
	TripayPrivateKey   string `json:"tripayPrivateKey"`
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	var body updateConfigBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	err := h.service.UpdateConfig(r.Context(), infrastructure.UpdateConfigInput{
		ActiveProvider: body.ActiveProvider, IsSandbox: body.IsSandbox, TripayMerchantCode: body.TripayMerchantCode,
		TripayAPIKey: body.TripayAPIKey, TripayPrivateKey: body.TripayPrivateKey,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Konfigurasi gateway berhasil disimpan", nil)
}

func (h *Handler) Config(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetConfig(w, r)
	case http.MethodPatch:
		h.UpdateConfig(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

func requirePlatformAdmin(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "platform_admin" {
		response.Error(w, http.StatusForbidden, "Hanya admin platform yang dapat melakukan aksi ini", nil)
		return false
	}
	return true
}

func writeAppError(w http.ResponseWriter, err error) {
	status := apperror.HTTPStatus(err)
	if appErr, ok := apperror.As(err); ok {
		if appErr.Kind == apperror.KindValidation {
			response.Error(w, status, appErr.Message, appErr.Fields)
			return
		}
		response.Error(w, status, appErr.Message, nil)
		return
	}
	response.Error(w, status, "Terjadi kesalahan pada server", nil)
}
