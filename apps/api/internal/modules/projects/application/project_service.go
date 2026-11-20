package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
)

type ProjectRepository interface {
	List(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64) ([]domain.Project, error)
	// CountAll backs the dashboard's TotalProjects stat.
	CountAll(ctx context.Context, tenantID int64) (int64, error)
	// ListForDashboard backs the dashboard's trend/near-D-day/upcoming/lagging
	// computations — bounded to rows that could matter for any of them
	// instead of List's entire unbounded tenant history. See the
	// infrastructure implementation's doc comment for `since`'s exact meaning.
	ListForDashboard(ctx context.Context, tenantID int64, since time.Time) ([]domain.Project, error)
	ListPaginated(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64, params pagination.Params, filter domain.ProjectListFilter, sort pagination.Sort) ([]domain.Project, int64, error)
	// ListLabels names a set of the tenant's projects (id, name, event date) in
	// one query; ids of other tenants or that do not exist are absent.
	ListLabels(ctx context.Context, tenantID int64, ids []int64) ([]domain.ProjectLabel, error)
	// ListSortSpec is the ?sort= whitelist (public name -> fixed SQL) and the
	// default order of ListPaginated.
	ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// ListByStaffPIC backs the staff hard-delete "impact" endpoint (PLAN.md)
	// -- see its infrastructure implementation's doc comment.
	ListByStaffPIC(ctx context.Context, tenantID, staffID int64) ([]domain.ProjectRef, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Project, error)
	Create(ctx context.Context, p *domain.Project) error
	// Patch writes the changed fields (domain field keys) of p in one statement; with
	// expected it is a compare-and-set on updated_at (see infrastructure.patchRow).
	Patch(ctx context.Context, p *domain.Project, changed []string, expected *time.Time) (bool, error)
	// ChangeStatus is conditional on the current status (see the MySQL implementation).
	ChangeStatus(ctx context.Context, tenantID, id int64, from, to domain.ProjectStatus) (bool, error)
	// DeleteCascade permanently removes the project and every same-module row
	// referencing it — see the infrastructure implementation's doc comment
	// (ADR-0013) for the exact deletion order and why it matters. found is
	// false when the project is not in the tenant; a foreign key that still
	// refuses is domain.ErrProjectDeleteBlocked.
	DeleteCascade(ctx context.Context, tenantID, id int64) (found bool, err error)
	// CountActive counts projects that are not completed, cancelled or
	// archived (trial cap, PLAN.md K6).
	CountActive(ctx context.Context, tenantID int64) (int, error)
}

// ClientCleaner is the narrow shape ProjectService needs from `clients` to
// best-effort clean up every client tied to a project being hard-deleted —
// deliberately a local interface (not an import of clients/contracts) so
// `projects` never has to import `clients`. Bridged from main.go via
// Module.SetClientCleaner, exactly like the existing presentation-layer
// SetClientAccessResolver two-phase wiring (see projects.module.go and
// handler.go's ClientAccessResolver) — needed because `clients` itself
// depends on `projects.Contracts()`, so `clients` must be built after
// `projects`, meaning `projects` can't take this as a constructor argument.
type ClientCleaner interface {
	DeleteAllForProject(ctx context.Context, tenantID, projectID int64) error
}

// StaffNameResolver is the narrow shape ProjectService needs from `staff` to
// embed a project's PIC name directly in its own response (PLAN.md's Client
// Portal restructure). Deliberately typed with primitives only (not
// staffcontracts.Summary) -- staff/application already imports
// projects/contracts (for its own hard-delete impact-lookup bridge), so
// projects/application importing staff/contracts back would create an
// import cycle (projects/application -> staff/contracts -> staff/application
// -> projects/contracts -> projects/application). main.go bridges the real
// staffcontracts.Contracts into this shape with a small adapter instead.
// Unlike ClientCleaner, `staff` is already built before
// `projects` in main.go, so this is a direct constructor argument, no
// two-phase setter needed.
type StaffNameResolver interface {
	// GetName resolves a staff member's name. ok is false when the ID
	// doesn't resolve (e.g. the staff row was hard-deleted, PLAN.md's Staff
	// hard delete) -- never an error, so a stale pic_staff_id degrades to
	// "no name available" rather than breaking the whole project response.
	GetName(ctx context.Context, tenantID, staffID int64) (name string, ok bool, err error)
	// GetPhone resolves the same member's contact number ("" when none).
	GetPhone(ctx context.Context, tenantID, staffID int64) (phone string, ok bool, err error)
	// StaffNames resolves several names in one query; ids that do not resolve
	// are absent from the map.
	StaffNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
}

type MilestoneRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectMilestone, error)
	// ListByProjects backs ComputeProgressBatch: every matching row across
	// the given projects in one query (WHERE project_id IN (...)).
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectMilestone, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectMilestone, error)
	Create(ctx context.Context, m *domain.ProjectMilestone) error
	// Patch writes the changed fields of a milestone (see ProjectRepository.Patch).
	Patch(ctx context.Context, m *domain.ProjectMilestone, changed []string, expected *time.Time) (bool, error)
	NextSortOrder(ctx context.Context, projectID int64) (int, error)
	Reorder(ctx context.Context, projectID int64, orderedIDs []int64) error
}

type ProjectService struct {
	repo               ProjectRepository
	milestones         MilestoneRepository
	milestoneTemplates MilestoneTemplateRepository
	suppliers          SupplierRepository
	vendorMilestones   VendorMilestoneRepository
	issues             IssueRepository
	payments           SupplierPaymentRepository
	events             EventRepository
	evidence           *EvidenceService
	activity           *ActivityService
	clients            ClientCleaner
	staff              StaffNameResolver
	quota              TrialQuota
	// eventService validates and writes the events of a new project (SetEvents).
	eventService *EventService
	// tx/terms back CreateAtomic (wired by SetAtomicSupport).
	tx    Transactor
	terms PaymentTermRepository
	// listMoney backs the derived columns of the list (SetListMoney).
	listMoney ListMoneySource
	// workspace says whether the workspace is PKP (copied onto a new project).
	workspace WorkspaceProfileSource
	// lines copies a duplicated project's Rincian (SetLines).
	lines *LineService
	// quotations is `offers` for "Jadikan proyek" and the purge mark
	// (SetQuotations, main.go).
	quotations QuotationSource
}

// SetQuotations wires `offers`' quotations (built after this module).
func (s *ProjectService) SetQuotations(q QuotationSource) { s.quotations = q }

// WorkspaceProfile is what a project copies from its workspace when it is
// created (PLAN §3.2): whether the WO is PKP, so its contract carries PPN.
type WorkspaceProfile struct {
	IsPKP bool
}

// WorkspaceProfileSource reads it from `platform` (adapted in main.go).
type WorkspaceProfileSource interface {
	WorkspaceProfile(ctx context.Context, tenantID int64) (WorkspaceProfile, error)
}

// SetWorkspaceProfile wires the workspace profile (platform is built first,
// but the adaptor lives in main.go).
func (s *ProjectService) SetWorkspaceProfile(w WorkspaceProfileSource) { s.workspace = w }

