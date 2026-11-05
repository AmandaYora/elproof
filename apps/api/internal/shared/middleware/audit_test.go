package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type memRecorder struct{ recs []AuditRecord }

func (m *memRecorder) Record(_ context.Context, rec AuditRecord) error {
	m.recs = append(m.recs, rec)
	return nil
}

func TestAuditTrail(t *testing.T) {
	rec := &memRecorder{}
	h := AuditTrail(rec, "platform_admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	do := func(principal, method string) {
		req := httptest.NewRequest(method, "/api/v1/tenants/4/toggle-suspension", nil)
		req.Header.Set("X-Real-IP", "1.2.3.4")
		req = req.WithContext(NewContextWithClaims(req.Context(), &Claims{PrincipalType: principal, PrincipalID: "9", Role: "Support"}))
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	do("platform_admin", http.MethodGet)    // reads are not audited
	do("staff", http.MethodPost)            // other principals are not audited
	do("platform_admin", http.MethodPost)   // audited, 200
	do("platform_admin", http.MethodDelete) // audited, refused attempt
	if len(rec.recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(rec.recs), rec.recs)
	}
	if r := rec.recs[0]; r.StatusCode != 200 || r.ActorID != "9" || r.IP != "1.2.3.4" || r.ActorRole != "Support" {
		t.Errorf("unexpected record: %+v", r)
	}
	if rec.recs[1].StatusCode != http.StatusForbidden {
		t.Errorf("refused attempt must be recorded with its status, got %d", rec.recs[1].StatusCode)
	}
}
