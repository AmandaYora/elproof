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
		{http.MethodGet, []string{"payments"}, "", ""},
		{http.MethodPatch, nil, "project.edit", ""},
		{http.MethodDelete, nil, "project.delete", ""},
		{http.MethodPost, []string{"cancel"}, "project.edit", ""},
		{http.MethodPost, []string{"toggle-archive"}, "project.archive", ""},
		{http.MethodPost, []string{"duplicate"}, "project.duplicate", ""},
		{http.MethodPost, []string{"payments"}, "finance.record", ""},
		{http.MethodPost, []string{"client-payments", "4", "verify"}, "finance.record", ""},
		{http.MethodDelete, []string{"client-payments", "4"}, "finance.payment.delete", ""},
		{http.MethodPatch, []string{"venue-payments", "4"}, "finance.record", ""},
		{http.MethodPost, []string{"evidence"}, "project.edit", "finance.record"},
		{http.MethodPost, []string{"evidence", "9", "toggle-client-visible"}, "project.edit", ""},
		{http.MethodPost, []string{"uploads"}, "project.edit", "finance.record"},
		{http.MethodPut, []string{"uploads", "tok.sig"}, "project.edit", "finance.record"},
		{http.MethodPost, []string{"milestones"}, "project.edit", ""},
		{http.MethodPost, []string{"vendors", "2", "cancel"}, "project.edit", ""},
		{http.MethodPatch, []string{"issues", "9"}, "project.edit", ""},
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
		"":                "",
		"milestones":      "portal.timeline.view",
		"vendors":         "portal.vendor.view",
		"venue":           "portal.vendor.view",
		"client-payments": "portal.finance.view",
		"payment-terms":   "portal.finance.view",
		"issues":          "portal.issues.view",
		"evidence":        "portal.documents.view",
		"documents":       "portal.documents.view",
	} {
		var r []string
		if rest != "" {
			r = []string{rest}
		}
		if got := clientReadPermission(r); got != want {
			t.Errorf("%q: got %q, want %q", rest, got, want)
		}
	}
	// A single file is judged per file (clientCanSeeEvidence), not by the documents tab's permission.
	for _, tail := range []string{"file", "link"} {
		if got := clientReadPermission([]string{"evidence", "9", tail}); got != "" {
			t.Errorf("evidence/9/%s: got %q, want no up-front permission", tail, got)
		}
	}
}
