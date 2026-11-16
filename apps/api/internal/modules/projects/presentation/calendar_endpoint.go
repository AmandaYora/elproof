package presentation

import (
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/response"
)

// calendarItemResponse is one entry of GET /calendar.
type calendarItemResponse struct {
	Date        string `json:"date"`
	Kind        string `json:"kind"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	// Amount (the installment) and Remaining (what is still owed on it) are
	// present for payments only.
	Amount    *int64 `json:"amount,omitempty"`
	Remaining *int64 `json:"remaining,omitempty"`
}

type calendarResponse struct {
	From  string                 `json:"from"`
	To    string                 `json:"to"`
	Items []calendarItemResponse `json:"items"`
}

// SetCalendar gives the handler the cross-project calendar.
func (h *Handler) SetCalendar(s *application.CalendarService) { h.calendar = s }

// Calendar is GET /calendar?from=&to=[&picStaffId=&kinds=]: the wedding days,
// open schedule steps and client installments of the projects the caller
// reaches, by day. Open to every staff role and scoped like GET /projects: a
// role that does not reach every project sees only the projects the person
// holds. A client principal is refused (requireStaff).
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}
	q, kinds, err := calendarQueryFrom(r, claims)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items, err := h.calendar.List(r.Context(), q, kinds, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	out := calendarResponse{From: q.From.Format(dateLayout), To: q.To.Format(dateLayout), Items: make([]calendarItemResponse, 0, len(items))}
	for _, it := range items {
		out.Items = append(out.Items, calendarItemResponse{
			Date: it.Date.Format(dateLayout), Kind: string(it.Kind), ProjectID: it.ProjectID, ProjectName: it.ProjectName,
			Title: it.Title, Status: it.Status, Amount: it.Amount, Remaining: it.Remaining,
		})
	}
	response.OK(w, "ok", out)
}

// calendarQueryFrom reads and checks the query string; every problem is one 422
// naming each offending key. The role's scope comes from the claims, never
// from the request.
func calendarQueryFrom(r *http.Request, claims staffClaims) (application.CalendarQuery, domain.CalendarKinds, error) {
	v := r.URL.Query()
	q := application.CalendarQuery{TenantID: claims.tenantID}
	bad := map[string][]string{}
	date := func(key string) (time.Time, bool) {
		raw := v.Get(key)
		if raw == "" {
			bad[key] = []string{"Wajib diisi (YYYY-MM-DD)"}
			return time.Time{}, false
		}
		t, err := parseDate(raw)
		if err != nil {
			bad[key] = []string{"Tanggal harus berformat YYYY-MM-DD"}
			return time.Time{}, false
		}
		return t, true
	}
	from, okFrom := date("from")
	to, okTo := date("to")
	if okFrom && okTo {
		if problem := domain.CalendarRangeProblem(from, to); problem != "" {
			bad["to"] = []string{problem}
		}
	}
	q.From, q.To = from, to
	if raw := v.Get("picStaffId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			bad["picStaffId"] = []string{"PIC tidak valid"}
		} else {
			q.PICStaffID = &id
		}
	}
	kinds, ok := domain.ParseCalendarKinds(v.Get("kinds"))
	if !ok {
		bad["kinds"] = []string{"Jenis hanya boleh event, milestone, atau payment"}
	}
	if len(bad) > 0 {
		return q, nil, apperror.Validation("Filter kalender tidak valid", bad)
	}
	if claims.assignedOnly() {
		staffID := claims.staffID
		q.AssignedStaffID = &staffID
	}
	return q, kinds, nil
}
