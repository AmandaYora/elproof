package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// ---- fakes for creating a project ----

type e7DB struct {
	projects   []domain.Project
	milestones int
	terms      []domain.PaymentTerm
	events     *memEvents
	suppliers  *memSuppliers
	activity   *memActivity
	// failMilestones makes the default steps' write fail (after the project
	// and its events are written).
	failMilestones bool
}

func (d *e7DB) snapshot() func() {
	p, m, tm := append([]domain.Project(nil), d.projects...), d.milestones, append([]domain.PaymentTerm(nil), d.terms...)
	act := append([]domain.ActivityLogEntry(nil), d.activity.entries...)
	undoEvents, undoSuppliers := d.events.snapshot(), d.suppliers.snapshot()
	return func() {
		d.projects, d.milestones, d.terms, d.activity.entries = p, m, tm, act
		undoEvents()
		undoSuppliers()
	}
}

type e7Tx struct{ db *e7DB }

func (t e7Tx) InTx(ctx context.Context, fn func(context.Context) error) error {
	undo := t.db.snapshot()
	if err := fn(ctx); err != nil {
		undo()
		return err
	}
	return nil
}
func (e7Tx) LockProject(context.Context, int64) error { return nil }

type e7Projects struct {
	ProjectRepository
	db *e7DB
}

func (r e7Projects) Create(_ context.Context, p *domain.Project) error {
	p.ID = int64(len(r.db.projects) + 1)
	r.db.projects = append(r.db.projects, *p)
	return nil
}

type e7Milestones struct {
	MilestoneRepository
	db *e7DB
}

func (r e7Milestones) Create(context.Context, *domain.ProjectMilestone) error {
	if r.db.failMilestones {
		return apperror.Internal("disk full")
	}
	r.db.milestones++
	return nil
}

type e7Templates struct{ MilestoneTemplateRepository }

func (e7Templates) List(context.Context, int64) ([]domain.ProjectMilestoneTemplate, error) {
	return []domain.ProjectMilestoneTemplate{{Name: "Fitting", DaysBeforeEvent: 30}, {Name: "Gladi resik", DaysBeforeEvent: 7}}, nil
}

type e7Terms struct {
	PaymentTermRepository
	db *e7DB
}

func (r e7Terms) ReplaceAll(_ context.Context, _ int64, terms []domain.PaymentTerm) error {
	r.db.terms = terms
	return nil
}

func newE7() (*ProjectService, *e7DB) {
	db := &e7DB{activity: &memActivity{}, events: &memEvents{}, suppliers: &memSuppliers{}}
	act := NewActivityService(activityRepoAdapter{db.activity})
	svc := NewProjectService(e7Projects{db: db}, e7Milestones{db: db}, e7Templates{}, nil, nil, nil, nil, nil, nil, act,
		staffByTenant{1: {10: "Ani"}, 2: {20: "Budi"}})
	tx := e7Tx{db: db}
	svc.SetAtomicSupport(tx, e7Terms{db: db})
	wireEventsWith(svc, db.events, db.suppliers, tx, act)
	return svc, db
}

func baseInput() ProjectInput {
	return ProjectInput{
		Name: "Pernikahan A & B", PrepStartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), PICStaffID: 10,
	}
}

var (
	akadDay    = time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	resepsiDay = time.Date(2027, 3, 2, 0, 0, 0, 0, time.UTC)
)

// twoEvents is the acceptance example: Akad at a mosque, Resepsi at a hall.
func twoEvents() []EventInput {
	return []EventInput{akadAtMasjid(akadDay), resepsiAtGedung(resepsiDay, 550, 550)}
}

func day7(d int) time.Time { return time.Date(2026, 12, d, 0, 0, 0, 0, time.UTC) }

