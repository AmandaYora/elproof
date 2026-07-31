package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
)

type ProjectRepository interface {
	List(ctx context.Context, tenantID int64, picStaffID *int64) ([]domain.Project, error)
	ListPaginated(ctx context.Context, tenantID int64, picStaffID *int64, params pagination.Params, search, status string, showArchived bool) ([]domain.Project, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Project, error)
	Create(ctx context.Context, p *domain.Project) error
	Update(ctx context.Context, p *domain.Project) error
	SetStatus(ctx context.Context, tenantID, id int64, status domain.ProjectStatus) error
	SetArchived(ctx context.Context, tenantID, id int64, archived bool) error
	// DeleteCascade permanently removes the project and every same-module row
	// referencing it — see the infrastructure implementation's doc comment
	// (ADR-0013) for the exact deletion order and why it matters.
	DeleteCascade(ctx context.Context, tenantID, id int64) error
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

type MilestoneRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectMilestone, error)
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
	evidence           *EvidenceService
	activity           *ActivityService
	clients            ClientCleaner
	venues             VenueResolver
}

func NewProjectService(
	repo ProjectRepository,
	milestones MilestoneRepository,
	milestoneTemplates MilestoneTemplateRepository,
	vendorEngagements VendorEngagementRepository,
	vendorMilestones VendorMilestoneRepository,
	issues IssueRepository,
	payments PaymentRepository,
	evidence *EvidenceService,
	activity *ActivityService,
) *ProjectService {
	return &ProjectService{
		repo: repo, milestones: milestones, milestoneTemplates: milestoneTemplates, vendorEngagements: vendorEngagements,
		vendorMilestones: vendorMilestones, issues: issues, payments: payments,
		evidence: evidence, activity: activity,
	}
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
// failure.
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
		return nil, err
	}
	return &summary, nil
}

// picStaffID scopes results to a single PIC's own projects — nil means every
// project in the tenant (see MySQLProjectRepository.List's own doc comment).
func (s *ProjectService) List(ctx context.Context, tenantID int64, picStaffID *int64) ([]domain.Project, error) {
	return s.repo.List(ctx, tenantID, picStaffID)
}

// ListPaginated backs the real project list page — showArchived splits the
// result into two disjoint views (active vs. archived), never merged, so
// archived projects stay genuinely out of the way of day-to-day work (see
// ADR-0013) rather than just visually deprioritized in a mixed list.
func (s *ProjectService) ListPaginated(ctx context.Context, tenantID int64, picStaffID *int64, params pagination.Params, search, status string, showArchived bool) ([]domain.Project, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, picStaffID, params, search, status, showArchived)
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
	Description   string
	// VenueID is only ever read by Update (Create always leaves a new
	// project's VenueID nil — attaching a venue is a separate, post-creation
	// action, see ADR-0016). nil means the caller's JSON body omitted the
	// key entirely — leave the project's current attachment untouched; `0`
	// explicitly detaches; a positive ID attaches that venue. Venue IDs are
	// AUTO_INCREMENT starting at 1, so `0` is never a real one.
	VenueID *int64
}

func (s *ProjectService) Create(ctx context.Context, tenantID int64, actorStaffID int64, input ProjectInput) (*domain.Project, error) {
	p := &domain.Project{
		TenantID: tenantID, Name: input.Name, BrideName: input.BrideName, GroomName: input.GroomName,
		EventDate: input.EventDate, Venue: input.Venue, PrepStartDate: input.PrepStartDate,
		PackageName: input.PackageName, ContractValue: input.ContractValue, Status: input.Status,
		PICStaffID: input.PICStaffID, Description: input.Description,
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

// Update rejects a Wedding Planner ("Staff" role) reassigning PICStaffID —
// confirmed role rule: only Owner/Admin assign or reassign who a project's
// PIC is, even on a project the Wedding Planner already manages day to day.
// callerRole == "" (or any non-"Staff" value) skips the check entirely, so
// Owner/Admin keep reassigning freely exactly as before.
func (s *ProjectService) Update(ctx context.Context, tenantID, id int64, actorStaffID int64, callerRole string, input ProjectInput) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if callerRole == "Staff" && input.PICStaffID != p.PICStaffID {
		return nil, apperror.Forbidden("Hanya Owner atau Admin yang dapat menugaskan ulang PIC project")
	}
	p.Name = input.Name
	p.BrideName = input.BrideName
	p.GroomName = input.GroomName
	p.EventDate = input.EventDate
	p.Venue = input.Venue
	p.PrepStartDate = input.PrepStartDate
	p.PackageName = input.PackageName
	p.ContractValue = input.ContractValue
	p.Status = input.Status
	p.PICStaffID = input.PICStaffID
	p.Description = input.Description
	if input.VenueID != nil {
		if *input.VenueID == 0 {
			p.VenueID = nil
		} else {
			p.VenueID = input.VenueID
		}
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectUpdated, actorStaffID, "project", formatID(p.ID), p.Name,
		"Informasi project diperbarui")
	return p, nil
}

func (s *ProjectService) Cancel(ctx context.Context, tenantID, id, actorStaffID int64) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetStatus(ctx, tenantID, id, domain.StatusCancelled); err != nil {
		return nil, err
	}
	p.Status = domain.StatusCancelled
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectStatusChanged, actorStaffID, "project", formatID(p.ID), p.Name,
		"Project dibatalkan")
	return p, nil
}

// SetArchived toggles a project's archive flag — reversible, orthogonal to
// Status (see ADR-0013). No restriction on which status can be archived.
func (s *ProjectService) SetArchived(ctx context.Context, tenantID, id, actorStaffID int64, archived bool) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
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

	p := &domain.Project{
		TenantID: tenantID, Name: input.Name, BrideName: input.BrideName, GroomName: input.GroomName,
		EventDate: input.EventDate, Venue: input.Venue, PrepStartDate: input.PrepStartDate,
		PackageName: input.PackageName, ContractValue: input.ContractValue, Status: input.Status,
		PICStaffID: input.PICStaffID, Description: input.Description,
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
			BookingDate: pv.BookingDate, EventDate: newEventDate, DPAmount: 0, PaidAmount: 0,
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
	incompleteCount := 0
	for _, p := range payments {
		if !p.IsEvidenceComplete() {
			incompleteCount++
		}
	}

	progress := domain.ComputeProjectProgress(projectStats, vendorStats, openIssues, incompleteCount)
	return &progress, nil
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
