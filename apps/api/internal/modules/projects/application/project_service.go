package application

import (
	"context"
	"fmt"
	"time"

	"elproof/internal/modules/projects/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
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
	ListPaginated(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64, params pagination.Params, search, status string, showArchived bool) ([]domain.Project, int64, error)
	// ListByVenueID/ListByStaffPIC back the hard-delete "impact" endpoints
	// (PLAN.md) -- see their infrastructure implementations' doc comments.
	ListByVenueID(ctx context.Context, tenantID, venueID int64) ([]domain.ProjectRef, error)
	ListByStaffPIC(ctx context.Context, tenantID, staffID int64) ([]domain.ProjectRef, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Project, error)
	Create(ctx context.Context, p *domain.Project) error
	Update(ctx context.Context, p *domain.Project) error
	// ChangeStatus is conditional on the current status (see the MySQL implementation).
	ChangeStatus(ctx context.Context, tenantID, id int64, from, to domain.ProjectStatus) (bool, error)
	SetArchived(ctx context.Context, tenantID, id int64, archived bool) error
	// DeleteCascade permanently removes the project and every same-module row
	// referencing it — see the infrastructure implementation's doc comment
	// (ADR-0013) for the exact deletion order and why it matters.
	DeleteCascade(ctx context.Context, tenantID, id int64) error
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

// VenueResolver is the narrow shape ProjectService needs from `vendors` to
// resolve a project's attached venue_id into display data (ADR-0016) --
// deliberately a local interface (not an import of vendors/application) so
// this file only depends on vendors' public contracts package for the return
// type. Bridged from main.go via Module.SetVenueResolver, the same two-phase
// idiom as SetClientCleaner above -- needed because `vendors` itself depends
// on `projects.Contracts()` (built after `projects`), so `projects` can't
// take this as a constructor argument.
type VenueResolver interface {
	GetVenueSummary(ctx context.Context, tenantID, venueID int64) (vendorscontracts.VenueSummary, error)
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
// Unlike ClientCleaner/VenueResolver, `staff` is already built before
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
	Update(ctx context.Context, m *domain.ProjectMilestone) error
	NextSortOrder(ctx context.Context, projectID int64) (int, error)
	Reorder(ctx context.Context, projectID int64, orderedIDs []int64) error
}

type ProjectService struct {
	repo               ProjectRepository
	milestones         MilestoneRepository
	milestoneTemplates MilestoneTemplateRepository
	vendorEngagements  VendorEngagementRepository
	vendorMilestones   VendorMilestoneRepository
	issues             IssueRepository
	payments           PaymentRepository
	venuePayments      VenuePaymentRepository
	evidence           *EvidenceService
	activity           *ActivityService
	clients            ClientCleaner
	venues             VenueResolver
	staff              StaffNameResolver
	quota              TrialQuota
}

func NewProjectService(
	repo ProjectRepository,
	milestones MilestoneRepository,
	milestoneTemplates MilestoneTemplateRepository,
	vendorEngagements VendorEngagementRepository,
	vendorMilestones VendorMilestoneRepository,
	issues IssueRepository,
	payments PaymentRepository,
	venuePayments VenuePaymentRepository,
	evidence *EvidenceService,
	activity *ActivityService,
	staff StaffNameResolver,
) *ProjectService {
	return &ProjectService{
		repo: repo, milestones: milestones, milestoneTemplates: milestoneTemplates, vendorEngagements: vendorEngagements,
		vendorMilestones: vendorMilestones, issues: issues, payments: payments, venuePayments: venuePayments,
		evidence: evidence, activity: activity, staff: staff,
	}
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

// SetVenueResolver completes the two-phase wiring described on VenueResolver
// above. main.go calls this right after vendorsModule is built, the same
// slot as SetClientCleaner/SetClientAccessResolver.
func (s *ProjectService) SetVenueResolver(resolver VenueResolver) {
	s.venues = resolver
}

// GetVenue resolves this project's attached venue (if any) into display
// data via the vendors module's contract. Returns (nil, nil) — not an error
// — when no venue is attached, so callers (both the WO Console tab and
// Client Portal's) can render an empty state rather than treat it as a
// failure. A NotFound from the vendors module (the venue itself has since
// been hard-deleted — PLAN.md's Vendor/Venue hard delete, which never
// cleans up this stale reference) is treated exactly the same way: the
// venue is simply no longer resolvable, not a real error to surface.
func (s *ProjectService) GetVenue(ctx context.Context, tenantID, projectID int64) (*vendorscontracts.VenueSummary, error) {
	p, err := s.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if p.VenueID == nil || s.venues == nil {
		return nil, nil
	}
	summary, err := s.venues.GetVenueSummary(ctx, tenantID, *p.VenueID)
	if err != nil {
		if appErr, ok := apperror.As(err); ok && appErr.Kind == apperror.KindNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &summary, nil
}

// picStaffID/picSalesStaffID scope results to a single PIC's own projects —
// nil/nil means every project in the tenant (see
// MySQLProjectRepository.List's own doc comment).
func (s *ProjectService) List(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64) ([]domain.Project, error) {
	return s.repo.List(ctx, tenantID, picStaffID, picSalesStaffID)
}

// ListAffectedProjectsForVenue backs the hard-delete "impact" endpoint for
// Venue (PLAN.md) -- see ListByVenueID's doc comment.
func (s *ProjectService) ListAffectedProjectsForVenue(ctx context.Context, tenantID, venueID int64) ([]domain.ProjectRef, error) {
	return s.repo.ListByVenueID(ctx, tenantID, venueID)
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
func (s *ProjectService) ListPaginated(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64, params pagination.Params, search, status string, showArchived bool) ([]domain.Project, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, picStaffID, picSalesStaffID, params, search, status, showArchived)
}

func (s *ProjectService) Get(ctx context.Context, tenantID, id int64) (*domain.Project, error) {
	p, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperror.NotFound("Project tidak ditemukan")
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
	Name          string
	BrideName     string
	GroomName     string
	EventDate     time.Time
	Venue         string
	PrepStartDate time.Time
	PackageName   string
	ContractValue int64
	Status        domain.ProjectStatus
	PICStaffID    int64
	// PICSalesStaffID is the "PIC Sales" slot -- see Project.PICSalesStaffID's
	// doc comment. Create forces this to the caller's own staff id when
	// callerRole is "Sales" (ignoring whatever's sent here); Owner/Admin's
	// value passes through untouched.
	PICSalesStaffID int64
	Description     string
	// VenueID is only ever read by Update (Create always leaves a new
	// project's VenueID nil — attaching a venue is a separate, post-creation
	// action, see ADR-0016). nil means the caller's JSON body omitted the
	// key entirely — leave the project's current attachment untouched; `0`
	// explicitly detaches; a positive ID attaches that venue. Venue IDs are
	// AUTO_INCREMENT starting at 1, so `0` is never a real one.
	VenueID *int64
	// VenueRentalPrice/VenueCharge ride along with VenueID whenever it's
	// present at all (see Update below) -- the per-project cost snapshot,
	// PLAN.md "Financial Calculation Correctness". Only meaningful when
	// VenueID is a positive attach/change; ignored on detach (force-cleared
	// regardless) and left nil-safe when VenueID itself is nil (every other
	// project edit that never touches venue attachment at all).
	VenueRentalPrice *int64
	VenueCharge      *int64
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

// Create: a person limited to their own projects must stay able to reach
// the project they create, so unless they made themselves PIC they become
// its PIC Sales (the former Sales rule, now for any such role).
func (s *ProjectService) Create(ctx context.Context, tenantID int64, actor ProjectActor, input ProjectInput) (*domain.Project, error) {
	actorStaffID := actor.StaffID
	picSalesStaffID := input.PICSalesStaffID
	if actor.AssignedOnly && input.PICStaffID != actorStaffID {
		picSalesStaffID = actorStaffID
	}
	// A new project always starts as Draft: later steps are explicit
	// transitions, so creating cannot skip them (ADR-0027).
	input.Status = domain.StatusDraft
	if err := s.checkPICs(ctx, tenantID, input.PICStaffID, picSalesStaffID); err != nil {
		return nil, err
	}
	if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
		return nil, err
	}
	p := &domain.Project{
		TenantID: tenantID, Name: input.Name, BrideName: input.BrideName, GroomName: input.GroomName,
		EventDate: input.EventDate, Venue: input.Venue, PrepStartDate: input.PrepStartDate,
		PackageName: input.PackageName, ContractValue: input.ContractValue, Status: input.Status,
		PICStaffID: input.PICStaffID, PICSalesStaffID: picSalesStaffID, Description: input.Description,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	if err := s.seedDefaultMilestones(ctx, tenantID, p.ID, p.EventDate, p.PrepStartDate); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectCreated, actorStaffID, "project", formatID(p.ID), p.Name,
		"Project baru dibuat: "+p.Name)
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
			return apperror.Validation("Staf tidak ditemukan", map[string][]string{"picStaffId": {"Staf tidak ditemukan"}})
		}
	}
	return nil
}

// Update rejects a Wedding Planner ("Staff" role) reassigning PICStaffID —
// confirmed role rule: only Owner/Admin assign or reassign who a project's
// PIC is, even on a project the Wedding Planner already manages day to day.
// callerRole == "" (or any non-"Staff"/"Sales" value) skips both checks
// entirely, so Owner/Admin keep reassigning freely exactly as before. A
// Sales caller may NOT move PICSalesStaffID to someone else (their own PIC
// Sales slot is immutable from their side), but MAY freely change
// PICStaffID -- that's the handover mechanism (PLAN.md
// revisi-timeline-vendor-role-sales); reaching this method at all as Sales
// already implies resolveProjectAccess confirmed they own this project.
func (s *ProjectService) Update(ctx context.Context, tenantID, id int64, actor ProjectActor, input ProjectInput) (*domain.Project, error) {
	actorStaffID := actor.StaffID
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !actor.CanAssignPIC && (input.PICStaffID != p.PICStaffID || input.PICSalesStaffID != p.PICSalesStaffID) {
		return nil, apperror.Forbidden("Peran Anda tidak dapat mengganti PIC project").WithCode("forbidden_permission")
	}
	// Only a PIC that changes is looked up: a PIC whose staff row was deleted
	// since must not make every later edit of the project fail.
	if input.PICStaffID != p.PICStaffID || input.PICSalesStaffID != p.PICSalesStaffID {
		var changed []int64
		if input.PICStaffID != p.PICStaffID {
			changed = append(changed, input.PICStaffID)
		}
		if input.PICSalesStaffID != p.PICSalesStaffID {
			changed = append(changed, input.PICSalesStaffID)
		}
		if err := s.checkPICs(ctx, tenantID, changed...); err != nil {
			return nil, err
		}
	}
	// Status is deliberately not taken from the input: it changes only through
	// Transition / Cancel, so editing a project's details can never skip a
	// step of its life (FLOW-IDEAL 2.4).
	p.Name = input.Name
	p.BrideName = input.BrideName
	p.GroomName = input.GroomName
	p.EventDate = input.EventDate
	p.Venue = input.Venue
	p.PrepStartDate = input.PrepStartDate
	p.PackageName = input.PackageName
	p.ContractValue = input.ContractValue
	p.PICStaffID = input.PICStaffID
	p.PICSalesStaffID = input.PICSalesStaffID
	p.Description = input.Description
	if input.VenueID != nil {
		if *input.VenueID == 0 {
			// Detach always force-clears the cost snapshot too, regardless of
			// whatever the request body happened to send for these two
			// fields -- there is no cost without a venue.
			p.VenueID = nil
			p.VenueRentalPrice = nil
			p.VenueCharge = nil
		} else {
			// Same venue as before: keep the lenient behaviour below (a venue
			// deleted since must not make every later edit fail). A different
			// one is being attached now and must be this workspace's own.
			sameVenue := p.VenueID != nil && *p.VenueID == *input.VenueID
			p.VenueID = input.VenueID
			p.VenueRentalPrice = input.VenueRentalPrice
			p.VenueCharge = input.VenueCharge
			// One place for the venue: while one is attached from the
			// directory, the project's venue text IS its name, so the two can
			// never disagree (PLAN.md Fase 4c "venue tunggal"). If the venue
			// cannot be resolved, the old text stays.
			if s.venues != nil {
				summary, err := s.venues.GetVenueSummary(ctx, tenantID, *input.VenueID)
				switch {
				case err == nil:
					if summary.Name != "" {
						p.Venue = summary.Name
					}
				case !sameVenue:
					if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
						return nil, apperror.Validation("Venue tidak ditemukan", map[string][]string{"venueId": {"Venue tidak ditemukan"}})
					}
					return nil, err
				}
			}
		}
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectUpdated, actorStaffID, "project", formatID(p.ID), p.Name,
		"Informasi project diperbarui")
	return p, nil
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
			"Aksi ini tidak bisa dilakukan pada project berstatus %s. Muat ulang halaman untuk melihat status terbaru.", projectStatusLabel(p.Status),
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
		fmt.Sprintf("Status project diubah: %s → %s", projectStatusLabel(from), projectStatusLabel(next)))
	return p, nil
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
			"Project berstatus %s tidak bisa dibatalkan. Muat ulang halaman untuk melihat status terbaru.", projectStatusLabel(p.Status),
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
		"Project dibatalkan")
	return p, nil
}

