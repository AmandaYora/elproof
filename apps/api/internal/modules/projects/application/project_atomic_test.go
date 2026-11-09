package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
)

// ---- fakes for creating a project ----

type e7DB struct {
	projects   []domain.Project
	milestones int
	terms      []domain.PaymentTerm
	activity   *memActivity
	failTerms  bool
}

func (d *e7DB) snapshot() func() {
	p, m, tm := append([]domain.Project(nil), d.projects...), d.milestones, append([]domain.PaymentTerm(nil), d.terms...)
	act := append([]domain.ActivityLogEntry(nil), d.activity.entries...)
	return func() { d.projects, d.milestones, d.terms, d.activity.entries = p, m, tm, act }
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
	if r.db.failTerms {
		return apperror.Internal("disk full")
	}
	r.db.terms = terms
	return nil
}

// e7Venues knows the venues of each workspace.
type e7Venues struct {
	byTenant map[int64]map[int64]vendorscontracts.VenueAttachInfo
}

func (v e7Venues) GetVenueSummary(context.Context, int64, int64) (vendorscontracts.VenueSummary, error) {
	return vendorscontracts.VenueSummary{}, nil
}
func (v e7Venues) GetVenueForAttach(_ context.Context, tenantID, id int64) (vendorscontracts.VenueAttachInfo, error) {
	if info, ok := v.byTenant[tenantID][id]; ok {
		return info, nil
	}
	return vendorscontracts.VenueAttachInfo{}, apperror.NotFound("Venue tidak ditemukan")
}

func newE7() (*ProjectService, *e7DB) {
	db := &e7DB{activity: &memActivity{}}
	act := NewActivityService(activityRepoAdapter{db.activity})
	svc := NewProjectService(e7Projects{db: db}, e7Milestones{db: db}, e7Templates{}, nil, nil, nil, nil, nil, nil, act,
		staffByTenant{1: {10: "Ani"}, 2: {20: "Budi"}})
	svc.SetVenueResolver(e7Venues{byTenant: map[int64]map[int64]vendorscontracts.VenueAttachInfo{
		1: {5: {ID: 5, Name: "Gedung Sabuga", RentalPrice: 10_000_000, Charge: 500_000}},
		2: {6: {ID: 6, Name: "Venue lawan"}},
	}})
	svc.SetAtomicSupport(e7Tx{db: db}, e7Terms{db: db})
	return svc, db
}

func p64(n int64) *int64 { return &n }

func baseInput() ProjectInput {
	return ProjectInput{
		Name: "Pernikahan A & B", EventDate: time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC), Venue: "-",
		PrepStartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), ContractValue: 100_000_000, PICStaffID: 10,
	}
}

func day7(d int) time.Time { return time.Date(2026, 12, d, 0, 0, 0, 0, time.UTC) }

