package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/middleware"
)

// ADR-0047: the Client Portal's four writes.

func TestClientWritePermission(t *testing.T) {
	allowed := []struct {
		method string
		rest   []string
		perm   string
	}{
		{http.MethodPost, []string{"uploads"}, "portal.finance.submit"},
		{http.MethodPut, []string{"uploads", "tok.sig"}, "portal.finance.submit"},
		{http.MethodPost, []string{"client-payments", "submissions"}, "portal.finance.submit"},
		{http.MethodPost, []string{"milestones", "3", "complete"}, "portal.tasks.complete"},
	}
	for _, c := range allowed {
		perm, ok := clientWritePermission(c.method, c.rest)
		if !ok || perm != c.perm {
			t.Errorf("%s %v: got (%q,%v), want %q", c.method, c.rest, perm, ok, c.perm)
		}
	}
	denied := []struct {
		method string
		rest   []string
	}{
		{http.MethodPost, []string{"client-payments"}},                // the staff create
		{http.MethodPost, []string{"client-payments", "3", "verify"}}, // verifying is staff's
		{http.MethodPatch, []string{"client-payments", "3"}},
		{http.MethodDelete, []string{"client-payments", "3"}},
		{http.MethodPut, []string{"client-payments", "submissions"}},
		{http.MethodPatch, []string{"milestones", "3"}},
		{http.MethodPost, []string{"milestones"}},
		{http.MethodPost, []string{"milestones", "x", "complete"}},
		{http.MethodPost, []string{"milestones", "0", "complete"}},
		{http.MethodPut, []string{"milestones", "3", "complete"}},
		{http.MethodDelete, []string{"uploads", "tok"}},
		{http.MethodPut, []string{"uploads"}},
		{http.MethodPut, []string{"uploads", ""}},
		{http.MethodPost, []string{"evidence"}},
		{http.MethodPut, []string{"cover"}},
		{http.MethodDelete, []string{"cover"}},
		{http.MethodPatch, nil},
		{http.MethodPost, []string{"issues"}},
	}
	for _, c := range denied {
		if _, ok := clientWritePermission(c.method, c.rest); ok {
			t.Errorf("%s %v must not be a client write", c.method, c.rest)
		}
	}
}

type ownProject struct{ projectID int64 }

func (o ownProject) ProjectIDForClient(context.Context, int64, int64) (int64, error) {
	return o.projectID, nil
}

type failingAccess struct{}

func (failingAccess) ProjectIDForClient(context.Context, int64, int64) (int64, error) {
	return 0, errors.New("no project")
}

func clientRequest(method, path string, perms ...string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	ctx := middleware.NewContextWithClaims(req.Context(), &middleware.Claims{PrincipalType: "client", PrincipalID: "5", TenantID: "1"})
	ctx = middleware.WithGrants(ctx, grantsOf(perms...))
	return req.WithContext(ctx)
}

