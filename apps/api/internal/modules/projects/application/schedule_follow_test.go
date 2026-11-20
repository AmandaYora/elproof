package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// ---- fakes: one project with its Timeline steps, schedule and client payments ----

type followDB struct {
	project    domain.Project
	milestones []domain.ProjectMilestone
	terms      []domain.PaymentTerm
	payments   []domain.ClientPayment
	patched    []string // "<milestone id>:<keys>" of every milestone write
	events     *memEvents
	suppliers  *memSuppliers
	eventSvc   *EventService
	tx         *followTx
	// lines is the Rincian (line_service_test.go); nil elsewhere.
	lines *memLines
}

// followTx is a Transactor over followDB: a failed InTx puts the project, its
// steps, events and suppliers back.
type followTx struct {
	db            *followDB
	began, locked int
}

func (t *followTx) InTx(ctx context.Context, fn func(context.Context) error) error {
	t.began++
	p, ms := t.db.project, append([]domain.ProjectMilestone(nil), t.db.milestones...)
	undoEvents, undoSuppliers := t.db.events.snapshot(), t.db.suppliers.snapshot()
	undoLines := func() {}
	if t.db.lines != nil {
		undoLines = t.db.lines.snapshot()
	}
	if err := fn(ctx); err != nil {
		t.db.project, t.db.milestones = p, ms
		undoEvents()
		undoSuppliers()
		undoLines()
		return err
	}
	return nil
}
func (t *followTx) LockProject(context.Context, int64) error {
	t.locked++
	return nil
}

type followProjects struct {
	ProjectRepository
	db *followDB
}

func (r followProjects) FindByID(_ context.Context, tenantID, id int64) (*domain.Project, error) {
	if tenantID != r.db.project.TenantID || id != r.db.project.ID {
		return nil, nil
	}
	p := r.db.project
	return &p, nil
}
func (r followProjects) Create(_ context.Context, p *domain.Project) error {
	p.ID = 1
	r.db.project = *p
	return nil
}
func (r followProjects) Patch(_ context.Context, p *domain.Project, _ []string, _ *time.Time) (bool, error) {
	r.db.project = *p
	return true, nil
}

type followMilestones struct {
	MilestoneRepository
	db *followDB
}

