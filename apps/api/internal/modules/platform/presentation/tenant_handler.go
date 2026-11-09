package presentation

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type TenantHandler struct {
	tenants *application.TenantService
	// auditor records actions that are not state-changing requests (the CSV export).
	auditor middleware.AuditRecorder
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
	BrandColorPreset      string  `json:"brandColorPreset"`
	HasLogo               bool    `json:"hasLogo"`
	TrialEndsAt           *string `json:"trialEndsAt"`
	// SignupSource: "self_service" (Fase 2 signup) or "platform".
	SignupSource string `json:"signupSource"`
	// Lifecycle is computed at read time (domain.ComputeLifecycle);
	// SubscriptionStatus above mirrors Lifecycle.Status.
	Lifecycle lifecycleResponse `json:"lifecycle"`
}

// lifecycleResponse is the subscription state any staff member of the
// tenant may see (GET /tenants/me/subscription) — it powers the WO
// Console's global banner for every role, not only the Owner.
type lifecycleResponse struct {
	Status       string  `json:"status"`
	Access       string  `json:"access"`
	IsTrial      bool    `json:"isTrial"`
	PeriodEndsAt *string `json:"periodEndsAt"`
	DaysLeft     int     `json:"daysLeft"`
	ReadOnlyAt   *string `json:"readOnlyAt"`
	SuspendAt    *string `json:"suspendAt"`
	Frozen       bool    `json:"frozen"`
}

func toLifecycleResponse(lc domain.Lifecycle) lifecycleResponse {
	return lifecycleResponse{
		Status: string(lc.Status), Access: string(lc.Access), IsTrial: lc.IsTrial,
		PeriodEndsAt: dateOrNil(lc.PeriodEndsAt), DaysLeft: lc.DaysLeft,
		ReadOnlyAt: dateOrNil(lc.ReadOnlyAt), SuspendAt: dateOrNil(lc.SuspendAt), Frozen: lc.Frozen,
	}
}

// brandingResponse is the minimal, non-sensitive branding shape any
// tenant-scoped principal (staff of any role, or client) can read — unlike
// tenantResponse/me, which stays Owner-only since it also carries
// subscription/billing data.
type brandingResponse struct {
	BusinessName     string `json:"businessName"`
	BrandColorPreset string `json:"brandColorPreset"`
	HasLogo          bool   `json:"hasLogo"`
}