// SetLines wires the Rincian (built after this service).
func (s *ProjectService) SetLines(l *LineService) { s.lines = l }

// workspaceIsPKP is the workspace's PKP flag now (false when no profile is
// wired: a contract without PPN is the safe default).
func (s *ProjectService) workspaceIsPKP(ctx context.Context, tenantID int64) (bool, error) {
	if s.workspace == nil {
		return false, nil
	}
	p, err := s.workspace.WorkspaceProfile(ctx, tenantID)
	if err != nil {
		return false, err
	}
	return p.IsPKP, nil
}

// SetAtomicSupport wires what creating a project with its events and payment
// schedule in one transaction needs.
func (s *ProjectService) SetAtomicSupport(tx Transactor, terms PaymentTermRepository) {
	s.tx, s.terms = tx, terms
}

// SetEvents wires the event service (it needs this service back, so it is
// built after it).
func (s *ProjectService) SetEvents(e *EventService) { s.eventService = e }

func NewProjectService(
	repo ProjectRepository,
	milestones MilestoneRepository,
	milestoneTemplates MilestoneTemplateRepository,
	suppliers SupplierRepository,
	vendorMilestones VendorMilestoneRepository,
	issues IssueRepository,
	payments SupplierPaymentRepository,
	events EventRepository,
	evidence *EvidenceService,
	activity *ActivityService,
	staff StaffNameResolver,
) *ProjectService {
	return &ProjectService{
		repo: repo, milestones: milestones, milestoneTemplates: milestoneTemplates, suppliers: suppliers,
		vendorMilestones: vendorMilestones, issues: issues, payments: payments, events: events,
		evidence: evidence, activity: activity, staff: staff,
	}
}

func (s *ProjectService) txOrDirect() Transactor {
	if s.tx == nil {
		return noTransactor{}
	}
	return s.tx
}

// ResolvePICName backs the `picName` field on a project's response (PLAN.md's
// Client Portal restructure) -- returns "" on any failure (unresolved ID,
// nil resolver) rather than erroring, since this is display-only metadata,
// never a required field.
func (s *ProjectService) ResolvePICName(ctx context.Context, tenantID, staffID int64) string {
	if s.staff == nil {
		return ""
	}
	name, ok, err := s.staff.GetName(ctx, tenantID, staffID)
	if err != nil || !ok {
		return ""
	}
	return name
}

// StaffNames resolves the names of the given staff ids in one query, best
// effort like ResolvePICName: a failure, a nil resolver or an id that no longer
// resolves just leaves that id out of the map. Zero ids are skipped and
// duplicates asked once.
func (s *ProjectService) StaffNames(ctx context.Context, tenantID int64, ids []int64) map[int64]string {
	if s.staff == nil {
		return map[int64]string{}
	}
	seen := make(map[int64]bool, len(ids))
	var want []int64
	for _, id := range ids {
		if id != 0 && !seen[id] {
			seen[id] = true
			want = append(want, id)
		}
	}
	if len(want) == 0 {
		return map[int64]string{}
	}
	names, err := s.staff.StaffNames(ctx, tenantID, want)
	if err != nil || names == nil {
		return map[int64]string{}
	}
	return names
}

// ResolvePICPhone is the PIC's contact number for the Client Portal's
// WhatsApp button; "" on any failure, like ResolvePICName.
func (s *ProjectService) ResolvePICPhone(ctx context.Context, tenantID, staffID int64) string {
	if s.staff == nil {
		return ""
	}
	phone, ok, err := s.staff.GetPhone(ctx, tenantID, staffID)
	if err != nil || !ok {
		return ""
	}
	return phone
}

// SetClientCleaner completes the two-phase wiring needed because `clients`
// depends on `projects.Contracts()` (built after `projects`) — see
// ClientCleaner's doc comment. main.go calls this right after clientsModule
// is built, the same slot as SetClientAccessResolver.
func (s *ProjectService) SetClientCleaner(cleaner ClientCleaner) {
	s.clients = cleaner
}

// picStaffID/picSalesStaffID scope results to a single PIC's own projects —
// nil/nil means every project in the tenant (see
// MySQLProjectRepository.List's own doc comment).
func (s *ProjectService) List(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64) ([]domain.Project, error) {
	return s.repo.List(ctx, tenantID, picStaffID, picSalesStaffID)
}

// ListAffectedProjectsForStaff backs the hard-delete "impact" endpoint for
// Staff (PLAN.md) -- see ListByStaffPIC's doc comment.
func (s *ProjectService) ListAffectedProjectsForStaff(ctx context.Context, tenantID, staffID int64) ([]domain.ProjectRef, error) {
	return s.repo.ListByStaffPIC(ctx, tenantID, staffID)
}

// CountAll backs the dashboard's TotalProjects stat.
func (s *ProjectService) CountAll(ctx context.Context, tenantID int64) (int64, error) {
	return s.repo.CountAll(ctx, tenantID)
}

// ListForDashboard backs every other dashboard computation — see
// ProjectRepository.ListForDashboard's doc comment.
func (s *ProjectService) ListForDashboard(ctx context.Context, tenantID int64, since time.Time) ([]domain.Project, error) {
	return s.repo.ListForDashboard(ctx, tenantID, since)
}

// ListPaginated backs the real project list page — showArchived splits the
// result into two disjoint views (active vs. archived), never merged, so
// archived projects stay genuinely out of the way of day-to-day work (see
// ADR-0013) rather than just visually deprioritized in a mixed list.
func (s *ProjectService) ListPaginated(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64, params pagination.Params, filter domain.ProjectListFilter, sort pagination.Sort) ([]domain.Project, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, picStaffID, picSalesStaffID, params, filter, sort)
}

// Labels names the tenant's projects among ids (id -> label); an id that is not
// the tenant's (or does not exist) is simply absent.
func (s *ProjectService) Labels(ctx context.Context, tenantID int64, ids []int64) (map[int64]domain.ProjectLabel, error) {
	out := make(map[int64]domain.ProjectLabel, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	list, err := s.repo.ListLabels(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	for _, l := range list {
		out[l.ID] = l
	}
	return out, nil
}

// ListSortSpec is the sort whitelist and default of the paginated list.
func (s *ProjectService) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.ListSortSpec()
}

// ListMoneySource is where a page of the list reads the raw numbers behind its
// derived columns (see infrastructure.MySQLListMoneyRepository).
type ListMoneySource interface {
	PaymentLinesByProjects(ctx context.Context, projectIDs []int64) (map[int64][]domain.PaymentLine, error)
	TermsByProjects(ctx context.Context, projectIDs []int64) (map[int64][]domain.PaymentTerm, error)
}

// SetListMoney wires the source of the list's derived columns.
func (s *ProjectService) SetListMoney(src ListMoneySource) { s.listMoney = src }

// ListMoney derives `remaining` and `nextDueDate` for every project of a page
// with the domain rules (ComputeListMoney): two queries for the whole page.
// Without a source it answers nothing (the callers then show no derived
// columns), never a wrong number.
func (s *ProjectService) ListMoney(ctx context.Context, projects []domain.Project) (map[int64]domain.ListMoney, error) {
	out := make(map[int64]domain.ListMoney, len(projects))
	if s.listMoney == nil || len(projects) == 0 {
		return out, nil
	}
	ids := make([]int64, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	lines, err := s.listMoney.PaymentLinesByProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	terms, err := s.listMoney.TermsByProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		out[p.ID] = domain.ComputeListMoney(p.ContractTotal, lines[p.ID], terms[p.ID])
	}
	return out, nil
}

func (s *ProjectService) Get(ctx context.Context, tenantID, id int64) (*domain.Project, error) {
	p, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperror.NotFound("Proyek tidak ditemukan")
	}
	return p, nil
}

