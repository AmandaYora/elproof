package domain

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

func TestRestoreWindowIsExactlyRetention(t *testing.T) {
	if got := RestorableUntil(t0); !got.Equal(t0.Add(24 * time.Hour)) {
		t.Fatalf("RestorableUntil = %v", got)
	}
	cases := []struct {
		name string
		now  time.Time
		can  bool
	}{
		{"just deleted", t0, true},
		{"one second before the end", t0.Add(24*time.Hour - time.Second), true},
		{"exactly at the end", t0.Add(24 * time.Hour), false},
		{"long after", t0.Add(72 * time.Hour), false},
	}
	for _, c := range cases {
		if got := CanRestore(t0, c.now); got != c.can {
			t.Errorf("%s: CanRestore = %v, want %v", c.name, got, c.can)
		}
		// A row is always in exactly one state: restorable, or due for the purge.
		if IsPurgeDue(t0, c.now) == CanRestore(t0, c.now) {
			t.Errorf("%s: a row must be either restorable or due for purge, never both or neither", c.name)
		}
	}
}

// The purge deletes `deleted_at <= PurgeCutoff(now)` and restore accepts
// `deleted_at > PurgeCutoff(now)`: they must agree with IsPurgeDue/CanRestore.
func TestPurgeCutoffMatchesTheRestoreRule(t *testing.T) {
	now := t0.Add(48 * time.Hour)
	cutoff := PurgeCutoff(now)
	for _, deletedAt := range []time.Time{t0, t0.Add(23 * time.Hour), t0.Add(24 * time.Hour), t0.Add(24*time.Hour + time.Second), t0.Add(47 * time.Hour), now} {
		purged := !deletedAt.After(cutoff) // deleted_at <= cutoff
		if purged != IsPurgeDue(deletedAt, now) {
			t.Errorf("deletedAt %v: SQL-style cutoff says purge=%v, IsPurgeDue says %v", deletedAt, purged, IsPurgeDue(deletedAt, now))
		}
		restorable := deletedAt.After(cutoff) // deleted_at > cutoff
		if restorable != CanRestore(deletedAt, now) {
			t.Errorf("deletedAt %v: restorable mismatch", deletedAt)
		}
	}
}

func TestPurgeReportTotals(t *testing.T) {
	a := PurgeReport{ClientPayments: 1, Engagements: 2, EngagementPayments: 3, Evidence: 4, Objects: 9}
	b := PurgeReport{VendorPayments: 1, VenuePayments: 1, EngagementIssues: 2, EngagementMilestones: 1}
	a.Add(b)
	if a.Total() != 1+2+3+4+1+1+2+1 {
		t.Errorf("Total = %d (stored files are not rows)", a.Total())
	}
	if a.Objects != 9 {
		t.Errorf("Objects = %d", a.Objects)
	}
	if (PurgeReport{Objects: 5}).Total() != 0 {
		t.Error("a run that removed only objects removed no rows")
	}
}

func TestClientVisibleIssuesKeepsOnlyShared(t *testing.T) {
	list := []VendorIssue{{ID: 1, IsClientVisible: true}, {ID: 2}, {ID: 3, IsClientVisible: true}}
	got := ClientVisibleIssues(list)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("got %+v", got)
	}
	if got := ClientVisibleIssues(nil); got == nil || len(got) != 0 {
		t.Error("an empty result must be an empty slice")
	}
	if (VendorIssue{}).HasVendor() || !(VendorIssue{ProjectVendorID: 4}).HasVendor() {
		t.Error("HasVendor: 0 means none")
	}
}

func TestHiddenIssueProofIsNeverVisibleToAClient(t *testing.T) {
	can := func(string) bool { return true }
	shown := Evidence{RelatedKind: RelatedIssue}
	hidden := Evidence{RelatedKind: RelatedIssue, ClientHidden: true}
	if !ClientCanSeeEvidence(shown, can) {
		t.Error("proof of a shared kendala is visible with portal.issues.view")
	}
	if ClientCanSeeEvidence(hidden, can) {
		t.Error("proof of a kendala not shared with the client must stay hidden")
	}
}
