package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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
	// AccessRoleID / AccessRoleName: the portal role (what they see).
	AccessRoleID   int64  `json:"accessRoleId"`
	AccessRoleName string `json:"accessRoleName,omitempty"`
	// ProjectName / ProjectEventDate name the client's project; only the
	// paginated list (GET /clients?page=…) fills them, one batch lookup per page.
	ProjectName      string `json:"projectName,omitempty"`
	ProjectEventDate string `json:"projectEventDate,omitempty"`
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
	if names := h.clients.RoleNames(r.Context(), c.TenantID); names != nil {
		resp.AccessRoleName = names[c.AccessRoleID]
	}
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
		AccessRoleID: c.AccessRoleID,
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

// requireTenantWideClients gates the tenant-wide client list (and global
// search): client.manage with every-project scope — a person limited to
// their own projects has no single project to scope that list to.
func requireTenantWideClients(w http.ResponseWriter, r *http.Request) bool {
	g := middleware.GrantsFrom(r.Context())
	if !g.Can("client.manage") || !g.AllProjects {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat melihat seluruh client", map[string]string{"code": "forbidden_permission"})
		return false
	}
	return true
}

// requireClientManager is the gate every Client write needs: client.manage,
// and — for a role scoped to its own projects — being PIC or PIC Sales of
// the client's project (ADR-0026).
func (h *Handler) requireClientManager(w http.ResponseWriter, r *http.Request, tenantID, projectID int64) bool {
	g := middleware.GrantsFrom(r.Context())
	if !g.Can("client.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak memiliki izin mengelola client", map[string]string{"code": "forbidden_permission"})
		return false
	}
	if g.AllProjects {
		return true
	}
	claims, _ := middleware.FromContext(r.Context())
	staffID, err := strconv.ParseInt(claims.PrincipalID, 10, 64)
	if err == nil && h.clients.VerifyAssignedProject(r.Context(), tenantID, projectID, staffID) == nil {
		return true
	}
	response.Error(w, http.StatusForbidden, "Anda tidak memiliki akses ke project ini", nil)
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

// pagedListParams are the parameters that turn `GET /clients?projectId=` from the
// project tab's plain array into the paginated list filtered to that project.
var pagedListParams = []string{"page", "limit", "sort", "q", "role", "loginStatus"}

