package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type TenantHandler struct {
	tenants *application.TenantService
}

func NewTenantHandler(tenants *application.TenantService) *TenantHandler {
	return &TenantHandler{tenants: tenants}
}

type tenantResponse struct {
	ID                    int64   `json:"id"`
	BusinessName          string  `json:"businessName"`
	OwnerName             string  `json:"ownerName"`
	Username              string  `json:"username"`
	Email                 string  `json:"email"`
	Phone                 string  `json:"phone"`
	City                  string  `json:"city"`
	JoinedAt              string  `json:"joinedAt"`
	PlanID                *int64  `json:"planId"`
	SubscriptionStatus    string  `json:"subscriptionStatus"`
	SubscriptionExpiresAt *string `json:"subscriptionExpiresAt"`
	IsSuspended           bool    `json:"isSuspended"`
	LastCredentialResetAt *string `json:"lastCredentialResetAt"`
}

func dateOrNil(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func toTenantResponse(t domain.Tenant) tenantResponse {
	return tenantResponse{
		ID: t.ID, BusinessName: t.BusinessName, OwnerName: t.OwnerName, Username: t.Username,
		Email: t.Email, Phone: t.Phone, City: t.City, JoinedAt: t.JoinedAt.Format("2006-01-02"),
		PlanID: t.PlanID, SubscriptionStatus: string(t.SubscriptionStatus),
		SubscriptionExpiresAt: dateOrNil(t.SubscriptionExpiresAt), IsSuspended: t.IsSuspended,
		LastCredentialResetAt: dateOrNil(t.LastCredentialResetAt),
	}
}

type chargeResponse struct {
	OrderRef    string `json:"orderRef"`
	ProviderRef string `json:"providerRef"`
	Channel     string `json:"channel"`
	QRImageURL  string `json:"qrImageUrl"`
	PayCode     string `json:"payCode"`
	CheckoutURL string `json:"checkoutUrl"`
	Amount      int64  `json:"amount"`
	FeeAmount   int64  `json:"feeAmount"`
	ExpiresAt   string `json:"expiresAt"`
	Status      string `json:"status"`
}

func toChargeResponse(c paymentcontracts.ChargeResult) chargeResponse {
	return chargeResponse{
		OrderRef: c.OrderRef, ProviderRef: c.ProviderRef, Channel: c.Channel,
		QRImageURL: c.QRImageURL, PayCode: c.PayCode, CheckoutURL: c.CheckoutURL,
		Amount: c.Amount, FeeAmount: c.FeeAmount, ExpiresAt: c.ExpiresAt.Format(time.RFC3339), Status: c.Status,
	}
}

func (h *TenantHandler) Collection(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.register(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

func (h *TenantHandler) list(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("all") == "true" {
		tenants, err := h.tenants.List(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]tenantResponse, 0, len(tenants))
		for _, t := range tenants {
			result = append(result, toTenantResponse(t))
		}
		response.OK(w, "ok", result)
		return
	}
	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	tenants, total, err := h.tenants.ListPaginated(r.Context(), params, search, status)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]tenantResponse, 0, len(tenants))
	for _, t := range tenants {
		result = append(result, toTenantResponse(t))
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
}

type registerTenantBody struct {
	BusinessName string `json:"businessName"`
	OwnerName    string `json:"ownerName"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	City         string `json:"city"`
	PlanID       int64  `json:"planId"`
	Password     string `json:"password"`
}

func (h *TenantHandler) register(w http.ResponseWriter, r *http.Request) {
	var body registerTenantBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	result, err := h.tenants.Register(r.Context(), application.RegisterTenantInput{
		BusinessName: body.BusinessName, OwnerName: body.OwnerName, Username: body.Username, Email: body.Email,
		Phone: body.Phone, City: body.City, PlanID: body.PlanID, Password: body.Password,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Tenant berhasil didaftarkan", map[string]interface{}{
		"tenant":   toTenantResponse(result.Tenant),
		"username": result.Username,
	})
}

func (h *TenantHandler) Item(w http.ResponseWriter, r *http.Request) {
	segments := httpx.Segments(r.URL.Path, "/api/v1/tenants/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Tenant tidak ditemukan", nil)
		return
	}

	// "me" is the tenant Owner's own self-service read — everything else under
	// /tenants/ is platform_admin only.
	if segments[0] == "me" && len(segments) == 1 && r.Method == http.MethodGet {
		h.me(w, r)
		return
	}
	if !requirePlatformAdmin(w, r) {
		return
	}

	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		h.get(w, r, id)
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.update(w, r, id)
	case len(segments) == 2 && segments[1] == "toggle-suspension" && r.Method == http.MethodPost:
		h.toggleSuspension(w, r, id)
	case len(segments) == 2 && segments[1] == "reset-credential" && r.Method == http.MethodPost:
		h.resetCredential(w, r, id)
	case len(segments) == 2 && segments[1] == "activate-subscription" && r.Method == http.MethodPost:
		h.activateSubscription(w, r, id)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// me is the tenant Owner's self-service read of their own tenant record
// (plan, subscription status/expiry) — scoped from the JWT claim, never a
// request parameter. Powers the WO Console's own "Langganan" page.
func (h *TenantHandler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || claims.Role != "Owner" {
		response.Error(w, http.StatusForbidden, "Hanya akun Owner yang dapat mengakses data ini", nil)
		return
	}
	tenantID, err := application.ParseTenantID(claims.TenantID)
	if err != nil {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}
	tenant, err := h.tenants.Get(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toTenantResponse(*tenant))
}

func (h *TenantHandler) get(w http.ResponseWriter, r *http.Request, id int64) {
	tenant, err := h.tenants.Get(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toTenantResponse(*tenant))
}

type updateTenantBody struct {
	BusinessName string `json:"businessName"`
	OwnerName    string `json:"ownerName"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	City         string `json:"city"`
}

func (h *TenantHandler) update(w http.ResponseWriter, r *http.Request, id int64) {
	var body updateTenantBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	tenant, err := h.tenants.Update(r.Context(), id, application.UpdateTenantInput{
		BusinessName: body.BusinessName, OwnerName: body.OwnerName, Email: body.Email, Phone: body.Phone, City: body.City,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tenant berhasil diperbarui", toTenantResponse(*tenant))
}

func (h *TenantHandler) toggleSuspension(w http.ResponseWriter, r *http.Request, id int64) {
	current, err := h.tenants.Get(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	tenant, err := h.tenants.SetSuspended(r.Context(), id, !current.IsSuspended)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status tenant diperbarui", toTenantResponse(*tenant))
}

type resetCredentialBody struct {
	Password string `json:"password"`
}

func (h *TenantHandler) resetCredential(w http.ResponseWriter, r *http.Request, id int64) {
	var body resetCredentialBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	result, err := h.tenants.ResetCredential(r.Context(), id, body.Password)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kredensial tenant berhasil direset", map[string]string{"username": result.Username, "password": body.Password})
}

type activateSubscriptionBody struct {
	PlanID int64 `json:"planId"`
}

func (h *TenantHandler) activateSubscription(w http.ResponseWriter, r *http.Request, id int64) {
	var body activateSubscriptionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	tenant, err := h.tenants.ActivateSubscription(r.Context(), id, body.PlanID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Langganan tenant berhasil diaktifkan", toTenantResponse(*tenant))
}

// Pay is the tenant Owner's own self-service subscription payment — scoped to
// their own tenant via the JWT claim, never a request parameter.
func (h *TenantHandler) Pay(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || claims.Role != "Owner" {
		response.Error(w, http.StatusForbidden, "Hanya akun Owner yang dapat mengelola langganan", nil)
		return
	}
	tenantID, err := application.ParseTenantID(claims.TenantID)
	if err != nil {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}

	var body activateSubscriptionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	charge, err := h.tenants.Pay(r.Context(), tenantID, body.PlanID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tagihan pembayaran dibuat, selesaikan lewat QRIS di bawah", toChargeResponse(*charge))
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
