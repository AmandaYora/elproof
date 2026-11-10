package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	RoleID   int64  `json:"roleId"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	IsActive bool   `json:"isActive"`
	// LoginStatus: "invited" until the invitation was accepted, then
	// "active" (or "inactive").
	LoginStatus string `json:"loginStatus,omitempty"`
	// RoleName is the role's current name (what `role` shows too); the list
	// fills it so the sort key and the column share one name.
	RoleName string `json:"roleName,omitempty"`
}

type invitationResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

func toInvitationResponse(inv *application.Invitation) invitationResponse {
	return invitationResponse{Token: inv.Token, ExpiresAt: inv.ExpiresAt.Format(time.RFC3339)}
}

// withLoginStatus adds the invitation state (Pengguna list).
func (h *Handler) withLoginStatus(r *http.Request, m domain.StaffMember, roleNames map[int64]string) staffResponse {
	resp := toStaffResponse(m)
	resp.LoginStatus = h.staff.LoginStatus(r.Context(), m.ID)
	if name, ok := roleNames[m.RoleID]; ok {
		resp.Role = name // current name (roles can be renamed)
		resp.RoleName = name
	}
	return resp
}

func toStaffResponse(m domain.StaffMember) staffResponse {
	return staffResponse{
		ID: m.ID, Name: m.Name, Title: m.Title, Initials: m.Initials,
		Role: string(m.Role), RoleID: m.RoleID, Username: m.Username, Email: m.Email, Phone: m.Phone, IsActive: m.IsActive,
	}
}

// requireTeamManager gates the Pengguna (staff directory) surface to roles
// holding team.manage (Owner by default; a WO may grant it to others —
// ADR-0026). The Owner's own row stays editable only by the Owner
// (StaffService.Update's isSelf check).
func requireTeamManager(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || !middleware.GrantsFrom(r.Context()).Can("team.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak memiliki izin mengelola tim", map[string]string{"code": "forbidden_permission"})
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

// requireTenant is the plain "any authenticated staff" gate — used only by
// Summary below, since PIC-name resolution (project PIC pickers, milestone/
// vendor-engagement/issue/activity "assigned to" labels throughout the
// `projects` module) is needed by every role, not just Owner.
func requireTenant(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" {
		response.Error(w, http.StatusForbidden, "Hanya staff WO yang dapat mengakses endpoint ini", nil)
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

type staffSummaryResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// Summary backs every PIC picker/label across the `projects` module —
// deliberately excludes username/email/phone/isActive (Pengguna's
// management-only fields), reachable by any staff role regardless of the
// Owner-only gate on the rest of this module (see requireOwnerTenant).
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}
	members, err := h.staff.List(r.Context(), tenantID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]staffSummaryResponse, 0, len(members))
	for _, m := range members {
		result = append(result, staffSummaryResponse{ID: m.ID, Name: m.Name, Title: m.Title})
	}
	response.OK(w, "ok", result)
}

func (h *Handler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTeamManager(w, r)
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
			names := h.staff.RoleNames(r.Context(), tenantID)
			for _, m := range members {
				result = append(result, h.withLoginStatus(r, m, names))
			}
			response.OK(w, "ok", result)
			return
		}
		params := pagination.FromRequest(r)
		filter, err := staffListFilter(r)
		if err != nil {
			writeAppError(w, err)
			return
		}
		columns, defaultSort := h.staff.StaffSortSpec()
		sort, err := pagination.SortFromRequest(r, columns, defaultSort)
		if err != nil {
			writeAppError(w, err)
			return
		}
		members, total, err := h.staff.ListPaginated(r.Context(), tenantID, params, filter, sort)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]staffResponse, 0, len(members))
		names := h.staff.RoleNames(r.Context(), tenantID)
		statuses := h.staff.LoginStatuses(r.Context(), tenantID, members)
		for _, m := range members {
			resp := toStaffResponse(m)
			resp.LoginStatus = statuses[m.ID]
			if name, ok := names[m.RoleID]; ok {
				resp.Role, resp.RoleName = name, name
			}
			result = append(result, resp)
		}
		response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total).WithSort(sort))
	case http.MethodPost:
		h.create(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

// staffListFilter reads q/search, roleId (or the older role), isActive; a bad
// value is a 422 under its own key.
func staffListFilter(r *http.Request) (domain.StaffListFilter, error) {
	q := r.URL.Query()
	f := domain.StaffListFilter{Search: pagination.SearchFromRequest(r)}
	bad := map[string][]string{}
	roleKey, raw := "roleId", q.Get("roleId")
	if raw == "" {
		roleKey, raw = "role", q.Get("role") // the filter's first name
	}
	if raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			bad[roleKey] = []string{"Peran tidak valid"}
		} else {
			f.RoleID = id
		}
	}
	switch q.Get("isActive") {
	case "":
	case "true":
		v := true
		f.IsActive = &v
	case "false":
		v := false
		f.IsActive = &v
	default:
		bad["isActive"] = []string{"isActive harus true atau false"}
	}
	if len(bad) > 0 {
		return f, apperror.Validation("Filter tim tidak valid", bad)
	}
	return f, nil
}

type staffInputBody struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	RoleID int64  `json:"roleId"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

