package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
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
	List(ctx context.Context, appID string) ([]domain.Plan, error)
	ListPaginated(ctx context.Context, params pagination.Params, search, appID string) ([]domain.Plan, int64, error)
	Get(ctx context.Context, id int64) (*domain.Plan, error)
	Create(ctx context.Context, input application.PlanInput) (*domain.Plan, error)
	Update(ctx context.Context, id int64, input application.PlanInput) (*domain.Plan, error)
	ToggleActive(ctx context.Context, id int64) (*domain.Plan, error)
}

type PlanHandler struct {
	plans planLister
	apps  paymentcontracts.AppRegistry
}

func NewPlanHandler(plans planLister, apps paymentcontracts.AppRegistry) *PlanHandler {
	return &PlanHandler{plans: plans, apps: apps}
}

type planResponse struct {
	ID int64 `json:"id"`
	// AppID is the one App that sells this plan (ADR-0040) — ElProof's own
	// internal App for the WO tenant catalog.
	AppID          string   `json:"appId"`
	Name           string   `json:"name"`
	DurationMonths int      `json:"durationMonths"`
	Price          int64    `json:"price"`
	Features       []string `json:"features"`
	IsActive       bool     `json:"isActive"`
}

func toPlanResponseValue(p domain.Plan) planResponse {
	features := p.Features
	if features == nil {
		features = make([]string, 0)
	}
	return planResponse{
		ID:             p.ID,
		AppID:          p.AppID,
		Name:           p.Name,
		DurationMonths: p.DurationMonths,
		Price:          p.Price,
		Features:       features,
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
	AppID          string   `json:"appId"`
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

// listAppID picks whose plans a list call returns. It is NOT an access-
// denying guard — every principal may list plans. platform_admin sees every
// App's plans, or one App's with `?appId=`; anyone else — including no claim
// at all, the safe default — always gets ElProof's own catalog, whatever
// `appId` they send. Used by BOTH the all=true and paginated branches: a leak
// through either door is still a leak.
func listAppID(r *http.Request) string {
	claims, ok := middleware.FromContext(r.Context())
	if ok && claims.PrincipalType == "platform_admin" {
		return r.URL.Query().Get("appId")
	}
	return application.TenantPlansApp
}

func (h *PlanHandler) list(w http.ResponseWriter, r *http.Request) {
	appID := listAppID(r)

	if r.URL.Query().Get("all") == "true" {
		plans, err := h.plans.List(r.Context(), appID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", toPlanResponses(plans))
		return
	}

	params := pagination.FromRequest(r)
	plans, total, err := h.plans.ListPaginated(r.Context(), params, r.URL.Query().Get("search"), appID)
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
	if !h.requireKnownApp(w, r, body.AppID) {
		return
	}
	plan, err := h.plans.Create(r.Context(), application.PlanInput{
		AppID: body.AppID, Name: body.Name, DurationMonths: body.DurationMonths, Price: body.Price, Features: body.Features,
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
	if !h.requireKnownApp(w, r, body.AppID) {
		return
	}
	plan, err := h.plans.Update(r.Context(), id, application.PlanInput{
		AppID: body.AppID, Name: body.Name, DurationMonths: body.DurationMonths, Price: body.Price, Features: body.Features,
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

// requireKnownApp answers 422 on appId unless it names an App of the
// registry: app_id is a primitive ID with no FK to payment_apps, so this is
// the only guard against a typo producing a plan nobody can ever see. An
// empty appId is left to the service (same 422). ElProof's own App is always
// known — it is bootstrapped at every start, and `api seed` empties the
// registry until then. A deactivated App may still own plans.
//
// Registry IDs are lower case ("app_<hex>", "platform-billing") and the
// registry lookup ignores case, so an ID in any other case is refused here:
// the stored app_id is always byte-identical to the registry's.
func (h *PlanHandler) requireKnownApp(w http.ResponseWriter, r *http.Request, appID string) bool {
	if appID == "" || appID == application.TenantPlansApp {
		return true
	}
	unknown := apperror.Validation("Aplikasi tidak ditemukan", map[string][]string{"appId": {"Pilih aplikasi yang terdaftar"}})
	if appID != strings.ToLower(appID) {
		writeAppError(w, unknown)
		return false
	}
	exists, err := h.apps.AppExists(r.Context(), appID)
	if err != nil {
		writeAppError(w, err)
		return false
	}
	if !exists {
		writeAppError(w, unknown)
		return false
	}
	return true
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
	response.FromError(w, err)
}