// errStatusMoved: someone else changed the project's status between this
// request reading it and writing it.
func errStatusMoved() error {
	return apperror.Conflict("Status project baru saja diubah orang lain. Muat ulang halaman untuk melihat status terbaru.").WithCode("invalid_status_transition")
}

// SetArchived toggles a project's archive flag — reversible, orthogonal to
// Status (see ADR-0013). No restriction on which status can be archived.
func (s *ProjectService) SetArchived(ctx context.Context, tenantID, id, actorStaffID int64, archived bool) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !archived && p.IsArchived && isActiveProject(p.Status, false) {
		if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.SetArchived(ctx, tenantID, id, archived); err != nil {
		return nil, err
	}
	p.IsArchived = archived
	action := "diarsipkan"
	if !archived {
		action = "dipulihkan dari arsip"
	}
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectStatusChanged, actorStaffID, "project", formatID(p.ID), p.Name,
		"Project "+action)
	return p, nil
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
		return apperror.Validation("Project belum bisa dihapus permanen", map[string][]string{
			"status": {"Arsipkan atau batalkan project ini terlebih dahulu sebelum menghapusnya secara permanen"},
		})
	}

	evidences, err := s.evidence.List(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteCascade(ctx, tenantID, id); err != nil {
		return err
	}

	// Both cleanup steps below run only after the DB cascade has committed —
	// a failure in either leaves, at worst, an orphaned S3 object or client
	// row, never a dangling reference back to a project that no longer
	// exists (the opposite ordering would risk exactly that).
	s.evidence.DeleteStorageObjects(ctx, evidences)

	if s.clients != nil {
		if err := s.clients.DeleteAllForProject(ctx, tenantID, id); err != nil {
			logger.Error("failed to clean up clients for deleted project %d: %v", id, err)
		}
	}

	return nil
}

