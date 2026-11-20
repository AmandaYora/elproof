package presentation

import (
	"net/http"
	"testing"
)

func TestStaffWritePermission(t *testing.T) {
	cases := []struct {
		method string
		rest   []string
		perm   string
		alt    string
	}{
		{http.MethodGet, []string{"supplier-payments"}, "", ""},
		// Tahap 8: a project PATCH carries details (project.edit) and/or the archive
		// state (project.archive); updateProject judges each field on its own.
		{http.MethodPatch, nil, "project.edit", "project.archive"},
		{http.MethodDelete, nil, "project.delete", ""},
		{http.MethodPost, []string{"cancel"}, "project.edit", ""},
		// The removed toggle-archive route has no routing of its own any more: it
		// falls to the subtree default, like any unknown path.
		{http.MethodPost, []string{"toggle-archive"}, "project.edit", ""},
		{http.MethodPost, []string{"duplicate"}, "project.duplicate", ""},
		{http.MethodPost, []string{"supplier-payments"}, "finance.record", "finance.margin.view"},
		{http.MethodPost, []string{"client-payments", "4", "verify"}, "finance.record", ""},
		{http.MethodDelete, []string{"client-payments", "4"}, "finance.payment.delete", ""},
		{http.MethodPatch, []string{"supplier-payments", "4"}, "finance.record", "finance.margin.view"},
		{http.MethodPost, []string{"evidence"}, "project.edit", "finance.record"},
		{http.MethodPost, []string{"evidence", "9", "toggle-client-visible"}, "project.edit", "finance.record"},
		{http.MethodPatch, []string{"evidence", "9"}, "project.edit", ""},
		{http.MethodPatch, []string{"milestones", "3"}, "project.edit", ""},
		{http.MethodPatch, []string{"suppliers", "2", "milestones", "3"}, "project.edit", ""},
		{http.MethodPost, []string{"events"}, "project.edit", ""},
		{http.MethodPut, []string{"events", "3"}, "project.edit", ""},
		{http.MethodPost, []string{"events", "3", "set-main"}, "project.edit", ""},
		{http.MethodPost, []string{"suppliers", "2", "refresh-capacity"}, "project.edit", ""},
		{http.MethodPatch, []string{"client-payments", "4"}, "finance.record", ""},
		{http.MethodPost, []string{"uploads"}, "project.edit", "finance.record"},
		{http.MethodPut, []string{"uploads", "tok.sig"}, "project.edit", "finance.record"},
		{http.MethodPost, []string{"milestones"}, "project.edit", ""},
		{http.MethodPost, []string{"suppliers"}, "project.edit", ""},
		{http.MethodPatch, []string{"issues", "9"}, "project.edit", ""},
		// Tahap 5: restoring a payment is the same decision as deleting it.
		{http.MethodPost, []string{"client-payments", "4", "restore"}, "finance.payment.delete", ""},
		{http.MethodPost, []string{"supplier-payments", "4", "restore"}, "finance.payment.delete", ""},
		{http.MethodDelete, []string{"supplier-payments", "4"}, "finance.payment.delete", ""},
		{http.MethodDelete, []string{"suppliers", "2"}, "project.edit", ""},
		{http.MethodPost, []string{"suppliers", "2", "restore"}, "project.edit", ""},
		{http.MethodDelete, []string{"evidence", "9"}, "project.edit", "finance.record"},
		{http.MethodGet, []string{"overview"}, "", ""},
		{http.MethodGet, []string{"activity"}, "", ""},
	}
	for _, c := range cases {
		perm, alt := staffWritePermission(c.method, c.rest)
		if perm != c.perm || alt != c.alt {
			t.Errorf("%s %v: got (%q,%q), want (%q,%q)", c.method, c.rest, perm, alt, c.perm, c.alt)
		}
	}
}

func TestClientReadPermission(t *testing.T) {
	for rest, want := range map[string]string{
		"":                  "",
		"milestones":        "portal.timeline.view",
		"events":            "portal.timeline.view",
		"suppliers":         "portal.vendor.view",
		"client-payments":   "portal.finance.view",
		"supplier-payments": "portal.finance.view",
		"payment-terms":     "portal.finance.view",
		"issues":            "portal.issues.view",
		"evidence":          "portal.documents.view",
		"documents":         "portal.documents.view",
	} {
		var r []string
		if rest != "" {
			r = []string{rest}
		}
		if got := clientReadPermission(r); got != want {
			t.Errorf("%q: got %q, want %q", rest, got, want)
		}
	}
	// A single file is judged per file (canSeeEvidence), not by the documents tab's permission.
	for _, tail := range []string{"file", "link"} {
		if got := clientReadPermission([]string{"evidence", "9", tail}); got != "" {
			t.Errorf("evidence/9/%s: got %q, want no up-front permission", tail, got)
		}
	}
}