// ExistsForTenant is used by the `clients` module (via contracts) to validate
// a project_id before creating a client row — no cross-module FK, so this is
// the only way `clients` can be sure the project it's pointed at is real and
// belongs to the same tenant.
func (s *ProjectService) ExistsForTenant(ctx context.Context, tenantID, id int64) (bool, error) {
	p, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return false, err
	}
	return p != nil, nil
}

type ProjectInput struct {
	Name      string
	BrideName string
	GroomName string
	// EventDate is the main event's date. CreateAtomic takes it from the first
	// event; Duplicate shifts the copied events so the main one lands on it.
	EventDate     time.Time
	PrepStartDate time.Time
	Status        domain.ProjectStatus
	PICStaffID    int64
	// PICSalesStaffID is the "PIC Sales" slot -- see Project.PICSalesStaffID's
	// doc comment.
	PICSalesStaffID int64
	Description     string
}

// ProjectActor is the staff member acting and what their workspace role
// allows on projects (ADR-0026). Internal callers use the zero value plus
// CanAssignPIC.
type ProjectActor struct {
	StaffID int64
	// AssignedOnly: the role reaches only projects where the person is PIC
	// or PIC Sales.
	AssignedOnly bool
	// CanAssignPIC: may change PIC / PIC Sales (project.assign_pic).
	CanAssignPIC bool
}

// ProjectSetup is what is saved together with a new project, in the same
// transaction: its events (at least one; the first is the main event). The
// client's payment schedule is made after the Rincian is saved (PLAN §9.1).
type ProjectSetup struct {
	Events []EventInput
	// FieldErrors are problems the caller already found while reading the
	// request (a date that is not YYYY-MM-DD, keyed "events.0.eventDate", ...).
	// They are reported together with everything the service finds, as one
	// validation error.
	FieldErrors map[string][]string
}

// Create: a person limited to their own projects must stay able to reach
// the project they create, so unless they made themselves PIC they become
// its PIC Sales (the former Sales rule, now for any such role).
func (s *ProjectService) Create(ctx context.Context, tenantID int64, actor ProjectActor, input ProjectInput) (*domain.Project, error) {
	return s.CreateAtomic(ctx, tenantID, actor, input, ProjectSetup{})
}

