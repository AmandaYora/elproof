package presentation

import (
	"context"
	"encoding/json"
	"net/http"

	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/response"
)

// appPlanMapper is the narrow slice of *application.PlanService this
// handler needs — declared here so a stub can be injected for tests (A34).
type appPlanMapper interface {
	GetAppPlanIDs(ctx context.Context, appID string) ([]int64, error)
	SetAppPlans(ctx context.Context, appID string, planIDs []int64) error
}

// AppPlanHandler is the platform_admin-only mapping editor behind
// GET|PUT /api/v1/app-plans (A8) — Platform Console's "Kelola Paket" tab
// inside AppDetailModal (D13) uses this, never SQL by hand.
type AppPlanHandler struct {
	plans appPlanMapper
	apps  paymentcontracts.AppRegistry
}

func NewAppPlanHandler(plans appPlanMapper, apps paymentcontracts.AppRegistry) *AppPlanHandler {
	return &AppPlanHandler{plans: plans, apps: apps}
}

type appPlanGetResponse struct {
	AppID   string  `json:"appId"`
	PlanIDs []int64 `json:"planIds"`
}

type appPlanPutBody struct {
	AppID   string  `json:"appId"`
	PlanIDs []int64 `json:"planIds"`
}

// Collection dispatches GET (read the mapping) and PUT (replace it) on
// /api/v1/app-plans.
func (h *AppPlanHandler) Collection(w http.ResponseWriter, r *http.Request) {
	// The platform_admin guard runs first, before either branch does
	// anything else — an unauthorized caller must never trigger a query,
	// not even AppExists (task 7.2).
	if !requirePlatformAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPut:
		h.put(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

func (h *AppPlanHandler) get(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("appId")
	if appID == "" {
		response.Error(w, http.StatusBadRequest, "appId wajib diisi", nil)
		return
	}
	planIDs, err := h.plans.GetAppPlanIDs(r.Context(), appID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if planIDs == nil {
		planIDs = make([]int64, 0)
	}
	response.OK(w, "ok", appPlanGetResponse{AppID: appID, PlanIDs: planIDs})
}

func (h *AppPlanHandler) put(w http.ResponseWriter, r *http.Request) {
	var body appPlanPutBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	if body.AppID == "" {
		response.Error(w, http.StatusBadRequest, "appId wajib diisi", nil)
		return
	}

	// Without an FK to payment_apps (A2), this is the only guard against a
	// typo'd app_id producing a dead mapping that looks like success (A11).
	// Three outcomes, same shape as requireActiveApp: err != nil means "we
	// don't know" (500), false means "that app_id doesn't exist" (422 — a
	// client mistake, not a server failure), true means proceed.
	exists, err := h.apps.AppExists(r.Context(), body.AppID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if !exists {
		writeAppError(w, apperror.Validation("appId tidak ditemukan", map[string][]string{"appId": {"appId tidak ditemukan"}}))
		return
	}

	if err := h.plans.SetAppPlans(r.Context(), body.AppID, body.PlanIDs); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pemetaan paket berhasil disimpan", nil)
}
