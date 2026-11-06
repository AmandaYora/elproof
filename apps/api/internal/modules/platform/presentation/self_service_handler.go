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

// ownerTenantID returns the caller's tenant when the caller is that
// tenant's Owner; otherwise it writes a 403.
func ownerTenantID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || claims.Role != "Owner" {
		response.Error(w, http.StatusForbidden, "Hanya Owner WO yang dapat melakukan aksi ini", nil)
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
