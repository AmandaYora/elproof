package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

func parseDate(s string) (time.Time, error) {
	return time.Parse(dateLayout, s)
}

// listProjects scopes the result to the caller's own PIC'd projects when
// their role is "Staff" (Wedding Planner) — nil (Owner/Admin) means every
// project in the tenant, unchanged from before this role existed.
func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request, claims staffClaims) {
	var picStaffID *int64
	if claims.role == "Staff" {
		picStaffID = &claims.staffID
	}
	if r.URL.Query().Get("all") == "true" {
		projects, err := h.projects.List(r.Context(), claims.tenantID, picStaffID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result, err := h.toProjectResponsesWithProgress(r, claims.tenantID, projects)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", result)
		return
	}
	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	showArchived := r.URL.Query().Get("archived") == "true"
	projects, total, err := h.projects.ListPaginated(r.Context(), claims.tenantID, picStaffID, params, search, status, showArchived)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := h.toProjectResponsesWithProgress(r, claims.tenantID, projects)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
}

// toProjectResponsesWithProgress sets `.Progress` on every returned row using
// one ComputeProgressBatch call over exactly this page's project ids —
// PLAN.md "Performance remediation": the list endpoint now returns
// progress-complete rows in one round trip (the same contract shape GET
// /projects/{id} already has), instead of the frontend enriching each row
// with its own GET /projects/{id} call.
func (h *Handler) toProjectResponsesWithProgress(r *http.Request, tenantID int64, projects []domain.Project) ([]projectResponse, error) {
	ids := make([]int64, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	progressByID, err := h.projects.ComputeProgressBatch(r.Context(), tenantID, ids, time.Now())
	if err != nil {
		return nil, err
	}
	result := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		resp := toProjectResponse(p)
		if progress, ok := progressByID[p.ID]; ok {
			progressResp := toProgressResponse(progress)
			resp.Progress = &progressResp
		}
		result = append(result, resp)
	}
	return result, nil
}

type projectInputBody struct {
	Name          string `json:"name"`
	BrideName     string `json:"brideName"`
	GroomName     string `json:"groomName"`
	EventDate     string `json:"eventDate"`
	Venue         string `json:"venue"`
	PrepStartDate string `json:"prepStartDate"`
	PackageName   string `json:"packageName"`
	ContractValue int64  `json:"contractValue"`
	Status        string `json:"status"`
	PICStaffID    int64  `json:"picStaffId"`
	Description   string `json:"description"`
	// VenueID is only meaningful on update -- see ProjectInput.VenueID's doc
	// comment for the omitted/0/positive tri-state. create/duplicate bodies
	// never send this key, so it decodes to nil and Create simply never
	// reads it (ADR-0016: attaching a venue is a separate, post-creation
	// action).
	VenueID *int64 `json:"venueId"`
	// VenueRentalPrice/VenueCharge ride along with VenueID whenever it's
	// present at all -- the per-project cost snapshot (PLAN.md "Financial
	// Calculation Correctness"). Meaningless (and ignored by
	// ProjectService.Update) when VenueID itself is nil or the detach
	// sentinel 0.
	VenueRentalPrice *int64 `json:"venueRentalPrice"`
	VenueCharge      *int64 `json:"venueCharge"`
}

func toProjectInput(body projectInputBody) (application.ProjectInput, error) {
	eventDate, err := parseDate(body.EventDate)
	if err != nil {
		return application.ProjectInput{}, err
	}
	prepStartDate, err := parseDate(body.PrepStartDate)
	if err != nil {
		return application.ProjectInput{}, err
	}
	return application.ProjectInput{
		Name: body.Name, BrideName: body.BrideName, GroomName: body.GroomName, EventDate: eventDate,
		Venue: body.Venue, PrepStartDate: prepStartDate, PackageName: body.PackageName,
		ContractValue: body.ContractValue, Status: domain.ProjectStatus(body.Status),
		PICStaffID: body.PICStaffID, Description: body.Description, VenueID: body.VenueID,
		VenueRentalPrice: body.VenueRentalPrice, VenueCharge: body.VenueCharge,
	}, nil
}

// createProject is Owner/Admin only (confirmed role rule) — a Wedding
// Planner only ever operates within a project already assigned to them by
// someone else, never creates one themselves.
func (h *Handler) createProject(w http.ResponseWriter, r *http.Request, claims staffClaims) {
	if claims.role == "Staff" {
		response.Error(w, http.StatusForbidden, "Hanya Owner atau Admin yang dapat membuat project baru", nil)
		return
	}
	var body projectInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	input, err := toProjectInput(body)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"eventDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	p, err := h.projects.Create(r.Context(), claims.tenantID, claims.staffID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Project berhasil dibuat", toProjectResponse(*p))
}

