package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"elproof/internal/modules/staff/application"
	"elproof/internal/modules/staff/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type Handler struct {
	staff *application.StaffService
}

func NewHandler(staff *application.StaffService) *Handler {
	return &Handler{staff: staff}
}

type staffResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Initials string `json:"initials"`
	Role     string `json:"role"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	IsActive bool   `json:"isActive"`
}

func toStaffResponse(m domain.StaffMember) staffResponse {
	return staffResponse{
		ID: m.ID, Name: m.Name, Title: m.Title, Initials: m.Initials,
		Role: string(m.Role), Username: m.Username, Email: m.Email, Phone: m.Phone, IsActive: m.IsActive,
	}
}

func requireTenant(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

func (h *Handler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("all") == "true" {
			members, err := h.staff.List(r.Context(), tenantID)
			if err != nil {
				writeAppError(w, err)
				return
			}
			result := make([]staffResponse, 0, len(members))
			for _, m := range members {
				result = append(result, toStaffResponse(m))
			}
			response.OK(w, "ok", result)
			return
		}
		params := pagination.FromRequest(r)
		search := r.URL.Query().Get("search")
		role := r.URL.Query().Get("role")
		members, total, err := h.staff.ListPaginated(r.Context(), tenantID, params, search, role)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]staffResponse, 0, len(members))
		for _, m := range members {
			result = append(result, toStaffResponse(m))
		}
		response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
	case http.MethodPost:
		h.create(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

type staffInputBody struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Role     string `json:"role"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body staffInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	member, err := h.staff.Create(r.Context(), tenantID, application.StaffInput{
		Name: body.Name, Title: body.Title, Role: domain.StaffRole(body.Role),
		Username: body.Username, Password: body.Password, Email: body.Email, Phone: body.Phone,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Pengguna berhasil ditambahkan", toStaffResponse(*member))
}

func (h *Handler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/staff/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Pengguna tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.update(w, r, tenantID, id)
	case len(segments) == 2 && segments[1] == "toggle-active" && r.Method == http.MethodPost:
		h.toggleActive(w, r, tenantID, id)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body staffInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	claims, _ := middleware.FromContext(r.Context())
	isSelf := claims != nil && claims.PrincipalID == strconv.FormatInt(id, 10)
	member, err := h.staff.Update(r.Context(), tenantID, id, isSelf, application.StaffInput{
		Name: body.Name, Title: body.Title, Role: domain.StaffRole(body.Role), Email: body.Email, Phone: body.Phone,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pengguna berhasil diperbarui", toStaffResponse(*member))
}

func (h *Handler) toggleActive(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	current, err := h.staff.Get(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	member, err := h.staff.SetActive(r.Context(), tenantID, id, !current.IsActive)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status pengguna diperbarui", toStaffResponse(*member))
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