func TestCreateAtomicMakesProjectVenueAndScheduleTogether(t *testing.T) {
	svc, db := newE7()
	in := baseInput()
	in.VenueID = p64(5) // prices omitted: the venue's current prices
	p, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, in, ProjectSetup{Terms: []PaymentTermInput{
		{Label: "DP", Amount: 30_000_000, DueDate: day7(1)}, {Label: "Pelunasan", Amount: 70_000_000, DueDate: day7(20)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(db.projects) != 1 || db.milestones != 2 || len(db.terms) != 2 {
		t.Fatalf("project, default milestones and schedule: %d/%d/%d", len(db.projects), db.milestones, len(db.terms))
	}
	if p.VenueID == nil || *p.VenueID != 5 || *p.VenueRentalPrice != 10_000_000 || *p.VenueCharge != 500_000 || p.Venue != "Gedung Sabuga" || p.Status != domain.StatusDraft {
		t.Errorf("venue attached with its prices and name: %+v", p)
	}
	for _, tm := range db.terms {
		if tm.ProjectID != p.ID {
			t.Errorf("terms belong to the new project: %+v", tm)
		}
	}
	// Prices that are sent win over the venue's (0 included).
	in.VenueRentalPrice, in.VenueCharge = p64(9_000_000), p64(0)
	p, err = svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, in, ProjectSetup{})
	if err != nil || *p.VenueRentalPrice != 9_000_000 || *p.VenueCharge != 0 {
		t.Errorf("explicit prices: %+v %v", p, err)
	}
}

func TestCreateAtomicReportsEveryProblemAtOnceAndCreatesNothing(t *testing.T) {
	svc, db := newE7()
	in := baseInput()
	in.PICStaffID = 20  // another workspace's staff
	in.VenueID = p64(6) // another workspace's venue
	in.ContractValue = 50_000_000
	_, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, in, ProjectSetup{
		FieldErrors: map[string][]string{"eventDate": {"Gunakan format YYYY-MM-DD"}, "terms[1].dueDate": {"Gunakan format YYYY-MM-DD"}},
		Terms: []PaymentTermInput{
			{Label: "DP", Amount: 30_000_000, DueDate: day7(1)},
			{Label: "", Amount: 30_000_000, DueDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
			{Label: "Pelunasan", Amount: 0, DueDate: day7(20)},
		},
	})
	f := fieldsOf(t, err)
	want := "eventDate,picStaffId,terms[1].dueDate,terms[1].label,terms[2].amount,venueId"
	if got := strings.Join(sortedKeys(f), ","); got != want {
		t.Errorf("keys: got %s\nwant %s", got, want)
	}
	if ae, _ := apperror.As(err); !strings.Contains(ae.Message, "isian lain") {
		t.Errorf("the message says how many more there are: %q", ae.Message)
	}
	if len(db.projects) != 0 || db.milestones != 0 || len(db.terms) != 0 || len(db.activity.entries) != 0 {
		t.Errorf("nothing may be created: %d/%d/%d", len(db.projects), db.milestones, len(db.terms))
	}
}

func TestCreateAtomicScheduleRules(t *testing.T) {
	svc, db := newE7()
	actor := ProjectActor{StaffID: 10, CanAssignPIC: true}
	// Sum above the contract value.
	_, err := svc.CreateAtomic(context.Background(), 1, actor, baseInput(), ProjectSetup{Terms: []PaymentTermInput{
		{Label: "a", Amount: 60_000_000, DueDate: day7(1)}, {Label: "b", Amount: 60_000_000, DueDate: day7(2)},
	}})
	if f := fieldsOf(t, err); len(f["terms"]) == 0 {
		t.Errorf("sum above contract: %v", f)
	}
	// More than 20.
	many := make([]PaymentTermInput, 21)
	for i := range many {
		many[i] = PaymentTermInput{Label: "t", Amount: 1, DueDate: day7(1)}
	}
	_, err = svc.CreateAtomic(context.Background(), 1, actor, baseInput(), ProjectSetup{Terms: many})
	if f := fieldsOf(t, err); len(f["terms"]) == 0 {
		t.Errorf("more than 20: %v", f)
	}
	// Negative price and a venue that does not exist, with no schedule.
	in := baseInput()
	in.VenueID, in.VenueRentalPrice, in.VenueCharge = p64(5), p64(-1), p64(-5)
	_, err = svc.CreateAtomic(context.Background(), 1, actor, in, ProjectSetup{})
	if got := strings.Join(sortedKeys(fieldsOf(t, err)), ","); got != "venueCharge,venueRentalPrice" {
		t.Errorf("prices: %s", got)
	}
	if len(db.projects) != 0 {
		t.Error("nothing created")
	}
	// Contract value 0 means "not set": the schedule is not capped.
	in = baseInput()
	in.ContractValue = 0
	if _, err := svc.CreateAtomic(context.Background(), 1, actor, in, ProjectSetup{Terms: []PaymentTermInput{{Label: "a", Amount: 5, DueDate: day7(1)}}}); err != nil {
		t.Errorf("no contract value, no cap: %v", err)
	}
}

func TestCreateAtomicRollsBackWhenAWriteFails(t *testing.T) {
	svc, db := newE7()
	db.failTerms = true
	_, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, baseInput(), ProjectSetup{Terms: []PaymentTermInput{
		{Label: "DP", Amount: 1_000, DueDate: day7(1)},
	}})
	if err == nil {
		t.Fatal("must fail")
	}
	if len(db.projects) != 0 || db.milestones != 0 || len(db.activity.entries) != 0 {
		t.Errorf("a failed schedule write must take the project and its milestones with it: %d/%d", len(db.projects), db.milestones)
	}
}

func TestCreateWithoutSetupIsUnchanged(t *testing.T) {
	svc, db := newE7()
	p, err := svc.Create(context.Background(), 1, ProjectActor{StaffID: 10}, baseInput())
	if err != nil || p.VenueID != nil || p.VenueRentalPrice != nil || len(db.projects) != 1 || len(db.terms) != 0 {
		t.Errorf("plain create: %+v %v", p, err)
	}
	// A person limited to their own projects still becomes PIC Sales.
	in := baseInput()
	in.PICStaffID = 0
	p, _ = svc.Create(context.Background(), 1, ProjectActor{StaffID: 10, AssignedOnly: true}, in)
	if p.PICSalesStaffID != 10 {
		t.Errorf("assigned-only creator becomes PIC Sales: %+v", p)
	}
}