// Duplicate creates a new project from `input` (the caller's, possibly
// user-edited, copy of the source project's fields — see ADR-0014) and
// clones the source's Project Milestones and Vendor Engagements (with their
// own Vendor Milestones) as a reusable structural template. Deliberately
// excluded: vendor_payments, vendor_issues, evidence, activity_log, and
// clients — all historical/transactional data tied to what actually happened
// on the source project, not to a template. Every cloned row is reset to its
// initial state (milestone status, engagement status, paid/DP amounts);
// every date (including the source's own milestone/booking/due dates) is
// copied verbatim, not shifted — the user adjusts whatever's stale by hand
// afterward, same as any other edit.
func (s *ProjectService) Duplicate(ctx context.Context, tenantID int64, actorStaffID int64, sourceID int64, input ProjectInput) (*domain.Project, error) {
	source, err := s.Get(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}
	input.Status = domain.StatusDraft // a copy always starts as Draft (ADR-0027)
	if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
		return nil, err
	}

	p := &domain.Project{
		TenantID: tenantID, Name: input.Name, BrideName: input.BrideName, GroomName: input.GroomName,
		EventDate: input.EventDate, Venue: input.Venue, PrepStartDate: input.PrepStartDate,
		PackageName: input.PackageName, ContractValue: input.ContractValue, Status: input.Status,
		PICStaffID: input.PICStaffID, PICSalesStaffID: input.PICSalesStaffID, Description: input.Description,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	if err := s.cloneMilestonesFrom(ctx, sourceID, p.ID); err != nil {
		return nil, err
	}
	if err := s.cloneVendorEngagementsFrom(ctx, sourceID, p.ID, p.EventDate); err != nil {
		return nil, err
	}

	s.activity.Record(ctx, &p.ID, domain.ActivityProjectCreated, actorStaffID, "project", formatID(p.ID), p.Name,
		"Project baru dibuat dari duplikasi project: "+source.Name)
	return p, nil
}

