package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type PlatformAdminHandler struct {
	admins *application.PlatformAdminService
}

func NewPlatformAdminHandler(admins *application.PlatformAdminService) *PlatformAdminHandler {
	return &PlatformAdminHandler{admins: admins}
}

type platformAdminResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Role     string `json:"role"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	IsActive bool   `json:"isActive"`
}

func toPlatformAdminResponse(a domain.PlatformAdmin) platformAdminResponse {
	return platformAdminResponse{
		ID: a.ID, Name: a.Name, Title: a.Title, Role: string(a.Role),
		Username: a.Username, Email: a.Email, Phone: a.Phone, IsActive: a.IsActive,
	}
}

func (h *PlatformAdminHandler) Collection(w http.ResponseWriter, r *http.Request) {
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

func (h *PlatformAdminHandler) list(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("all") == "true" {
		admins, err := h.admins.List(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]platformAdminResponse, 0, len(admins))
		for _, a := range admins {
			result = append(result, toPlatformAdminResponse(a))
		}
		response.OK(w, "ok", result)
		return
	}
	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")
	admins, total, err := h.admins.ListPaginated(r.Context(), params, search, role)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]platformAdminResponse, 0, len(admins))
	for _, a := range admins {
		result = append(result, toPlatformAdminResponse(a))
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
}

type registerPlatformAdminBody struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Role     string `json:"role"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (h *PlatformAdminHandler) register(w http.ResponseWriter, r *http.Request) {
	var body registerPlatformAdminBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	admin, err := h.admins.Register(r.Context(), application.RegisterPlatformAdminInput{
		Name: body.Name, Title: body.Title, Role: domain.PlatformAdminRole(body.Role), Username: body.Username,
		Email: body.Email, Phone: body.Phone, Password: body.Password,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Admin platform berhasil dibuat", map[string]interface{}{
		"admin":    toPlatformAdminResponse(*admin),
		"username": admin.Username,
	})
}

func (h *PlatformAdminHandler) Item(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/platform-admins/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Admin platform tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.update(w, r, id)
	case len(segments) == 2 && segments[1] == "toggle-active" && r.Method == http.MethodPost:
		h.toggleActive(w, r, id)
	case len(segments) == 2 && segments[1] == "reset-password" && r.Method == http.MethodPost:
		h.resetPassword(w, r, id)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

type updatePlatformAdminBody struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Role  string `json:"role"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (h *PlatformAdminHandler) update(w http.ResponseWriter, r *http.Request, id int64) {
	var body updatePlatformAdminBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	admin, err := h.admins.Update(r.Context(), id, application.UpdatePlatformAdminInput{
		Name: body.Name, Title: body.Title, Role: domain.PlatformAdminRole(body.Role), Email: body.Email, Phone: body.Phone,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Admin platform berhasil diperbarui", toPlatformAdminResponse(*admin))
}

func (h *PlatformAdminHandler) toggleActive(w http.ResponseWriter, r *http.Request, id int64) {
	claims, _ := middleware.FromContext(r.Context())
	if claims != nil && claims.PrincipalID == strconv.FormatInt(id, 10) {
		response.Error(w, http.StatusForbidden, "Anda tidak dapat menonaktifkan akun Anda sendiri", nil)
		return
	}

	current, err := h.admins.Get(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	admin, err := h.admins.SetActive(r.Context(), id, !current.IsActive)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status admin platform diperbarui", toPlatformAdminResponse(*admin))
}

type resetPlatformAdminPasswordBody struct {
	Password string `json:"password"`
}

func (h *PlatformAdminHandler) resetPassword(w http.ResponseWriter, r *http.Request, id int64) {
	var body resetPlatformAdminPasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	admin, err := h.admins.ResetPassword(r.Context(), id, body.Password)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Password admin platform berhasil direset", map[string]string{"username": admin.Username, "password": body.Password})
}