func wantsPagedList(r *http.Request) bool {
	q := r.URL.Query()
	for _, key := range pagedListParams {
		if _, ok := q[key]; ok {
			return true
		}
	}
	return false
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, tenantID int64) {
	projectIDRaw := r.URL.Query().Get("projectId")

	if projectIDRaw != "" && !wantsPagedList(r) {
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
		if !middleware.GrantsFrom(r.Context()).AllProjects {
			if err := h.clients.VerifyAssignedProject(r.Context(), tenantID, projectID, staffID); err != nil {
				writeAppError(w, err)
				return
			}
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

	if projectIDRaw != "" {
		// The paginated list narrowed to one project: the same access rule as the
		// project tab's array (a role limited to its own projects must be PIC or
		// PIC Sales of it); nobody gets another workspace's clients (every query
		// carries the tenant).
		projectID, err := strconv.ParseInt(projectIDRaw, 10, 64)
		if err != nil || projectID < 1 {
			writeAppError(w, apperror.Validation("Filter client tidak valid", map[string][]string{"projectId": {"Project tidak valid"}}))
			return
		}
		if !middleware.GrantsFrom(r.Context()).AllProjects {
			claims, _ := middleware.FromContext(r.Context())
			staffID, _ := strconv.ParseInt(claims.PrincipalID, 10, 64)
			if err := h.clients.VerifyAssignedProject(r.Context(), tenantID, projectID, staffID); err != nil {
				writeAppError(w, err)
				return
			}
		}
		h.listPage(w, r, tenantID, &projectID)
		return
	}

	// No projectId filter — list every client for the tenant. `all=true` is
	// used by WO Console's global search, which matches names across all
	// projects in one shot; otherwise this is the tenant-wide Client list
	// page. Both are Owner/Admin only (a Wedding Planner has no legitimate
	// need to see clients across projects that aren't theirs).
	if !requireTenantWideClients(w, r) {
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
	h.listPage(w, r, tenantID, nil)
}

// listPage answers the paginated, filterable, sortable client list:
// ?page&limit&sort&q (or search)&role&loginStatus, plus projectId when the caller
// narrowed it (and has been checked).
func (h *Handler) listPage(w http.ResponseWriter, r *http.Request, tenantID int64, projectID *int64) {
	params := pagination.FromRequest(r)
	filter, err := clientListFilter(r, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	columns, defaultSort := h.clients.ClientSortSpec()
	sort, err := pagination.SortFromRequest(r, columns, defaultSort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	page, err := h.clients.ListPage(r.Context(), tenantID, params, filter, sort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	names := h.clients.RoleNames(r.Context(), tenantID)
	result := make([]clientResponse, 0, len(page.Rows))
	for _, row := range page.Rows {
		resp := toClientResponse(row.Client)
		resp.LoginStatus = row.LoginStatus
		if names != nil {
			resp.AccessRoleName = names[row.Client.AccessRoleID]
		}
		resp.ProjectName = row.ProjectName
		if row.ProjectEventDate != nil {
			resp.ProjectEventDate = row.ProjectEventDate.Format("2006-01-02")
		}
		result = append(result, resp)
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, page.Total).WithSort(sort))
}

// clientListFilter reads q/search, role and loginStatus; an unknown role or
// loginStatus is a 422 under its own key.
func clientListFilter(r *http.Request, projectID *int64) (domain.ListFilter, error) {
	q := r.URL.Query()
	f := domain.ListFilter{Search: pagination.SearchFromRequest(r), Role: q.Get("role"), LoginStatus: q.Get("loginStatus"), ProjectID: projectID}
	bad := map[string][]string{}
	if f.Role != "" && !domain.IsClientRole(f.Role) {
		bad["role"] = []string{"Hubungan harus salah satu dari: Bride, Groom, Family Representative"}
	}
	if f.LoginStatus != "" && !domain.IsLoginStatus(f.LoginStatus) {
		bad["loginStatus"] = []string{"Status akses harus salah satu dari: " + strings.Join(domain.LoginStatuses, ", ")}
	}
	if len(bad) > 0 {
		return f, apperror.Validation("Filter client tidak valid", bad)
	}
	return f, nil
}

type createClientBody struct {
	ProjectID    int64  `json:"projectId"`
	Role         string `json:"role"`
	RelationNote string `json:"relationNote"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	AccessRoleID int64  `json:"accessRoleId"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body createClientBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	if !h.requireClientManager(w, r, tenantID, body.ProjectID) {
		return
	}
	c, link, err := h.clients.Create(r.Context(), tenantID, application.CreateClientInput{
		ProjectID: body.ProjectID, Role: domain.ClientRole(body.Role), RelationNote: body.RelationNote,
		Name: body.Name, Phone: body.Phone, Username: body.Username, Email: body.Email, AccessRoleID: body.AccessRoleID,
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
	// A deleted client is only reachable through restore: look it up (for the
	// permission check on its project) instead of the live row.
	restoring := len(segments) == 2 && segments[1] == "restore" && r.Method == http.MethodPost
	var c *domain.Client
	if restoring {
		c, err = h.clients.GetDeleted(r.Context(), tenantID, id)
	} else {
		c, err = h.clients.Get(r.Context(), tenantID, id)
	}
	if err != nil {
		writeAppError(w, err)
		return
	}
	if !h.requireClientManager(w, r, tenantID, c.ProjectID) {
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.updateContact(w, r, tenantID, id)
	case len(segments) == 1 && r.Method == http.MethodDelete:
		h.delete(w, r, tenantID, id)
	case restoring:
		h.restore(w, r, tenantID, id)
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
	// AccessRoleID, when set, changes the client's portal role.
	AccessRoleID int64 `json:"accessRoleId"`
}

func (h *Handler) updateContact(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body contactBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	if body.AccessRoleID > 0 {
		if _, err := h.clients.SetAccessRole(r.Context(), tenantID, id, body.AccessRoleID); err != nil {
			writeAppError(w, err)
			return
		}
	}
	c, err := h.clients.UpdateContact(r.Context(), tenantID, id, application.UpdateContactInput{Name: body.Name, Phone: body.Phone, Email: body.Email})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kontak client berhasil diperbarui", h.withLoginStatus(r, *c))
}

// deletedResponse is the body of a soft DELETE: what was deleted and until when
// it can be restored.
type deletedResponse struct {
	ID              int64  `json:"id"`
	RestorableUntil string `json:"restorableUntil"`
}

// delete soft-deletes the client (ADR-0034): it leaves every list at once, the
// login is switched off, and POST /clients/{id}/restore brings it back for 24 hours.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	until, err := h.clients.Delete(r.Context(), tenantID, id, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Client berhasil dihapus", deletedResponse{ID: id, RestorableUntil: until.Format(time.RFC3339)})
}

// restore is POST /clients/{id}/restore: the client as the list shows it. 404
// when it is not deleted or was purged; 422 when the project is gone or the
// workspace's trial quota has no room for it.
func (h *Handler) restore(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	c, err := h.clients.Restore(r.Context(), tenantID, id, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Client dipulihkan", h.withLoginStatus(r, *c))
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
	response.FromError(w, err)
}
