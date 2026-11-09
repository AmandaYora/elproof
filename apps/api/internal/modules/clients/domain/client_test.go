package domain

import (
	"testing"
	"time"
)

func TestClientRestoreWindow(t *testing.T) {
	deleted := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	if !CanRestore(deleted, deleted.Add(24*time.Hour-time.Second)) || CanRestore(deleted, deleted.Add(24*time.Hour)) {
		t.Error("restorable for exactly 24 hours")
	}
	if got := RestorableUntil(deleted); !got.Equal(deleted.Add(24 * time.Hour)) {
		t.Errorf("RestorableUntil = %v", got)
	}
	now := deleted.Add(30 * time.Hour)
	if cutoff := PurgeCutoff(now); !cutoff.Equal(deleted.Add(6 * time.Hour)) {
		t.Errorf("PurgeCutoff = %v", cutoff)
	}
	// purge: deleted_at <= cutoff; restore: deleted_at > cutoff — complementary.
	for _, at := range []time.Time{deleted, deleted.Add(6 * time.Hour), deleted.Add(7 * time.Hour)} {
		purged := !at.After(PurgeCutoff(now))
		if purged == CanRestore(at, now) {
			t.Errorf("deleted at %v: purge=%v restorable=%v must differ", at, purged, CanRestore(at, now))
		}
	}
}