func (s *ProjectService) cloneMilestonesFrom(ctx context.Context, sourceProjectID, newProjectID int64) error {
	milestones, err := s.milestones.ListByProject(ctx, sourceProjectID)
	if err != nil {
		return err
	}
	for _, m := range milestones {
		clone := &domain.ProjectMilestone{
			ProjectID: newProjectID, SortOrder: m.SortOrder, Name: m.Name,
			Status: domain.MilestoneNotStarted, TargetDate: m.TargetDate,
		}
		if err := s.milestones.Create(ctx, clone); err != nil {
			return err
		}
	}
	return nil
}

// cloneVendorEngagementsFrom clones every vendor engagement (and each one's
// own vendor milestones) from the source project. EventDate is synced to the
// new project's own event date (newEventDate) rather than copied from the
// source engagement — it's a denormalized mirror of the parent project's
// event date, always set that way on every other creation path (see
// useProjectStore.ts's vendorEngagementInputBody), not an independent piece
// of information the "copy dates verbatim" choice applies to. BookingDate and
// DueDate, which are genuinely independent, are copied as-is.
func (s *ProjectService) cloneVendorEngagementsFrom(ctx context.Context, sourceProjectID, newProjectID int64, newEventDate time.Time) error {
	engagements, err := s.vendorEngagements.ListByProject(ctx, sourceProjectID)
	if err != nil {
		return err
	}
	for _, pv := range engagements {
		clone := &domain.ProjectVendor{
			ProjectID: newProjectID, VendorID: pv.VendorID, CategoryID: pv.CategoryID, Scope: pv.Scope,
			ContractValue: pv.ContractValue, PricingTier: pv.PricingTier, EngagementStatus: domain.EngagementPlanned,
			BookingDate: pv.BookingDate, EventDate: newEventDate, DPAmount: 0,
			DueDate: pv.DueDate, PICStaffID: pv.PICStaffID, Notes: pv.Notes,
		}
		if err := s.vendorEngagements.Create(ctx, clone); err != nil {
			return err
		}

		vendorMilestones, err := s.vendorMilestones.ListByProjectVendor(ctx, pv.ID)
		if err != nil {
			return err
		}
		for _, vm := range vendorMilestones {
			vmClone := &domain.VendorMilestone{
				ProjectVendorID: clone.ID, SortOrder: vm.SortOrder, Name: vm.Name, Description: vm.Description,
				Status: domain.MilestoneNotStarted, TargetDate: vm.TargetDate,
				PICStaffID: vm.PICStaffID, Notes: vm.Notes,
			}
			if err := s.vendorMilestones.Create(ctx, vmClone); err != nil {
				return err
			}
		}
	}
	return nil
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
		"Timeline project ditambahkan: "+m.Name)
	return m, nil
}

