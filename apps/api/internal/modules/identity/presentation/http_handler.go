package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/identity/application"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

type Handler struct {
	auth *application.AuthService
	// loginIP caps login and two-factor attempts per caller IP. Failed
	// attempts per *account* are capped inside AuthService (by credential
	// id), so spelling variants of one identifier share one budget.
	loginIP *middleware.RateLimiter
}

const (
	loginWindow   = 15 * time.Minute
	loginMaxPerIP = 30
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
