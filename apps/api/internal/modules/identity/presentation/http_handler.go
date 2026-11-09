package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/identity/application"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

type Handler struct {
	auth *application.AuthService
	// loginIP caps login and two-factor attempts per caller IP. Failed
	// attempts per *account* are capped inside AuthService (by credential
	// id), so spelling variants of one identifier share one budget.
	loginIP *middleware.RateLimiter
	// linkIP caps one-time link requests per IP separately, so opening an
	// invitation page never uses up the visitor's login attempts.
	linkIP *middleware.RateLimiter
}

const (
	loginWindow   = 15 * time.Minute
	loginMaxPerIP = 30
	linkMaxPerIP  = 60
	// Auth request bodies are tiny; anything larger is refused before
	// decoding (memory safety).
	maxAuthBodyBytes = 16 << 10
	maxIdentifierLen = 254
	maxPasswordLen   = 1024
)

func NewHandler(auth *application.AuthService) *Handler {
	return &Handler{
		auth:    auth,
		loginIP: middleware.NewRateLimiter(loginMaxPerIP, loginWindow),
		linkIP:  middleware.NewRateLimiter(linkMaxPerIP, loginWindow),
	}
}

func limitBody(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
}

func (h *Handler) tooManyAttempts(w http.ResponseWriter, limiter *middleware.RateLimiter) {
	w.Header().Set("Retry-After", strconv.Itoa(limiter.RetryAfterSeconds()))
	response.Error(w, http.StatusTooManyRequests,
		"Terlalu banyak percobaan masuk. Silakan coba lagi dalam 15 menit.",
		map[string]string{"code": "rate_limited"})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type sessionResponse struct {
	AccessToken   string `json:"accessToken"`
	RefreshToken  string `json:"refreshToken"`
	PrincipalType string `json:"principalType"`
	PrincipalID   string `json:"principalId"`
	TenantID      *int64 `json:"tenantId"`
	Role          string `json:"role"`
	DisplayName   string `json:"displayName"`
	// MFA: the session was established with a second factor.
	// MFASetupRequired: a Platform Console admin must enrol before using
	// the console (the API refuses everything else until then).
	MFA              bool `json:"mfa"`
	MFASetupRequired bool `json:"mfaSetupRequired"`
}

// mfaChallengeResponse is returned by login instead of a session when the
// account has two-factor on; answer it at POST /auth/mfa/verify.
type mfaChallengeResponse struct {
	MFARequired bool   `json:"mfaRequired"`
	MFAToken    string `json:"mfaToken"`
}

func toSessionResponse(s *application.Session) sessionResponse {
	return sessionResponse{
		AccessToken:      s.AccessToken,
		RefreshToken:     s.RefreshToken,
		PrincipalType:    s.PrincipalType,
		PrincipalID:      s.PrincipalID,
		TenantID:         s.TenantID,
		Role:             s.Role,
		DisplayName:      s.DisplayName,
		MFA:              s.MFA,
		MFASetupRequired: s.MFASetupRequired,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r)
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	if req.Username == "" || req.Password == "" {
		response.Error(w, http.StatusUnprocessableEntity, "Username/email dan password wajib diisi", map[string][]string{
			"username": {"Username atau email wajib diisi"},
			"password": {"Password wajib diisi"},
		})
		return
	}

	if len(req.Username) > maxIdentifierLen || len(req.Password) > maxPasswordLen {
		response.Error(w, http.StatusUnprocessableEntity, "Username/email atau password terlalu panjang", nil)
		return
	}
	if !h.loginIP.Allow(middleware.ClientIP(r)) {
		h.tooManyAttempts(w, h.loginIP)
		return
	}

	result, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if result.MFARequired {
		response.OK(w, "Masukkan kode verifikasi dari aplikasi authenticator", mfaChallengeResponse{MFARequired: true, MFAToken: result.MFAToken})
		return
	}
	response.OK(w, "Login berhasil", toSessionResponse(result.Session))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r)
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "Refresh token wajib diisi", nil)
		return
	}

	session, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Sesi diperbarui", toSessionResponse(session))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r)
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.OK(w, "Berhasil keluar", nil)
		return
	}
	_ = h.auth.Logout(r.Context(), req.RefreshToken)
	response.OK(w, "Berhasil keluar", nil)
}

func writeAppError(w http.ResponseWriter, err error) {
	response.FromError(w, err)
}

type mfaVerifyRequest struct {
	MFAToken string `json:"mfaToken"`
	Code     string `json:"code"`
}

// VerifyMFA is POST /auth/mfa/verify (public): second login step.
func (h *Handler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r)
	var req mfaVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MFAToken == "" || req.Code == "" {
		response.Error(w, http.StatusUnprocessableEntity, "Kode verifikasi wajib diisi", map[string][]string{"code": {"Masukkan 6 digit kode"}})
		return
	}
	if !h.loginIP.Allow(middleware.ClientIP(r)) {
		h.tooManyAttempts(w, h.loginIP)
		return
	}
	session, err := h.auth.VerifyMFA(r.Context(), req.MFAToken, req.Code)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Login berhasil", toSessionResponse(session))
}

type mfaSetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauthUrl"`
}

// SetupMFA is POST /auth/mfa/setup (authenticated admin, not yet enrolled).
func (h *Handler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return
	}
	setup, err := h.auth.SetupMFA(r.Context(), claims.PrincipalType, claims.PrincipalID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tambahkan kunci ini ke aplikasi authenticator Anda", mfaSetupResponse{Secret: setup.Secret, OTPAuthURL: setup.OTPAuthURL})
}

type mfaEnableRequest struct {
	Code string `json:"code"`
}

