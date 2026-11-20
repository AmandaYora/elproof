package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

// Events of a project ("Acara & rincian", portal Jadwal):
//
//	GET    /projects/{id}/events                     list (staff; client with portal.timeline.view)
//	POST   /projects/{id}/events                     add an event
//	PUT    /projects/{id}/events/{eid}               replace an event and its sessions (If-Match)
//	DELETE /projects/{id}/events/{eid}               (a project keeps at least one)
//	POST   /projects/{id}/events/{eid}/set-main      make it the main event (its date is Hari H)
//	POST   /projects/{id}/events/{eid}/move          {"direction":"up"|"down"}
//	POST   /projects/{id}/events/{eid}/duplicate
//
// Choosing a venue for an event adds that venue to Vendor & venue in the same
// transaction; the capacity rule is checked on every write (a numbered message
// on the session or the venue that does not fit).
func (h *Handler) routeEvents(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, rest []string) {
	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		h.listEvents(w, r, claims, projectID)
	case len(rest) == 0 && r.Method == http.MethodPost:
		h.createEvent(w, r, claims, projectID)
	case len(rest) == 1 && r.Method == http.MethodPut:
		h.updateEvent(w, r, claims, projectID, rest[0])
	case len(rest) == 1 && r.Method == http.MethodDelete:
		h.deleteEvent(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "set-main" && r.Method == http.MethodPost:
		h.setMainEvent(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "move" && r.Method == http.MethodPost:
		h.moveEvent(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "duplicate" && r.Method == http.MethodPost:
		h.duplicateEvent(w, r, claims, projectID, rest[0])
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

type eventSessionBody struct {
	Label     string  `json:"label"`
	StartTime *string `json:"startTime"`
	EndTime   *string `json:"endTime"`
	Guests    int     `json:"guests"`
}

type eventBody struct {
	EventKindID int64              `json:"eventKindId"`
	EventDate   string             `json:"eventDate"`
	StartTime   *string            `json:"startTime"`
	EndTime     *string            `json:"endTime"`
	VenueID     *int64             `json:"venueId"`
	Layout      string             `json:"layout"`
	Sessions    []eventSessionBody `json:"sessions"`
}

// toEventInput reads one event body; a date that is not YYYY-MM-DD is added to
// fields under prefix+"eventDate" (the service reports everything else).
func toEventInput(b eventBody, prefix string, fields map[string][]string) application.EventInput {
	in := application.EventInput{
		EventKindID: b.EventKindID, StartTime: b.StartTime, EndTime: b.EndTime, VenueID: b.VenueID, Layout: domain.Layout(b.Layout),
	}
	if b.VenueID != nil && *b.VenueID == 0 {
		in.VenueID = nil
	}
	if d, err := parseDate(b.EventDate); err != nil {
		fields[prefix+"eventDate"] = append(fields[prefix+"eventDate"], "Gunakan format YYYY-MM-DD")
	} else {
		in.EventDate = d
	}
	for _, s := range b.Sessions {
		in.Sessions = append(in.Sessions, application.SessionInput{Label: s.Label, StartTime: s.StartTime, EndTime: s.EndTime, Guests: s.Guests})
	}
	return in
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	events, err := h.events.List(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	rows, err := h.eventRows(r.Context(), claims, projectID, events)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", rows)
}

// eventRows adds each event's venue: staff get the project supplier with the
// capacities copied when it was chosen; a client gets the venue's name,
// address and city from the directory.
func (h *Handler) eventRows(ctx context.Context, claims staffClaims, projectID int64, events []domain.ProjectEvent) ([]eventResponse, error) {
	suppliers, err := h.suppliers.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	venues := map[int64]domain.ProjectSupplier{}
	venueIDs := []int64{}
	for _, s := range suppliers {
		if s.Kind == domain.SupplierVenue {
			venues[s.ID] = s
			venueIDs = append(venueIDs, s.SupplierID)
		}
	}
	summaries := h.refs.VenueSummaries(ctx, claims.tenantID, venueIDs)
	out := make([]eventResponse, 0, len(events))
	for _, e := range events {
		resp := toEventResponse(e)
		if e.VenueSupplierID != nil {
			if v, ok := venues[*e.VenueSupplierID]; ok {
				sum := summaries[v.SupplierID]
				ev := &eventVenueResponse{Name: v.Name, Address: sum.Address, City: sum.City}
				if !claims.isClient {
					ev.SupplierID, ev.VenueID = v.ID, v.SupplierID
					ev.CapacityStanding, ev.CapacitySeated = v.CapacityStanding, v.CapacitySeated
				}
				resp.Venue = ev
			}
		}
		out = append(out, resp)
	}
	return out, nil
}

func (h *Handler) eventItem(ctx context.Context, claims staffClaims, projectID int64, e domain.ProjectEvent) (eventResponse, error) {
	rows, err := h.eventRows(ctx, claims, projectID, []domain.ProjectEvent{e})
	if err != nil {
		return eventResponse{}, err
	}
	return rows[0], nil
}

// eventMutated is the answer to an event write: the item, the project's
// progress and its event date (the main event's, which the write may have
// moved).
func (h *Handler) eventMutated(ctx context.Context, claims staffClaims, projectID int64, item any) withExtras {
	more := []extra{}
	if p, err := h.projects.Get(ctx, claims.tenantID, projectID); err == nil {
		more = append(more, extra{"projectEventDate", p.EventDate.Format(dateLayout)})
	}
	return h.mutated(ctx, claims.tenantID, projectID, item, more...)
}

// writeEventResult answers an event write with the event as listed.
func (h *Handler) writeEventResult(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, e *domain.ProjectEvent, created bool, msg string) {
	item, err := h.eventItem(r.Context(), claims, projectID, *e)
	if err != nil {
		writeAppError(w, err)
		return
	}
	body := h.eventMutated(r.Context(), claims, projectID, item)
	if created {
		response.Created(w, msg, body)
		return
	}
	response.OK(w, msg, body)
}

func decodeEventBody(w http.ResponseWriter, r *http.Request) (application.EventInput, bool) {
	var body eventBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return application.EventInput{}, false
	}
	fields := map[string][]string{}
	in := toEventInput(body, "", fields)
	if len(fields) > 0 {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", fields)
		return application.EventInput{}, false
	}
	return in, true
}

func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	in, ok := decodeEventBody(w, r)
	if !ok {
		return
	}
	e, err := h.events.Create(r.Context(), claims.tenantID, projectID, claims.staffID, in)
	if err != nil {
		writeAppError(w, err)
		return
	}
	h.writeEventResult(w, r, claims, projectID, e, true, "Acara berhasil ditambahkan")
}

// updateEvent is PUT /projects/{id}/events/{eid}: every field of the event and
// its sessions as sent. If-Match is required (428 without; 409 with the
// current event when it changed meanwhile).
func (h *Handler) updateEvent(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	expected, ok := requireIfMatch(w, r, func(ctx context.Context) error {
		_, err := h.events.Get(ctx, claims.tenantID, projectID, id)
		return err
	})
	if !ok {
		return
	}
	in, ok := decodeEventBody(w, r)
	if !ok {
		return
	}
	e, err := h.events.Update(r.Context(), claims.tenantID, projectID, id, claims.staffID, in, expected)
	if err != nil {
		writePatchError(w, err, func(cur any) (any, error) {
			return h.eventItem(r.Context(), claims, projectID, *cur.(*domain.ProjectEvent))
		})
		return
	}
	h.writeEventResult(w, r, claims, projectID, e, false, "Acara berhasil diperbarui")
}

func (h *Handler) deleteEvent(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	if err := h.events.Delete(r.Context(), claims.tenantID, projectID, id, claims.staffID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Acara dihapus", h.eventMutated(r.Context(), claims, projectID, map[string]int64{"id": id}))
}

func (h *Handler) setMainEvent(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	e, err := h.events.SetMain(r.Context(), claims.tenantID, projectID, id, claims.staffID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	h.writeEventResult(w, r, claims, projectID, e, false, fmt.Sprintf("%s menjadi acara utama", e.KindName))
}

// moveEvent answers with every event of the project in their new order.
func (h *Handler) moveEvent(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body struct {
		Direction string `json:"direction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	events, err := h.events.Move(r.Context(), claims.tenantID, projectID, id, claims.staffID, body.Direction)
	if err != nil {
		writeAppError(w, err)
		return
	}
	rows, err := h.eventRows(r.Context(), claims, projectID, events)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Urutan acara diperbarui", rows)
}

func (h *Handler) duplicateEvent(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	e, err := h.events.Duplicate(r.Context(), claims.tenantID, projectID, id, claims.staffID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	h.writeEventResult(w, r, claims, projectID, e, true, "Acara berhasil diduplikat")
}