// CreateAtomic creates a project and, in the same database transaction, its
// default milestones and its events (with the venues they bring in). The
// project copies the workspace's PKP flag; its contract starts at 0 until the
// Rincian is saved. Everything is validated first and every problem of every
// part is returned together as ONE validation error (keys "picStaffId",
// "events", "events.1.venueId", ...); nothing is written
// when anything is wrong, and if a write fails the transaction rolls back, so
// no half-made project is ever left. The project's event date is the first
// (main) event's.
// The trial quota and PIC rules are unchanged.
func (s *ProjectService) CreateAtomic(ctx context.Context, tenantID int64, actor ProjectActor, input ProjectInput, setup ProjectSetup) (*domain.Project, error) {
	actorStaffID := actor.StaffID
	picSalesStaffID := input.PICSalesStaffID
	if actor.AssignedOnly && input.PICStaffID != actorStaffID {
		picSalesStaffID = actorStaffID
	}
	// A new project always starts as Draft: later steps are explicit
	// transitions, so creating cannot skip them (ADR-0027).
	input.Status = domain.StatusDraft

	fe := newFieldErrors()
	fe.mergeFields(setup.FieldErrors)
	if err := s.checkPICs(ctx, tenantID, input.PICStaffID, picSalesStaffID); err != nil && !fe.mergeErr(err) {
		return nil, err
	}
	if s.eventService == nil {
		return nil, apperror.Internal("Layanan acara belum tersedia")
	}
	events, err := s.eventService.PrepareForProject(ctx, tenantID, setup.Events, fe)
	if err != nil {
		return nil, err
	}
	if len(events) > 0 {
		input.EventDate = events[0].event.EventDate
	}
	if err := fe.err(); err != nil {
		return nil, err
	}
	if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
		return nil, err
	}
	isPKP, err := s.workspaceIsPKP(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	p := &domain.Project{
		TenantID: tenantID, Name: input.Name, BrideName: input.BrideName, GroomName: input.GroomName,
		EventDate: input.EventDate, PrepStartDate: input.PrepStartDate, IsPKP: isPKP, Status: input.Status,
		PICStaffID: input.PICStaffID, PICSalesStaffID: picSalesStaffID, Description: input.Description,
	}
	err = s.txOrDirect().InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		if err := s.eventService.createForProject(ctx, p.ID, actorStaffID, events); err != nil {
			return err
		}
		if err := s.seedDefaultMilestones(ctx, tenantID, p.ID, p.EventDate, p.PrepStartDate); err != nil {
			return err
		}
		s.activity.Record(ctx, &p.ID, domain.ActivityProjectCreated, actorStaffID, "project", formatID(p.ID), p.Name,
			"Proyek baru dibuat: "+p.Name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// checkPICs confirms every named PIC is a staff member of this workspace (0 =
// none). Without it a body could attach another workspace's staff id to a
// project and read that person's name back through the project's responses.
func (s *ProjectService) checkPICs(ctx context.Context, tenantID int64, ids ...int64) error {
	for _, id := range ids {
		if id == 0 {
			continue
		}
		_, ok, err := s.staff.GetName(ctx, tenantID, id)
		if err != nil {
			return err
		}
		if !ok {
			return apperror.Validation("Anggota tim tidak ditemukan", map[string][]string{"picStaffId": {"Anggota tim tidak ditemukan"}})
		}
	}
	return nil
}

// ProjectPatch is what PATCH /projects/{id} sent: only the fields present are
// changed (ADR-0036). Status is deliberately not here: it changes only through
// Transition / Cancel, so editing a project's details can never skip a step of its
// life (FLOW-IDEAL 2.4).
type ProjectPatch struct {
	Name, BrideName, GroomName, Description Opt[string]
	// PrepStartDate is the booking date the early steps count from. The event
	// date is not here: it is the main event's, changed on the event.
	PrepStartDate Opt[time.Time]
	// PICStaffID / PICSalesStaffID: 0 means "none"; PICSalesStaffID may be null.
	PICStaffID, PICSalesStaffID Opt[int64]
	// IsArchived is the explicit archive state.
	IsArchived Opt[bool]
}

// HasDetails reports whether the patch touches anything besides the archive state
// (which has its own permission).
func (p ProjectPatch) HasDetails() bool {
	return p.Name.Present() || p.BrideName.Present() || p.GroomName.Present() ||
		p.Description.Present() || p.PrepStartDate.Present() ||
		p.PICStaffID.Present() || p.PICSalesStaffID.Present()
}

// Patch changes the fields of a project that were sent, and only those. The
// business rules are judged on the merged result: a Wedding Planner ("Staff" role)
// still cannot reassign a PIC (only Owner/Admin assign who a project's PIC is, and
// a Sales caller keeps handover through PICStaffID), a PIC that changes must be
// this workspace's staff, and un-archiving counts against the trial cap. A patch that changes nothing
// writes nothing and answers with the project as it is. With `expected` the write
// is a compare-and-set on updated_at: a stale version is a 409 (apperror.Stale)
// carrying the current project.
func (s *ProjectService) Patch(ctx context.Context, tenantID, id int64, actor ProjectActor, patch ProjectPatch, expected *time.Time) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	fe := newFieldErrors()
	requiredText(fe, "name", "Nama proyek wajib diisi", patch.Name)
	requiredText(fe, "brideName", "Nama pengantin wanita wajib diisi", patch.BrideName)
	requiredText(fe, "groomName", "Nama pengantin pria wajib diisi", patch.GroomName)
	if patch.PICStaffID.Present() && patch.PICStaffID.Value() < 0 {
		fe.add("Anggota tim tidak ditemukan", "picStaffId")
	}
	if patch.PICSalesStaffID.Present() && patch.PICSalesStaffID.Value() < 0 {
		fe.add("Anggota tim tidak ditemukan", "picSalesStaffId")
	}
	if err := fe.err(); err != nil {
		return nil, err
	}

	wasArchived := p.IsArchived
	loaded := *p
	var ch changeSet
	setVal(&ch, "name", "nama", &p.Name, patch.Name)
	setVal(&ch, "brideName", "nama pengantin wanita", &p.BrideName, patch.BrideName)
	setVal(&ch, "groomName", "nama pengantin pria", &p.GroomName, patch.GroomName)
	setDate(&ch, "prepStartDate", "mulai persiapan", &p.PrepStartDate, patch.PrepStartDate)
	setVal(&ch, "description", "deskripsi", &p.Description, patch.Description)
	setVal(&ch, "picStaffId", "PIC", &p.PICStaffID, patch.PICStaffID)
	setVal(&ch, "picSalesStaffId", "PIC Sales", &p.PICSalesStaffID, orZero(patch.PICSalesStaffID))
	setVal(&ch, "isArchived", "status arsip", &p.IsArchived, patch.IsArchived)
	if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
		return nil, err
	}

	// Only a PIC that changes is looked up: a PIC whose staff row was deleted since
	// must not make every later edit of the project fail.
	if ch.has("picStaffId") || ch.has("picSalesStaffId") {
		if !actor.CanAssignPIC {
			return nil, apperror.Forbidden("Peran Anda tidak dapat mengganti PIC proyek").WithCode("forbidden_permission")
		}
		var changed []int64
		if ch.has("picStaffId") {
			changed = append(changed, p.PICStaffID)
		}
		if ch.has("picSalesStaffId") {
			changed = append(changed, p.PICSalesStaffID)
		}
		if err := s.checkPICs(ctx, tenantID, changed...); err != nil {
			return nil, err
		}
	}
	if wasArchived && !p.IsArchived && isActiveProject(p.Status, false) {
		if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	if !ch.any() {
		return p, nil
	}

	// The Timeline steps counted from the booking date move with it, in the same
	// transaction as the project's own write (EP-06). Nothing moves when the write
	// did not happen (a stale If-Match is answered by finishPatch below).
	var wrote bool
	moved := 0
	err = s.txOrDirect().InTx(ctx, func(ctx context.Context) error {
		w, err := s.repo.Patch(ctx, p, ch.keys, expected)
		if err != nil {
			return err
		}
		wrote = w
		if w && ch.has("prepStartDate") {
			moved, err = s.shiftMilestones(ctx, p.ID, p.EventDate, p.PrepStartDate)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.Project, error) {
		return s.repo.FindByID(ctx, tenantID, id)
	}, func(p *domain.Project) time.Time { return p.UpdatedAt }, apperror.NotFound("Proyek tidak ditemukan"))
	if err != nil {
		return nil, err
	}
	if moved > 0 {
		s.activity.Record(ctx, &fresh.ID, domain.ActivityMilestoneUpdated, actor.StaffID, "project_milestone", "", fresh.Name,
			strconv.Itoa(moved)+" tahapan mengikuti tanggal booking yang baru")
	}
	if len(ch.keys) == 1 && ch.has("isArchived") {
		action := "diarsipkan"
		if !fresh.IsArchived {
			action = "dipulihkan dari arsip"
		}
		s.activity.Record(ctx, &fresh.ID, domain.ActivityProjectStatusChanged, actor.StaffID, "project", formatID(fresh.ID), fresh.Name, "Proyek "+action)
	} else {
		s.activity.Record(ctx, &fresh.ID, domain.ActivityProjectUpdated, actor.StaffID, "project", formatID(fresh.ID), fresh.Name,
			"Informasi proyek diperbarui: "+ch.summary())
	}
	return fresh, nil
}

// orZero reads an optional id where null means "none" (0).
func orZero(o Opt[int64]) Opt[int64] {
	if o.IsNull() {
		return Some(int64(0))
	}
	return o
}

// syncEventDate makes the project's event date the main event's and moves the
// Timeline steps that follow it, inside the caller's transaction (EventService
// calls it whenever the main event or its date changes).
func (s *ProjectService) syncEventDate(ctx context.Context, tenantID, projectID, actorStaffID int64, date time.Time) error {
	p, err := s.Get(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	if p.EventDate.Format("2006-01-02") == date.Format("2006-01-02") {
		return nil
	}
	p.EventDate = date
	if _, err := s.repo.Patch(ctx, p, []string{"eventDate"}, nil); err != nil {
		return err
	}
	moved, err := s.shiftMilestones(ctx, p.ID, p.EventDate, p.PrepStartDate)
	if err != nil {
		return err
	}
	if moved > 0 {
		s.activity.Record(ctx, &p.ID, domain.ActivityMilestoneUpdated, actorStaffID, "project_milestone", "", p.Name,
			strconv.Itoa(moved)+" tahapan mengikuti tanggal acara utama yang baru")
	}
	return nil
}

// projectStatusLabel is the Indonesian name shown in the activity log.
func projectStatusLabel(status domain.ProjectStatus) string {
	switch status {
	case domain.StatusDraft:
		return "Draf"
	case domain.StatusPreparation:
		return "Persiapan"
	case domain.StatusReady:
		return "Siap"
	case domain.StatusCompleted:
		return "Selesai"
	case domain.StatusCancelled:
		return "Dibatalkan"
	}
	return string(status)
}

// Transition moves a project one explicit step through its life (Draft ->
// Preparation -> Ready -> Completed, or Reopen). An action that does not
// apply to the current status is refused with `invalid_status_transition`.
// Reopening counts as one more active project, so the trial cap applies.
func (s *ProjectService) Transition(ctx context.Context, tenantID, id, actorStaffID int64, action domain.StatusAction) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !domain.IsStatusAction(action) {
		return nil, apperror.Validation("Aksi status tidak dikenal", map[string][]string{
			"action": {"Gunakan start_preparation, mark_ready, complete, atau reopen"},
		})
	}
	next, ok := domain.NextStatus(p.Status, action)
	if !ok {
		return nil, apperror.Conflict(fmt.Sprintf(
			"Aksi ini tidak bisa dilakukan pada proyek berstatus %s. Muat ulang halaman untuk melihat status terbaru.", projectStatusLabel(p.Status),
		)).WithCode("invalid_status_transition")
	}
	if !isActiveProject(p.Status, p.IsArchived) && isActiveProject(next, p.IsArchived) {
		if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	from := p.Status
	changed, err := s.repo.ChangeStatus(ctx, tenantID, id, from, next)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, errStatusMoved()
	}
	p.Status = next
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectStatusChanged, actorStaffID, "project", formatID(p.ID), p.Name,
		fmt.Sprintf("Status proyek diubah: %s → %s", projectStatusLabel(from), projectStatusLabel(next)))
	return s.afterWrite(ctx, tenantID, p), nil
}

func (s *ProjectService) Cancel(ctx context.Context, tenantID, id, actorStaffID int64) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	// Only an open project can be cancelled; a finished or already cancelled
	// one would just log a second "dibatalkan" (reopen it first).
	if p.Status == domain.StatusCompleted || p.Status == domain.StatusCancelled {
		return nil, apperror.Conflict(fmt.Sprintf(
			"Proyek berstatus %s tidak bisa dibatalkan. Muat ulang halaman untuk melihat status terbaru.", projectStatusLabel(p.Status),
		)).WithCode("invalid_status_transition")
	}
	changed, err := s.repo.ChangeStatus(ctx, tenantID, id, p.Status, domain.StatusCancelled)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, errStatusMoved()
	}
	p.Status = domain.StatusCancelled
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectStatusChanged, actorStaffID, "project", formatID(p.ID), p.Name,
		"Proyek dibatalkan")
	return s.afterWrite(ctx, tenantID, p), nil
}

