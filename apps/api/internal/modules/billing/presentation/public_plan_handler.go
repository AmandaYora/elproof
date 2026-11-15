package presentation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

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

// publicPlansCacheControl: a browser or proxy may keep the response but must
// ask again before reusing it (EP-11). It used to be "max-age=300", so a plan
// the team had just deactivated stayed on the homepage for up to five minutes.
// The ETag is a hash of the body, so creating, editing or (de)activating a plan
// changes it at once — there is no in-process cache to bust — and an unchanged
// catalog costs only a 304.
const publicPlansCacheControl = "public, no-cache"

// publicEnvelope is the standard success envelope, encoded here (not through
// response.OK) because the ETag is computed from the exact bytes sent.
type publicEnvelope struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Data    []publicPlanResponse `json:"data"`
}

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

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(publicEnvelope{Success: true, Message: "ok", Data: result}); err != nil {
		response.Error(w, http.StatusInternalServerError, "Terjadi kesalahan pada server", nil)
		return
	}
	sum := sha256.Sum256(body.Bytes())
	tag := `"` + hex.EncodeToString(sum[:16]) + `"`

	w.Header().Set("Cache-Control", publicPlansCacheControl)
	w.Header().Set("ETag", tag)
	if etagMatches(r.Header.Get("If-None-Match"), tag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body.Bytes())
}

// etagMatches implements If-None-Match's weak comparison (RFC 9110 §13.1.2):
// a list of tags, each possibly W/-prefixed, or "*".
func etagMatches(header, tag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == "*" || candidate == tag {
			return true
		}
	}
	return false
}
