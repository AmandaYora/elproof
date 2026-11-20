package application

import (
	"context"
	"strings"
	"testing"

	"elproof/internal/modules/projects/domain"
)

// PLAN §4.1: "Perbarui kapasitas" checks the events held at the venue against
// the new numbers; a session that no longer fits refuses it and nothing changes.
func TestRefreshingAVenueBelowItsSessionsIsRefused(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()
	if _, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-13"), 550, 550)); err != nil {
		t.Fatal(err)
	}
	gedung := venueSuppliers(db.suppliers.rows)[venueGedung]

	_, err := events.suppliers.RefreshCapacity(ctx, p.ID, gedung.ID, 10, intp(500), intp(400))
	f := validationFields(t, err)
	msgs := f["capacity"]
	if len(msgs) != 2 || msgs[0] != "Gedung Melati muat 500 orang berdiri, Sesi 1 Resepsi 550" {
		t.Fatalf("both sessions are named, with the numbers: %v", f)
	}
	if got := venueSuppliers(db.suppliers.rows)[venueGedung]; got.CapacityStanding == nil || *got.CapacityStanding != 600 {
		t.Errorf("the copied capacity is unchanged: %+v", got.CapacityStanding)
	}

	// Numbers that still hold every session are copied.
	fresh, err := events.suppliers.RefreshCapacity(ctx, p.ID, gedung.ID, 10, intp(560), intp(400))
	if err != nil || fresh.CapacityStanding == nil || *fresh.CapacityStanding != 560 {
		t.Fatalf("560 holds 550: %+v %v", fresh, err)
	}
}

// PLAN §3.2: a duplicate re-checks the capacity of the events it copies, at the
// venues it copies; a cancelled venue is not copied, so its event goes without one.
func TestDuplicateRechecksCapacity(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()
	if _, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-13"), 550, 550)); err != nil {
		t.Fatal(err)
	}
	svc := events.projects
	svc.suppliers, svc.events = db.suppliers, db.events
	if err := svc.checkCopiedCapacity(ctx, p.ID); err != nil {
		t.Fatalf("everything fits: %v", err)
	}

	// A copied capacity smaller than a session (saved before the rule, say).
	for i := range db.suppliers.rows {
		if db.suppliers.rows[i].Kind == domain.SupplierVenue && db.suppliers.rows[i].SupplierID == venueGedung {
			db.suppliers.rows[i].CapacityStanding = intp(500)
		}
	}
	err := svc.checkCopiedCapacity(ctx, p.ID)
	f := validationFields(t, err)
	if len(f["capacity"]) != 2 || !strings.HasPrefix(err.Error(), "Proyek belum bisa diduplikat. Gedung Melati muat 500 orang berdiri, Sesi 1 Resepsi 550") {
		t.Fatalf("refused with the numbers: %v / %v", err, f)
	}

	// Cancelled, the hall is left out of the copy, and so is the check.
	for i := range db.suppliers.rows {
		if db.suppliers.rows[i].SupplierID == venueGedung {
			db.suppliers.rows[i].EngagementStatus = domain.EngagementCancelled
		}
	}
	if err := svc.checkCopiedCapacity(ctx, p.ID); err != nil {
		t.Errorf("a cancelled venue is not copied: %v", err)
	}
}
