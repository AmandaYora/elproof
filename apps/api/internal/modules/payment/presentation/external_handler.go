package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

type externalContextKey string

const externalAppIDContextKey externalContextKey = "elproof.payment.externalAppID"

// RequireActiveApp gates the three /external/payments/* routes: the caller
// must present a token minted for an `app` principal, AND that App must
// still be active *right now* — checked fresh against the registry on every
// request (never just trusting the JWT), so deactivating an App from
// Platform Console takes effect immediately, not at token expiry. See
// knowledge/MODULE_PAYMENT.md §7.1/§7.2. Exported: payment.module.go
// (composition root for this module's own routes) wires it around each
// external handler.
func (h *Handler) RequireActiveApp(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.FromContext(r.Context())
		if !ok || claims.PrincipalType != "app" {
			writeExternalError(w, http.StatusForbidden, "forbidden", "Endpoint ini hanya untuk aplikasi eksternal terdaftar")
			return
		}
		active, err := h.service.IsAppActive(r.Context(), claims.PrincipalID)
		if err != nil {
			writeExternalAppError(w, err)
			return
		}
		if !active {
			writeExternalError(w, http.StatusForbidden, "forbidden", "Aplikasi telah dinonaktifkan")
			return
		}
		ctx := context.WithValue(r.Context(), externalAppIDContextKey, claims.PrincipalID)
		next(w, r.WithContext(ctx))
	}
}

func externalAppIDFromContext(ctx context.Context) string {
	appID, _ := ctx.Value(externalAppIDContextKey).(string)
	return appID
}

type externalChargeResponse struct {
	OrderRef    string `json:"orderRef"`
	ProviderRef string `json:"providerRef"`
	Channel     string `json:"channel"`
	QRImageURL  string `json:"qrImageUrl,omitempty"`
	PayCode     string `json:"payCode,omitempty"`
	CheckoutURL string `json:"checkoutUrl,omitempty"`
	Amount      int64  `json:"amount"`
	FeeAmount   int64  `json:"feeAmount"`
	ExpiresAt   string `json:"expiresAt"`
	Status      string `json:"status"`
}

func toExternalChargeResponse(c *contracts.ChargeResult) externalChargeResponse {
	return externalChargeResponse{
		OrderRef: c.OrderRef, ProviderRef: c.ProviderRef, Channel: c.Channel,
		QRImageURL: c.QRImageURL, PayCode: c.PayCode, CheckoutURL: c.CheckoutURL,
		Amount: c.Amount, FeeAmount: c.FeeAmount, ExpiresAt: c.ExpiresAt.Format(time.RFC3339), Status: c.Status,
	}
}

const defaultExternalChannel = "QRIS"

// ExternalCreateCharge is `POST /api/v1/external/payments/charges` — appId
// always comes from the token (via requireActiveApp), never the body, so an
// App can never create a charge on another App's behalf.
func (h *Handler) ExternalCreateCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeExternalError(w, http.StatusMethodNotAllowed, "bad_request", "Metode HTTP tidak diizinkan untuk endpoint ini")
		return
	}
	appID := externalAppIDFromContext(r.Context())

	var body struct {
		OrderRef      string `json:"orderRef"`
		Amount        int64  `json:"amount"`
		Channel       string `json:"channel"`
		CustomerName  string `json:"customerName"`
		CustomerEmail string `json:"customerEmail"`
		CustomerPhone string `json:"customerPhone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeExternalError(w, http.StatusBadRequest, "bad_request", "Body permintaan tidak valid")
		return
	}
	if body.OrderRef == "" || body.Amount <= 0 {
		writeExternalError(w, http.StatusBadRequest, "bad_request", "orderRef dan amount (>0) wajib diisi")
		return
	}
	channel := body.Channel
	if channel == "" {
		channel = defaultExternalChannel
	}

	result, err := h.service.CreateChannelCharge(r.Context(), appID, "", body.OrderRef, body.Amount, channel, contracts.ChargeOptions{
		CustomerName: body.CustomerName, CustomerEmail: body.CustomerEmail, CustomerPhone: body.CustomerPhone,
	})
	if err != nil {
		writeExternalAppError(w, err)
		return
	}
	response.Created(w, "ok", toExternalChargeResponse(result))
}

// ExternalChargeStatus is `GET /api/v1/external/payments/charges/{orderRef}/status`.
func (h *Handler) ExternalChargeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeExternalError(w, http.StatusMethodNotAllowed, "bad_request", "Metode HTTP tidak diizinkan untuk endpoint ini")
		return
	}
	appID := externalAppIDFromContext(r.Context())

	segments := httpx.Segments(r.URL.Path, "/api/v1/external/payments/charges/")
	if len(segments) != 2 || segments[1] != "status" {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	orderRef := segments[0]

	result, err := h.service.CheckStatusForApp(r.Context(), appID, orderRef)
	if err != nil {
		writeExternalAppError(w, err)
		return
	}
	response.OK(w, "ok", toExternalChargeResponse(result))
}

// ExternalListChannels is `GET /api/v1/external/payments/channels`.
func (h *Handler) ExternalListChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeExternalError(w, http.StatusMethodNotAllowed, "bad_request", "Metode HTTP tidak diizinkan untuk endpoint ini")
		return
	}
	channels, err := h.service.ListChannels(r.Context())
	if err != nil {
		writeExternalAppError(w, err)
		return
	}
	response.OK(w, "ok", channels)
}

// externalErrorCode maps an AppError.Kind to the fixed vocabulary external
// consumers can branch on programmatically — see knowledge/MODULE_PAYMENT.md
// §7.6. Internal routes keep using writeAppError (message-only, no code);
// this is additive, scoped to the three /external/payments/* + /auth/app/token
// routes only.
func externalErrorCode(kind apperror.Kind) string {
	switch kind {
	case apperror.KindValidation:
		return "bad_request"
	case apperror.KindUnauthorized:
		return "unauthorized"
	case apperror.KindForbidden:
		return "forbidden"
	case apperror.KindNotFound:
		return "not_found"
	case apperror.KindConflict:
		return "conflict"
	case apperror.KindRateLimited:
		return "rate_limited"
	default:
		return "internal"
	}
}

func writeExternalAppError(w http.ResponseWriter, err error) {
	status := apperror.HTTPStatus(err)
	if appErr, ok := apperror.As(err); ok {
		response.Error(w, status, appErr.Message, map[string]string{"code": externalErrorCode(appErr.Kind)})
		return
	}
	response.Error(w, status, "Terjadi kesalahan pada server", map[string]string{"code": "internal"})
}

func writeExternalError(w http.ResponseWriter, status int, code, message string) {
	response.Error(w, status, message, map[string]string{"code": code})
}
