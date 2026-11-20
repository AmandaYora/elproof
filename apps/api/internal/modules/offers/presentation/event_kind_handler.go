package presentation

import (
	"encoding/json"
	"net/http"

	"elproof/internal/modules/offers/application"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/response"
)

type EventKindHandler struct {
	kinds *application.EventKindService
}

func NewEventKindHandler(kinds *application.EventKindService) *EventKindHandler {
	return &EventKindHandler{kinds: kinds}
}

type eventKindResponse struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	IsDefault      bool   `json:"isDefault"`
	IsActive       bool   `json:"isActive"`
	PackageCount   int    `json:"packageCount"`
	QuotationCount int    `json:"quotationCount"`
	UpdatedAt      string `json:"updatedAt"`
}

func (r eventKindResponse) ETag() string { return quoted(r.UpdatedAt) }

func toEventKindResponse(v application.EventKindView) eventKindResponse {
	return eventKindResponse{ID: v.ID, Name: v.Name, IsDefault: v.DefaultKey != "", IsActive: v.IsActive,
		PackageCount: v.PackageCount, QuotationCount: v.QuotationCount, UpdatedAt: etag.Format(v.UpdatedAt)}
}

func eventKindShape(v *application.EventKindView) any { return toEventKindResponse(*v) }

type eventKindBody struct {
	Name string `json:"name"`
}

// Collection is GET /event-kinds (any staff: the kinds are picked in
// packages, quotations, and projects) and POST (package.manage). The list is
// short and has no paging: a WO has a handful of kinds.
func (h *EventKindHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		kinds, err := h.kinds.List(r.Context(), tenantID)
		if err != nil {
			response.FromError(w, err)
			return
		}
		out := make([]eventKindResponse, 0, len(kinds))
		for _, k := range kinds {
			out = append(out, toEventKindResponse(k))
		}
		response.OK(w, "ok", out)
	case http.MethodPost:
		if !requirePermission(w, r, PermManage) {
			return
		}
		var body eventKindBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		k, err := h.kinds.Create(r.Context(), tenantID, body.Name)
		if err != nil {
			response.FromError(w, err)
			return
		}
		response.Created(w, "Jenis acara ditambahkan", toEventKindResponse(*k))
	default:
		methodNotAllowed(w)
	}
}

// Item is PATCH /event-kinds/{id} ({"isActive": bool} alone sets the state,
// ADR-0036; {"name"} renames with If-Match) and DELETE.
func (h *EventKindHandler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok || !requirePermission(w, r, PermManage) {
		return
	}
	id, segments, ok := itemID(w, r, "/api/v1/event-kinds/", application.MsgEventKindNotFound)
	if !ok {
		return
	}
	if len(segments) != 1 {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		sp, err := httpx.ReadStatePatch(r, "isActive")
		if err != nil {
			httpx.WriteStatePatchError(w, err)
			return
		}
		if sp.IsState {
			k, err := h.kinds.SetActive(r.Context(), tenantID, id, sp.Value)
			if err != nil {
				response.FromError(w, err)
				return
			}
			msg := "Jenis acara dinonaktifkan"
			if sp.Value {
				msg = "Jenis acara diaktifkan"
			}
			response.OK(w, msg, toEventKindResponse(*k))
			return
		}
		expected, ok := ifMatch(w, r, "jenis acara", func() error { _, err := h.kinds.Get(r.Context(), tenantID, id); return err })
		if !ok {
			return
		}
		var body eventKindBody
		if err := json.Unmarshal(sp.Body, &body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		k, err := h.kinds.Rename(r.Context(), tenantID, id, body.Name, expected)
		if err != nil {
			response.FromError(w, staleAs(err, eventKindShape))
			return
		}
		response.OK(w, "Jenis acara disimpan", toEventKindResponse(*k))
	case http.MethodDelete:
		if err := h.kinds.Delete(r.Context(), tenantID, id); err != nil {
			response.FromError(w, err)
			return
		}
		response.OK(w, "Jenis acara dihapus", nil)
	default:
		methodNotAllowed(w)
	}
}
