package presentation

import (
	"context"
	"net/http"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/response"
)

// publicPlanLister is the narrow slice of *application.PlanService this
// handler needs — declared here so a stub can be injected for tests (A34).
type publicPlanLister interface {
	ListPublicActive(ctx context.Context) ([]domain.Plan, error)
}

// PublicPlanHandler serves GET /api/v1/public/plans: the active plans of
// ElProof's own catalog, for the public homepage's pricing section. Pre-auth,
// so it exposes only what a visitor needs to read a price — never the App
// mapping, the active flag or timestamps.
type PublicPlanHandler struct {
	plans publicPlanLister
}

func NewPublicPlanHandler(plans publicPlanLister) *PublicPlanHandler {
	return &PublicPlanHandler{plans: plans}
}

// publicPlanResponse mirrors the web contract
// (apps/web/src/modules/homepage/schemas/plan.schema.ts).
type publicPlanResponse struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	DurationMonths int      `json:"durationMonths"`
	Price          int64    `json:"price"`
	Features       []string `json:"features"`
}

// publicPlansCacheControl: prices change rarely and every homepage visit hits this
// endpoint, so browsers and any proxy may reuse a response for five minutes.
const publicPlansCacheControl = "public, max-age=300"

// List is GET /api/v1/public/plans (method guarded by httpx.Method at
// registration, same as the other /public/* routes).
func (h *PublicPlanHandler) List(w http.ResponseWriter, r *http.Request) {
	plans, err := h.plans.ListPublicActive(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	// make(..., 0, ...) so an empty catalog is `[]`, never `null`.
	result := make([]publicPlanResponse, 0, len(plans))
	for _, p := range plans {
		features := p.Features
		if features == nil {
			features = make([]string, 0)
		}
		result = append(result, publicPlanResponse{
			ID: p.ID, Name: p.Name, DurationMonths: p.DurationMonths,
			Price: p.Price, Features: features,
		})
	}
	w.Header().Set("Cache-Control", publicPlansCacheControl)
	response.OK(w, "ok", result)
}