func TestCreateAtomicMakesProjectEventsAndVenuesTogether(t *testing.T) {
	svc, db := newE7()
	p, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, baseInput(), ProjectSetup{
		Events: twoEvents(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// The schedule comes after the Rincian (Tahap 3): none at creation.
	if len(db.projects) != 1 || db.milestones != 2 || len(db.terms) != 0 || len(db.events.rows) != 2 || len(db.suppliers.rows) != 2 {
		t.Fatalf("project, default milestones, no schedule, events and venues: %d/%d/%d/%d/%d",
			len(db.projects), db.milestones, len(db.terms), len(db.events.rows), len(db.suppliers.rows))
	}
	if !p.EventDate.Equal(akadDay) || p.Status != domain.StatusDraft {
		t.Errorf("the project's date is the first (main) event's: %+v", p)
	}
	main := domain.MainEvent(db.events.rows)
	if main == nil || main.KindName != "Akad" || main.ProjectID != p.ID || db.events.rows[1].IsMain {
		t.Errorf("the first event is the one main event: %+v", db.events.rows)
	}
	for _, sup := range db.suppliers.rows {
		if sup.Kind != domain.SupplierVenue || sup.ProjectID != p.ID || sup.PaidBy != domain.PayerWO {
			t.Errorf("each chosen venue becomes a venue supplier the WO pays: %+v", sup)
		}
	}
	// Two events at the same venue share one venue supplier.
	_, err = svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, baseInput(), ProjectSetup{
		Events: []EventInput{resepsiAtGedung(akadDay, 300), resepsiAtGedung(resepsiDay, 400)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(db.suppliers.rows) != 3 || len(db.events.rows) != 4 {
		t.Fatalf("one venue supplier for both events: %d suppliers, %d events", len(db.suppliers.rows), len(db.events.rows))
	}
	if a, b := db.events.rows[2].VenueSupplierID, db.events.rows[3].VenueSupplierID; a == nil || b == nil || *a != *b || *a != db.suppliers.rows[2].ID {
		t.Errorf("both events point at the same venue supplier: %v %v", a, b)
	}
}

func TestCreateAtomicReportsEveryProblemAtOnceAndCreatesNothing(t *testing.T) {
	svc, db := newE7()
	in := baseInput()
	in.PICStaffID = 20 // another workspace's staff
	events := twoEvents()
	events[0].VenueID = i64p(venueOtherTeam)       // another workspace's venue
	events[1].Sessions[1] = session("Sesi 2", 650) // above the hall
	_, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, in, ProjectSetup{
		Events:      events,
		FieldErrors: map[string][]string{"events.1.eventDate": {"Gunakan format YYYY-MM-DD"}},
	})
	f := fieldsOf(t, err)
	want := "events.0.venueId,events.1.eventDate,events.1.sessions.1.guests,picStaffId"
	if got := strings.Join(sortedKeys(f), ","); got != want {
		t.Errorf("keys: got %s\nwant %s", got, want)
	}
	if ae, _ := apperror.As(err); !strings.Contains(ae.Message, "isian lain") {
		t.Errorf("the message says how many more there are: %q", ae.Message)
	}
	if len(db.projects) != 0 || db.milestones != 0 || len(db.terms) != 0 || len(db.activity.entries) != 0 ||
		len(db.events.rows) != 0 || len(db.suppliers.rows) != 0 {
		t.Errorf("nothing may be created: %d/%d/%d/%d/%d", len(db.projects), db.milestones, len(db.terms), len(db.events.rows), len(db.suppliers.rows))
	}
}

func TestCreateAtomicEventRules(t *testing.T) {
	svc, db := newE7()
	actor := ProjectActor{StaffID: 10, CanAssignPIC: true}
	// At least one event.
	_, err := svc.CreateAtomic(context.Background(), 1, actor, baseInput(), ProjectSetup{})
	if f := fieldsOf(t, err); len(f["events"]) == 0 {
		t.Errorf("no event: %v", f)
	}
	// An event kind of another workspace, a missing date, no session, an inactive venue.
	bad := []EventInput{
		{EventKindID: 9, EventDate: akadDay, Sessions: []SessionInput{session("", 10)}},
		{EventKindID: kindResepsi, VenueID: i64p(venueNonaktif)},
	}
	_, err = svc.CreateAtomic(context.Background(), 1, actor, baseInput(), ProjectSetup{Events: bad})
	want := "events.0.eventKindId,events.1.eventDate,events.1.sessions,events.1.venueId"
	if got := strings.Join(sortedKeys(fieldsOf(t, err)), ","); got != want {
		t.Errorf("keys: got %s\nwant %s", got, want)
	}
	if len(db.projects) != 0 || len(db.events.rows) != 0 {
		t.Error("nothing created")
	}
}

// Tahap 3: a project is created without a contract: its value comes from the
// Rincian added afterwards. Whether the workspace is PKP is copied at creation,
// so a later change of the workspace setting does not reprice it.
func TestCreateAtomicCopiesIsPKPAndStartsWithoutAContract(t *testing.T) {
	svc, db := newE7()
	svc.SetWorkspaceProfile(fixedProfile{pkp: true})
	p, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, baseInput(), ProjectSetup{Events: twoEvents()})
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsPKP || !db.projects[0].IsPKP {
		t.Errorf("is_pkp copied from the workspace: %+v", p)
	}
	if p.ContractSubtotal != 0 || p.ContractTax != 0 || p.ContractTotal != 0 || len(db.terms) != 0 {
		t.Errorf("no contract and no schedule before the Rincian: %+v, %d terms", p, len(db.terms))
	}
	svc.SetWorkspaceProfile(fixedProfile{pkp: false})
	p, err = svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, baseInput(), ProjectSetup{Events: twoEvents()})
	if err != nil || p.IsPKP {
		t.Errorf("a workspace that is not PKP: %+v %v", p, err)
	}
}

type fixedProfile struct{ pkp bool }

func (f fixedProfile) WorkspaceProfile(context.Context, int64) (WorkspaceProfile, error) {
	return WorkspaceProfile{IsPKP: f.pkp}, nil
}

func TestCreateAtomicRollsBackWhenAWriteFails(t *testing.T) {
	svc, db := newE7()
	db.failMilestones = true
	_, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, baseInput(), ProjectSetup{
		Events: twoEvents(),
	})
	if err == nil {
		t.Fatal("must fail")
	}
	if len(db.projects) != 0 || db.milestones != 0 || len(db.activity.entries) != 0 || len(db.events.rows) != 0 || len(db.suppliers.rows) != 0 {
		t.Errorf("a failed write must take the project, its milestones, events and venues with it: %d/%d/%d/%d",
			len(db.projects), db.milestones, len(db.events.rows), len(db.suppliers.rows))
	}
}

func TestCreateNeedsAnEventAndKeepsThePICRules(t *testing.T) {
	svc, db := newE7()
	if _, err := svc.Create(context.Background(), 1, ProjectActor{StaffID: 10}, baseInput()); len(fieldsOf(t, err)["events"]) == 0 {
		t.Errorf("a project without an event is refused: %v", err)
	}
	p, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10}, baseInput(), ProjectSetup{Events: twoEvents()})
	if err != nil || len(db.projects) != 1 || len(db.terms) != 0 {
		t.Errorf("no schedule: %+v %v", p, err)
	}
	// A person limited to their own projects still becomes PIC Sales.
	in := baseInput()
	in.PICStaffID = 0
	p, _ = svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, AssignedOnly: true}, in, ProjectSetup{Events: twoEvents()})
	if p.PICSalesStaffID != 10 {
		t.Errorf("assigned-only creator becomes PIC Sales: %+v", p)
	}
}