func dateOrNil(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func toTenantResponse(t domain.Tenant) tenantResponse {
	lc := domain.ComputeLifecycle(t, time.Now(), domain.CurrentSubscriptionPolicy())
	return tenantResponse{
		ID: t.ID, BusinessName: t.BusinessName, OwnerName: t.OwnerName, Username: t.Username,
		Email: t.Email, Phone: t.Phone, City: t.City, JoinedAt: t.JoinedAt.Format("2006-01-02"),
		PlanID: t.PlanID, SubscriptionStatus: string(lc.Status),
		TrialEndsAt: dateOrNil(t.TrialEndsAt), Lifecycle: toLifecycleResponse(lc),
		SubscriptionExpiresAt: dateOrNil(t.SubscriptionExpiresAt), IsSuspended: t.IsSuspended,
		LastCredentialResetAt: dateOrNil(t.LastCredentialResetAt),
		BrandColorPreset:      t.BrandColorPreset, HasLogo: t.LogoStoragePath != nil,
		SignupSource: string(t.SignupSource),
	}
}

func toBrandingResponse(t domain.Tenant) brandingResponse {
	return brandingResponse{
		BusinessName: t.BusinessName, BrandColorPreset: t.BrandColorPreset, HasLogo: t.LogoStoragePath != nil,
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
	params := pagination.FromRequest(r)
	tenants, total, err := h.tenants.ListPaginated(r.Context(), params, tenantFilterFrom(r))
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
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	City         string `json:"city"`
}

type ownerInvitationResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	// Email: the Google account that must accept the invitation.
	Email string `json:"email"`
}

func toOwnerInvitation(inv application.OwnerInvitation) ownerInvitationResponse {
	return ownerInvitationResponse{Token: inv.Token, ExpiresAt: inv.ExpiresAt.Format(time.RFC3339), Email: inv.Email}
}

func (h *TenantHandler) register(w http.ResponseWriter, r *http.Request) {
	var body registerTenantBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	result, err := h.tenants.Register(r.Context(), application.RegisterTenantInput{
		BusinessName: body.BusinessName, OwnerName: body.OwnerName, Email: body.Email, Phone: body.Phone, City: body.City,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Tenant didaftarkan. Kirim link undangan ke Owner.", map[string]interface{}{
		"tenant":     toTenantResponse(result.Tenant),
		"invitation": toOwnerInvitation(result.Invitation),
	})
}

func (h *TenantHandler) Item(w http.ResponseWriter, r *http.Request) {
	segments := httpx.Segments(r.URL.Path, "/api/v1/tenants/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Tenant tidak ditemukan", nil)
		return
	}

	// "me" is the authenticated principal's own self-service read — never a
	// platform_admin-only path. /me itself (full tenant incl. subscription)
	// stays Owner-only; /me/branding and /me/logo are open to any tenant-scoped
	// principal (any staff role, or client), since branding must render for
	// everyone inside WO Console / Client Portal, not just the Owner.
	if segments[0] == "me" {
		switch {
		case len(segments) == 1 && r.Method == http.MethodGet:
			h.me(w, r)
		case len(segments) == 2 && segments[1] == "branding" && r.Method == http.MethodGet:
			h.myBranding(w, r)
		case len(segments) == 2 && segments[1] == "logo" && r.Method == http.MethodGet:
			h.myLogo(w, r)
		case len(segments) == 2 && segments[1] == "subscription" && r.Method == http.MethodGet:
			h.mySubscription(w, r)
		case len(segments) == 2 && segments[1] == "profile" && r.Method == http.MethodPatch:
			h.updateMyProfile(w, r)
		case len(segments) == 2 && segments[1] == "business-details" && r.Method == http.MethodGet:
			h.myBusinessDetails(w, r)
		case len(segments) == 2 && segments[1] == "business-details" && r.Method == http.MethodPut:
			h.updateMyBusinessDetails(w, r)
		case len(segments) == 2 && segments[1] == "contact" && r.Method == http.MethodGet:
			h.myContact(w, r)
		case len(segments) == 2 && segments[1] == "brand-color" && r.Method == http.MethodPatch:
			h.updateMyBrandColor(w, r)
		case len(segments) == 2 && segments[1] == "logo" && r.Method == http.MethodPut:
			h.uploadMyLogo(w, r)
		case len(segments) == 2 && segments[1] == "onboarding" && r.Method == http.MethodGet:
			h.myOnboarding(w, r)
		case len(segments) == 3 && segments[1] == "onboarding" && segments[2] == "dismiss" && r.Method == http.MethodPost:
			h.dismissMyOnboarding(w, r)
		default:
			response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		}
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
	case len(segments) == 2 && segments[1] == "logo" && r.Method == http.MethodPut:
		h.uploadLogo(w, r, id)
	case len(segments) == 2 && segments[1] == "logo" && r.Method == http.MethodGet:
		h.downloadLogo(w, r, id)
	case len(segments) == 2 && segments[1] == "toggle-suspension" && r.Method == http.MethodPost:
		h.toggleSuspension(w, r, id)
	case len(segments) == 2 && segments[1] == "owner-invite" && r.Method == http.MethodPost:
		h.reinviteOwner(w, r, id)
	case len(segments) == 2 && segments[1] == "activate-subscription" && r.Method == http.MethodPost:
		h.activateSubscription(w, r, id)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// myBranding is the self-service read powering WO Console/Client Portal
// theming — open to any authenticated tenant-scoped principal (unlike me,
// which stays Owner-only since it also carries subscription/billing data).
func (h *TenantHandler) myBranding(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := selfTenantID(r)
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}
	tenant, err := h.tenants.Get(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toBrandingResponse(*tenant))
}

// mySubscription backs GET /tenants/me/subscription — the lifecycle summary
// (status, access level, days left) every staff role needs for the WO
// Console banner. Unlike /me it carries no billing history, so it is not
// Owner-only; it is staff-only (clients never see the WO's billing state).
func (h *TenantHandler) mySubscription(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" {
		response.Error(w, http.StatusForbidden, "Hanya staff WO yang dapat melihat status langganan", nil)
		return
	}
	tenantID, ok := selfTenantID(r)
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}
	tenant, err := h.tenants.Get(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toLifecycleResponse(h.tenants.Lifecycle(*tenant)))
}

// myLogo streams the caller's own tenant's logo — same auth scoping as
// myBranding above.
func (h *TenantHandler) myLogo(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := selfTenantID(r)
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}
	streamLogo(w, r, h.tenants, tenantID)
}

// selfTenantID resolves the calling principal's own tenant from the JWT
// claim — never a request parameter. Any tenant-bound principal (staff of
// any role, or client) qualifies; platform_admin principals have none.
func selfTenantID(r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		return 0, false
	}
	return claims.TenantIDInt()
}

