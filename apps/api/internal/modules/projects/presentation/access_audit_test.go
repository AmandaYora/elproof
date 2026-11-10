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

// uploadRefused runs the upload and reports whether the sharing rule refused it.
// A request the rule lets through goes on to the evidence service, which this
// handler does not have: that panic means "not refused by the rule".
func uploadRefused(t *testing.T, kind string, visible bool, perms ...string) (refused bool) {
	t.Helper()
	body := `{"name":"Bukti","type":"Transfer Proof","fileName":"b.pdf","mimeType":"application/pdf","base64Data":"JVBERg==","relatedKind":"` + kind + `","relatedId":1,"isClientVisible":` + map[bool]string{true: "true", false: "false"}[visible] + `}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/1/evidence", strings.NewReader(body))
	defer func() {
		if recover() != nil {
			refused = false
		}
	}()
	(&Handler{}).uploadEvidence(rec, req, staffClaims{tenantID: 1, staffID: 5, grants: grantsOf(perms...)}, 1)
	return rec.Code == http.StatusForbidden
}

// Review round 1: the rule refuses only sharing without project.edit; proofs and
// shared documents by whoever may edit the project still go through.
func TestUploadSharingRuleLetsLegitimateUploadsThrough(t *testing.T) {
	if uploadRefused(t, "general", true, "project.edit") {
		t.Error("project.edit may share a document while uploading it")
	}
	if uploadRefused(t, "general", false, "finance.record") {
		t.Error("finance.record may add a document that stays hidden from the client")
	}
	if uploadRefused(t, "clientPayment", true, "finance.record") {
		t.Error("a payment proof's visibility follows the payment, not the flag: finance.record may upload it")
	}
	if !uploadRefused(t, "general", true, "finance.record") {
		t.Error("finance.record alone must not share a document with the client")
	}
}
