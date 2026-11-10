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
	ListPaginated(ctx context.Context, tenantID int64, picStaffID, picSalesStaffID *int64, params pagination.Params, filter domain.ProjectListFilter, sort pagination.Sort) ([]domain.Project, int64, error)
	// ListLabels names a set of the tenant's projects (id, name, event date) in
	// one query; ids of other tenants or that do not exist are absent.
	ListLabels(ctx context.Context, tenantID int64, ids []int64) ([]domain.ProjectLabel, error)
	// ListSortSpec is the ?sort= whitelist (public name -> fixed SQL) and the
	// default order of ListPaginated.
	ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// ListByVenueID/ListByStaffPIC back the hard-delete "impact" endpoints
	// (PLAN.md) -- see their infrastructure implementations' doc comments.
	ListByVenueID(ctx context.Context, tenantID, venueID int64) ([]domain.ProjectRef, error)
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
	// GetVenueForAttach is the venue's name and current prices (staff-only
	// data, so not part of the public-safe summary); used when a project is
	// created with its venue.
	GetVenueForAttach(ctx context.Context, tenantID, venueID int64) (vendorscontracts.VenueAttachInfo, error)
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
	// Patch writes the changed fields of a milestone (see ProjectRepository.Patch).
	Patch(ctx context.Context, m *domain.ProjectMilestone, changed []string, expected *time.Time) (bool, error)
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
	// tx/terms back CreateAtomic (wired by SetAtomicSupport).
	tx    Transactor
	terms PaymentTermRepository
	// listMoney backs the derived columns of the list (SetListMoney).
	listMoney ListMoneySource
}

// SetAtomicSupport wires what creating a project with its venue and payment
// schedule in one transaction needs.
func (s *ProjectService) SetAtomicSupport(tx Transactor, terms PaymentTermRepository) {
	s.tx, s.terms = tx, terms
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
		out[p.ID] = domain.ComputeListMoney(p.ContractValue, lines[p.ID], terms[p.ID])
	}
	return out, nil
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

// ProjectSetup is what can be saved together with a new project, in the same
// transaction: the client's payment schedule. (The venue travels in
// ProjectInput.VenueID / VenueRentalPrice / VenueCharge.)
type ProjectSetup struct {
	// Terms is the payment schedule (nil or empty: none).
	Terms []PaymentTermInput
	// FieldErrors are problems the caller already found while reading the
	// request (a date that is not YYYY-MM-DD, keyed "eventDate",
	// "terms[1].dueDate", ...). They are reported together with everything the
	// service finds, as one validation error.
	FieldErrors map[string][]string
}

// Create: a person limited to their own projects must stay able to reach
// the project they create, so unless they made themselves PIC they become
// its PIC Sales (the former Sales rule, now for any such role).
func (s *ProjectService) Create(ctx context.Context, tenantID int64, actor ProjectActor, input ProjectInput) (*domain.Project, error) {
	return s.CreateAtomic(ctx, tenantID, actor, input, ProjectSetup{})
}

