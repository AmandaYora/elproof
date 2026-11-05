// Package presentation exposes the audit log to the Platform Console.
package presentation

import (
	"net/http"

	"elproof/internal/modules/audit/application"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

type entryResponse struct {
	ID         int64  `json:"id"`
	OccurredAt string `json:"occurredAt"`
	ActorType  string `json:"actorType"`
	ActorID    string `json:"actorId"`
	ActorRole  string `json:"actorRole"`
	Action     string `json:"action"`
	Path       string `json:"path"`
	StatusCode int    `json:"statusCode"`
	Succeeded  bool   `json:"succeeded"`
	IP         string `json:"ip"`
}

// List is GET /api/v1/audit-log — Super Admin only (who did what is itself
// sensitive). Optional ?actorId= filter.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "platform_admin" || claims.Role != "Super Admin" {
		response.Error(w, http.StatusForbidden, "Hanya Super Admin yang dapat melihat audit log", nil)
		return
	}
	params := pagination.FromRequest(r)
	entries, total, err := h.service.List(r.Context(), params, r.URL.Query().Get("actorId"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat audit log", nil)
		return
	}
	out := make([]entryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, entryResponse{
			ID: e.ID, OccurredAt: e.OccurredAt.Format("2006-01-02T15:04:05Z07:00"), ActorType: e.ActorType,
			ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, Path: e.Path,
			StatusCode: e.StatusCode, Succeeded: e.Succeeded(), IP: e.IP,
		})
	}
	response.OKPaginated(w, "ok", out, pagination.BuildMeta(params, total))
}
