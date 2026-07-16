package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type PlanHandler struct {
	plans *application.PlanService
}

func NewPlanHandler(plans *application.PlanService) *PlanHandler {
	return &PlanHandler{plans: plans}
}

type planResponse struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	DurationMonths int      `json:"durationMonths"`
	Price          int64    `json:"price"`
	Features       []string `json:"features"`
	IsActive       bool     `json:"isActive"`
}

func toPlanResponseValue(p domain.Plan) planResponse {
	return planResponse{
		ID:             p.ID,
		Name:           p.Name,
		DurationMonths: p.DurationMonths,
		Price:          p.Price,
		Features:       p.Features,
		IsActive:       p.IsActive,
	}
}

func toPlanResponses(plans []domain.Plan) []planResponse {
	result := make([]planResponse, 0, len(plans))
	for _, p := range plans {
		result = append(result, toPlanResponseValue(p))
	}
	return result
}

type planInputBody struct {
	Name           string   `json:"name"`
	DurationMonths int      `json:"durationMonths"`
	Price          int64    `json:"price"`
	Features       []string `json:"features"`
}

// Collection dispatches GET (list, any authenticated principal) and POST
// (create, platform_admin only) on /api/v1/plans.
func (h *PlanHandler) Collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

// Item dispatches on /api/v1/plans/{id} and /api/v1/plans/{id}/toggle-active.
func (h *PlanHandler) Item(w http.ResponseWriter, r *http.Request) {
	segments := httpx.Segments(r.URL.Path, "/api/v1/plans/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Paket tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID paket tidak valid", nil)
		return
	}

	if len(segments) == 2 && segments[1] == "toggle-active" && r.Method == http.MethodPost {
		h.toggleActive(w, r, id)
		return
	}
	if len(segments) == 1 && r.Method == http.MethodPatch {
		h.update(w, r, id)
		return
	}
	response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
}

func (h *PlanHandler) list(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("all") == "true" {
		plans, err := h.plans.List(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", toPlanResponses(plans))
		return
	}
	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	plans, total, err := h.plans.ListPaginated(r.Context(), params, search)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OKPaginated(w, "ok", toPlanResponses(plans), pagination.BuildMeta(params, total))
}

func (h *PlanHandler) create(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	var body planInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	plan, err := h.plans.Create(r.Context(), application.PlanInput{
		Name: body.Name, DurationMonths: body.DurationMonths, Price: body.Price, Features: body.Features,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Paket berhasil dibuat", toPlanResponseValue(*plan))
}

func (h *PlanHandler) update(w http.ResponseWriter, r *http.Request, id int64) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	var body planInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	plan, err := h.plans.Update(r.Context(), id, application.PlanInput{
		Name: body.Name, DurationMonths: body.DurationMonths, Price: body.Price, Features: body.Features,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Paket berhasil diperbarui", toPlanResponseValue(*plan))
}

func (h *PlanHandler) toggleActive(w http.ResponseWriter, r *http.Request, id int64) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	plan, err := h.plans.ToggleActive(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status paket diperbarui", toPlanResponseValue(*plan))
}

func requirePlatformAdmin(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "platform_admin" {
		response.Error(w, http.StatusForbidden, "Hanya admin platform yang dapat melakukan aksi ini", nil)
		return false
	}
	return true
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
