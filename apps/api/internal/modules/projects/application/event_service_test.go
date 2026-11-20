package application

import (
	"context"
	"strings"
	"testing"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// wireEventsWith gives a ProjectService its EventService over in-memory events
// and suppliers, with workspace 1's event kinds and venues (testCatalog,
// testVenues) and the real capacity rule.
func wireEventsWith(p *ProjectService, events *memEvents, suppliers *memSuppliers, tx Transactor, act *ActivityService) *EventService {
	sup := NewSupplierService(suppliers, nil, events, suppliers.pays, act)
	ev := NewEventService(events, sup, p, tx, act)
	ev.SetCatalog(testCatalog())
	ev.SetVenues(testVenues())
	sup.SetCapacity(ev)
	p.SetEvents(ev)
	return ev
}

// newEventProject is a project of workspace 1 created with only its Akad at
// the mosque (150 seated, one session of 120).
func newEventProject(t *testing.T) (*EventService, *followDB, *domain.Project) {
	t.Helper()
	svc, db, _ := newFollow()
	p, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, ProjectInput{
		Name: "Ayu & Bima", PrepStartDate: ymd("2026-11-15"),
	}, ProjectSetup{Events: []EventInput{akadAtMasjid(ymd("2027-06-12"))}})
	if err != nil {
		t.Fatal(err)
	}
	return db.eventSvc, db, p
}

func venueSuppliers(rows []domain.ProjectSupplier) map[int64]domain.ProjectSupplier {
	out := map[int64]domain.ProjectSupplier{}
	for _, s := range rows {
		if s.Kind == domain.SupplierVenue {
			out[s.SupplierID] = s
		}
	}
	return out
}

func validationFields(t *testing.T, err error) map[string][]string {
	t.Helper()
	ae, ok := apperror.As(err)
	if err == nil || !ok || ae.Kind != apperror.KindValidation {
		t.Fatalf("want a 422, got %v", err)
	}
	return ae.Fields
}

// Tahap 2: Akad at a mosque holding 150 seated (one session of 120) and Resepsi
// at a hall holding 600 standing (two sessions of 550) both save.
func TestMosqueAkadAndHallResepsiSave(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()

	res, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-13"), 550, 550))
	if err != nil {
		t.Fatalf("two sessions of 550 fit a hall of 600 standing: %v", err)
	}
	if res.KindName != "Resepsi" || res.IsMain || len(res.Sessions) != 2 || res.TotalGuests() != 1100 || res.VenueSupplierID == nil {
		t.Errorf("resepsi as saved: %+v", res)
	}
	list, err := events.List(ctx, 1, p.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("two events: %+v %v", list, err)
	}
	akad := list[0]
	if akad.KindName != "Akad" || !akad.IsMain || akad.Layout != domain.LayoutSeated || len(akad.Sessions) != 1 || akad.Sessions[0].Guests != 120 {
		t.Errorf("akad as saved: %+v", akad)
	}
	venues := venueSuppliers(db.suppliers.rows)
	masjid, gedung := venues[venueMasjid], venues[venueGedung]
	if masjid.Name != "Masjid Al-Ikhlas" || masjid.CapacitySeated == nil || *masjid.CapacitySeated != 150 || akad.VenueSupplierID == nil || *akad.VenueSupplierID != masjid.ID {
		t.Errorf("the mosque is the akad's venue, its capacity copied: %+v / %+v", masjid, akad)
	}
	if gedung.Name != "Gedung Melati" || gedung.CapacityStanding == nil || *gedung.CapacityStanding != 600 || *res.VenueSupplierID != gedung.ID {
		t.Errorf("the hall is the resepsi's venue, its capacity copied: %+v", gedung)
	}
	if !db.project.EventDate.Equal(ymd("2027-06-12")) {
		t.Errorf("the project's date stays the main event's: %s", db.project.EventDate.Format("2006-01-02"))
	}
}

