package presentation

import (
	"context"
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

// planLister is the narrow slice of *application.PlanService this handler
// needs — declared here (not the concrete struct) so a stub can be injected
// for tests without a database (A34/task 6.7). *application.PlanService
// satisfies it implicitly; billing.module.go's wiring is unchanged.
type planLister interface {
	List(ctx context.Context) ([]domain.Plan, error)
	ListPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error)
	ListPublic(ctx context.Context) ([]domain.Plan, error)
	ListPublicPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error)
	Get(ctx context.Context, id int64) (*domain.Plan, error)
	Create(ctx context.Context, input application.PlanInput) (*domain.Plan, error)
	Update(ctx context.Context, id int64, input application.PlanInput) (*domain.Plan, error)
	ToggleActive(ctx context.Context, id int64) (*domain.Plan, error)
}

type PlanHandler struct {
	plans planLister
}

func NewPlanHandler(plans planLister) *PlanHandler {
	return &PlanHandler{plans: plans}
}

type planResponse struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	DurationMonths int      `json:"durationMonths"`
	Price          int64    `json:"price"`
	Features       []string `json:"features"`
	IsActive       bool     `json:"isActive"`
	// AppIDs is populated only when the caller is platform_admin (D12) —
	// [] otherwise. A non-empty value marks a plan as private to those
	// apps (D3); this is the field that lets Platform Console tell public
	// and private plans apart (T16).
	AppIDs []string `json:"appIds"`
}

func toPlanResponseValue(p domain.Plan) planResponse {
	appIDs := p.AppIDs
	if appIDs == nil {
		appIDs = make([]string, 0)
	}
	return planResponse{
		ID:             p.ID,
		Name:           p.Name,
		DurationMonths: p.DurationMonths,
		Price:          p.Price,
		Features:       p.Features,
		IsActive:       p.IsActive,
		AppIDs:         appIDs,
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

// list is NOT an access-denying guard — every principal may call it, same
// as today. The claim is only used to pick the data source (D3/A12):
// platform_admin sees every plan (including private ones, since it's the
// only principal that manages the mapping); anyone else — including no
// claim at all (6.3, the safe default) — sees only plans with no App
// mapping. This branching is applied to BOTH the all=true and paginated
// branches (6.2): fixing only one just moves the leak (T9) to the other door.
func (h *PlanHandler) list(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	isPlatformAdmin := ok && claims.PrincipalType == "platform_admin"

	if r.URL.Query().Get("all") == "true" {
		var (
			plans []domain.Plan
			err   error
		)
		if isPlatformAdmin {
			plans, err = h.plans.List(r.Context())
		} else {
			plans, err = h.plans.ListPublic(r.Context())
		}
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", toPlanResponses(plans))
		return
	}

	params := pagination.FromRequest(r)
	search := r.URL.Query().Get("search")
	var (
		plans []domain.Plan
		total int64
		err   error
	)
	if isPlatformAdmin {
		plans, total, err = h.plans.ListPaginated(r.Context(), params, search)
	} else {
		plans, total, err = h.plans.ListPublicPaginated(r.Context(), params, search)
	}
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
		response.Error(w, status, appErr.Message, appErr.Details())
		return
	}
	response.Error(w, status, "Terjadi kesalahan pada server", nil)
}