// EnableMFA is POST /auth/mfa/enable: confirms enrolment with a first code
// and returns a new two-factor session (all other sessions are revoked).
func (h *Handler) EnableMFA(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r)
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return
	}
	var req mfaEnableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		response.Error(w, http.StatusUnprocessableEntity, "Kode verifikasi wajib diisi", map[string][]string{"code": {"Masukkan 6 digit kode"}})
		return
	}
	session, err := h.auth.EnableMFA(r.Context(), claims.PrincipalType, claims.PrincipalID, req.Code)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Verifikasi dua langkah aktif", toSessionResponse(session))
}

type googleLoginRequest struct {
	Credential string `json:"credential"`
}

// googleSignupResponse is returned instead of a session when the Google
// account has no ElProof login yet: the frontend continues to the
// workspace form (POST /public/signup/workspace) with the same credential.
type googleSignupResponse struct {
	SignupRequired bool   `json:"signupRequired,omitempty"`
	LinkRequired   bool   `json:"linkRequired,omitempty"`
	Invited        bool   `json:"invited,omitempty"`
	Email          string `json:"email"`
	Name           string `json:"name"`
}

// maxGoogleCredentialLen bounds the ID token (a few KB in practice).
const maxGoogleCredentialLen = 8 << 10

// LoginGoogle is POST /auth/google (public): Sign in with Google for WO
// Owners and staff.
func (h *Handler) LoginGoogle(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r)
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Credential == "" || len(req.Credential) > maxGoogleCredentialLen {
		response.Error(w, http.StatusBadRequest, "Data login Google tidak valid", nil)
		return
	}
	if !h.loginIP.Allow(middleware.ClientIP(r)) {
		h.tooManyAttempts(w, h.loginIP)
		return
	}
	result, err := h.auth.LoginWithGoogle(r.Context(), req.Credential)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if result.LinkRequired {
		response.OK(w, "Konfirmasi kata sandi untuk menghubungkan akun Google", googleSignupResponse{
			LinkRequired: true, Email: result.Identity.Email, Name: result.Identity.Name,
		})
		return
	}
	if result.SignupRequired {
		response.OK(w, "Akun Google ini belum terdaftar", googleSignupResponse{
			SignupRequired: true, Invited: result.Invited, Email: result.Identity.Email, Name: result.Identity.Name,
		})
		return
	}
	response.OK(w, "Login berhasil", toSessionResponse(result.Session))
}

type googleLinkRequest struct {
	Credential string `json:"credential"`
	Password   string `json:"password"`
}

// LinkGoogle is POST /auth/google/link (public): connect a Google account to
// the existing staff login with the same email by confirming that login's
// ElProof password once.
func (h *Handler) LinkGoogle(w http.ResponseWriter, r *http.Request) {
	limitBody(w, r)
	var req googleLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Credential == "" || len(req.Credential) > maxGoogleCredentialLen {
		response.Error(w, http.StatusBadRequest, "Data login Google tidak valid", nil)
		return
	}
	if req.Password == "" || len(req.Password) > maxPasswordLen {
		response.Error(w, http.StatusUnprocessableEntity, "Kata sandi wajib diisi", map[string][]string{"password": {"Kata sandi wajib diisi"}})
		return
	}
	if !h.loginIP.Allow(middleware.ClientIP(r)) {
		h.tooManyAttempts(w, h.loginIP)
		return
	}
	session, err := h.auth.LinkGoogleWithPassword(r.Context(), req.Credential, req.Password)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Akun Google terhubung", toSessionResponse(session))
}

type linkInfoResponse struct {
	Purpose     string `json:"purpose"`
	TenantName  string `json:"tenantName"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	ExpiresAt   string `json:"expiresAt"`
}

type linkPasswordRequest struct {
	Password string `json:"password"`
}

const maxLinkTokenLen = 128

// Links serves the public one-time access links (PLAN.md Fase 3):
//
//	GET  /auth/links/{token}           what the link is for (invitation page)
//	POST /auth/links/{token}/google    staff: accept with Google  {credential}
//	POST /auth/links/{token}/password  client: set own password   {password}
func (h *Handler) Links(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/auth/links/"), "/")
	token, action, _ := strings.Cut(rest, "/")
	if token == "" || len(token) > maxLinkTokenLen {
		response.Error(w, http.StatusNotFound, "Link tidak dikenali", map[string]string{"code": "link_invalid"})
		return
	}
	if !h.linkIP.Allow(middleware.ClientIP(r)) {
		h.tooManyAttempts(w, h.linkIP)
		return
	}
	// The token itself is a credential: never let a shared cache keep it.
	w.Header().Set("Cache-Control", "no-store")
	switch {
	case action == "" && r.Method == http.MethodGet:
		info, err := h.auth.AccessLinkInfo(r.Context(), token)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", linkInfoResponse{
			Purpose: string(info.Purpose), TenantName: info.TenantName, DisplayName: info.DisplayName,
			Username: info.Username, Email: info.Email, ExpiresAt: info.ExpiresAt.Format(time.RFC3339),
		})
	case action == "google" && r.Method == http.MethodPost:
		limitBody(w, r)
		var req googleLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Credential == "" || len(req.Credential) > maxGoogleCredentialLen {
			response.Error(w, http.StatusBadRequest, "Data login Google tidak valid", nil)
			return
		}
		session, err := h.auth.AcceptStaffInvite(r.Context(), token, req.Credential)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Undangan diterima", toSessionResponse(session))
	case action == "password" && r.Method == http.MethodPost:
		limitBody(w, r)
		var req linkPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		session, err := h.auth.SetPasswordWithLink(r.Context(), token, req.Password)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Kata sandi tersimpan", toSessionResponse(session))
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}