type MilestoneUpdateInput struct {
	Status        domain.MilestoneStatus
	TargetDate    time.Time
	CompletedDate *time.Time
}

// UpdateMilestone lets the WO managing this project correct its own
// schedule — Status, TargetDate, and CompletedDate are all set directly from
// client input (no more server-side auto-stamping of CompletedDate), mirroring
// VendorEngagementService.UpdateMilestone's shape for the sibling
// vendor-milestone entity.
func (s *ProjectService) UpdateMilestone(ctx context.Context, tenantID, projectID, milestoneID int64, actorStaffID int64, input MilestoneUpdateInput) (*domain.ProjectMilestone, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	m, err := s.milestones.FindByID(ctx, projectID, milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, apperror.NotFound("Timeline tidak ditemukan")
	}
	m.Status = input.Status
	m.TargetDate = input.TargetDate
	m.CompletedDate = input.CompletedDate
	if err := s.milestones.Update(ctx, m); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "project_milestone", formatID(m.ID), m.Name,
		"Timeline project diperbarui: "+m.Name)
	return m, nil
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
		return apperror.Validation("Urutan timeline tidak valid", nil)
	}
	existingIDs := make(map[int64]bool, len(existing))
	for _, m := range existing {
		existingIDs[m.ID] = true
	}
	seen := make(map[int64]bool, len(orderedIDs))
	for _, id := range orderedIDs {
		if !existingIDs[id] || seen[id] {
			return apperror.Validation("Urutan timeline tidak valid", nil)
		}
		seen[id] = true
	}
	if err := s.milestones.Reorder(ctx, projectID, orderedIDs); err != nil {
		return err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "project_milestone", "", "",
		"Urutan timeline diubah")
	return nil
}