// CreateAtomic creates a project and, in the same database transaction, its
// default milestones, its venue (when input.VenueID is positive) and its client
// payment schedule (setup.Terms). Everything is validated first and every
// problem of every part is returned together as ONE validation error (keys
// "picStaffId", "venueId", "venueRentalPrice", "venueCharge", "terms",
// "terms[2].amount", ...); nothing is written when anything is wrong, and if a
// write fails the transaction rolls back, so no half-made project is ever left.
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
	venue, err := s.resolveVenueForCreate(ctx, tenantID, input, fe)
	if err != nil {
		return nil, err
	}
	terms, err := s.checkSetupTerms(input.ContractValue, setup.Terms, fe)
	if err != nil {
		return nil, err
	}
	if err := fe.err(); err != nil {
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
	if venue != nil {
		// One place for the venue: while one is attached from the directory,
		// the project's venue text IS its name (same rule as Update).
		id := venue.ID
		p.VenueID, p.VenueRentalPrice, p.VenueCharge = &id, &venue.rental, &venue.charge
		if venue.name != "" {
			p.Venue = venue.name
		}
	}
	tx := s.tx
	if tx == nil {
		tx = noTransactor{}
	}
	err = tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		if err := s.seedDefaultMilestones(ctx, tenantID, p.ID, p.EventDate, p.PrepStartDate); err != nil {
			return err
		}
		if len(terms) > 0 {
			for i := range terms {
				terms[i].ProjectID = p.ID
			}
			if err := s.terms.ReplaceAll(ctx, p.ID, terms); err != nil {
				return err
			}
		}
		s.activity.Record(ctx, &p.ID, domain.ActivityProjectCreated, actorStaffID, "project", formatID(p.ID), p.Name,
			"Project baru dibuat: "+p.Name)
		if len(terms) > 0 {
			s.activity.Record(ctx, &p.ID, domain.ActivityProjectUpdated, actorStaffID, "payment_terms", formatID(p.ID), p.Name,
				"Jadwal pembayaran client diperbarui")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// venueForCreate is a venue verified for attaching, with the prices the project
// will snapshot.
type venueForCreate struct {
	ID             int64
	name           string
	rental, charge int64
}

// resolveVenueForCreate validates the venue part of a create request. It
// returns nil (and no error) when no venue is asked for. A venue that is not
// this workspace's is reported on "venueId"; prices left out default to the
// venue's current ones (the project's own snapshot, freely different later).
func (s *ProjectService) resolveVenueForCreate(ctx context.Context, tenantID int64, input ProjectInput, fe *fieldErrors) (*venueForCreate, error) {
	if input.VenueID == nil || *input.VenueID == 0 {
		return nil, nil
	}
	if *input.VenueID < 0 {
		fe.add("Venue tidak ditemukan", "venueId")
		return nil, nil
	}
	if s.venues == nil {
		return nil, apperror.Internal("Direktori venue belum tersedia")
	}
	info, err := s.venues.GetVenueForAttach(ctx, tenantID, *input.VenueID)
	if err != nil {
		if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
			fe.add("Venue tidak ditemukan", "venueId")
			return nil, nil
		}
		return nil, err
	}
	v := &venueForCreate{ID: info.ID, name: info.Name, rental: info.RentalPrice, charge: info.Charge}
	if input.VenueRentalPrice != nil {
		if *input.VenueRentalPrice < 0 || *input.VenueRentalPrice > domain.MaxMoney {
			fe.add("Harga sewa venue harus 0 atau lebih dan wajar", "venueRentalPrice")
		}
		v.rental = *input.VenueRentalPrice
	}
	if input.VenueCharge != nil {
		if *input.VenueCharge < 0 || *input.VenueCharge > domain.MaxMoney {
			fe.add("Biaya tambahan venue harus 0 atau lebih dan wajar", "venueCharge")
		}
		v.charge = *input.VenueCharge
	}
	return v, nil
}

// checkSetupTerms validates a payment schedule sent with a new project exactly
// as PUT /payment-terms does (buildTerms: every row's problems at once, keyed
// "terms[i].label|amount|dueDate"; at most 20; together not above the contract
// value), adding to fe.
func (s *ProjectService) checkSetupTerms(contractValue int64, inputs []PaymentTermInput, fe *fieldErrors) ([]domain.PaymentTerm, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if s.terms == nil {
		return nil, apperror.Internal("Jadwal pembayaran belum tersedia")
	}
	if len(inputs) > maxPaymentTerms {
		fe.add("Maksimal 20 termin per project", "terms")
		return nil, nil
	}
	terms, total, err := buildTerms(0, inputs)
	if err != nil {
		if !fe.mergeErr(err) {
			return nil, err
		}
		return nil, nil
	}
	if contractValue > 0 && total > contractValue {
		fe.add("Jumlah termin "+formatRupiah(total)+" melebihi nilai kontrak "+formatRupiah(contractValue)+". Kurangi nominal salah satu termin.", "terms")
	}
	return terms, nil
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

// ProjectPatch is what PATCH /projects/{id} sent: only the fields present are
// changed (ADR-0036). Status is deliberately not here: it changes only through
// Transition / Cancel, so editing a project's details can never skip a step of its
// life (FLOW-IDEAL 2.4).
type ProjectPatch struct {
	Name, BrideName, GroomName, Venue, PackageName, Description Opt[string]
	EventDate, PrepStartDate                                    Opt[time.Time]
	ContractValue                                               Opt[int64]
	// PICStaffID / PICSalesStaffID: 0 means "none"; PICSalesStaffID may be null.
	PICStaffID, PICSalesStaffID Opt[int64]
	// VenueID: null or 0 detaches the venue (and clears its cost snapshot), a
	// positive id attaches that venue. VenueRentalPrice/VenueCharge: the cost
	// snapshot of the attached venue (null clears).
	VenueID, VenueRentalPrice, VenueCharge Opt[int64]
	// IsArchived is the explicit archive state.
	IsArchived Opt[bool]
}

// HasDetails reports whether the patch touches anything besides the archive state
// (which has its own permission).
func (p ProjectPatch) HasDetails() bool {
	return p.Name.Present() || p.BrideName.Present() || p.GroomName.Present() || p.Venue.Present() || p.PackageName.Present() ||
		p.Description.Present() || p.EventDate.Present() || p.PrepStartDate.Present() || p.ContractValue.Present() ||
		p.PICStaffID.Present() || p.PICSalesStaffID.Present() || p.VenueID.Present() || p.VenueRentalPrice.Present() || p.VenueCharge.Present()
}

// Patch changes the fields of a project that were sent, and only those. The
// business rules are judged on the merged result: a Wedding Planner ("Staff" role)
// still cannot reassign a PIC (only Owner/Admin assign who a project's PIC is, and
// a Sales caller keeps handover through PICStaffID), a PIC that changes must be
// this workspace's staff, a venue is attached or detached with its cost snapshot,
// and un-archiving counts against the trial cap. A patch that changes nothing
// writes nothing and answers with the project as it is. With `expected` the write
// is a compare-and-set on updated_at: a stale version is a 409 (apperror.Stale)
// carrying the current project.
func (s *ProjectService) Patch(ctx context.Context, tenantID, id int64, actor ProjectActor, patch ProjectPatch, expected *time.Time) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	fe := newFieldErrors()
	requiredText(fe, "name", "Nama project wajib diisi", patch.Name)
	requiredText(fe, "brideName", "Nama pengantin wanita wajib diisi", patch.BrideName)
	requiredText(fe, "groomName", "Nama pengantin pria wajib diisi", patch.GroomName)
	requiredText(fe, "packageName", "Nama paket wajib diisi", patch.PackageName)
	if patch.ContractValue.Present() && (patch.ContractValue.Value() < 0 || patch.ContractValue.Value() > domain.MaxMoney) {
		fe.add("Nilai kontrak harus 0 atau lebih dan wajar", "contractValue")
	}
	for _, f := range []struct {
		o   Opt[int64]
		key string
		msg string
	}{
		{patch.VenueRentalPrice, "venueRentalPrice", "Harga sewa venue harus 0 atau lebih dan wajar"},
		{patch.VenueCharge, "venueCharge", "Biaya tambahan venue harus 0 atau lebih dan wajar"},
	} {
		if f.o.Present() && !f.o.IsNull() && (f.o.Value() < 0 || f.o.Value() > domain.MaxMoney) {
			fe.add(f.msg, f.key)
		}
	}
	if patch.PICStaffID.Present() && patch.PICStaffID.Value() < 0 {
		fe.add("Staf tidak ditemukan", "picStaffId")
	}
	if patch.PICSalesStaffID.Present() && patch.PICSalesStaffID.Value() < 0 {
		fe.add("Staf tidak ditemukan", "picSalesStaffId")
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
	setDate(&ch, "eventDate", "tanggal acara", &p.EventDate, patch.EventDate)
	setDate(&ch, "prepStartDate", "mulai persiapan", &p.PrepStartDate, patch.PrepStartDate)
	setVal(&ch, "packageName", "paket", &p.PackageName, patch.PackageName)
	setVal(&ch, "contractValue", "nilai kontrak", &p.ContractValue, patch.ContractValue)
	setVal(&ch, "description", "deskripsi", &p.Description, patch.Description)
	setVal(&ch, "venue", "venue", &p.Venue, patch.Venue)
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
			return nil, apperror.Forbidden("Peran Anda tidak dapat mengganti PIC project").WithCode("forbidden_permission")
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
	if err := s.patchVenue(ctx, tenantID, p, patch, &ch); err != nil {
		return nil, err
	}
	if wasArchived && !p.IsArchived && isActiveProject(p.Status, false) {
		if err := s.checkActiveProjectQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	if !ch.any() {
		return p, nil
	}

	wrote, err := s.repo.Patch(ctx, p, ch.keys, expected)
	if err != nil {
		return nil, err
	}
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.Project, error) {
		return s.repo.FindByID(ctx, tenantID, id)
	}, func(p *domain.Project) time.Time { return p.UpdatedAt }, apperror.NotFound("Project tidak ditemukan"))
	if err != nil {
		return nil, err
	}
	if len(ch.keys) == 1 && ch.has("isArchived") {
		action := "diarsipkan"
		if !fresh.IsArchived {
			action = "dipulihkan dari arsip"
		}
		s.activity.Record(ctx, &fresh.ID, domain.ActivityProjectStatusChanged, actor.StaffID, "project", formatID(fresh.ID), fresh.Name, "Project "+action)
	} else {
		s.activity.Record(ctx, &fresh.ID, domain.ActivityProjectUpdated, actor.StaffID, "project", formatID(fresh.ID), fresh.Name,
			"Informasi project diperbarui: "+ch.summary())
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

// patchVenue applies the venue part of a project patch onto the merged project:
// attach, change or detach the directory venue, and the cost snapshot that rides
// with it. The rules are the ones the full edit had; what a patch adds is that a
// venue attached without prices snapshots the venue's current ones (as create
// does), and prices sent for a project with no venue are refused.
func (s *ProjectService) patchVenue(ctx context.Context, tenantID int64, p *domain.Project, patch ProjectPatch, ch *changeSet) error {
	venueSent := patch.VenueID.Present()
	detach := venueSent && (patch.VenueID.IsNull() || patch.VenueID.Value() == 0)
	switch {
	case detach:
		// Detach always force-clears the cost snapshot too, whatever the request
		// sent for those two fields: there is no cost without a venue.
		if p.VenueID != nil {
			p.VenueID = nil
			ch.note("venueId", "venue")
		}
		if p.VenueRentalPrice != nil {
			p.VenueRentalPrice = nil
			ch.note("venueRentalPrice", "harga sewa venue")
		}
		if p.VenueCharge != nil {
			p.VenueCharge = nil
			ch.note("venueCharge", "biaya venue")
		}
		return nil
	case venueSent:
		venueID := patch.VenueID.Value()
		if venueID < 0 {
			return apperror.Validation("Venue tidak ditemukan", map[string][]string{"venueId": {"Venue tidak ditemukan"}})
		}
		sameVenue := p.VenueID != nil && *p.VenueID == venueID
		if !sameVenue && s.venues != nil {
			// A different venue is being attached now and must be this workspace's own.
			info, err := s.venues.GetVenueForAttach(ctx, tenantID, venueID)
			if err != nil {
				if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindNotFound {
					return apperror.Validation("Venue tidak ditemukan", map[string][]string{"venueId": {"Venue tidak ditemukan"}})
				}
				return err
			}
			id := venueID
			p.VenueID = &id
			ch.note("venueId", "venue")
			if !patch.VenueRentalPrice.Present() {
				v := info.RentalPrice
				p.VenueRentalPrice = &v
				ch.note("venueRentalPrice", "harga sewa venue")
			}
			if !patch.VenueCharge.Present() {
				v := info.Charge
				p.VenueCharge = &v
				ch.note("venueCharge", "biaya venue")
			}
			// One place for the venue: while one is attached from the directory, the
			// project's venue text IS its name, so the two can never disagree.
			if info.Name != "" && p.Venue != info.Name {
				p.Venue = info.Name
				ch.note("venue", "venue")
			}
		} else if sameVenue && s.venues != nil {
			// Same venue as before: lenient (a venue deleted since must not make every
			// later edit fail); the text follows the venue's name while it resolves.
			if summary, err := s.venues.GetVenueSummary(ctx, tenantID, venueID); err == nil && summary.Name != "" && p.Venue != summary.Name {
				p.Venue = summary.Name
				ch.note("venue", "venue")
			}
		} else if !sameVenue {
			id := venueID
			p.VenueID = &id
			ch.note("venueId", "venue")
		}
	}
	if patch.VenueRentalPrice.Present() || patch.VenueCharge.Present() {
		if p.VenueID == nil {
			if (patch.VenueRentalPrice.Present() && !patch.VenueRentalPrice.IsNull()) || (patch.VenueCharge.Present() && !patch.VenueCharge.IsNull()) {
				return apperror.Validation("Pilih venue lebih dulu", map[string][]string{"venueId": {"Pilih venue lebih dulu sebelum mengisi biayanya"}})
			}
			return nil // clearing a snapshot that is already empty
		}
		setNullable(ch, "venueRentalPrice", "harga sewa venue", &p.VenueRentalPrice, patch.VenueRentalPrice)
		setNullable(ch, "venueCharge", "biaya venue", &p.VenueCharge, patch.VenueCharge)
	}
	// The venue text follows the directory venue's name while one is attached, so a
	// patch of just the text cannot make the two disagree.
	if patch.Venue.Present() && !venueSent && p.VenueID != nil && s.venues != nil {
		if summary, err := s.venues.GetVenueSummary(ctx, tenantID, *p.VenueID); err == nil && summary.Name != "" && p.Venue != summary.Name {
			p.Venue = summary.Name
			ch.note("venue", "venue")
		}
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
	return apperror.Conflict("Status project baru saja diubah orang lain. Muat ulang halaman untuk melihat status terbaru.").WithCode("invalid_status_transition")
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

	// Every file of the project, the ones hidden because their payment or
	// vendor waits in the "deleted" state included: their objects must go too.
	evidences, err := s.evidence.ListAll(ctx, id)
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

// MilestonePatch is what PATCH /projects/{id}/milestones/{mid} sent (ADR-0036).
type MilestonePatch struct {
	Status Opt[domain.MilestoneStatus]
	// TargetDate is required (cannot be null); CompletedDate may be null (clear it).
	TargetDate    Opt[time.Time]
	CompletedDate Opt[time.Time]
}

// PatchMilestone lets the WO managing this project correct one step of its own
// schedule. Status, TargetDate and CompletedDate are set directly from what was
// sent (no server-side auto-stamping of CompletedDate); fields left out keep their
// value. A patch that changes nothing writes nothing; with `expected` the write is
// a compare-and-set on the milestone's updated_at (a stale one is a 409).
func (s *ProjectService) PatchMilestone(ctx context.Context, tenantID, projectID, milestoneID int64, actorStaffID int64, patch MilestonePatch, expected *time.Time) (*domain.ProjectMilestone, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	missing := apperror.NotFound("Timeline tidak ditemukan")
	m, err := s.milestones.FindByID(ctx, projectID, milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, missing
	}
	if patch.Status.Present() && !domain.IsValidMilestoneStatus(patch.Status.Value()) {
		return nil, apperror.Validation("Status timeline tidak valid", map[string][]string{"status": {"Status timeline tidak valid"}})
	}
	loaded := *m
	var ch changeSet
	setVal(&ch, "status", "status", &m.Status, patch.Status)
	setDate(&ch, "targetDate", "tanggal target", &m.TargetDate, patch.TargetDate)
	setNullableDate(&ch, "completedDate", "tanggal selesai", &m.CompletedDate, patch.CompletedDate)
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
		"Timeline project diperbarui: "+fresh.Name+" ("+ch.summary()+")")
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
// Deleted engagements, payments and the kendala of a deleted engagement are not
// in it (the repositories skip them), so they count for nothing.
type progressData struct {
	milestones       map[int64][]domain.ProjectMilestone
	engagements      map[int64][]domain.ProjectVendor
	vendorMilestones map[int64][]domain.VendorMilestone // by engagement
	issues           map[int64][]domain.VendorIssue
	payments         map[int64][]domain.VendorPayment
	venuePayments    map[int64][]domain.VenuePayment
	evidence         map[int64][]domain.Evidence
}

// loadProgressData reads the seven sources in seven queries.
func (s *ProjectService) loadProgressData(ctx context.Context, projectIDs []int64) (*progressData, error) {
	d := &progressData{
		milestones: map[int64][]domain.ProjectMilestone{}, engagements: map[int64][]domain.ProjectVendor{},
		vendorMilestones: map[int64][]domain.VendorMilestone{}, issues: map[int64][]domain.VendorIssue{},
		payments: map[int64][]domain.VendorPayment{}, venuePayments: map[int64][]domain.VenuePayment{},
		evidence: map[int64][]domain.Evidence{},
	}
	milestones, err := s.milestones.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, m := range milestones {
		d.milestones[m.ProjectID] = append(d.milestones[m.ProjectID], m)
	}
	engagements, err := s.vendorEngagements.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	engagementIDs := make([]int64, 0, len(engagements))
	for _, pv := range engagements {
		d.engagements[pv.ProjectID] = append(d.engagements[pv.ProjectID], pv)
		engagementIDs = append(engagementIDs, pv.ID)
	}
	vendorMilestones, err := s.vendorMilestones.ListByProjectVendors(ctx, engagementIDs)
	if err != nil {
		return nil, err
	}
	for _, vm := range vendorMilestones {
		d.vendorMilestones[vm.ProjectVendorID] = append(d.vendorMilestones[vm.ProjectVendorID], vm)
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
	venuePayments, err := s.venuePayments.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, p := range venuePayments {
		d.venuePayments[p.ProjectID] = append(d.venuePayments[p.ProjectID], p)
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
// counts only the kendala shared with the client.
func (d *progressData) progressFor(projectID int64, asOf time.Time, clientView bool) domain.ProjectProgress {
	projectStats := domain.ComputeMilestoneStats(toMilestoneLikes(d.milestones[projectID]), asOf)

	var vendorMilestones []domain.VendorMilestone
	for _, pv := range d.engagements[projectID] {
		vendorMilestones = append(vendorMilestones, d.vendorMilestones[pv.ID]...)
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

	evidences := d.evidence[projectID]
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedPayment)
	incompleteCount := 0
	for _, p := range d.payments[projectID] {
		if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof) {
			incompleteCount++
		}
	}
	vHasInvoice, vHasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedVenuePayment)
	for _, p := range d.venuePayments[projectID] {
		if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, vHasInvoice, vHasProof) {
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