func TestClientWritesAreScopedAndNeedTheirPermission(t *testing.T) {
	h := &Handler{clientAccess: ownProject{projectID: 12}}
	cases := []struct {
		method, path string
		perms        []string
		want         int // 0 = let through
	}{
		{http.MethodPost, "/api/v1/projects/12/client-payments/submissions", []string{"portal.finance.submit"}, 0},
		{http.MethodPost, "/api/v1/projects/12/uploads", []string{"portal.finance.submit"}, 0},
		{http.MethodPut, "/api/v1/projects/12/uploads/tok.sig", []string{"portal.finance.submit"}, 0},
		{http.MethodPost, "/api/v1/projects/12/milestones/3/complete", []string{"portal.tasks.complete"}, 0},
		// The cover is the hero of every portal page: no portal permission.
		{http.MethodGet, "/api/v1/projects/12/cover", nil, 0},
		// Without the permission (a Keluarga role): 403.
		{http.MethodPost, "/api/v1/projects/12/client-payments/submissions", []string{"portal.finance.view"}, http.StatusForbidden},
		{http.MethodPost, "/api/v1/projects/12/uploads", []string{"portal.tasks.complete"}, http.StatusForbidden},
		{http.MethodPost, "/api/v1/projects/12/milestones/3/complete", []string{"portal.finance.submit", "portal.timeline.view"}, http.StatusForbidden},
		// Another couple's project: 403 even with every permission.
		{http.MethodPost, "/api/v1/projects/13/client-payments/submissions", []string{"portal.finance.submit"}, http.StatusForbidden},
		{http.MethodPost, "/api/v1/projects/13/milestones/3/complete", []string{"portal.tasks.complete"}, http.StatusForbidden},
		{http.MethodGet, "/api/v1/projects/13/cover", nil, http.StatusForbidden},
		// Every other write stays refused.
		{http.MethodPatch, "/api/v1/projects/12/milestones/3", []string{"portal.tasks.complete", "portal.finance.submit"}, http.StatusForbidden},
		{http.MethodPost, "/api/v1/projects/12/client-payments", []string{"portal.finance.submit"}, http.StatusForbidden},
		{http.MethodPost, "/api/v1/projects/12/client-payments/3/verify", []string{"portal.finance.submit"}, http.StatusForbidden},
		{http.MethodDelete, "/api/v1/projects/12/client-payments/3", []string{"portal.finance.submit"}, http.StatusForbidden},
		{http.MethodPut, "/api/v1/projects/12/cover", []string{"portal.finance.submit"}, http.StatusForbidden},
		{http.MethodPost, "/api/v1/projects/12/evidence", []string{"portal.finance.submit"}, http.StatusForbidden},
	}
	for _, c := range cases {
		rest := strings.Split(strings.TrimPrefix(c.path, "/api/v1/projects/"), "/")
		var projectID int64 = 12
		if rest[0] == "13" {
			projectID = 13
		}
		rec := httptest.NewRecorder()
		claims, ok := h.resolveProjectAccess(rec, clientRequest(c.method, c.path, c.perms...), projectID, rest[1:])
		switch {
		case c.want == 0 && (!ok || !claims.isClient || claims.clientID != 5 || claims.staffID != 0 || claims.tenantID != 1):
			t.Errorf("%s %s: want let through as client 5, got ok=%v %+v (%d %s)", c.method, c.path, ok, claims, rec.Code, rec.Body.String())
		case c.want != 0 && (ok || rec.Code != c.want):
			t.Errorf("%s %s: want %d, got ok=%v %d", c.method, c.path, c.want, ok, rec.Code)
		}
	}

	// A client whose project cannot be resolved writes nothing.
	h = &Handler{clientAccess: failingAccess{}}
	rec := httptest.NewRecorder()
	if _, ok := h.resolveProjectAccess(rec, clientRequest(http.MethodPost, "/api/v1/projects/12/client-payments/submissions", "portal.finance.submit"),
		12, []string{"client-payments", "submissions"}); ok || rec.Code != http.StatusForbidden {
		t.Errorf("unresolved client: %d", rec.Code)
	}
}

// The client-only routes refuse staff, and the staff-only cover writes refuse a
// client, before any service is touched (the handler here has none).
func TestPortalRoutesRefuseTheOtherPrincipal(t *testing.T) {
	h := &Handler{}
	staff := staffClaims{tenantID: 1, staffID: 9, grants: grantsOf("finance.record", "project.edit")}
	client := staffClaims{tenantID: 1, isClient: true, clientID: 5, grants: grantsOf("portal.finance.submit", "portal.tasks.complete")}
	calls := []struct {
		name string
		run  func(w http.ResponseWriter, r *http.Request)
	}{
		{"staff submits a proof", func(w http.ResponseWriter, r *http.Request) { h.submitClientPayment(w, r, staff, 12) }},
		{"staff completes a client task", func(w http.ResponseWriter, r *http.Request) { h.completeMilestoneByClient(w, r, staff, 12, "3") }},
		{"client uploads a cover", func(w http.ResponseWriter, r *http.Request) { h.putCover(w, r, client, 12) }},
		{"client removes the cover", func(w http.ResponseWriter, r *http.Request) { h.deleteCover(w, r, client, 12) }},
	}
	for _, c := range calls {
		rec := httptest.NewRecorder()
		c.run(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}")))
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "forbidden_permission") {
			t.Errorf("%s: got %d %s", c.name, rec.Code, rec.Body.String())
		}
	}
}