// --- Progress computation (mirrors mock/selectors.ts computeProjectProgress) ---

func (s *ProjectService) ComputeProgress(ctx context.Context, tenantID, projectID int64, asOf time.Time) (*domain.ProjectProgress, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}

	projectMilestones, err := s.milestones.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	projectStats := domain.ComputeMilestoneStats(toMilestoneLikes(projectMilestones), asOf)

	vendorEngagements, err := s.vendorEngagements.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var allVendorMilestones []domain.VendorMilestone
	for _, pv := range vendorEngagements {
		vms, err := s.vendorMilestones.ListByProjectVendor(ctx, pv.ID)
		if err != nil {
			return nil, err
		}
		allVendorMilestones = append(allVendorMilestones, vms...)
	}
	vendorStats := domain.ComputeMilestoneStats(toVendorMilestoneLikes(allVendorMilestones), asOf)

	issues, err := s.issues.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var openIssues []domain.VendorIssue
	for _, i := range issues {
		if i.Status.IsOpen() {
			openIssues = append(openIssues, i)
		}
	}

	payments, err := s.payments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	evidences, err := s.evidence.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedPayment)
	incompleteCount := 0
	for _, p := range payments {
		if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof) {
			incompleteCount++
		}
	}

	venuePayments, err := s.venuePayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	vHasInvoice, vHasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedVenuePayment)
	for _, p := range venuePayments {
		if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, vHasInvoice, vHasProof) {
			incompleteCount++
		}
	}

	progress := domain.ComputeProjectProgress(projectStats, vendorStats, openIssues, incompleteCount)
	return &progress, nil
}

