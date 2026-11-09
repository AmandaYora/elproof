// Package presentation serves the workspace's roles API (Pengaturan → Peran
// & Akses) and the signed-in person's own grants.
package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"elproof/internal/modules/access/application"
	"elproof/internal/modules/access/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

type Handler struct {
	roles *application.RoleService
}

func NewHandler(roles *application.RoleService) *Handler {
	return &Handler{roles: roles}
}

type roleResponse struct {
	ID           int64    `json:"id"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	TemplateKey  string   `json:"templateKey,omitempty"`
	IsSystem     bool     `json:"isSystem"`
	ProjectScope string   `json:"projectScope"`
	Permissions  []string `json:"permissions"`
	MemberCount  int      `json:"memberCount"`
}

func toRoleResponse(r *domain.Role, members int) roleResponse {
	return roleResponse{
		ID: r.ID, Kind: string(r.Kind), Name: r.Name, Description: r.Description, TemplateKey: r.TemplateKey,
		IsSystem: r.IsSystem, ProjectScope: string(r.Scope), Permissions: r.Permissions, MemberCount: members,
	}
}

type permissionResponse struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"`
	Group       string `json:"group"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type grantsResponse struct {
	RoleID       int64    `json:"roleId"`
	RoleName     string   `json:"roleName"`
	IsOwner      bool     `json:"isOwner"`
	ProjectScope string   `json:"projectScope"`
	Permissions  []string `json:"permissions"`
}

func writeAppError(w http.ResponseWriter, err error) {
	response.FromError(w, err)
}

// staffTenant returns the caller's workspace when the caller is staff.
func staffTenant(w http.ResponseWriter, r *http.Request) (int64, *middleware.Grants, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" {
		response.Error(w, http.StatusForbidden, "Hanya staf WO yang dapat mengakses data ini", nil)
		return 0, nil, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, nil, false
	}
	return tenantID, middleware.GrantsFrom(r.Context()), true
}

func forbidden(w http.ResponseWriter) {
	response.Error(w, http.StatusForbidden, "Peran Anda tidak memiliki izin untuk aksi ini", map[string]string{"code": "forbidden_permission"})
}

// Me is GET /access/me: the caller's own role and permissions (staff and
// clients) — the web app shows only what these allow.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	g := middleware.GrantsFrom(r.Context())
	if g == nil {
		response.Error(w, http.StatusForbidden, "Akun ini tidak memiliki peran workspace", nil)
		return
	}
	perms := make([]string, 0, len(g.Permissions))
	for _, p := range domain.Catalog { // catalog order
		if g.Permissions[p.Key] {
			perms = append(perms, p.Key)
		}
	}
	scope := string(domain.ScopeAssigned)
	if g.AllProjects {
		scope = string(domain.ScopeAll)
	}
	response.OK(w, "ok", grantsResponse{RoleID: g.RoleID, RoleName: g.RoleName, IsOwner: g.Owner, ProjectScope: scope, Permissions: perms})
}

// Catalog is GET /access/catalog: every permission with its plain-language
// description, grouped for the Peran & Akses matrix.
func (h *Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := staffTenant(w, r); !ok {
		return
	}
	out := make([]permissionResponse, 0, len(domain.Catalog))
	for _, p := range domain.Catalog {
		out = append(out, permissionResponse{Key: p.Key, Kind: string(p.Kind), Group: p.Group, Label: p.Label, Description: p.Description})
	}
	response.OK(w, "ok", out)
}

type roleBody struct {
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	ProjectScope string   `json:"projectScope"`
	Permissions  []string `json:"permissions"`
}

func (b roleBody) input() application.RoleInput {
	return application.RoleInput{
		Kind: domain.Kind(b.Kind), Name: b.Name, Description: b.Description,
		Scope: domain.Scope(b.ProjectScope), Permissions: b.Permissions,
	}
}

// canReadRoles: the role lists feed the Peran & Akses page and the role
// pickers of the team and client forms.
func canReadRoles(g *middleware.Grants) bool {
	return g.Can("role.manage") || g.Can("team.manage") || g.Can("client.manage")
}

func (h *Handler) withCounts(r *http.Request, tenantID int64, roles []*domain.Role) []roleResponse {
	out := make([]roleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, toRoleResponse(role, h.roles.MemberCount(r.Context(), tenantID, role.ID)))
	}
	return out
}

// Roles is /access/roles: GET list (?kind=staff|client), POST create.
func (h *Handler) Roles(w http.ResponseWriter, r *http.Request) {
	tenantID, g, ok := staffTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !canReadRoles(g) {
			forbidden(w)
			return
		}
		roles, err := h.roles.List(r.Context(), tenantID, domain.Kind(r.URL.Query().Get("kind")))
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", h.withCounts(r, tenantID, roles))
	case http.MethodPost:
		if !g.Can("role.manage") {
			forbidden(w)
			return
		}
		var body roleBody
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		role, err := h.roles.Create(r.Context(), tenantID, body.input(), g)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.Created(w, "Peran dibuat", toRoleResponse(role, 0))
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan", nil)
	}
}

// ownRole refuses a non-Owner changing the role they hold themselves: with
// role.manage they could otherwise grant themselves anything (ADR-0026).
func ownRole(w http.ResponseWriter, g *middleware.Grants, id int64) bool {
	if g == nil || g.Owner || g.RoleID != id {
		return false
	}
	response.Error(w, http.StatusForbidden, "Anda tidak dapat mengubah peran yang Anda pakai sendiri. Minta Owner untuk mengubahnya.",
		map[string]string{"code": "forbidden_permission"})
	return true
}

// Role is /access/roles/{id}: GET, PATCH (role.manage), DELETE (role.manage).
func (h *Handler) Role(w http.ResponseWriter, r *http.Request) {
	tenantID, g, ok := staffTenant(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/access/roles/"), "/"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Peran tidak ditemukan", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !canReadRoles(g) {
			forbidden(w)
			return
		}
		role, err := h.roles.Get(r.Context(), tenantID, id)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", toRoleResponse(role, h.roles.MemberCount(r.Context(), tenantID, id)))
	case http.MethodPatch:
		if !g.Can("role.manage") {
			forbidden(w)
			return
		}
		if ownRole(w, g, id) {
			return
		}
		var body roleBody
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		role, err := h.roles.Update(r.Context(), tenantID, id, body.input(), g)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Peran disimpan", toRoleResponse(role, h.roles.MemberCount(r.Context(), tenantID, id)))
	case http.MethodDelete:
		if !g.Can("role.manage") {
			forbidden(w)
			return
		}
		if ownRole(w, g, id) {
			return
		}
		if err := h.roles.Delete(r.Context(), tenantID, id, g); err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Peran dihapus", nil)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan", nil)
	}
}
