package presentation

import (
	"context"
	"net/http"

	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

// externalPlanLister is the narrow slice of *application.PlanService this
// handler needs — declared here (not *application.PlanService, a concrete
// struct) so a stub can be injected for tests without a database (A34).
type externalPlanLister interface {
	ListForApp(ctx context.Context, appID string) ([]domain.Plan, error)
}

// ExternalPlanHandler serves GET /api/v1/external/subscription-plans (D9,
// docs/plan/external-subscription-plans/PLAN.md) — the first file in
// `billing` to import `payment/contracts` (A5); AppRegistry is already an
// interface there, so it isn't wrapped again here.
type ExternalPlanHandler struct {
	plans externalPlanLister
	apps  paymentcontracts.AppRegistry
}

func NewExternalPlanHandler(plans externalPlanLister, apps paymentcontracts.AppRegistry) *ExternalPlanHandler {
	return &ExternalPlanHandler{plans: plans, apps: apps}
}

// externalPlanResponse is deliberately a separate struct from planResponse
// (plan_handler.go) — the external contract uses `active` (not `isActive`,
// T3) and always includes `features` (D4).
type externalPlanResponse struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	DurationMonths int      `json:"durationMonths"`
	Price          int64    `json:"price"`
	Active         bool     `json:"active"`
	Features       []string `json:"features"`
}

func toExternalPlanResponses(plans []domain.Plan) []externalPlanResponse {
	// make(..., 0, ...) so an App with no mapped plans gets `[]`, never
	// `null` (A4) — a valid state (R2 — Elkasir), not an error.
	result := make([]externalPlanResponse, 0, len(plans))
	for _, p := range plans {
		features := p.Features
		if features == nil {
			features = make([]string, 0)
		}
		result = append(result, externalPlanResponse{
			ID: p.ID, Name: p.Name, DurationMonths: p.DurationMonths,
			Price: p.Price, Active: p.IsActive, Features: features,
		})
	}
	return result
}

// List is GET /api/v1/external/subscription-plans. Not wrapped in
// httpx.Method (T17/A24) — the method check happens here so a 405 still
// carries {code:"bad_request"}, matching the vocabulary every other
// external endpoint uses.
func (h *ExternalPlanHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeExternalError(w, http.StatusMethodNotAllowed, "bad_request", "Metode HTTP tidak diizinkan untuk endpoint ini")
		return
	}

	appID, ok := requireActiveApp(w, r, h.apps)
	if !ok {
		return
	}

	plans, err := h.plans.ListForApp(r.Context(), appID)
	if err != nil {
		writeExternalAppError(w, err)
		return
	}
	response.OK(w, "ok", toExternalPlanResponses(plans))
}

// requireActiveApp is this module's own guard (A7 — importing another
// module's `presentation` package is forbidden by the module-boundary
// rule, so `payment`'s RequireActiveApp can't be reused directly). Mirrors
// its shape and error handling exactly: principal must be an `app`, then
// IsAppActive is checked fresh against the registry on every request.
// Shared by every external, App-token handler in this package (currently
// ExternalPlanHandler and ExternalSubscriptionHandler) — same pattern as
// requirePlatformAdmin below being shared across PlanHandler/AppPlanHandler.
// Returns ("", false) once it has already written a response.
func requireActiveApp(w http.ResponseWriter, r *http.Request, apps paymentcontracts.AppRegistry) (string, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "app" {
		writeExternalError(w, http.StatusForbidden, "forbidden", "Endpoint ini hanya untuk aplikasi eksternal terdaftar")
		return "", false
	}

	active, err := apps.IsAppActive(r.Context(), claims.PrincipalID)
	if err != nil {
		// A registry error means "we don't know", not "not allowed" —
		// distinct from the false branch below (meniru
		// payment/presentation/external_handler.go:36-39).
		writeExternalAppError(w, err)
		return "", false
	}
	if !active {
		writeExternalError(w, http.StatusForbidden, "forbidden", "Aplikasi telah dinonaktifkan")
		return "", false
	}
	return claims.PrincipalID, true
}

// externalErrorCode maps an AppError.Kind to the fixed vocabulary external
// consumers can branch on programmatically — identical vocabulary to
// payment/presentation/external_handler.go's own externalErrorCode (T3/A7);
// duplicated here (2 lines) rather than imported, since importing another
// module's `presentation` package is forbidden.
func externalErrorCode(kind apperror.Kind) string {
	switch kind {
	case apperror.KindValidation:
		return "bad_request"
	case apperror.KindUnauthorized:
		return "unauthorized"
	case apperror.KindForbidden:
		return "forbidden"
	case apperror.KindNotFound:
		return "not_found"
	case apperror.KindConflict:
		return "conflict"
	case apperror.KindRateLimited:
		return "rate_limited"
	default:
		return "internal"
	}
}

func writeExternalAppError(w http.ResponseWriter, err error) {
	status := apperror.HTTPStatus(err)
	if appErr, ok := apperror.As(err); ok {
		response.Error(w, status, appErr.Message, map[string]string{"code": externalErrorCode(appErr.Kind)})
		return
	}
	response.Error(w, status, "Terjadi kesalahan pada server", map[string]string{"code": "internal"})
}

func writeExternalError(w http.ResponseWriter, status int, code, message string) {
	response.Error(w, status, message, map[string]string{"code": code})
}
