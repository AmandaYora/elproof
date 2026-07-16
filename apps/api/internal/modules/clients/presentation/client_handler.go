package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"elproof/internal/modules/clients/application"
	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type Handler struct {
	clients *application.ClientService
}

func NewHandler(clients *application.ClientService) *Handler {
	return &Handler{clients: clients}
}

type clientResponse struct {
	ID                    int64   `json:"id"`
	ProjectID             int64   `json:"projectId"`
	Role                  string  `json:"role"`
	RelationNote          string  `json:"relationNote"`
	Name                  string  `json:"name"`
	Phone                 string  `json:"phone"`
	Email                 string  `json:"email"`
	IsActive              bool    `json:"isActive"`
	LastCredentialResetAt *string `json:"lastCredentialResetAt"`
}

func toClientResponse(c domain.Client) clientResponse {
	var lastReset *string
	if c.LastCredentialResetAt != nil {
		s := c.LastCredentialResetAt.Format("2006-01-02")
		lastReset = &s
	}
	return clientResponse{
		ID: c.ID, ProjectID: c.ProjectID, Role: string(c.Role), RelationNote: c.RelationNote,
		Name: c.Name, Phone: c.Phone, Email: c.Email, IsActive: c.IsActive, LastCredentialResetAt: lastReset,
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
		h.list(w, r, tenantID)
	case http.MethodPost:
		h.create(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, tenantID int64) {
	projectIDRaw := r.URL.Query().Get("projectId")

	if projectIDRaw != "" {
		projectID, err := strconv.ParseInt(projectIDRaw, 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "projectId tidak valid", nil)
			return
		}
		list, err := h.clients.ListByProject(r.Context(), tenantID, projectID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]clientResponse, 0, len(list))
		for _, c := range list {
			result = append(result, toClientResponse(c))
		}
		response.OK(w, "ok", result)
		return
	}

	// No projectId filter — list every client for the tenant. `all=true` is
	// used by WO Console's global search, which matches names across all
	// projects in one shot; otherwise this is the tenant-wide Client list page.
	if r.URL.Query().Get("all") == "true" {
		list, err := h.clients.ListByTenant(r.Context(), tenantID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]clientResponse, 0, len(list))
		for _, c := range list {
			result = append(result, toClientResponse(c))
		}
		response.OK(w, "ok", result)
		return
	}

	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	list, total, err := h.clients.ListByTenantPaginated(r.Context(), tenantID, params, search)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]clientResponse, 0, len(list))
	for _, c := range list {
		result = append(result, toClientResponse(c))
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
}

type createClientBody struct {
	ProjectID    int64  `json:"projectId"`
	Role         string `json:"role"`
	RelationNote string `json:"relationNote"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Password     string `json:"password"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body createClientBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	c, err := h.clients.Create(r.Context(), tenantID, application.CreateClientInput{
		ProjectID: body.ProjectID, Role: domain.ClientRole(body.Role), RelationNote: body.RelationNote,
		Name: body.Name, Phone: body.Phone, Email: body.Email, Password: body.Password,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Client berhasil ditambahkan", toClientResponse(*c))
}

func (h *Handler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/clients/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Client tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.updateContact(w, r, tenantID, id)
	case len(segments) == 2 && segments[1] == "toggle-active" && r.Method == http.MethodPost:
		h.toggleActive(w, r, tenantID, id)
	case len(segments) == 2 && segments[1] == "reset-credential" && r.Method == http.MethodPost:
		h.resetCredential(w, r, tenantID, id)
	case len(segments) == 2 && segments[1] == "replace-representative" && r.Method == http.MethodPost:
		h.replaceRepresentative(w, r, tenantID, id)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

type contactBody struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

func (h *Handler) updateContact(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body contactBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	c, err := h.clients.UpdateContact(r.Context(), tenantID, id, application.UpdateContactInput{Name: body.Name, Phone: body.Phone, Email: body.Email})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kontak client berhasil diperbarui", toClientResponse(*c))
}

func (h *Handler) toggleActive(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	current, err := h.clients.Get(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	c, err := h.clients.SetActive(r.Context(), tenantID, id, !current.IsActive)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status client diperbarui", toClientResponse(*c))
}

type resetCredentialBody struct {
	Password string `json:"password"`
}

func (h *Handler) resetCredential(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body resetCredentialBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	c, err := h.clients.ResetCredential(r.Context(), tenantID, id, body.Password)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kredensial client berhasil direset", toClientResponse(*c))
}

type replaceRepresentativeBody struct {
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	RelationNote string `json:"relationNote"`
}

func (h *Handler) replaceRepresentative(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body replaceRepresentativeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	c, err := h.clients.ReplaceRepresentative(r.Context(), tenantID, id,
		application.UpdateContactInput{Name: body.Name, Phone: body.Phone, Email: body.Email}, body.RelationNote)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Perwakilan keluarga berhasil diganti", toClientResponse(*c))
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
