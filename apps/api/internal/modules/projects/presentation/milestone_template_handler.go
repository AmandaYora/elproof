package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

// MilestoneTemplateHandler backs Pengaturan -> Timeline Default (PLAN.md) --
// a tenant's own configurable checklist seeded into every new project's
// Timeline tab. Registered directly in projects.module.go's RegisterRoutes,
// not funneled through Handler.Item's /projects/{id}/... dispatcher, since
// this data is tenant-level configuration, not scoped to one project.
type MilestoneTemplateHandler struct {
	templates *application.MilestoneTemplateService
}

func NewMilestoneTemplateHandler(templates *application.MilestoneTemplateService) *MilestoneTemplateHandler {
	return &MilestoneTemplateHandler{templates: templates}
}

type milestoneTemplateResponse struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	DaysBeforeEvent int    `json:"daysBeforeEvent"`
	SortOrder       int    `json:"sortOrder"`
}

func toMilestoneTemplateResponse(t domain.ProjectMilestoneTemplate) milestoneTemplateResponse {
	return milestoneTemplateResponse{
		ID: t.ID, Name: t.Name, DaysBeforeEvent: t.DaysBeforeEvent, SortOrder: t.SortOrder,
	}
}

// requireOwnerForTemplates gates every method on this handler -- unlike
// Kategori Vendor's read-open-to-all-staff GET, a milestone template has no
// dropdown/display use case elsewhere the way a vendor category name does,
// so there's no reason to relax the read side (PLAN.md §3.5).
func requireOwnerForTemplates(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return 0, false
	}
	if !claims.HasRole("Owner") {
		response.Error(w, http.StatusForbidden, "Hanya Owner yang dapat mengelola timeline default", nil)
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

type milestoneTemplateInputBody struct {
	Name            string `json:"name"`
	DaysBeforeEvent int    `json:"daysBeforeEvent"`
}

type milestoneTemplateReorderBody struct {
	OrderedIDs []int64 `json:"orderedIds"`
}

func (h *MilestoneTemplateHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireOwnerForTemplates(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		templates, err := h.templates.List(r.Context(), tenantID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]milestoneTemplateResponse, 0, len(templates))
		for _, t := range templates {
			result = append(result, toMilestoneTemplateResponse(t))
		}
		response.OK(w, "ok", result)
	case http.MethodPost:
		var body milestoneTemplateInputBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		t, err := h.templates.Create(r.Context(), tenantID, application.MilestoneTemplateInput{
			Name: body.Name, DaysBeforeEvent: body.DaysBeforeEvent,
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.Created(w, "Template timeline berhasil ditambahkan", toMilestoneTemplateResponse(*t))
	case http.MethodPatch:
		var body milestoneTemplateReorderBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		if err := h.templates.Reorder(r.Context(), tenantID, body.OrderedIDs); err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Urutan template timeline berhasil diperbarui", nil)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

func (h *MilestoneTemplateHandler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireOwnerForTemplates(w, r)
	if !ok {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/milestone-templates/")
	if len(segments) != 1 {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var body milestoneTemplateInputBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		t, err := h.templates.Update(r.Context(), tenantID, id, application.MilestoneTemplateInput{
			Name: body.Name, DaysBeforeEvent: body.DaysBeforeEvent,
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Template timeline berhasil diperbarui", toMilestoneTemplateResponse(*t))
	case http.MethodDelete:
		if err := h.templates.Delete(r.Context(), tenantID, id); err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Template timeline berhasil dihapus", nil)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}
