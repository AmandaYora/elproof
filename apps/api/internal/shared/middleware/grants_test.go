package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeGrantsResolver struct {
	g   *Grants
	err error
}

func (f fakeGrantsResolver) ResolveGrants(context.Context, int64, int64) (*Grants, error) { return f.g, f.err }

func runGrants(t *testing.T, claims *Claims, res GrantsResolver) (int, *Grants) {
	t.Helper()
	l := &LoadGrants{}
	if res != nil {
		l.SetResolver(res)
	}
	var seen *Grants
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = GrantsFrom(r.Context()) }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), claimsContextKey, claims))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, seen
}

func TestLoadGrants(t *testing.T) {
	g := &Grants{RoleID: 7, Permissions: map[string]bool{"project.create": true}}
	staff := &Claims{PrincipalType: "staff", PrincipalID: "1", TenantID: "3", RoleID: 7}

	if code, got := runGrants(t, staff, fakeGrantsResolver{g: g}); code != 200 || !got.Can("project.create") || got.Can("team.manage") {
		t.Fatalf("staff grants not loaded: %d %+v", code, got)
	}
	if code, _ := runGrants(t, &Claims{PrincipalType: "staff", TenantID: "3"}, fakeGrantsResolver{g: g}); code != 401 {
		t.Fatalf("token without role id must be refreshed: %d", code)
	}
	if code, _ := runGrants(t, staff, fakeGrantsResolver{}); code != 403 {
		t.Fatalf("deleted role must be refused: %d", code)
	}
	if code, _ := runGrants(t, staff, fakeGrantsResolver{err: errors.New("db")}); code != 503 {
		t.Fatalf("resolver failure must fail closed: %d", code)
	}
	if code, got := runGrants(t, &Claims{PrincipalType: "platform_admin", PrincipalID: "1"}, nil); code != 200 || got != nil {
		t.Fatalf("platform admins carry no workspace grants: %d %+v", code, got)
	}
	var none *Grants
	if none.Can("anything") {
		t.Fatal("nil grants must grant nothing")
	}
}

func TestGrantsCovers(t *testing.T) {
	hr := &Grants{Permissions: map[string]bool{"team.manage": true}}
	if hr.Covers([]string{"team.manage", "finance.record"}, false) {
		t.Error("a role with only team.manage covers finance.record")
	}
	if !hr.Covers([]string{"team.manage"}, false) {
		t.Error("a role does not cover its own permissions")
	}
	if hr.Covers([]string{"team.manage"}, true) {
		t.Error("an assigned-scope role covers the every-project scope")
	}
	if !(&Grants{Owner: true}).Covers([]string{"anything"}, true) {
		t.Error("the Owner does not cover everything")
	}
	var none *Grants
	if none.Covers(nil, false) {
		t.Error("nil grants cover something")
	}
}
