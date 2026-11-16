package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/middleware"
)

// fakeCalendarRepo records the query it was given and which reads ran.
type fakeCalendarRepo struct {
	got   []application.CalendarQuery
	calls []string
}

func (f *fakeCalendarRepo) ListEventDays(_ context.Context, q application.CalendarQuery) ([]domain.CalendarItem, error) {
	f.got, f.calls = append(f.got, q), append(f.calls, "event")
	d, _ := time.Parse(dateLayout, "2026-11-21")
	return []domain.CalendarItem{{Date: d, Kind: domain.CalendarEvent, ProjectID: 1, ProjectName: "Citra & Bima", Title: "Hari H", Status: "Preparation"}}, nil
}

func (f *fakeCalendarRepo) ListOpenMilestones(_ context.Context, q application.CalendarQuery) ([]domain.CalendarItem, error) {
	f.got, f.calls = append(f.got, q), append(f.calls, "milestone")
	d, _ := time.Parse(dateLayout, "2026-11-05")
	return []domain.CalendarItem{{Date: d, Kind: domain.CalendarMilestone, ProjectID: 1, ProjectName: "Citra & Bima", RefID: 4, Title: "Fitting baju", Status: "In Progress"}}, nil
}

func (f *fakeCalendarRepo) ListTermsTouching(_ context.Context, q application.CalendarQuery) ([]domain.TermWithProject, map[int64]int64, error) {
	f.got, f.calls = append(f.got, q), append(f.calls, "payment")
	d, _ := time.Parse(dateLayout, "2026-11-10")
	return []domain.TermWithProject{{Term: domain.PaymentTerm{ID: 2, ProjectID: 1, Label: "Termin 2", Amount: 40, DueDate: d}, ProjectName: "Citra & Bima"}}, map[int64]int64{}, nil
}

func staffRequest(target string, grants *middleware.Grants) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := middleware.NewContextWithClaims(req.Context(), &middleware.Claims{PrincipalType: "staff", PrincipalID: "5", TenantID: "1"})
	return req.WithContext(middleware.WithGrants(ctx, grants))
}

func runCalendar(t *testing.T, req *http.Request) (*httptest.ResponseRecorder, *fakeCalendarRepo) {
	t.Helper()
	repo := &fakeCalendarRepo{}
	h := &Handler{}
	h.SetCalendar(application.NewCalendarService(repo))
	rec := httptest.NewRecorder()
	h.Calendar(rec, req)
	return rec, repo
}

type calendarBody struct {
	Success bool                `json:"success"`
	Errors  map[string][]string `json:"errors"`
	Data    calendarResponse    `json:"data"`
}

func decodeCalendar(t *testing.T, rec *httptest.ResponseRecorder) calendarBody {
	t.Helper()
	var body calendarBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return body
}

const novemberGrid = "/api/v1/calendar?from=2026-10-26&to=2026-12-06"

// A role that reaches every project gets the workspace; a role limited to its
// own projects is scoped to the projects the person holds, whatever it sends.
func TestCalendarScopesAssignedRoles(t *testing.T) {
	rec, repo := runCalendar(t, staffRequest(novemberGrid, grantsOf()))
	if rec.Code != http.StatusOK {
		t.Fatalf("owner: %d %s", rec.Code, rec.Body.String())
	}
	for _, q := range repo.got {
		if q.TenantID != 1 || q.AssignedStaffID != nil {
			t.Fatalf("all-projects role must not be scoped: %+v", q)
		}
	}

	limited := &middleware.Grants{AllProjects: false, Permissions: map[string]bool{"project.view": true}}
	rec, repo = runCalendar(t, staffRequest(novemberGrid+"&picStaffId=9", limited))
	if rec.Code != http.StatusOK {
		t.Fatalf("assigned role: %d %s", rec.Code, rec.Body.String())
	}
	if len(repo.got) != 3 {
		t.Fatalf("every read must run, got %v", repo.calls)
	}
	for _, q := range repo.got {
		if q.AssignedStaffID == nil || *q.AssignedStaffID != 5 {
			t.Fatalf("assigned role must be limited to its own projects (staff 5): %+v", q)
		}
		// The PIC filter can only narrow the scope further, never widen it.
		if q.PICStaffID == nil || *q.PICStaffID != 9 {
			t.Fatalf("PIC filter: %+v", q)
		}
	}
}

func TestCalendarRefusesClients(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, novemberGrid, nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), &middleware.Claims{PrincipalType: "client", PrincipalID: "3", TenantID: "1"}))
	rec, repo := runCalendar(t, req)
	if rec.Code != http.StatusForbidden || len(repo.calls) != 0 {
		t.Fatalf("client principal: got %d (reads %v), want 403 and nothing read", rec.Code, repo.calls)
	}
}

func TestCalendarValidatesRange(t *testing.T) {
	cases := map[string]string{
		"/api/v1/calendar":                                                "from",
		"/api/v1/calendar?from=2026-11-01":                                "to",
		"/api/v1/calendar?from=01-11-2026&to=2026-11-30":                  "from",
		"/api/v1/calendar?from=2026-11-30&to=2026-11-01":                  "to",
		"/api/v1/calendar?from=2026-11-01&to=2027-03-01":                  "to",
		novemberGrid + "&kinds=event,meeting":                             "kinds",
		novemberGrid + "&picStaffId=abc":                                  "picStaffId",
		"/api/v1/calendar?from=2026-11-01&to=2026-11-30&x=1&picStaffId=0": "picStaffId",
	}
	for target, key := range cases {
		rec, repo := runCalendar(t, staffRequest(target, grantsOf()))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: got %d, want 422", target, rec.Code)
			continue
		}
		if body := decodeCalendar(t, rec); len(body.Errors[key]) == 0 {
			t.Errorf("%s: want an error on %q, got %v", target, key, body.Errors)
		}
		if len(repo.calls) != 0 {
			t.Errorf("%s: nothing may be read for a refused request, got %v", target, repo.calls)
		}
	}
}

func TestCalendarKindsFilter(t *testing.T) {
	rec, repo := runCalendar(t, staffRequest(novemberGrid+"&kinds=payment", grantsOf()))
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if strings.Join(repo.calls, ",") != "payment" {
		t.Fatalf("only the payment read may run, got %v", repo.calls)
	}
	body := decodeCalendar(t, rec)
	if len(body.Data.Items) != 1 || body.Data.Items[0].Kind != "payment" || body.Data.Items[0].Amount == nil || *body.Data.Items[0].Amount != 40 {
		t.Fatalf("payment items only: %+v", body.Data.Items)
	}

	rec, _ = runCalendar(t, staffRequest(novemberGrid, grantsOf()))
	body = decodeCalendar(t, rec)
	if body.Data.From != "2026-10-26" || body.Data.To != "2026-12-06" || len(body.Data.Items) != 3 {
		t.Fatalf("every kind by default: %+v", body.Data)
	}
	// By day: the step (5 Nov), the installment (10 Nov), the wedding (21 Nov).
	got := []string{body.Data.Items[0].Kind, body.Data.Items[1].Kind, body.Data.Items[2].Kind}
	if strings.Join(got, ",") != "milestone,payment,event" {
		t.Fatalf("order by day: %v", got)
	}
	if body.Data.Items[2].Amount != nil {
		t.Fatal("amount is for payments only")
	}
}
