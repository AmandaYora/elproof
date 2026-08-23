package presentation

import (
	"context"
	"net/http"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/response"
)

// appSubscriptionLister is the narrow slice this handler needs — declared
// here so a stub can be injected for tests (A34).
type appSubscriptionLister interface {
	LatestPaidPerApp(ctx context.Context) ([]domain.AppSubscriptionSummary, error)
}

// AppSubscriptionHandler serves GET /api/v1/app-subscriptions (T13/D7) —
// the "who's paying for what, until when" view Platform Console's
// AppDetailModal tab Langganan reads (D13).
type AppSubscriptionHandler struct {
	charges appSubscriptionLister
}

func NewAppSubscriptionHandler(charges appSubscriptionLister) *AppSubscriptionHandler {
	return &AppSubscriptionHandler{charges: charges}
}

type appSubscriptionResponse struct {
	AppID             string `json:"appId"`
	PlanName          string `json:"planName"`
	SubscriptionUntil string `json:"subscriptionUntil"`
	Status            string `json:"status"`
}

// List is GET /api/v1/app-subscriptions, platform_admin only. This is a
// console-facing route, so httpx.Method (billing.module.go's own
// convention for internal routes) applies here — unlike the external
// routes in this package (A24).
func (h *AppSubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	summaries, err := h.charges.LatestPaidPerApp(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	// make(..., 0, ...) so an App with no paid history yet renders as `[]`
	// (D11 — the ledger is intentionally empty until jwswedding switches
	// over), never `null`.
	result := make([]appSubscriptionResponse, 0, len(summaries))
	for _, s := range summaries {
		result = append(result, appSubscriptionResponse{
			AppID: s.AppID, PlanName: s.PlanName,
			SubscriptionUntil: s.SubscriptionUntil.Format("2006-01-02"),
			Status:            string(s.Status),
		})
	}
	response.OK(w, "ok", result)
}
