package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/identity/application"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

type Handler struct {
	auth *application.AuthService
	// loginIP caps login attempts per caller IP (any outcome), and
	// loginFailures caps *failed* attempts per identifier — together they
	// stop both spraying many accounts from one IP and guessing one
	// account's password from many IPs (OWASP Authentication Cheat Sheet).
	loginIP       *middleware.RateLimiter
	loginFailures *middleware.RateLimiter
}

// Login limits (in-memory, single instance — ADR-0011).
const (
	loginWindow          = 15 * time.Minute
	loginMaxPerIP        = 30
	loginMaxFailedPerKey = 10
)

func NewHandler(auth *application.AuthService) *Handler {
	return &Handler{
		auth:          auth,
		loginIP:       middleware.NewRateLimiter(loginMaxPerIP, loginWindow),
		loginFailures: middleware.NewRateLimiter(loginMaxFailedPerKey, loginWindow),
	}
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
}

func toSessionResponse(s *application.Session) sessionResponse {
	return sessionResponse{
		AccessToken:   s.AccessToken,
		RefreshToken:  s.RefreshToken,
		PrincipalType: s.PrincipalType,
		PrincipalID:   s.PrincipalID,
		TenantID:      s.TenantID,
		Role:          s.Role,
		DisplayName:   s.DisplayName,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
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

	if !h.loginIP.Allow(middleware.ClientIP(r)) {
		h.tooManyAttempts(w, h.loginIP)
		return
	}
	identifierKey := strings.ToLower(strings.TrimSpace(req.Username))
	if h.loginFailures.Exceeded(identifierKey) {
		h.tooManyAttempts(w, h.loginFailures)
		return
	}

	session, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if appErr, ok := apperror.As(err); ok && appErr.Kind == apperror.KindUnauthorized {
			h.loginFailures.Record(identifierKey)
		}
		writeAppError(w, err)
		return
	}
	h.loginFailures.Reset(identifierKey)
	response.OK(w, "Login berhasil", toSessionResponse(session))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
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
