package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

// SignupHandler serves the public self-service WO registration (PLAN.md §3).
type SignupHandler struct {
	tenants *application.TenantService
	// perIP caps workspace creation attempts from one address.
	perIP *middleware.RateLimiter
}

const (
	signupWindow   = time.Hour
	signupMaxPerIP = 10
	maxSignupBody  = 16 << 10
)

func NewSignupHandler(tenants *application.TenantService) *SignupHandler {
	return &SignupHandler{tenants: tenants, perIP: middleware.NewRateLimiter(signupMaxPerIP, signupWindow)}
}

type createWorkspaceBody struct {
	Credential     string `json:"credential"`
	BusinessName   string `json:"businessName"`
	AcceptTerms    bool   `json:"acceptTerms"`
	MarketingOptIn bool   `json:"marketingOptIn"`
}

type createWorkspaceResponse struct {
	TenantID     int64  `json:"tenantId"`
	Email        string `json:"email"`
	TermsVersion string `json:"termsVersion"`
}

// CreateWorkspace is POST /public/signup/workspace. On success the browser
// signs in with the same Google credential (POST /auth/google).
func (h *SignupHandler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSignupBody)
	var body createWorkspaceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Credential == "" {
		response.Error(w, http.StatusBadRequest, "Data pendaftaran tidak valid. Silakan daftar ulang dengan Google.", nil)
		return
	}
	ip := middleware.ClientIP(r)
	if !h.perIP.Allow(ip) {
		w.Header().Set("Retry-After", strconv.Itoa(h.perIP.RetryAfterSeconds()))
		response.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan pendaftaran. Silakan coba lagi nanti.",
			map[string]string{"code": "rate_limited"})
		return
	}
	result, err := h.tenants.CreateWorkspace(r.Context(), application.CreateWorkspaceInput{
		IDToken:        body.Credential,
		BusinessName:   body.BusinessName,
		AcceptTerms:    body.AcceptTerms,
		MarketingOptIn: body.MarketingOptIn,
		IPAddress:      ip,
		UserAgent:      r.UserAgent(),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Workspace berhasil dibuat", createWorkspaceResponse{
		TenantID: result.TenantID, Email: result.Email, TermsVersion: domain.TermsVersion,
	})
}

// ownerTenantID returns the caller's tenant when the caller's role may
// manage the business profile (business_profile.manage — the Owner by
// default, ADR-0026); otherwise it writes a 403.
func ownerTenantID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || !middleware.GrantsFrom(r.Context()).Can("business_profile.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat mengelola profil usaha", map[string]string{"code": "forbidden_permission"})
		return 0, false
	}
	id, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return id, true
}

type profileBody struct {
	BusinessName string `json:"businessName"`
	Phone        string `json:"phone"`
	City         string `json:"city"`
}