// me is the tenant Owner's self-service read of their own tenant record
// (plan, subscription status/expiry) — scoped from the JWT claim, never a
// request parameter. Powers the WO Console's own "Langganan" page.
func (h *TenantHandler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	g := middleware.GrantsFrom(r.Context())
	if !ok || claims.PrincipalType != "staff" || !(g.Can("subscription.manage") || g.Can("business_profile.manage")) {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat mengakses data workspace ini", map[string]string{"code": "forbidden_permission"})
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
	BusinessName     string `json:"businessName"`
	OwnerName        string `json:"ownerName"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	City             string `json:"city"`
	BrandColorPreset string `json:"brandColorPreset"`
}

func (h *TenantHandler) update(w http.ResponseWriter, r *http.Request, id int64) {
	var body updateTenantBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	// The Owner's email is the Google account that logs in as Owner after an
	// invitation, so changing it is handing over the workspace: only Super
	// Admin may, not the Support role that otherwise edits tenants.
	if claims, ok := middleware.FromContext(r.Context()); ok && claims.Role != string(domain.RoleSuperAdmin) {
		current, err := h.tenants.Get(r.Context(), id)
		if err != nil {
			writeAppError(w, err)
			return
		}
		if !strings.EqualFold(strings.TrimSpace(body.Email), current.Email) {
			response.Error(w, http.StatusForbidden, "Hanya Super Admin yang dapat mengganti email Owner. Email itu menentukan akun Google yang menjadi Owner.", map[string]string{"code": "forbidden_role"})
			return
		}
	}
	tenant, err := h.tenants.Update(r.Context(), id, application.UpdateTenantInput{
		BusinessName: body.BusinessName, OwnerName: body.OwnerName, Email: body.Email, Phone: body.Phone, City: body.City,
		BrandColorPreset: body.BrandColorPreset,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tenant berhasil diperbarui", toTenantResponse(*tenant))
}

type logoUploadBody struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	Base64Data string `json:"base64Data"`
}

// uploadLogo is the platform-admin action (superadmin sets a tenant's logo
// via the tenant edit form) — see PLAN.md §6 open question 1.
func (h *TenantHandler) uploadLogo(w http.ResponseWriter, r *http.Request, id int64) {
	var body logoUploadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	tenant, err := h.tenants.UploadLogo(r.Context(), id, application.UploadLogoInput{
		FileName: body.FileName, MimeType: body.MimeType, Base64Data: body.Base64Data,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Logo tenant berhasil diunggah", toTenantResponse(*tenant))
}

func (h *TenantHandler) downloadLogo(w http.ResponseWriter, r *http.Request, id int64) {
	streamLogo(w, r, h.tenants, id)
}

func streamLogo(w http.ResponseWriter, r *http.Request, tenants *application.TenantService, tenantID int64) {
	reader, err := tenants.DownloadLogo(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Disposition", `inline; filename="logo"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
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

// reinviteOwner is POST /tenants/{id}/owner-invite (Support/Super Admin):
// a new Owner invitation link; no password is ever set by the team.
func (h *TenantHandler) reinviteOwner(w http.ResponseWriter, r *http.Request, id int64) {
	inv, err := h.tenants.ReinviteOwner(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Link undangan Owner dibuat", toOwnerInvitation(*inv))
}

type activateSubscriptionBody struct {
	PlanID int64 `json:"planId"`
}

type manualActivationBody struct {
	PlanID         int64  `json:"planId"`
	Reason         string `json:"reason"`
	ProofReference string `json:"proofReference"`
	Note           string `json:"note"`
}

func (h *TenantHandler) activateSubscription(w http.ResponseWriter, r *http.Request, id int64) {
	var body manualActivationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	claims, _ := middleware.FromContext(r.Context())
	tenant, err := h.tenants.ActivateSubscription(r.Context(), id, application.ActivateSubscriptionInput{
		PlanID: body.PlanID, Reason: domain.ManualActivationReason(body.Reason),
		ProofReference: body.ProofReference, Note: body.Note, ActorAdminID: claims.PrincipalID,
	})
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
	if !ok || claims.PrincipalType != "staff" || !middleware.GrantsFrom(r.Context()).Can("subscription.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat mengelola langganan", map[string]string{"code": "forbidden_permission"})
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

func (h *TenantHandler) PendingCharge(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || !middleware.GrantsFrom(r.Context()).Can("subscription.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat mengelola langganan", map[string]string{"code": "forbidden_permission"})
		return
	}
	tenantID, err := application.ParseTenantID(claims.TenantID)
	if err != nil {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}

	charge, err := h.tenants.GetPendingCharge(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if charge == nil {
		response.OK(w, "ok", nil)
		return
	}
	response.OK(w, "ok", toChargeResponse(*charge))
}

func (h *TenantHandler) CancelPendingCharge(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || !middleware.GrantsFrom(r.Context()).Can("subscription.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat mengelola langganan", map[string]string{"code": "forbidden_permission"})
		return
	}
	tenantID, err := application.ParseTenantID(claims.TenantID)
	if err != nil {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}

	if err := h.tenants.CancelPendingCharge(r.Context(), tenantID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tagihan pembayaran dibatalkan", nil)
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
	response.FromError(w, err)
}

// tenantFilterFrom reads ?search=, ?status=, ?planId= and ?source= (the same
// filters the list page and the CSV export share).
func tenantFilterFrom(r *http.Request) domain.TenantFilter {
	q := r.URL.Query()
	f := domain.TenantFilter{Search: strings.TrimSpace(q.Get("search")), Status: q.Get("status"), Source: q.Get("source")}
	if id, err := strconv.ParseInt(q.Get("planId"), 10, 64); err == nil && id > 0 {
		f.PlanID = &id
	}
	return f
}
