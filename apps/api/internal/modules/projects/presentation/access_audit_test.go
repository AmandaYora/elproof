package presentation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elproof/internal/shared/middleware"
)

func grantsOf(perms ...string) *middleware.Grants {
	g := &middleware.Grants{AllProjects: true, Permissions: map[string]bool{}}
	for _, p := range perms {
		g.Permissions[p] = true
	}
	return g
}

// T-20261110-12: sharing a document with the client is a project decision
// (project.edit, ADR-0026; the visibility toggle already needs it). A role that
// may only record payments (Keuangan) must not publish a document to the Client
// Portal by ticking "Tampilkan ke portal klien" while uploading it. The refusal
// comes before the evidence service is touched (the handler here has none).
func TestUploadSharedWithClientNeedsProjectEdit(t *testing.T) {
	h := &Handler{}
	for _, kind := range []string{"general", "projectMilestone"} {
		body := `{"name":"Kontrak","type":"Contract","fileName":"k.pdf","mimeType":"application/pdf","base64Data":"JVBERg==","relatedKind":"` + kind + `","relatedId":1,"isClientVisible":true}`
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/1/evidence", strings.NewReader(body))
		h.uploadEvidence(rec, req, staffClaims{tenantID: 1, staffID: 5, grants: grantsOf("finance.record")}, 1)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "forbidden_permission") {
			t.Errorf("%s: finance.record alone sharing with the client: got %d %s, want 403 forbidden_permission", kind, rec.Code, rec.Body.String())
		}
	}
}
