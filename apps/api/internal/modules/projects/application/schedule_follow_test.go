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

func newFollow() (*ProjectService, *followDB, *memActivity) {
	db := &followDB{}
	mem := &memActivity{}
	act := NewActivityService(activityRepoAdapter{mem})
	svc := NewProjectService(followProjects{db: db}, followMilestones{db: db}, followTemplates{}, nil, nil, nil, nil, nil, nil, act,
		staffByTenant{1: {10: "Ani"}})
	return svc, db, mem
}

func createFollowProject(t *testing.T, svc *ProjectService) *domain.Project {
	t.Helper()
	p, err := svc.Create(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, ProjectInput{
		Name: "Ayu & Bima", BrideName: "Ayu", GroomName: "Bima", EventDate: ymd("2027-06-12"), Venue: "-",
		PrepStartDate: ymd("2026-11-15"), PackageName: "Gold", ContractValue: 100_000_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
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

// EP-06: moving the event date moves the steps that follow it, not the ones
// dated by hand or already done.
func TestEventDateChangeShiftsFollowingSteps(t *testing.T) {
	svc, db, mem := newFollow()
	p := createFollowProject(t, svc)
	db.milestones[0].Status = domain.MilestoneCompleted // Survei done: stays
	manual := ymd("2027-05-01")
	db.milestones = append(db.milestones, domain.ProjectMilestone{ID: 4, Name: "Fitting", Status: domain.MilestoneNotStarted, TargetDate: manual})

	_, err := svc.Patch(context.Background(), 1, p.ID, ProjectActor{StaffID: 10, CanAssignPIC: true},
		ProjectPatch{EventDate: Some(ymd("2027-08-01"))}, nil)
	if err != nil {
		t.Fatal(err)
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
	if _, err := svc.Patch(ctx, 1, p.ID, ProjectActor{StaffID: 10}, ProjectPatch{EventDate: Some(ymd("2027-07-01"))}, nil); err != nil {
		t.Fatal(err)
	}
	if db.milestones[2].TargetDate.Format("2006-01-02") != "2027-06-13" {
		t.Errorf("the hand-dated step stays put: %s", db.milestones[2].TargetDate.Format("2006-01-02"))
	}
}

// EP-10: a negative contract value is a 422 on contractValue, on create and on edit.
func TestNegativeContractValueIsRefused(t *testing.T) {
	svc, db, _ := newFollow()
	_, err := svc.Create(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, ProjectInput{
		Name: "X", EventDate: ymd("2027-06-12"), PrepStartDate: ymd("2026-11-15"), ContractValue: -5_000_000,
	})
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation || len(ae.Fields["contractValue"]) == 0 || ae.Fields["contractValue"][0] != "Nilai kontrak tidak boleh negatif" {
		t.Fatalf("create: want 422 on contractValue, got %v", err)
	}
	if len(db.milestones) != 0 {
		t.Error("nothing is written")
	}
	p := createFollowProject(t, svc)
	_, err = svc.Patch(context.Background(), 1, p.ID, ProjectActor{StaffID: 10}, ProjectPatch{ContractValue: Some(int64(-1))}, nil)
	if ae, ok := apperror.As(err); !ok || len(ae.Fields["contractValue"]) == 0 {
		t.Fatalf("edit: want 422 on contractValue, got %v", err)
	}
}

// EP-05: recalculating fits the schedule to the new contract value, keeping paid installments.
func TestRecalculateTermsFollowsContractValue(t *testing.T) {
	svc, db, mem := newFollow()
	p := createFollowProject(t, svc)
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