func (r followMilestones) ListByProject(context.Context, int64) ([]domain.ProjectMilestone, error) {
	return append([]domain.ProjectMilestone(nil), r.db.milestones...), nil
}
func (r followMilestones) FindByID(_ context.Context, _, id int64) (*domain.ProjectMilestone, error) {
	for _, m := range r.db.milestones {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, nil
}
func (r followMilestones) Create(_ context.Context, m *domain.ProjectMilestone) error {
	m.ID = int64(len(r.db.milestones) + 1)
	r.db.milestones = append(r.db.milestones, *m)
	return nil
}
func (r followMilestones) Patch(_ context.Context, m *domain.ProjectMilestone, changed []string, _ *time.Time) (bool, error) {
	for i := range r.db.milestones {
		if r.db.milestones[i].ID == m.ID {
			r.db.milestones[i] = *m
		}
	}
	r.db.patched = append(r.db.patched, formatID(m.ID)+":"+strings.Join(changed, ","))
	return true, nil
}

type followTemplates struct{ MilestoneTemplateRepository }

func (followTemplates) List(context.Context, int64) ([]domain.ProjectMilestoneTemplate, error) {
	return []domain.ProjectMilestoneTemplate{
		{Name: "Survei Venue & Vendor", DaysBeforeEvent: 90},
		{Name: "Technical Meeting", DaysBeforeEvent: 14},
		{Name: "Hari-H Pernikahan", DaysBeforeEvent: 0},
	}, nil
}

type followTerms struct {
	PaymentTermRepository
	db *followDB
}

func (r followTerms) ListByProject(context.Context, int64) ([]domain.PaymentTerm, error) {
	return append([]domain.PaymentTerm(nil), r.db.terms...), nil
}
func (r followTerms) ReplaceAll(_ context.Context, _ int64, terms []domain.PaymentTerm) error {
	r.db.terms = append([]domain.PaymentTerm(nil), terms...)
	for i := range r.db.terms {
		r.db.terms[i].ID = int64(100 + i)
	}
	return nil
}

type followPayments struct {
	ClientPaymentRepository
	db *followDB
}

func (r followPayments) ListByProject(context.Context, int64) ([]domain.ClientPayment, error) {
	return r.db.payments, nil
}

func ymd(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

// newFollow is a project service of workspace 1 with its event service (see
// wireEventsWith), all in memory.
func newFollow() (*ProjectService, *followDB, *memActivity) {
	db := &followDB{events: &memEvents{}, suppliers: &memSuppliers{pays: &memSupplierPayments{}}}
	db.tx = &followTx{db: db}
	mem := &memActivity{}
	act := NewActivityService(activityRepoAdapter{mem})
	svc := NewProjectService(followProjects{db: db}, followMilestones{db: db}, followTemplates{}, nil, nil, nil, nil, nil, nil, act,
		staffByTenant{1: {10: "Ani"}})
	svc.SetAtomicSupport(db.tx, followTerms{db: db})
	db.eventSvc = wireEventsWith(svc, db.events, db.suppliers, db.tx, act)
	return svc, db, mem
}

// akadOn is a one-session Akad with no venue yet.
func akadOn(date time.Time) EventInput {
	return EventInput{EventKindID: kindAkad, EventDate: date, Layout: domain.LayoutSeated, Sessions: []SessionInput{session("", 100)}}
}

func createFollowProject(t *testing.T, svc *ProjectService) *domain.Project {
	t.Helper()
	p, err := svc.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, ProjectInput{
		Name: "Ayu & Bima", BrideName: "Ayu", GroomName: "Bima",
		PrepStartDate: ymd("2026-11-15"),
	}, ProjectSetup{Events: []EventInput{akadOn(ymd("2027-06-12"))}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// moveMainEvent changes the date of the project's main event.
func moveMainEvent(t *testing.T, db *followDB, projectID int64, date time.Time) {
	t.Helper()
	main := domain.MainEvent(db.events.rows)
	if main == nil {
		t.Fatal("no main event")
	}
	if _, err := db.eventSvc.Update(context.Background(), 1, projectID, main.ID, 10, akadOn(date), nil); err != nil {
		t.Fatal(err)
	}
}

func targets(ms []domain.ProjectMilestone) string {
	var out []string
	for _, m := range ms {
		out = append(out, m.TargetDate.Format("2006-01-02"))
	}
	return strings.Join(out, " ")
}

// EP-06: steps are dated from the event date, not the booking date.
func TestNewProjectStepsFollowTheEventDate(t *testing.T) {
	svc, db, _ := newFollow()
	createFollowProject(t, svc)
	if got, want := targets(db.milestones), "2027-03-14 2027-05-29 2027-06-12"; got != want {
		t.Fatalf("targets %s, want %s (H-90, H-14, Hari-H)", got, want)
	}
	for _, m := range db.milestones {
		if m.DaysBeforeEvent == nil {
			t.Errorf("%s must remember its offset", m.Name)
		}
	}
}

// EP-06: moving the main event's date moves the project's date and the steps
// that follow it, not the ones dated by hand or already done.
func TestEventDateChangeShiftsFollowingSteps(t *testing.T) {
	svc, db, mem := newFollow()
	p := createFollowProject(t, svc)
	db.milestones[0].Status = domain.MilestoneCompleted // Survei done: stays
	manual := ymd("2027-05-01")
	db.milestones = append(db.milestones, domain.ProjectMilestone{ID: 4, Name: "Fitting", Status: domain.MilestoneNotStarted, TargetDate: manual})

	moveMainEvent(t, db, p.ID, ymd("2027-08-01"))
	if !db.project.EventDate.Equal(ymd("2027-08-01")) {
		t.Errorf("the project's date is the main event's: %s", db.project.EventDate.Format("2006-01-02"))
	}
	if got, want := targets(db.milestones), "2027-03-14 2027-07-18 2027-08-01 2027-05-01"; got != want {
		t.Fatalf("targets %s, want %s", got, want)
	}
	if strings.Join(db.patched, " ") != "2:targetDate 3:targetDate" {
		t.Errorf("only the moved steps are written: %v", db.patched)
	}
	found := false
	for _, e := range mem.entries {
		if strings.Contains(e.Description, "2 tahapan mengikuti tanggal acara") {
			found = true
		}
	}
	if !found {
		t.Errorf("the move is in the activity log: %+v", mem.entries)
	}

	// A change that does not touch the dates moves nothing.
	db.patched = nil
	if _, err := svc.Patch(context.Background(), 1, p.ID, ProjectActor{StaffID: 10}, ProjectPatch{Name: Some("Ayu & Bima Wedding")}, nil); err != nil {
		t.Fatal(err)
	}
	if len(db.patched) != 0 {
		t.Errorf("no step written for a name change: %v", db.patched)
	}
}

// EP-06: a target set by hand fixes the step; status changes stamp the
// completed date.
func TestPatchMilestoneStampsCompletionAndFixesManualDates(t *testing.T) {
	svc, db, _ := newFollow()
	p := createFollowProject(t, svc)
	today := time.Date(2026, 11, 20, 9, 0, 0, 0, domain.WIB)
	ctx := context.Background()

	m, err := svc.PatchMilestone(ctx, 1, p.ID, 2, 10, MilestonePatch{Status: Some(domain.MilestoneCompleted)}, nil, today)
	if err != nil {
		t.Fatal(err)
	}
	if m.CompletedDate == nil || m.CompletedDate.Format("2006-01-02") != "2026-11-20" {
		t.Fatalf("Completed stamps today (WIB): %v", m.CompletedDate)
	}
	m, err = svc.PatchMilestone(ctx, 1, p.ID, 2, 10, MilestonePatch{Status: Some(domain.MilestoneInProgress)}, nil, today)
	if err != nil {
		t.Fatal(err)
	}
	if m.CompletedDate != nil {
		t.Errorf("leaving Completed clears the date: %v", m.CompletedDate)
	}
	// A completed date sent with the status wins over the stamp.
	sent := ymd("2026-11-18")
	m, _ = svc.PatchMilestone(ctx, 1, p.ID, 2, 10, MilestonePatch{Status: Some(domain.MilestoneCompleted), CompletedDate: Some(sent)}, nil, today)
	if m.CompletedDate == nil || m.CompletedDate.Format("2006-01-02") != "2026-11-18" {
		t.Errorf("sent completed date kept: %v", m.CompletedDate)
	}

	m, err = svc.PatchMilestone(ctx, 1, p.ID, 3, 10, MilestonePatch{TargetDate: Some(ymd("2027-06-13"))}, nil, today)
	if err != nil {
		t.Fatal(err)
	}
	if m.DaysBeforeEvent != nil {
		t.Error("a target set by hand no longer follows the event date")
	}
	moveMainEvent(t, db, p.ID, ymd("2027-07-01"))
	if db.milestones[2].TargetDate.Format("2006-01-02") != "2027-06-13" {
		t.Errorf("the hand-dated step stays put: %s", db.milestones[2].TargetDate.Format("2006-01-02"))
	}
}

// EP-05: recalculating fits the schedule to the new contract value, keeping paid installments.
func TestRecalculateTermsFollowsContractValue(t *testing.T) {
	svc, db, mem := newFollow()
	p := createFollowProject(t, svc)
	// The contract comes from the Rincian (Tahap 3): Rp 100.000.000.
	db.project.ContractSubtotal, db.project.ContractTotal = 100_000_000, 100_000_000
	terms := NewPaymentTermService(followTerms{db: db}, followPayments{db: db}, svc, NewActivityService(activityRepoAdapter{mem}))
	db.terms = []domain.PaymentTerm{
		{ID: 1, Label: "DP", Amount: 30, DueDate: ymd("2026-11-15")},
		{ID: 2, Label: "Termin 1", Amount: 40, DueDate: ymd("2027-04-13")},
		{ID: 3, Label: "Pelunasan", Amount: 30, DueDate: ymd("2027-06-05")},
	}
	db.payments = []domain.ClientPayment{{Type: domain.PaymentType("DP"), Amount: 30}}
	today := time.Date(2026, 11, 20, 9, 0, 0, 0, domain.WIB)

	list, err := terms.Recalculate(context.Background(), 1, p.ID, 10, today)
	if err != nil {
		t.Fatal(err)
	}
	got := []int64{list[0].Term.Amount, list[1].Term.Amount, list[2].Term.Amount}
	// Contract Rp 100.000.000; DP Rp 30 already paid stays; 99.999.970 split 40:30.
	if got[0] != 30 || got[1] != 57_142_840 || got[2] != 42_857_130 {
		t.Fatalf("amounts %v", got)
	}
	if list[0].Status != domain.TermPaid {
		t.Errorf("the paid DP stays paid: %s", list[0].Status)
	}

	// Every installment with money on it: refused with a message saying what to do.
	db.payments = []domain.ClientPayment{{Type: domain.PaymentType("DP"), Amount: 100_000_000}}
	_, err = terms.Recalculate(context.Background(), 1, p.ID, 10, today)
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation || !strings.Contains(ae.Fields["terms"][0], "manual") {
		t.Fatalf("want 422 on terms, got %v", err)
	}
}
