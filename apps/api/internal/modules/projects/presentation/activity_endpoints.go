package presentation

import (
	"net/http"

	"elproof/internal/shared/response"
)

func (h *Handler) listActivity(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.activity.ListByProject(r.Context(), projectID, 0)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]activityResponse, 0, len(list))
	for _, a := range list {
		result = append(result, toActivityResponse(a))
	}
	response.OK(w, "ok", result)
}