// ComputeProgressBatch computes progress for every id in projectIDs using
// ONE query per data source (WHERE project_id IN (...)) instead of
// ComputeProgress's ~7 queries repeated per project — see PLAN.md
// "Performance remediation". Every project in projectIDs is assumed to
// already belong to tenantID (callers fetch the roster themselves), so,
// unlike ComputeProgress, this does not re-verify each id via s.Get.
func (s *ProjectService) ComputeProgressBatch(ctx context.Context, tenantID int64, projectIDs []int64, asOf time.Time) (map[int64]domain.ProjectProgress, error) {
	result := make(map[int64]domain.ProjectProgress, len(projectIDs))
	if len(projectIDs) == 0 {
		return result, nil
	}

	allMilestones, err := s.milestones.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	milestonesByProject := make(map[int64][]domain.ProjectMilestone)
	for _, m := range allMilestones {
		milestonesByProject[m.ProjectID] = append(milestonesByProject[m.ProjectID], m)
	}

	allEngagements, err := s.vendorEngagements.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	engagementsByProject := make(map[int64][]domain.ProjectVendor)
	engagementIDs := make([]int64, 0, len(allEngagements))
	for _, pv := range allEngagements {
		engagementsByProject[pv.ProjectID] = append(engagementsByProject[pv.ProjectID], pv)
		engagementIDs = append(engagementIDs, pv.ID)
	}

	allVendorMilestones, err := s.vendorMilestones.ListByProjectVendors(ctx, engagementIDs)
	if err != nil {
		return nil, err
	}
	vendorMilestonesByEngagement := make(map[int64][]domain.VendorMilestone)
	for _, vm := range allVendorMilestones {
		vendorMilestonesByEngagement[vm.ProjectVendorID] = append(vendorMilestonesByEngagement[vm.ProjectVendorID], vm)
	}

	allIssues, err := s.issues.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	issuesByProject := make(map[int64][]domain.VendorIssue)
	for _, i := range allIssues {
		issuesByProject[i.ProjectID] = append(issuesByProject[i.ProjectID], i)
	}

	allPayments, err := s.payments.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	paymentsByProject := make(map[int64][]domain.VendorPayment)
	for _, p := range allPayments {
		paymentsByProject[p.ProjectID] = append(paymentsByProject[p.ProjectID], p)
	}

	allVenuePayments, err := s.venuePayments.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	venuePaymentsByProject := make(map[int64][]domain.VenuePayment)
	for _, p := range allVenuePayments {
		venuePaymentsByProject[p.ProjectID] = append(venuePaymentsByProject[p.ProjectID], p)
	}

	allEvidence, err := s.evidence.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	evidenceByProject := make(map[int64][]domain.Evidence)
	for _, e := range allEvidence {
		evidenceByProject[e.ProjectID] = append(evidenceByProject[e.ProjectID], e)
	}

	for _, projectID := range projectIDs {
		projectStats := domain.ComputeMilestoneStats(toMilestoneLikes(milestonesByProject[projectID]), asOf)

		var vendorMilestonesForProject []domain.VendorMilestone
		for _, pv := range engagementsByProject[projectID] {
			vendorMilestonesForProject = append(vendorMilestonesForProject, vendorMilestonesByEngagement[pv.ID]...)
		}
		vendorStats := domain.ComputeMilestoneStats(toVendorMilestoneLikes(vendorMilestonesForProject), asOf)

		var openIssues []domain.VendorIssue
		for _, i := range issuesByProject[projectID] {
			if i.Status.IsOpen() {
				openIssues = append(openIssues, i)
			}
		}

		evidences := evidenceByProject[projectID]
		hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedPayment)
		incompleteCount := 0
		for _, p := range paymentsByProject[projectID] {
			if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof) {
				incompleteCount++
			}
		}

		vHasInvoice, vHasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedVenuePayment)
		for _, p := range venuePaymentsByProject[projectID] {
			if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, vHasInvoice, vHasProof) {
				incompleteCount++
			}
		}

		result[projectID] = domain.ComputeProjectProgress(projectStats, vendorStats, openIssues, incompleteCount)
	}

	return result, nil
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