// me is the Client Portal's entry point (Fase 6) — a client principal has no
// other way to learn its own project id (login doesn't return one, and
// `identity` is deliberately profile-agnostic per ADR-0005), so this mirrors
// the `platform` module's `GET /tenants/me` self-service pattern from Fase 2
// (handled as a special segment inside Item, not a separate mux route).
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "client" {
		response.Error(w, http.StatusForbidden, "Hanya client yang dapat mengakses endpoint ini", nil)
		return
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}
	clientID, err := strconv.ParseInt(claims.PrincipalID, 10, 64)
	if err != nil {
		response.Error(w, http.StatusForbidden, "Identitas client tidak valid", nil)
		return
	}
	if h.clientAccess == nil {
		response.Error(w, http.StatusForbidden, "Akses client belum dikonfigurasi", nil)
		return
	}
	projectID, err := h.clientAccess.ProjectIDForClient(r.Context(), tenantID, clientID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	h.getProject(w, r, tenantID, projectID)
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request, tenantID, projectID int64) {
	p, err := h.projects.Get(r.Context(), tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	progress, err := h.projects.ComputeProgress(r.Context(), tenantID, projectID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp := toProjectResponse(*p)
	progressResp := toProgressResponse(*progress)
	resp.Progress = &progressResp
	response.OK(w, "ok", resp)
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body projectInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	input, err := toProjectInput(body)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"eventDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	p, err := h.projects.Update(r.Context(), claims.tenantID, projectID, claims.staffID, claims.role, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Project berhasil diperbarui", toProjectResponse(*p))
}

// getProjectVenue backs GET /projects/{id}/venue (ADR-0016) -- reachable by
// both staff and client principals (resolveProjectAccess in handler.go
// already scoped access before this is called), always the public-safe
// VenueSummary shape regardless of caller. Returns `data: null` (not an
// error) when no venue is attached, so both the WO Console tab and Client
// Portal's tab can render an empty state.
func (h *Handler) getProjectVenue(w http.ResponseWriter, r *http.Request, tenantID, projectID int64) {
	summary, err := h.projects.GetVenue(r.Context(), tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if summary == nil {
		response.OK(w, "ok", nil)
		return
	}
	response.OK(w, "ok", toVenueSummaryResponse(*summary))
}

func (h *Handler) cancelProject(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	p, err := h.projects.Cancel(r.Context(), claims.tenantID, projectID, claims.staffID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Project dibatalkan", toProjectResponse(*p))
}

// toggleArchiveProject is a single toggle (flips the current is_archived
// state), same convention as vendors'/staff's toggle-active — no body, no
// separate archive/unarchive endpoints. See ADR-0013.
func (h *Handler) toggleArchiveProject(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	current, err := h.projects.Get(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	p, err := h.projects.SetArchived(r.Context(), claims.tenantID, projectID, claims.staffID, !current.IsArchived)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status arsip project diperbarui", toProjectResponse(*p))
}

// deleteProject is the hard-delete endpoint (ADR-0013) — Owner-only, checked
// inline here since this module has no other role-gating to build on (same
// idiom as platform's tenant_handler.go Owner-only checks). The
// archived-or-cancelled precondition is enforced in ProjectService.Delete
// itself, not just here, so it can never be bypassed by a differently-shaped
// request.
func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	if claims.role != "Owner" {
		response.Error(w, http.StatusForbidden, "Hanya akun Owner yang dapat menghapus project secara permanen", nil)
		return
	}
	if err := h.projects.Delete(r.Context(), claims.tenantID, projectID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Project berhasil dihapus permanen", nil)
}

// duplicateProject clones the source project's Project Milestones and Vendor
// Engagements (with their Vendor Milestones) into a brand-new project — see
// ADR-0014. `body` is the new project's own fields (name/dates/etc.), same
// shape as create/update, since the frontend's duplicate form is the normal
// ProjectFormModal pre-filled from the source project — the caller decides
// what to change (e.g. name, event date) before submitting.
func (h *Handler) duplicateProject(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	// Duplicating produces a brand-new project, same restriction as
	// createProject — being the PIC of the source project (already verified
	// by resolveProjectAccess before this is ever reached) doesn't matter,
	// a Wedding Planner is never allowed to create the resulting copy.
	if claims.role == "Staff" {
		response.Error(w, http.StatusForbidden, "Hanya Owner atau Admin yang dapat menduplikasi project", nil)
		return
	}
	var body projectInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	input, err := toProjectInput(body)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"eventDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	p, err := h.projects.Duplicate(r.Context(), claims.tenantID, claims.staffID, projectID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Project berhasil diduplikasi", toProjectResponse(*p))
}

// --- Project milestones ---

func (h *Handler) listMilestones(w http.ResponseWriter, r *http.Request, tenantID, projectID int64) {
	milestones, err := h.projects.ListMilestones(r.Context(), tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]milestoneResponse, 0, len(milestones))
	for _, m := range milestones {
		result = append(result, toMilestoneResponse(m))
	}
	response.OK(w, "ok", result)
}

type milestoneInputBody struct {
	Name       string `json:"name"`
	TargetDate string `json:"targetDate"`
}

func (h *Handler) createMilestone(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body milestoneInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	targetDate, err := parseDate(body.TargetDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"targetDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	m, err := h.projects.CreateMilestone(r.Context(), claims.tenantID, projectID, claims.staffID, application.MilestoneInput{Name: body.Name, TargetDate: targetDate})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Timeline berhasil ditambahkan", toMilestoneResponse(*m))
}

type milestoneUpdateBody struct {
	Status        string `json:"status"`
	TargetDate    string `json:"targetDate"`
	CompletedDate string `json:"completedDate"`
}

func (h *Handler) updateMilestone(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, milestoneIDRaw string) {
	milestoneID, err := parseInt64(milestoneIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID timeline tidak valid", nil)
		return
	}
	var body milestoneUpdateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	targetDate, err := parseDate(body.TargetDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"targetDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	m, err := h.projects.UpdateMilestone(r.Context(), claims.tenantID, projectID, milestoneID, claims.staffID, application.MilestoneUpdateInput{
		Status: domain.MilestoneStatus(body.Status), TargetDate: targetDate, CompletedDate: parseOptionalDate(body.CompletedDate),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Timeline berhasil diperbarui", toMilestoneResponse(*m))
}

type milestoneReorderBody struct {
	OrderedIDs []int64 `json:"orderedIds"`
}

func (h *Handler) reorderMilestones(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body milestoneReorderBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	if err := h.projects.ReorderMilestones(r.Context(), claims.tenantID, projectID, claims.staffID, body.OrderedIDs); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Urutan timeline berhasil diperbarui", nil)
}