type createStaffResponse struct {
	Staff      staffResponse      `json:"staff"`
	Invitation invitationResponse `json:"invitation"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body staffInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	member, inv, err := h.staff.Create(r.Context(), tenantID, application.StaffInput{
		Name: body.Name, Title: body.Title, RoleID: body.RoleID, Email: body.Email, Phone: body.Phone,
	}, middleware.GrantsFrom(r.Context()))
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp := toStaffResponse(*member)
	resp.LoginStatus = "invited"
	response.Created(w, "Undangan dibuat. Kirim link-nya ke orang tersebut.", createStaffResponse{Staff: resp, Invitation: toInvitationResponse(inv)})
}

func (h *Handler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTeamManager(w, r)
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

	// Changing someone whose role allows more than the caller's is the
	// Owner's call (ADR-0026).
	if r.Method != http.MethodGet {
		if err := h.staff.CheckManageable(r.Context(), tenantID, id, middleware.GrantsFrom(r.Context())); err != nil {
			writeAppError(w, err)
			return
		}
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.update(w, r, tenantID, id)
	case len(segments) == 2 && segments[1] == "invite" && r.Method == http.MethodPost:
		inv, err := h.staff.Reinvite(r.Context(), tenantID, id)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Link undangan baru dibuat", toInvitationResponse(inv))
	case len(segments) == 2 && segments[1] == "delete-impact" && r.Method == http.MethodGet:
		h.deleteImpact(w, r, tenantID, id)
	case len(segments) == 1 && r.Method == http.MethodDelete:
		if err := h.staff.Delete(r.Context(), tenantID, id); err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Pengguna berhasil dihapus", nil)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

type staffDeleteImpactResponse struct {
	AffectedProjects []staffDeleteImpactProjectRef `json:"affectedProjects"`
}

type staffDeleteImpactProjectRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// deleteImpact backs the hard-delete confirmation dialog (PLAN.md) -- names
// exactly which projects still have this staff member as a live PIC
// assignment.
func (h *Handler) deleteImpact(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	if _, err := h.staff.Get(r.Context(), tenantID, id); err != nil {
		writeAppError(w, err)
		return
	}
	refs, err := h.staff.ListAffectedProjects(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]staffDeleteImpactProjectRef, 0, len(refs))
	for _, ref := range refs {
		result = append(result, staffDeleteImpactProjectRef{ID: strconv.FormatInt(ref.ID, 10), Name: ref.Name})
	}
	response.OK(w, "ok", staffDeleteImpactResponse{AffectedProjects: result})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	// {"isActive": true|false} on its own is the explicit active state (ADR-0036);
	// any other body is the ordinary edit.
	sp, err := httpx.ReadStatePatch(r, "isActive")
	if err != nil {
		httpx.WriteStatePatchError(w, err)
		return
	}
	if sp.IsState {
		h.setActive(w, r, tenantID, id, sp.Value)
		return
	}
	var body staffInputBody
	if err := json.Unmarshal(sp.Body, &body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	claims, _ := middleware.FromContext(r.Context())
	isSelf := claims != nil && claims.PrincipalID == strconv.FormatInt(id, 10)
	member, err := h.staff.Update(r.Context(), tenantID, id, isSelf, application.StaffInput{
		Name: body.Name, Title: body.Title, RoleID: body.RoleID, Email: body.Email, Phone: body.Phone,
	}, middleware.GrantsFrom(r.Context()))
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pengguna berhasil diperbarui", toStaffResponse(*member))
}

// setActive sets a staff member's active state to exactly `want` (PATCH
// {"isActive": ...}, ADR-0036), with the effects of the transition (a deactivated
// member loses their sessions and cannot sign in; the Owner cannot be deactivated;
// activating counts against the trial quota). Setting the state the member already
// has is a 200 with the member as they are and no write.
func (h *Handler) setActive(w http.ResponseWriter, r *http.Request, tenantID, id int64, want bool) {
	current, err := h.staff.Get(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if want == current.IsActive {
		response.OK(w, "Status pengguna diperbarui", toStaffResponse(*current))
		return
	}
	member, err := h.staff.SetActive(r.Context(), tenantID, id, want)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status pengguna diperbarui", toStaffResponse(*member))
}

func writeAppError(w http.ResponseWriter, err error) {
	response.FromError(w, err)
}