// updateMyProfile is PATCH /tenants/me/profile (Owner): "Lengkapi Profil".
func (h *TenantHandler) updateMyProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	var body profileBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	tenant, err := h.tenants.UpdateOwnProfile(r.Context(), tenantID, application.UpdateProfileInput{
		BusinessName: body.BusinessName, Phone: body.Phone, City: body.City,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Profil WO tersimpan", toTenantResponse(*tenant))
}

type brandColorBody struct {
	Preset string `json:"preset"`
}

// updateMyBrandColor is PATCH /tenants/me/brand-color (business_profile.manage).
func (h *TenantHandler) updateMyBrandColor(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	var body brandColorBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	tenant, err := h.tenants.UpdateOwnBrandColor(r.Context(), tenantID, body.Preset)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Warna tersimpan", toTenantResponse(*tenant))
}

type businessDetailsBody struct {
	Address           string `json:"address"`
	BankName          string `json:"bankName"`
	BankAccountNumber string `json:"bankAccountNumber"`
	BankAccountHolder string `json:"bankAccountHolder"`
}

type businessDetailsResponse struct {
	Address           string `json:"address"`
	BankName          string `json:"bankName"`
	BankAccountNumber string `json:"bankAccountNumber"`
	BankAccountHolder string `json:"bankAccountHolder"`
}

func toBusinessDetailsResponse(d domain.BusinessDetails) businessDetailsResponse {
	return businessDetailsResponse{Address: d.Address, BankName: d.BankName, BankAccountNumber: d.BankAccountNumber, BankAccountHolder: d.BankAccountHolder}
}

// myBusinessDetails is GET /tenants/me/business-details (business_profile.manage).
func (h *TenantHandler) myBusinessDetails(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	d, err := h.tenants.GetBusinessDetails(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toBusinessDetailsResponse(d))
}

// updateMyBusinessDetails is PUT /tenants/me/business-details (business_profile.manage).
func (h *TenantHandler) updateMyBusinessDetails(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	var body businessDetailsBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	d, err := h.tenants.SaveBusinessDetails(r.Context(), tenantID, application.BusinessDetailsInput{
		Address: body.Address, BankName: body.BankName, BankAccountNumber: body.BankAccountNumber, BankAccountHolder: body.BankAccountHolder,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Data usaha tersimpan", toBusinessDetailsResponse(d))
}

type contactResponse struct {
	BusinessName string `json:"businessName"`
	// WhatsApp is the business number, digits with the country code (62812…).
	WhatsApp string `json:"whatsapp"`
	City     string `json:"city"`
	Address  string `json:"address"`
	// Bank* are present only for people who may see what they owe: staff, or a
	// client whose portal role has portal.finance.view.
	BankName          string `json:"bankName,omitempty"`
	BankAccountNumber string `json:"bankAccountNumber,omitempty"`
	BankAccountHolder string `json:"bankAccountHolder,omitempty"`
}

// myContact is GET /tenants/me/contact: how to reach the business and, for
// those who may pay, where to pay. Open to any tenant-scoped principal (the
// Client Portal's contact and payment cards read it).
func (h *TenantHandler) myContact(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	tenantID, hasTenant := selfTenantID(r)
	if !ok || !hasTenant || (claims.PrincipalType != "staff" && claims.PrincipalType != "client") {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}
	tenant, err := h.tenants.Get(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	d, err := h.tenants.GetBusinessDetails(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	res := contactResponse{BusinessName: tenant.BusinessName, WhatsApp: tenant.Phone, City: tenant.City, Address: d.Address}
	if claims.PrincipalType == "staff" || middleware.GrantsFrom(r.Context()).Can("portal.finance.view") {
		res.BankName, res.BankAccountNumber, res.BankAccountHolder = d.BankName, d.BankAccountNumber, d.BankAccountHolder
	}
	response.OK(w, "ok", res)
}

// uploadMyLogo is PUT /tenants/me/logo (Owner).
func (h *TenantHandler) uploadMyLogo(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	var body logoUploadBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	tenant, err := h.tenants.UploadLogo(r.Context(), tenantID, application.UploadLogoInput{
		FileName: body.FileName, MimeType: body.MimeType, Base64Data: body.Base64Data,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Logo tersimpan", toTenantResponse(*tenant))
}

type trialLimitsResponse struct {
	MaxStaff          int `json:"maxStaff"`
	MaxActiveProjects int `json:"maxActiveProjects"`
	MaxClients        int `json:"maxClients"`
}

func toTrialLimitsResponse(l domain.TrialLimits) trialLimitsResponse {
	return trialLimitsResponse{MaxStaff: l.MaxStaff, MaxActiveProjects: l.MaxActiveProjects, MaxClients: l.MaxClients}
}

type usageResponse struct {
	Staff          int `json:"staff"`
	ActiveProjects int `json:"activeProjects"`
	Clients        int `json:"clients"`
}

type onboardingResponse struct {
	Dismissed       bool                 `json:"dismissed"`
	ProfileComplete bool                 `json:"profileComplete"`
	HasLogo         bool                 `json:"hasLogo"`
	HasPaidPlan     bool                 `json:"hasPaidPlan"`
	Usage           usageResponse        `json:"usage"`
	TrialLimits     *trialLimitsResponse `json:"trialLimits"`
}

// myOnboarding is GET /tenants/me/onboarding (Owner): the "Mulai di sini"
// checklist and trial usage.
func (h *TenantHandler) myOnboarding(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	st, err := h.tenants.Onboarding(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	res := onboardingResponse{
		Dismissed: st.Dismissed, ProfileComplete: st.ProfileComplete, HasLogo: st.HasLogo, HasPaidPlan: st.HasPaidPlan,
		Usage: usageResponse{Staff: st.Staff, ActiveProjects: st.ActiveProjects, Clients: st.Clients},
	}
	if st.Limits != nil {
		l := toTrialLimitsResponse(*st.Limits)
		res.TrialLimits = &l
	}
	response.OK(w, "ok", res)
}

// dismissMyOnboarding is POST /tenants/me/onboarding/dismiss (Owner).
func (h *TenantHandler) dismissMyOnboarding(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	if err := h.tenants.DismissOnboarding(r.Context(), tenantID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Panduan disembunyikan", nil)
}

// TrialLimits is GET/PUT /trial-limits (Platform Console; role rules in
// cmd/server/platform_scope.go).
func (h *TenantHandler) TrialLimits(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		l, err := h.tenants.GetTrialLimits(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", toTrialLimitsResponse(l))
	case http.MethodPut:
		var body trialLimitsResponse
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		l, err := h.tenants.UpdateTrialLimits(r.Context(), domain.TrialLimits{
			MaxStaff: body.MaxStaff, MaxActiveProjects: body.MaxActiveProjects, MaxClients: body.MaxClients,
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Batas masa coba tersimpan", toTrialLimitsResponse(l))
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan", nil)
	}
}