func TestIdempotencyKeysBelongToTheActor(t *testing.T) {
	if (staffClaims{staffID: 9}).actorID() != 9 {
		t.Error("staff: the staff id")
	}
	if (staffClaims{isClient: true, clientID: 5}).actorID() != 5 {
		t.Error("client: the client id, so the bride and the groom never replay each other's submission")
	}
}

func TestPayloadsCarryTheNewFields(t *testing.T) {
	at := time.Unix(1_800_000_000, 0)
	key := "k/cover/1.jpg"
	withCover := toProjectResponse(domain.Project{ID: 12, CoverPhotoPath: &key, CoverUpdatedAt: &at})
	if withCover.CoverURL == nil || *withCover.CoverURL != "/api/v1/projects/12/cover?v=1800000000" {
		t.Errorf("coverUrl: %v", withCover.CoverURL)
	}
	raw, _ := json.Marshal(toProjectResponse(domain.Project{ID: 12}))
	if !strings.Contains(string(raw), `"coverUrl":null`) {
		t.Errorf("no cover: coverUrl null, got %s", raw)
	}

	raw, _ = json.Marshal(toMilestoneResponse(domain.ProjectMilestone{ID: 3, ForClient: true, CompletedByClientID: 5, Status: domain.MilestoneCompleted}))
	if !strings.Contains(string(raw), `"forClient":true`) || !strings.Contains(string(raw), `"completedByClient":true`) {
		t.Errorf("milestone: %s", raw)
	}
	raw, _ = json.Marshal(toMilestoneResponse(domain.ProjectMilestone{ID: 4}))
	if !strings.Contains(string(raw), `"forClient":false`) || !strings.Contains(string(raw), `"completedByClient":false`) {
		t.Errorf("staff milestone: %s", raw)
	}

	raw, _ = json.Marshal(toClientPaymentResponse(domain.ClientPayment{ID: 1, SubmittedByClientID: 5}, true))
	if !strings.Contains(string(raw), `"source":"client"`) || !strings.Contains(string(raw), `"recordedBy":null`) {
		t.Errorf("client submission: %s", raw)
	}
	raw, _ = json.Marshal(toClientPaymentResponse(domain.ClientPayment{ID: 2, RecordedByStaffID: 9}, true))
	if !strings.Contains(string(raw), `"source":"staff"`) {
		t.Errorf("staff payment: %s", raw)
	}

	done := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	home := toClientHomeResponse(&application.ClientHome{
		Project: application.HomeProject{ID: 12, CoverPhotoPath: &key, CoverUpdatedAt: &at},
		ClientTasks: []domain.ProjectMilestone{
			{ID: 3, Name: "Kirim daftar tamu", Status: domain.MilestoneNotStarted, TargetDate: done},
			{ID: 4, Name: "Pilih menu", Status: domain.MilestoneCompleted, TargetDate: done, CompletedDate: &done},
		},
	})
	raw, _ = json.Marshal(home)
	for _, want := range []string{
		`"coverUrl":"/api/v1/projects/12/cover?v=1800000000"`,
		`"clientTasks":[{"id":3,"name":"Kirim daftar tamu","targetDate":"2026-11-02","status":"Not Started","completedDate":null},` +
			`{"id":4,"name":"Pilih menu","targetDate":"2026-11-02","status":"Completed","completedDate":"2026-11-02"}]`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("home lacks %s: %s", want, raw)
		}
	}
	raw, _ = json.Marshal(toClientHomeResponse(&application.ClientHome{}))
	if !strings.Contains(string(raw), `"clientTasks":[]`) || !strings.Contains(string(raw), `"coverUrl":null`) {
		t.Errorf("home without tasks or cover: %s", raw)
	}
}