// afterWrite is the project as the database holds it now. The object this request loaded carries the version from before its own
// write, and the answer's updatedAt/ETag is what the caller sends back as If-Match next time (ADR-0036): the stale one would make
// a person's next edit collide with their own status change. If the re-read fails the change has landed anyway, so the answer
// falls back to the object in hand rather than turning a success into an error.
func (s *ProjectService) afterWrite(ctx context.Context, tenantID int64, p *domain.Project) *domain.Project {
	if fresh, err := s.Get(ctx, tenantID, p.ID); err == nil {
		return fresh
	}
	return p
}

// errStatusMoved: someone else changed the project's status between this
// request reading it and writing it.
func errStatusMoved() error {
	return apperror.Conflict("Status proyek baru saja diubah orang lain. Muat ulang halaman untuk melihat status terbaru.").WithCode("invalid_status_transition")
}

// Delete permanently removes a project and every row across this module
// that references it (see ProjectRepository.DeleteCascade), plus best-effort
// cleans up evidence's object-storage files and every client tied to this
// project — a deliberate exception to this codebase's usual soft-state
// convention (knowledge/DATABASE_GUIDE.md), guarded accordingly: only a
// project already archived or cancelled may be hard-deleted (enforced here,
// not just the frontend), and only an Owner may call this at all (enforced
// by the handler, since role-gating doesn't otherwise exist in this module —
// see ADR-0013). No activity-log entry is recorded for the deletion itself:
// activity_log rows for this project are deleted in the same transaction,
// so one would just be wiped immediately after being written.
func (s *ProjectService) Delete(ctx context.Context, tenantID, id int64) error {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if !p.IsArchived && p.Status != domain.StatusCancelled {
		return apperror.Validation("Proyek belum bisa dihapus permanen", map[string][]string{
			"status": {"Arsipkan atau batalkan proyek ini terlebih dahulu sebelum menghapusnya secara permanen"},
		})
	}

	// Every file of the project, the ones hidden because their payment or
	// vendor waits in the "deleted" state included: their objects must go too.
	evidences, err := s.evidence.ListAll(ctx, id)
	if err != nil {
		return err
	}

	found, err := s.repo.DeleteCascade(ctx, tenantID, id)
	if errors.Is(err, domain.ErrProjectDeleteBlocked) {
		logger.Error("hard delete of project %d refused by a foreign key: %v", id, err)
		return apperror.Conflict("Proyek ini belum bisa dihapus permanen karena masih ada data lain yang terhubung. Hubungi dukungan ElProof agar datanya diperiksa.").WithCode("project_delete_blocked")
	}
	if err != nil {
		return err
	}
	if !found {
		return apperror.NotFound("Proyek tidak ditemukan")
	}

	// Both cleanup steps below run only after the DB cascade has committed —
	// a failure in either leaves, at worst, an orphaned S3 object or client
	// row, never a dangling reference back to a project that no longer
	// exists (the opposite ordering would risk exactly that).
	s.evidence.DeleteStorageObjects(ctx, evidences)
	if p.CoverPhotoPath != nil && s.evidence != nil && s.evidence.storage != nil {
		if err := s.evidence.storage.Delete(ctx, *p.CoverPhotoPath); err != nil {
			logger.Error("failed to delete cover object %q of deleted project %d: %v", *p.CoverPhotoPath, id, err)
		}
	}

	if s.clients != nil {
		if err := s.clients.DeleteAllForProject(ctx, tenantID, id); err != nil {
			logger.Error("failed to clean up clients for deleted project %d: %v", id, err)
		}
	}

	// A project made from a quotation: the quotation stays converted, with
	// "Proyek sudah dihapus" (PLAN §5.4). Best effort, like the steps above.
	if p.QuotationID != nil && s.quotations != nil {
		if err := s.quotations.MarkProjectDeleted(ctx, tenantID, id); err != nil {
			logger.Error("failed to mark quotation %d of deleted project %d: %v", *p.QuotationID, id, err)
		}
	}

	return nil
}

