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

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request, tenantID int64) {
	if r.URL.Query().Get("all") == "true" {
		projects, err := h.projects.List(r.Context(), tenantID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]projectResponse, 0, len(projects))
		for _, p := range projects {
			result = append(result, toProjectResponse(p))
		}
		response.OK(w, "ok", result)
		return
	}
	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	projects, total, err := h.projects.ListPaginated(r.Context(), tenantID, params, search, status)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		result = append(result, toProjectResponse(p))
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
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
		PICStaffID: body.PICStaffID, Description: body.Description,
	}, nil
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request, claims staffClaims) {
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
	p, err := h.projects.Update(r.Context(), claims.tenantID, projectID, claims.staffID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Project berhasil diperbarui", toProjectResponse(*p))
}

func (h *Handler) cancelProject(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	p, err := h.projects.Cancel(r.Context(), claims.tenantID, projectID, claims.staffID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Project dibatalkan", toProjectResponse(*p))
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
	response.Created(w, "Milestone berhasil ditambahkan", toMilestoneResponse(*m))
}

type milestoneStatusBody struct {
	Status string `json:"status"`
}

func (h *Handler) updateMilestoneStatus(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, milestoneIDRaw string) {
	milestoneID, err := parseInt64(milestoneIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID milestone tidak valid", nil)
		return
	}
	var body milestoneStatusBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	m, err := h.projects.UpdateMilestoneStatus(r.Context(), claims.tenantID, projectID, milestoneID, claims.staffID, domain.MilestoneStatus(body.Status), time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status milestone diperbarui", toMilestoneResponse(*m))
}
