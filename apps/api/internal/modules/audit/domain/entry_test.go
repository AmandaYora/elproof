package domain

import "testing"

// RT-08: an entry that carries its own sentence reads with it; older rows (no
// sentence) still read from the route.
func TestEntrySentence(t *testing.T) {
	stated := Entry{Method: "POST", Path: "/api/v1/tenants/12/toggle-suspension", Description: "Membekukan tenant WO Testing"}
	if got := stated.Sentence(); got != "Membekukan tenant WO Testing" {
		t.Errorf("stated: %q", got)
	}
	old := Entry{Method: "POST", Path: "/api/v1/tenants/12/toggle-suspension"}
	if got := old.Sentence(); got != "Membekukan atau mengaktifkan kembali tenant 12" {
		t.Errorf("old row: %q", got)
	}
}