// Duplicate creates a new project from `input` (the caller's, possibly
// user-edited, copy of the source project's fields — see ADR-0014) and copies
// the source's structure as a template, in one transaction: its Timeline steps,
// its vendors and venues (back to "Planned", with their payer, "tampilkan
// nama", agreed cost and steps) and its events. The events keep their spacing:
// they move by the distance between the source's event date and input's, so the
// main event lands on the new date. Deliberately excluded: payments, kendala,
// files, activity and clients — what actually happened on the source project.
// Every copied step starts "Not Started"; dates set by hand are copied as they
// are, a Timeline step that follows the event date (EP-06) is dated from the new
// project's own dates.
func (s *ProjectService) Duplicate(ctx context.Context, tenantID int64, actorStaffID int64, sourceID int64, input ProjectInput) (*domain.Project, error) {
	source, err := s.Get(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}
	input.Status = domain.StatusDraft // a copy always starts as Draft (ADR-0027)
	fe := newFieldErrors()
	if input.EventDate.IsZero() {
		fe.add("Tanggal acara wajib diisi", "eventDate")
	}
	if err := fe.err(); err != nil {
		return nil, err
	}
	if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
		return nil, err
	}
	if err := s.checkCopiedCapacity(ctx, source.ID); err != nil {
		return nil, err
	}
	isPKP, err := s.workspaceIsPKP(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	p := &domain.Project{
		TenantID: tenantID, Name: input.Name, BrideName: input.BrideName, GroomName: input.GroomName,
		EventDate: input.EventDate, PrepStartDate: input.PrepStartDate, IsPKP: isPKP, Status: input.Status,
		PICStaffID: input.PICStaffID, PICSalesStaffID: input.PICSalesStaffID, Description: input.Description,
	}
	err = s.txOrDirect().InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		if err := s.cloneMilestonesFrom(ctx, sourceID, p); err != nil {
			return err
		}
		suppliers, events, err := s.cloneSuppliersAndEvents(ctx, source, p)
		if err != nil {
			return err
		}
		// The Rincian comes along (amounts and rule copies; never a payment),
		// and the contract cache is written in this same transaction.
		if err := s.lines.CloneLines(ctx, tenantID, source.ID, p.ID, actorStaffID, events, suppliers); err != nil {
			return err
		}
		s.activity.Record(ctx, &p.ID, domain.ActivityProjectCreated, actorStaffID, "project", formatID(p.ID), p.Name,
			"Proyek baru dibuat dari duplikat proyek: "+source.Name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProjectService) cloneMilestonesFrom(ctx context.Context, sourceProjectID int64, target *domain.Project) error {
	milestones, err := s.milestones.ListByProject(ctx, sourceProjectID)
	if err != nil {
		return err
	}
	for _, m := range milestones {
		clone := &domain.ProjectMilestone{
			ProjectID: target.ID, SortOrder: m.SortOrder, Name: m.Name,
			Status: domain.MilestoneNotStarted, TargetDate: m.TargetDate,
			ForClient: m.ForClient,
		}
		if m.DaysBeforeEvent != nil {
			offset := *m.DaysBeforeEvent
			clone.DaysBeforeEvent = &offset
			clone.TargetDate = domain.MilestoneTarget(target.EventDate, target.PrepStartDate, offset)
		}
		if err := s.milestones.Create(ctx, clone); err != nil {
			return err
		}
	}
	return nil
}

// checkCopiedCapacity applies the capacity rule to the events the copy will
// have, at the venues it will have (PLAN §3.2 "Kapasitas dicek ulang"): the
// source's events against its live, not cancelled venues and their copied
// capacities. A copy that would not fit is refused, with the numbers, and
// nothing is created.
func (s *ProjectService) checkCopiedCapacity(ctx context.Context, sourceID int64) error {
	if s.eventService == nil {
		return nil
	}
	suppliers, err := s.suppliers.ListByProject(ctx, sourceID)
	if err != nil {
		return err
	}
	venues := map[int64]domain.ProjectSupplier{}
	for _, sup := range suppliers {
		if sup.Kind == domain.SupplierVenue && sup.EngagementStatus != domain.EngagementCancelled {
			venues[sup.ID] = sup
		}
	}
	events, err := s.events.ListByProject(ctx, sourceID)
	if err != nil {
		return err
	}
	domain.SortEvents(events)
	return s.eventService.capacityRefusal("Proyek belum bisa diduplikat. ", events, venues)
}

// cloneSuppliersAndEvents copies the source's live vendors and venues (with
// their steps) and its events onto target. A cancelled supplier is left out on
// purpose (the copy is a new start, and a cancelled booking is not part of it),
// so an event held at a cancelled venue is copied without a venue: its guests
// are then "belum dicek ke kapasitas venue" until one is chosen. Every other
// event's venue points at the copy of its venue.
//
// It returns the source → copy id maps of the suppliers and the events (the
// Rincian lines that point at them are remapped with them).
func (s *ProjectService) cloneSuppliersAndEvents(ctx context.Context, source, target *domain.Project) (map[int64]int64, map[int64]int64, error) {
	suppliers, err := s.suppliers.ListByProject(ctx, source.ID)
	if err != nil {
		return nil, nil, err
	}
	copied := map[int64]int64{}
	for _, sup := range suppliers {
		if sup.EngagementStatus == domain.EngagementCancelled {
			continue
		}
		clone := sup
		clone.ID, clone.ProjectID, clone.EngagementStatus = 0, target.ID, domain.EngagementPlanned
		clone.DeletedAt = nil
		if err := s.suppliers.Create(ctx, &clone); err != nil {
			return nil, nil, err
		}
		copied[sup.ID] = clone.ID
		steps, err := s.vendorMilestones.ListBySupplier(ctx, sup.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, vm := range steps {
			step := &domain.VendorMilestone{
				ProjectSupplierID: clone.ID, SortOrder: vm.SortOrder, Name: vm.Name, Description: vm.Description,
				Status: domain.MilestoneNotStarted, TargetDate: vm.TargetDate, PICStaffID: vm.PICStaffID, Notes: vm.Notes,
			}
			if err := s.vendorMilestones.Create(ctx, step); err != nil {
				return nil, nil, err
			}
		}
	}
	events, err := s.events.ListByProject(ctx, source.ID)
	if err != nil {
		return nil, nil, err
	}
	eventCopies := map[int64]int64{}
	shift := target.EventDate.Sub(source.EventDate)
	for _, e := range events {
		clone := e
		clone.ID, clone.ProjectID = 0, target.ID
		clone.EventDate = e.EventDate.Add(shift)
		clone.VenueSupplierID = nil
		if e.VenueSupplierID != nil {
			if id, ok := copied[*e.VenueSupplierID]; ok {
				clone.VenueSupplierID = &id
			}
		}
		clone.Sessions = append([]domain.EventSession(nil), e.Sessions...)
		for i := range clone.Sessions {
			clone.Sessions[i].ID, clone.Sessions[i].EventID = 0, 0
		}
		if err := s.events.Create(ctx, &clone); err != nil {
			return nil, nil, err
		}
		eventCopies[e.ID] = clone.ID
	}
	return copied, eventCopies, nil
}

// --- Project milestones ---

type MilestoneInput struct {
	Name       string
	TargetDate time.Time
}

func (s *ProjectService) ListMilestones(ctx context.Context, tenantID, projectID int64) ([]domain.ProjectMilestone, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return s.milestones.ListByProject(ctx, projectID)
}

func (s *ProjectService) CreateMilestone(ctx context.Context, tenantID, projectID int64, actorStaffID int64, input MilestoneInput) (*domain.ProjectMilestone, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	order, err := s.milestones.NextSortOrder(ctx, projectID)
	if err != nil {
		return nil, err
	}
	m := &domain.ProjectMilestone{
		ProjectID: projectID, SortOrder: order, Name: input.Name,
		Status: domain.MilestoneNotStarted, TargetDate: input.TargetDate,
	}
	if err := s.milestones.Create(ctx, m); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "project_milestone", formatID(m.ID), m.Name,
		"Tahapan proyek ditambahkan: "+m.Name)
	return m, nil
}

// MilestoneExists is nil when the project (of this tenant) has this milestone, a
// not found otherwise. The PATCH handler asks it before it answers a missing
// If-Match (428), so a step that is not there is a 404 with or without the header.
func (s *ProjectService) MilestoneExists(ctx context.Context, tenantID, projectID, milestoneID int64) error {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return err
	}
	m, err := s.milestones.FindByID(ctx, projectID, milestoneID)
	if err != nil {
		return err
	}
	if m == nil {
		return apperror.NotFound("Tahapan tidak ditemukan")
	}
	return nil
}

