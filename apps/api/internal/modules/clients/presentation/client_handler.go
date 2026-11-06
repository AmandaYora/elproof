package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	Username              string  `json:"username"`
	RelationNote          string  `json:"relationNote"`
	Name                  string  `json:"name"`
	Phone                 string  `json:"phone"`
	Email                 string  `json:"email"`
	IsActive              bool    `json:"isActive"`
	LastCredentialResetAt *string `json:"lastCredentialResetAt"`
	// LoginStatus: "invited" until the client set their password.
	LoginStatus string `json:"loginStatus,omitempty"`
}

type accessLinkResponse struct {
	Token     string `json:"token"`
	Purpose   string `json:"purpose"` // client_invite | client_reset
	ExpiresAt string `json:"expiresAt"`
}

func toAccessLinkResponse(l *application.AccessLinkResult) accessLinkResponse {
	return accessLinkResponse{Token: l.Token, Purpose: l.Purpose, ExpiresAt: l.ExpiresAt.Format(time.RFC3339)}
}

type clientWithLinkResponse struct {
	Client     clientResponse     `json:"client"`
	AccessLink accessLinkResponse `json:"accessLink"`
}

func (h *Handler) withLoginStatus(r *http.Request, c domain.Client) clientResponse {
	resp := toClientResponse(c)
	resp.LoginStatus = h.clients.LoginStatus(r.Context(), c.ID)
	return resp
}

func toClientResponse(c domain.Client) clientResponse {
	var lastReset *string
	if c.LastCredentialResetAt != nil {
		s := c.LastCredentialResetAt.Format("2006-01-02")
		lastReset = &s
	}
	return clientResponse{
		ID: c.ID, ProjectID: c.ProjectID, Role: string(c.Role), Username: c.Username, RelationNote: c.RelationNote,
		Name: c.Name, Phone: c.Phone, Email: c.Email, IsActive: c.IsActive, LastCredentialResetAt: lastReset,
	}
}

// requireTenant gates every /clients endpoint to a WO staff principal. A
// `client` principal (bride/groom/family) is rejected outright: before
// Fase 1, VerifyProjectReadAccess only special-cased the "Staff"/"Sales"
// roles, so a client's role ("Bride", ...) fell through to "allowed" and
// GET /clients?projectId=<any project> returned other couples' names,
// phone numbers, usernames and emails.
func requireTenant(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return 0, false
	}
	if claims.PrincipalType != "staff" {
		response.Error(w, http.StatusForbidden, "Hanya staff WO yang dapat mengakses data client", nil)
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

// requireManagerRole is the extra gate the standalone (non-project-scoped)
// list needs on top of requireTenant — mirrors vendors/venue_handler.go's
// own requireManagerRole exactly: Owner/Admin only, since a tenant-wide
// client list has no single project to scope a Wedding Planner or Sales
// staff member against.
func requireManagerRole(w http.ResponseWriter, r *http.Request) bool {
	claims, _ := middleware.FromContext(r.Context())
	if !claims.HasRole("Owner", "Admin") {
		response.Error(w, http.StatusForbidden, "Hanya Owner atau Admin yang dapat melakukan aksi ini", nil)
		return false
	}
	return true
}

// requireManagerOrProjectSales is the extra gate every Client **write**
// action needs on top of requireTenant — Owner/Admin write freely across
// every project (same as before); a Sales staff member additionally gets
// full write access, but only within the one project they're PIC Sales of
// (PLAN.md revisi-timeline-vendor-role-sales, confirmed: Sales gets full
// Client write, unlike a Wedding Planner who stays read-only everywhere —
// see VerifyProjectReadAccess). Every other role (notably "Staff") is
// rejected outright, same scope requireManagerRole always had.
func (h *Handler) requireManagerOrProjectSales(w http.ResponseWriter, r *http.Request, tenantID, projectID int64) bool {
	claims, _ := middleware.FromContext(r.Context())
	if claims.HasRole("Owner", "Admin") {
		return true
	}
	if claims.HasRole("Sales") {
		staffID, err := strconv.ParseInt(claims.PrincipalID, 10, 64)
		if err == nil && h.clients.VerifyProjectSalesOwner(r.Context(), tenantID, projectID, staffID) == nil {
			return true
		}
	}
	response.Error(w, http.StatusForbidden, "Anda tidak memiliki akses untuk melakukan aksi ini", nil)
	return false
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
		// A Wedding Planner may read clients for exactly the project they're
		// PIC of (used by Project Detail's own Client tab) — anyone else
		// (Owner/Admin) reads any project's clients freely.
		claims, _ := middleware.FromContext(r.Context())
		staffID, _ := strconv.ParseInt(claims.PrincipalID, 10, 64)
		if err := h.clients.VerifyProjectReadAccess(r.Context(), tenantID, projectID, staffID, claims.Role); err != nil {
			writeAppError(w, err)
			return
		}
		list, err := h.clients.ListByProject(r.Context(), tenantID, projectID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]clientResponse, 0, len(list))
		for _, c := range list {
			result = append(result, h.withLoginStatus(r, c))
		}
		response.OK(w, "ok", result)
		return
	}

	// No projectId filter — list every client for the tenant. `all=true` is
	// used by WO Console's global search, which matches names across all
	// projects in one shot; otherwise this is the tenant-wide Client list
	// page. Both are Owner/Admin only (a Wedding Planner has no legitimate
	// need to see clients across projects that aren't theirs).
	if !requireManagerRole(w, r) {
		return
	}
	if r.URL.Query().Get("all") == "true" {
		list, err := h.clients.ListByTenant(r.Context(), tenantID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]clientResponse, 0, len(list))
		for _, c := range list {
			result = append(result, h.withLoginStatus(r, c))
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
		result = append(result, h.withLoginStatus(r, c))
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
}