// Tahap 2: a session of 650 at that hall is refused with a numbered message.
func TestASessionAboveTheVenueIsRefusedWithTheNumbers(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()
	before := len(db.suppliers.rows)

	_, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-13"), 550, 650))
	f := validationFields(t, err)
	msgs := f["sessions.1.guests"]
	if len(msgs) != 1 || len(f) != 1 {
		t.Fatalf("only the second session is refused: %v", f)
	}
	if !strings.Contains(msgs[0], "650") || !strings.Contains(msgs[0], "600") || !strings.Contains(msgs[0], "Gedung Melati") {
		t.Errorf("the message names the venue and both numbers: %q", msgs[0])
	}
	if want := "Gedung Melati muat 600 orang berdiri, Sesi 2 Resepsi 650"; msgs[0] != want {
		t.Errorf("message = %q, want %q", msgs[0], want)
	}
	if list, _ := events.List(ctx, 1, p.ID); len(list) != 1 || len(db.suppliers.rows) != before {
		t.Errorf("nothing is written: %d events, %d suppliers", len(list), len(db.suppliers.rows))
	}

	// The same rule when the project is created with its events.
	svc, cdb, _ := newFollow()
	_, err = svc.CreateAtomic(ctx, 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, ProjectInput{Name: "X", PrepStartDate: ymd("2026-11-15")},
		ProjectSetup{Events: []EventInput{akadAtMasjid(ymd("2027-06-12")), resepsiAtGedung(ymd("2027-06-13"), 650)}})
	if f := validationFields(t, err); len(f["events.1.sessions.0.guests"]) == 0 || !strings.Contains(f["events.1.sessions.0.guests"][0], "650") {
		t.Errorf("keyed per event on create: %v", f)
	}
	if cdb.project.ID != 0 || len(cdb.events.rows) != 0 {
		t.Error("no project is created")
	}
}