// MilestonePatch is what PATCH /projects/{id}/milestones/{mid} sent (ADR-0036).
type MilestonePatch struct {
	Status Opt[domain.MilestoneStatus]
	// TargetDate is required (cannot be null); CompletedDate may be null (clear it).
	TargetDate    Opt[time.Time]
	CompletedDate Opt[time.Time]
	// ForClient marks the step as a task for the couple (ADR-0047).
	ForClient Opt[bool]
}

// PatchMilestone lets the WO managing this project correct one step of its own
// schedule. Status, TargetDate and CompletedDate are set from what was sent;
// fields left out keep their value, with two rules (EP-06):
//   - a status change without a completedDate keeps the date in line: becoming
//     Completed stamps `today` (WIB) when the step has no date yet, leaving
//     Completed clears it (domain.StampCompletion);
//   - a target date set by hand fixes the step: it no longer follows the event
//     date (DaysBeforeEvent becomes nil).
//
// A patch that changes nothing writes nothing; with `expected` the write is a
// compare-and-set on the milestone's updated_at (a stale one is a 409).
func (s *ProjectService) PatchMilestone(ctx context.Context, tenantID, projectID, milestoneID int64, actorStaffID int64, patch MilestonePatch, expected *time.Time, today time.Time) (*domain.ProjectMilestone, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	missing := apperror.NotFound("Tahapan tidak ditemukan")
	m, err := s.milestones.FindByID(ctx, projectID, milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, missing
	}
	if patch.Status.Present() && !domain.IsValidMilestoneStatus(patch.Status.Value()) {
		return nil, apperror.Validation("Status tahapan tidak valid", map[string][]string{"status": {"Status tahapan tidak valid"}})
	}
	loaded := *m
	var ch changeSet
	setVal(&ch, "status", "status", &m.Status, patch.Status)
	setDate(&ch, "targetDate", "tanggal target", &m.TargetDate, patch.TargetDate)
	setNullableDate(&ch, "completedDate", "tanggal selesai", &m.CompletedDate, patch.CompletedDate)
	setVal(&ch, "forClient", "tugas untuk pasangan", &m.ForClient, patch.ForClient)
	if !patch.CompletedDate.Present() && domain.StampCompletion(loaded.Status, m.Status, &m.CompletedDate, today) {
		ch.note("completedDate", "tanggal selesai")
	}
	// A status set by staff is staff's: "done by the couple" no longer describes it.
	if ch.has("status") && m.CompletedByClientID != 0 {
		m.CompletedByClientID = 0
		ch.note("completedByClientId", "")
	}
	if ch.has("targetDate") && m.DaysBeforeEvent != nil {
		m.DaysBeforeEvent = nil
		ch.note("daysBeforeEvent", "")
	}
	if !ch.any() {
		return m, nil
	}
	if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
		return nil, err
	}
	wrote, err := s.milestones.Patch(ctx, m, ch.keys, expected)
	if err != nil {
		return nil, err
	}
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.ProjectMilestone, error) {
		return s.milestones.FindByID(ctx, projectID, milestoneID)
	}, func(m *domain.ProjectMilestone) time.Time { return m.UpdatedAt }, missing)
	if err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "project_milestone", formatID(fresh.ID), fresh.Name,
		"Tahapan proyek diperbarui: "+fresh.Name+" ("+ch.summary()+")")
	return fresh, nil
}

// CompleteMilestoneByClient is the couple marking one of their tasks done from
// the Client Portal (ADR-0047): only a step staff flagged ForClient, never a
// cancelled one. It stamps Completed, today's Jakarta date and the client.
// Idempotent: a step already Completed (by anyone) is returned unchanged. Two
// clients completing at once: the second sees the first one's write and gets
// it back unchanged.
func (s *ProjectService) CompleteMilestoneByClient(ctx context.Context, tenantID, projectID, milestoneID, clientID int64, today time.Time) (*domain.ProjectMilestone, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	missing := apperror.NotFound("Tugas tidak ditemukan")
	m, err := s.milestones.FindByID(ctx, projectID, milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, missing
	}
	loaded := m.UpdatedAt
	changed, err := m.CompleteByClient(clientID, today)
	switch {
	case errors.Is(err, domain.ErrNotClientTask):
		return nil, apperror.Forbidden("Tahapan ini dikerjakan oleh WO, bukan tugas untuk Anda.").WithCode("not_client_task")
	case errors.Is(err, domain.ErrClientTaskCancelled):
		msg := "Tugas ini sudah dibatalkan oleh WO."
		return nil, apperror.Validation(msg, map[string][]string{"status": {msg}}).WithCode("task_cancelled")
	case err != nil:
		return nil, err
	}
	if !changed {
		return m, nil
	}
	wrote, err := s.milestones.Patch(ctx, m, []string{"status", "completedDate", "completedByClientId"}, &loaded)
	if err != nil {
		return nil, err
	}
	fresh, err := s.milestones.FindByID(ctx, projectID, milestoneID)
	if err != nil {
		return nil, err
	}
	if fresh == nil {
		return nil, missing
	}
	if !wrote {
		// Someone changed the step between our read and our write.
		if fresh.Status == domain.MilestoneCompleted {
			return fresh, nil
		}
		return nil, apperror.Stale(fresh)
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, 0, "project_milestone", formatID(fresh.ID), fresh.Name,
		"Diselesaikan oleh pasangan: "+fresh.Name)
	return fresh, nil
}

// ReorderMilestones rewrites every milestone's SortOrder for this project to
// match the position of its ID in orderedIDs. orderedIDs must be an exact
// permutation of the project's existing milestone IDs — rejected otherwise,
// so a stale/corrupted client payload can't silently drop or duplicate a row.
func (s *ProjectService) ReorderMilestones(ctx context.Context, tenantID, projectID int64, actorStaffID int64, orderedIDs []int64) error {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return err
	}
	existing, err := s.milestones.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	if len(orderedIDs) != len(existing) {
		return apperror.Validation("Urutan tahapan tidak valid", nil)
	}
	existingIDs := make(map[int64]bool, len(existing))
	for _, m := range existing {
		existingIDs[m.ID] = true
	}
	seen := make(map[int64]bool, len(orderedIDs))
	for _, id := range orderedIDs {
		if !existingIDs[id] || seen[id] {
			return apperror.Validation("Urutan tahapan tidak valid", nil)
		}
		seen[id] = true
	}
	if err := s.milestones.Reorder(ctx, projectID, orderedIDs); err != nil {
		return err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "project_milestone", "", "",
		"Urutan tahapan diubah")
	return nil
}

