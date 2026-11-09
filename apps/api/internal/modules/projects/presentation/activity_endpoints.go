package presentation

import (
	"net/http"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

// activityDefaultLimit is the page size of the activity feed (the shared
// pagination default of 10 is too short for a timeline).
const activityDefaultLimit = 20

// listActivity is GET /projects/{id}/activity?page=&limit=&entityType=&entityId=:
// the project's log, newest first, one page at a time (default 20, at most 100),
// optionally narrowed to one entity. Each entry carries the actor's name.
func (h *Handler) listActivity(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	params := pagination.FromRequest(r)
	if r.URL.Query().Get("limit") == "" {
		params.Limit = activityDefaultLimit
	}
	filter := domain.ActivityFilter{
		EntityType: r.URL.Query().Get("entityType"),
		EntityID:   r.URL.Query().Get("entityId"),
	}
	list, total, err := h.activity.ListPaged(r.Context(), projectID, filter, params.Page, params.Limit)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OKPaginated(w, "ok", h.activityResponses(r, claims.tenantID, list), pagination.BuildMeta(params, total))
}

// activityResponses maps entries to their response form with the actors' names
// from ONE staff lookup. An actor that cannot be resolved (never known, or the
// staff row is gone) has an empty name: no label is invented for it.
func (h *Handler) activityResponses(r *http.Request, tenantID int64, list []domain.ActivityLogEntry) []activityResponse {
	ids := make([]int64, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ActorStaffID)
	}
	names := h.projects.StaffNames(r.Context(), tenantID, ids)
	out := make([]activityResponse, 0, len(list))
	for _, a := range list {
		resp := toActivityResponse(a)
		resp.ActorName = names[a.ActorStaffID]
		out = append(out, resp)
	}
	return out
}