// Tahap 2: replacing the hall with a venue holding 500 is refused.
func TestReplacingTheHallWithASmallerVenueIsRefused(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()
	res, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-13"), 550, 550))
	if err != nil {
		t.Fatal(err)
	}
	before := len(db.suppliers.rows)

	in := resepsiAtGedung(ymd("2027-06-13"), 550, 550)
	in.VenueID = i64p(venueKecil) // Aula Kenanga, 500 standing
	_, err = events.Update(ctx, 1, p.ID, res.ID, 10, in, nil)
	f := validationFields(t, err)
	if len(f["sessions.0.guests"]) == 0 || len(f["sessions.1.guests"]) == 0 || !strings.Contains(f["sessions.0.guests"][0], "500") {
		t.Fatalf("both sessions are above 500: %v", f)
	}
	if want := "Aula Kenanga muat 500 orang berdiri, Sesi 1 Resepsi 550"; f["sessions.0.guests"][0] != want {
		t.Errorf("message = %q, want %q", f["sessions.0.guests"][0], want)
	}
	got, _ := events.Get(ctx, 1, p.ID, res.ID)
	if got.VenueSupplierID == nil || *got.VenueSupplierID != *res.VenueSupplierID || len(db.suppliers.rows) != before {
		t.Errorf("the event keeps its hall and no venue is added: %+v, %d suppliers", got, len(db.suppliers.rows))
	}

	// With guests that fit, the venue changes and the hall, which nothing else
	// uses or hangs off, leaves the project.
	in.Sessions = []SessionInput{session("Sesi 1", 450)}
	got, err = events.Update(ctx, 1, p.ID, res.ID, 10, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	venues := venueSuppliers(db.suppliers.rows)
	if _, still := venues[venueGedung]; still || got.VenueSupplierID == nil || *got.VenueSupplierID != venues[venueKecil].ID {
		t.Errorf("hall released, aula chosen: %+v / %+v", got, db.suppliers.rows)
	}
}

// A venue the event stops using stays when a payment already hangs off it.
func TestAReleasedVenueWithAPaymentStays(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()
	res, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-13"), 300))
	if err != nil {
		t.Fatal(err)
	}
	db.suppliers.pays.rows = append(db.suppliers.pays.rows, domain.SupplierPayment{ID: 1, ProjectID: p.ID, ProjectSupplierID: *res.VenueSupplierID, Payer: domain.PayerWO, Type: domain.PaymentDP, Amount: 1})
	in := resepsiAtGedung(ymd("2027-06-13"), 300)
	in.VenueID = i64p(venueKecil)
	if _, err := events.Update(ctx, 1, p.ID, res.ID, 10, in, nil); err != nil {
		t.Fatal(err)
	}
	if _, still := venueSuppliers(db.suppliers.rows)[venueGedung]; !still || len(db.suppliers.hardDeleted) != 0 {
		t.Errorf("the paid hall stays on the project: %+v", db.suppliers.rows)
	}
}

// Tahap 2: choosing a venue for an event adds it to the project's suppliers, in
// the same transaction; choosing it again for another event reuses it.
func TestChoosingAVenueAddsItToTheProjectSuppliers(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()
	sup := NewSupplierService(db.suppliers, nil, db.events, nil, nil)
	before, _ := sup.List(ctx, p.ID)

	res, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-13"), 400))
	if err != nil {
		t.Fatal(err)
	}
	after, _ := sup.List(ctx, p.ID)
	if len(after) != len(before)+1 {
		t.Fatalf("one supplier added: %d -> %d", len(before), len(after))
	}
	added := after[len(after)-1]
	if added.ID != *res.VenueSupplierID || added.Kind != domain.SupplierVenue || added.SupplierID != venueGedung ||
		added.PaidBy != domain.PayerWO || added.EngagementStatus != domain.EngagementPlanned || added.ProjectID != p.ID {
		t.Errorf("the hall is a venue supplier the WO pays, planned: %+v", added)
	}
	if db.tx.began == 0 || db.tx.locked == 0 {
		t.Errorf("written in a transaction holding the project lock: %+v", db.tx)
	}

	// A second event at the same hall uses the same supplier.
	dup, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-06-14"), 200))
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := sup.List(ctx, p.ID); len(again) != len(after) || *dup.VenueSupplierID != added.ID {
		t.Errorf("the hall is not added twice: %d suppliers, venue %d", len(again), *dup.VenueSupplierID)
	}

	// A failing write rolls the new venue back with the event.
	db.events.failCreate = true
	if _, err := events.Create(ctx, 1, p.ID, 10, EventInput{EventKindID: kindResepsi, EventDate: ymd("2027-06-15"), VenueID: i64p(venueKecil),
		Sessions: []SessionInput{session("", 100)}}); err == nil {
		t.Fatal("must fail")
	}
	if _, added := venueSuppliers(db.suppliers.rows)[venueKecil]; added {
		t.Error("the venue of an event that was not saved is not kept")
	}
}

// One main event per project, whose date is the project's event date.
func TestTheMainEventCarriesTheProjectDate(t *testing.T) {
	events, db, p := newEventProject(t)
	ctx := context.Background()
	res, err := events.Create(ctx, 1, p.ID, 10, resepsiAtGedung(ymd("2027-07-01"), 300))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := events.SetMain(ctx, 1, p.ID, res.ID, 10); err != nil {
		t.Fatal(err)
	}
	mains := 0
	for _, e := range db.events.rows {
		if e.IsMain {
			mains++
		}
	}
	if mains != 1 || !db.project.EventDate.Equal(ymd("2027-07-01")) {
		t.Errorf("one main event, the project follows it: %d mains, %s", mains, db.project.EventDate.Format("2006-01-02"))
	}
	// Deleting the main event makes the earliest remaining one main.
	if err := events.Delete(ctx, 1, p.ID, res.ID, 10); err != nil {
		t.Fatal(err)
	}
	if m := domain.MainEvent(db.events.rows); m == nil || m.KindName != "Akad" || !db.project.EventDate.Equal(ymd("2027-06-12")) {
		t.Errorf("akad is main again: %+v, %s", m, db.project.EventDate.Format("2006-01-02"))
	}
	// The last event cannot go.
	if err := events.Delete(ctx, 1, p.ID, db.events.rows[0].ID, 10); codeOf(err) != "last_event" {
		t.Errorf("want last_event, got %v", err)
	}
}