// --- Progress computation (mirrors mock/selectors.ts computeProjectProgress) ---

// ComputeProgress is a project's progress as staff see it (every kendala
// counts). It reads everything in a fixed number of queries (see
// loadProgressData), whatever the number of vendors.
func (s *ProjectService) ComputeProgress(ctx context.Context, tenantID, projectID int64, asOf time.Time) (*domain.ProjectProgress, error) {
	return s.computeOne(ctx, tenantID, projectID, asOf, false)
}

// ComputeClientProgress is the same figure as a client may see it: only the
// kendala staff shared with the client count, so a hidden kendala can neither be
// counted nor push the condition to "Attention"/"At Risk" (ADR-0034).
func (s *ProjectService) ComputeClientProgress(ctx context.Context, tenantID, projectID int64, asOf time.Time) (*domain.ProjectProgress, error) {
	return s.computeOne(ctx, tenantID, projectID, asOf, true)
}

func (s *ProjectService) computeOne(ctx context.Context, tenantID, projectID int64, asOf time.Time, clientView bool) (*domain.ProjectProgress, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	data, err := s.loadProgressData(ctx, []int64{projectID})
	if err != nil {
		return nil, err
	}
	progress := data.progressFor(projectID, asOf, clientView)
	return &progress, nil
}

// ComputeProgressBatch computes progress for every id in projectIDs using
// ONE query per data source (WHERE project_id IN (...)) -- see PLAN.md
// "Performance remediation". Every project in projectIDs is assumed to
// already belong to tenantID (callers fetch the roster themselves), so,
// unlike ComputeProgress, this does not re-verify each id via s.Get. Staff view.
func (s *ProjectService) ComputeProgressBatch(ctx context.Context, tenantID int64, projectIDs []int64, asOf time.Time) (map[int64]domain.ProjectProgress, error) {
	return s.computeBatch(ctx, projectIDs, asOf, false)
}

// ComputeClientProgressBatch is ComputeProgressBatch as a client may see it
// (see ComputeClientProgress).
func (s *ProjectService) ComputeClientProgressBatch(ctx context.Context, tenantID int64, projectIDs []int64, asOf time.Time) (map[int64]domain.ProjectProgress, error) {
	return s.computeBatch(ctx, projectIDs, asOf, true)
}

func (s *ProjectService) computeBatch(ctx context.Context, projectIDs []int64, asOf time.Time, clientView bool) (map[int64]domain.ProjectProgress, error) {
	result := make(map[int64]domain.ProjectProgress, len(projectIDs))
	if len(projectIDs) == 0 {
		return result, nil
	}
	data, err := s.loadProgressData(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, projectID := range projectIDs {
		result[projectID] = data.progressFor(projectID, asOf, clientView)
	}
	return result, nil
}

// progressData is every row progress is computed from, grouped per project.
// Deleted suppliers, payments and the kendala of a deleted supplier are not in
// it (the repositories skip them), so they count for nothing.
type progressData struct {
	milestones       map[int64][]domain.ProjectMilestone
	suppliers        map[int64][]domain.ProjectSupplier
	vendorMilestones map[int64][]domain.VendorMilestone // by supplier
	issues           map[int64][]domain.VendorIssue
	payments         map[int64][]domain.SupplierPayment
	evidence         map[int64][]domain.Evidence
}

// loadProgressData reads the six sources in six queries.
func (s *ProjectService) loadProgressData(ctx context.Context, projectIDs []int64) (*progressData, error) {
	d := &progressData{
		milestones: map[int64][]domain.ProjectMilestone{}, suppliers: map[int64][]domain.ProjectSupplier{},
		vendorMilestones: map[int64][]domain.VendorMilestone{}, issues: map[int64][]domain.VendorIssue{},
		payments: map[int64][]domain.SupplierPayment{}, evidence: map[int64][]domain.Evidence{},
	}
	milestones, err := s.milestones.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, m := range milestones {
		d.milestones[m.ProjectID] = append(d.milestones[m.ProjectID], m)
	}
	suppliers, err := s.suppliers.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	supplierIDs := make([]int64, 0, len(suppliers))
	for _, sup := range suppliers {
		d.suppliers[sup.ProjectID] = append(d.suppliers[sup.ProjectID], sup)
		supplierIDs = append(supplierIDs, sup.ID)
	}
	vendorMilestones, err := s.vendorMilestones.ListBySuppliers(ctx, supplierIDs)
	if err != nil {
		return nil, err
	}
	for _, vm := range vendorMilestones {
		d.vendorMilestones[vm.ProjectSupplierID] = append(d.vendorMilestones[vm.ProjectSupplierID], vm)
	}
	issues, err := s.issues.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, i := range issues {
		d.issues[i.ProjectID] = append(d.issues[i.ProjectID], i)
	}
	payments, err := s.payments.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, p := range payments {
		d.payments[p.ProjectID] = append(d.payments[p.ProjectID], p)
	}
	evidence, err := s.evidence.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, e := range evidence {
		d.evidence[e.ProjectID] = append(d.evidence[e.ProjectID], e)
	}
	return d, nil
}

// progressFor computes one project's progress from the loaded rows. clientView
// counts only the kendala shared with the client, and only the payments the
// client made to a supplier (what the WO pays its suppliers is not theirs).
func (d *progressData) progressFor(projectID int64, asOf time.Time, clientView bool) domain.ProjectProgress {
	projectStats := domain.ComputeMilestoneStats(toMilestoneLikes(d.milestones[projectID]), asOf)

	var vendorMilestones []domain.VendorMilestone
	for _, sup := range d.suppliers[projectID] {
		vendorMilestones = append(vendorMilestones, d.vendorMilestones[sup.ID]...)
	}
	vendorStats := domain.ComputeMilestoneStats(toVendorMilestoneLikes(vendorMilestones), asOf)

	issues := d.issues[projectID]
	if clientView {
		issues = domain.ClientVisibleIssues(issues)
	}
	var openIssues []domain.VendorIssue
	for _, i := range issues {
		if i.Status.IsOpen() {
			openIssues = append(openIssues, i)
		}
	}

	hasInvoice, hasProof := domain.PaymentEvidenceStatus(d.evidence[projectID], domain.RelatedSupplierPayment)
	incompleteCount := 0
	for _, p := range d.payments[projectID] {
		if clientView && p.Payer != domain.PayerClient {
			continue
		}
		if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof) {
			incompleteCount++
		}
	}
	return domain.ComputeProjectProgress(projectStats, vendorStats, openIssues, incompleteCount)
}

func toMilestoneLikes(ms []domain.ProjectMilestone) []domain.MilestoneLike {
	out := make([]domain.MilestoneLike, len(ms))
	for i, m := range ms {
		out[i] = domain.MilestoneLike{Status: m.Status, TargetDate: m.TargetDate}
	}
	return out
}

func toVendorMilestoneLikes(ms []domain.VendorMilestone) []domain.MilestoneLike {
	out := make([]domain.MilestoneLike, len(ms))
	for i, m := range ms {
		out[i] = domain.MilestoneLike{Status: m.Status, TargetDate: m.TargetDate}
	}
	return out
}