type createClientBody struct {
	ProjectID    int64  `json:"projectId"`
	Role         string `json:"role"`
	RelationNote string `json:"relationNote"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Username     string `json:"username"`
	Email        string `json:"email"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body createClientBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	if !h.requireManagerOrProjectSales(w, r, tenantID, body.ProjectID) {
		return
	}
	c, link, err := h.clients.Create(r.Context(), tenantID, application.CreateClientInput{
		ProjectID: body.ProjectID, Role: domain.ClientRole(body.Role), RelationNote: body.RelationNote,
		Name: body.Name, Phone: body.Phone, Username: body.Username, Email: body.Email,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp := toClientResponse(*c)
	resp.LoginStatus = "invited"
	response.Created(w, "Client ditambahkan. Kirim link undangan agar client mengatur kata sandinya sendiri.",
		clientWithLinkResponse{Client: resp, AccessLink: toAccessLinkResponse(link)})
}

// Item only ever dispatches write actions (update/delete/toggle-active/
// reset-credential/replace-representative — no GET case exists here), so
// the whole function is gated in one place — Owner/Admin unconditionally, or
// a Sales staff member scoped to this client's own project (see
// requireManagerOrProjectSales). The client row is fetched once here just to
// resolve its ProjectID for that check; every sub-handler below still does
// its own lookup when it needs the row's other fields, same as before.
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
	c, err := h.clients.Get(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if !h.requireManagerOrProjectSales(w, r, tenantID, c.ProjectID) {
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.updateContact(w, r, tenantID, id)
	case len(segments) == 1 && r.Method == http.MethodDelete:
		h.delete(w, r, tenantID, id)
	case len(segments) == 2 && segments[1] == "toggle-active" && r.Method == http.MethodPost:
		h.toggleActive(w, r, tenantID, id)
	case len(segments) == 2 && segments[1] == "access-link" && r.Method == http.MethodPost:
		h.accessLink(w, r, tenantID, id)
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

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	if err := h.clients.Delete(r.Context(), tenantID, id); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Client berhasil dihapus", nil)
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

// accessLink is POST /clients/{id}/access-link: a new invitation (client
// never set a password) or password-reset link to send by WhatsApp.
func (h *Handler) accessLink(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	link, err := h.clients.AccessLink(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	msg := "Link undangan baru dibuat"
	if link.Purpose == "client_reset" {
		msg = "Link atur ulang kata sandi dibuat"
	}
	response.OK(w, msg, toAccessLinkResponse(link))
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
	c, link, err := h.clients.ReplaceRepresentative(r.Context(), tenantID, id,
		application.UpdateContactInput{Name: body.Name, Phone: body.Phone, Email: body.Email}, body.RelationNote)
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp := toClientResponse(*c)
	resp.LoginStatus = "invited"
	response.OK(w, "Perwakilan keluarga diganti. Akses perwakilan lama sudah dicabut; kirim link undangan ke perwakilan baru.",
		clientWithLinkResponse{Client: resp, AccessLink: toAccessLinkResponse(link)})
}

func writeAppError(w http.ResponseWriter, err error) {
	status := apperror.HTTPStatus(err)
	if appErr, ok := apperror.As(err); ok {
		if appErr.Kind == apperror.KindValidation {
			response.Error(w, status, appErr.Message, appErr.Fields)
			return
		}
		response.Error(w, status, appErr.Message, appErr.Details())
		return
	}
	response.Error(w, status, "Terjadi kesalahan pada server", nil)
}
