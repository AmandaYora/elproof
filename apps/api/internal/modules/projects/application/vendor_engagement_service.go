package application

import (
	"context"
	"regexp"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type VendorEngagementRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectVendor, error)
	// ListByProjects backs ComputeProgressBatch: every matching row across
	// the given projects in one query (WHERE project_id IN (...)).
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectVendor, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectVendor, error)
	Create(ctx context.Context, pv *domain.ProjectVendor) error
	// Patch writes the changed fields of a live engagement (compare-and-set on
	// updated_at when expected is given; see infrastructure.patchRow).
	Patch(ctx context.Context, pv *domain.ProjectVendor, changed []string, expected *time.Time) (bool, error)
	SetStatus(ctx context.Context, projectID, id int64, status domain.EngagementStatus) error
	// SoftDelete hides a live engagement (deleted_at = at); false when there was none.
	SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error)
	// FindDeletedByID is the restore lookup (nil unless it waits in the "deleted" state).
	FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.ProjectVendor, error)
	// Restore clears the stamp of an engagement deleted after notBefore.
	Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error)
	ListByVendor(ctx context.Context, tenantID, vendorID int64) ([]domain.VendorEngagementHistoryRow, error)
}

type VendorMilestoneRepository interface {
	ListByProjectVendor(ctx context.Context, projectVendorID int64) ([]domain.VendorMilestone, error)
	// ListByProjectVendors backs ComputeProgressBatch: every matching row
	// across the given engagement ids in one query.
	ListByProjectVendors(ctx context.Context, projectVendorIDs []int64) ([]domain.VendorMilestone, error)
	FindByID(ctx context.Context, projectVendorID, id int64) (*domain.VendorMilestone, error)
	Create(ctx context.Context, m *domain.VendorMilestone) error
	// Patch writes the changed fields of a vendor milestone (see
	// VendorEngagementRepository.Patch).
	Patch(ctx context.Context, m *domain.VendorMilestone, changed []string, expected *time.Time) (bool, error)
	NextSortOrder(ctx context.Context, projectVendorID int64) (int, error)
}

type VendorEngagementService struct {
	repo       VendorEngagementRepository
	milestones VendorMilestoneRepository
	activity   *ActivityService
}

func NewVendorEngagementService(repo VendorEngagementRepository, milestones VendorMilestoneRepository, activity *ActivityService) *VendorEngagementService {
	return &VendorEngagementService{repo: repo, milestones: milestones, activity: activity}
}

func (s *VendorEngagementService) List(ctx context.Context, projectID int64) ([]domain.ProjectVendor, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *VendorEngagementService) Get(ctx context.Context, projectID, id int64) (*domain.ProjectVendor, error) {
	pv, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if pv == nil {
		return nil, apperror.NotFound("Kerja sama vendor tidak ditemukan")
	}
	return pv, nil
}

type VendorEngagementInput struct {
	VendorID         int64
	CategoryID       int64
	Scope            string
	ContractValue    int64
	PricingTier      domain.VendorPricingTier
	EngagementStatus domain.EngagementStatus
	BookingDate      *time.Time
	EventDate        time.Time
	EventStartTime   *string
	EventEndTime     *string
	DPAmount         int64
	DueDate          *time.Time
	PICStaffID       int64
	Notes            string
}

func (s *VendorEngagementService) Create(ctx context.Context, projectID int64, actorStaffID int64, input VendorEngagementInput) (*domain.ProjectVendor, error) {
	pv := &domain.ProjectVendor{
		ProjectID: projectID, VendorID: input.VendorID, CategoryID: input.CategoryID, Scope: input.Scope,
		ContractValue: input.ContractValue, PricingTier: input.PricingTier, EngagementStatus: input.EngagementStatus, BookingDate: input.BookingDate,
		EventDate: input.EventDate, EventStartTime: input.EventStartTime, EventEndTime: input.EventEndTime,
		DPAmount: input.DPAmount, DueDate: input.DueDate,
		PICStaffID: input.PICStaffID, Notes: input.Notes,
	}
	if err := s.repo.Create(ctx, pv); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorAdded, actorStaffID, "project_vendor", formatID(pv.ID), input.Scope,
		"Vendor ditambahkan ke project")
	return pv, nil
}

// VendorEngagementPatch is what PATCH /projects/{id}/vendors/{pvid} sent
// (ADR-0036). BookingDate, DueDate, EventStartTime and EventEndTime may be null
// (clear); Notes null clears to "".
type VendorEngagementPatch struct {
	VendorID, CategoryID, ContractValue, DPAmount, PICStaffID Opt[int64]
	Scope, Notes                                              Opt[string]
	PricingTier                                               Opt[domain.VendorPricingTier]
	EngagementStatus                                          Opt[domain.EngagementStatus]
	BookingDate, DueDate, EventDate                           Opt[time.Time]
	EventStartTime, EventEndTime                              Opt[string]
}

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func validEngagementStatus(s domain.EngagementStatus) bool {
	switch s {
	case domain.EngagementPlanned, domain.EngagementNegotiation, domain.EngagementBooked, domain.EngagementDPPaid,
		domain.EngagementInProgress, domain.EngagementFullyPaid, domain.EngagementReady, domain.EngagementCompleted, domain.EngagementCancelled:
		return true
	}
	return false
}

func validPricingTier(t domain.VendorPricingTier) bool {
	return t == domain.PricingTierAkad || t == domain.PricingTierAkadResepsi || t == domain.PricingTierResepsi
}

// Patch changes the fields of a vendor engagement that were sent, and only
// those; fields left out keep their value. A patch that changes nothing writes
// nothing; with `expected` the write is a compare-and-set on updated_at (a stale
// one is a 409 carrying the current engagement). The caller has already checked
// that the vendor, category and PIC named in the patch are this workspace's.
func (s *VendorEngagementService) Patch(ctx context.Context, projectID, id int64, actorStaffID int64, patch VendorEngagementPatch, expected *time.Time) (*domain.ProjectVendor, error) {
	pv, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	fe := newFieldErrors()
	requiredText(fe, "scope", "Lingkup pekerjaan wajib diisi", patch.Scope)
	if patch.ContractValue.Present() && (patch.ContractValue.Value() < 0 || patch.ContractValue.Value() > domain.MaxMoney) {
		fe.add("Nilai kontrak harus 0 atau lebih dan wajar", "contractValue")
	}
	if patch.DPAmount.Present() && (patch.DPAmount.Value() < 0 || patch.DPAmount.Value() > domain.MaxMoney) {
		fe.add("Nominal DP harus 0 atau lebih dan wajar", "dpAmount")
	}
	if patch.PricingTier.Present() && !validPricingTier(patch.PricingTier.Value()) {
		fe.add("Paket harga tidak valid", "pricingTier")
	}
	if patch.EngagementStatus.Present() && !validEngagementStatus(patch.EngagementStatus.Value()) {
		fe.add("Status kerja sama tidak valid", "engagementStatus")
	}
	for _, f := range []struct {
		o   Opt[string]
		key string
	}{{patch.EventStartTime, "eventStartTime"}, {patch.EventEndTime, "eventEndTime"}} {
		if f.o.Present() && !f.o.IsNull() && !hhmm.MatchString(f.o.Value()) {
			fe.add("Gunakan format HH:MM", f.key)
		}
	}
	if err := fe.err(); err != nil {
		return nil, err
	}

	loaded := *pv
	var ch changeSet
	setVal(&ch, "vendorId", "vendor", &pv.VendorID, patch.VendorID)
	setVal(&ch, "categoryId", "kategori", &pv.CategoryID, patch.CategoryID)
	setVal(&ch, "scope", "lingkup", &pv.Scope, patch.Scope)
	setVal(&ch, "contractValue", "nilai kontrak", &pv.ContractValue, patch.ContractValue)
	setVal(&ch, "pricingTier", "paket harga", &pv.PricingTier, patch.PricingTier)
	setVal(&ch, "engagementStatus", "status", &pv.EngagementStatus, patch.EngagementStatus)
	setNullableDate(&ch, "bookingDate", "tanggal booking", &pv.BookingDate, patch.BookingDate)
	setDate(&ch, "eventDate", "tanggal acara", &pv.EventDate, patch.EventDate)
	setNullable(&ch, "eventStartTime", "jam mulai", &pv.EventStartTime, patch.EventStartTime)
	setNullable(&ch, "eventEndTime", "jam selesai", &pv.EventEndTime, patch.EventEndTime)
	setVal(&ch, "dpAmount", "DP", &pv.DPAmount, patch.DPAmount)
	setNullableDate(&ch, "dueDate", "jatuh tempo", &pv.DueDate, patch.DueDate)
	setVal(&ch, "picStaffId", "PIC", &pv.PICStaffID, patch.PICStaffID)
	setVal(&ch, "notes", "catatan", &pv.Notes, patch.Notes)
	if !ch.any() {
		return pv, nil
	}
	if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
		return nil, err
	}
	wrote, err := s.repo.Patch(ctx, pv, ch.keys, expected)
	if err != nil {
		return nil, err
	}
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.ProjectVendor, error) {
		return s.repo.FindByID(ctx, projectID, id)
	}, func(p *domain.ProjectVendor) time.Time { return p.UpdatedAt }, apperror.NotFound("Kerja sama vendor tidak ditemukan"))
	if err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorStatusChanged, actorStaffID, "project_vendor", formatID(fresh.ID), fresh.Scope,
		"Informasi kerja sama vendor diperbarui: "+ch.summary())
	return fresh, nil
}

// ListHistoryForVendor backs the `vendors` module's "Lihat Project" feature
// via `contracts.Contracts.ListVendorEngagementHistory` — see that method's
// doc comment for why this lives here rather than in `vendors`.
func (s *VendorEngagementService) ListHistoryForVendor(ctx context.Context, tenantID, vendorID int64) ([]domain.VendorEngagementHistoryRow, error) {
	return s.repo.ListByVendor(ctx, tenantID, vendorID)
}

func (s *VendorEngagementService) Cancel(ctx context.Context, projectID, id int64, actorStaffID int64) (*domain.ProjectVendor, error) {
	pv, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetStatus(ctx, projectID, id, domain.EngagementCancelled); err != nil {
		return nil, err
	}
	pv.EngagementStatus = domain.EngagementCancelled
	s.activity.Record(ctx, &projectID, domain.ActivityVendorStatusChanged, actorStaffID, "project_vendor", formatID(pv.ID), pv.Scope,
		"Kerja sama vendor dibatalkan")
	// The version the caller sends back as If-Match is the one after this write, not the one loaded before it (ADR-0036).
	if fresh, err := s.Get(ctx, projectID, id); err == nil {
		return fresh, nil
	}
	return pv, nil
}

// Delete soft-deletes the engagement (ADR-0034): it leaves the vendor list, the
// progress, the dashboard and the vendor's project history, and so do its
// milestones, payments, kendala and their proof files, until it is restored (24
// hours) or purged. It returns the instant restoring stops being possible.
func (s *VendorEngagementService) Delete(ctx context.Context, projectID, id int64, actorStaffID int64, now time.Time) (*domain.ProjectVendor, time.Time, error) {
	pv, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, time.Time{}, err
	}
	ok, err := s.repo.SoftDelete(ctx, projectID, id, now)
	if err != nil {
		return nil, time.Time{}, err
	}
	if !ok {
		return nil, time.Time{}, apperror.NotFound("Kerja sama vendor tidak ditemukan")
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorDeleted, actorStaffID, "project_vendor", formatID(pv.ID), pv.Scope,
		"Kerja sama vendor dihapus")
	return pv, domain.RestorableUntil(now), nil
}

// Restore brings a deleted engagement back with everything that hung off it.
// check runs first, on the deleted engagement, and may refuse the restore with a
// validation error: the handler uses it to confirm that the vendor and category
// still exist (a vendor hard-deleted meanwhile cannot come back as a ghost). Not
// deleted, or past the retention: 404.
func (s *VendorEngagementService) Restore(ctx context.Context, projectID, id int64, actorStaffID int64, now time.Time, check func(ctx context.Context, pv *domain.ProjectVendor) error) (*domain.ProjectVendor, error) {
	pv, err := s.repo.FindDeletedByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if pv == nil || pv.DeletedAt == nil || !domain.CanRestore(*pv.DeletedAt, now) {
		return nil, errNotRestorable()
	}
	if check != nil {
		if err := check(ctx, pv); err != nil {
			return nil, err
		}
	}
	ok, err := s.repo.Restore(ctx, projectID, id, domain.PurgeCutoff(now))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errNotRestorable()
	}
	pv.DeletedAt = nil
	s.activity.Record(ctx, &projectID, domain.ActivityRestored, actorStaffID, "project_vendor", formatID(pv.ID), pv.Scope,
		"Kerja sama vendor dipulihkan")
	return pv, nil
}

// --- Vendor milestones ---

type VendorMilestoneInput struct {
	Name        string
	Description string
	TargetDate  time.Time
	PICStaffID  int64
}

func (s *VendorEngagementService) ListMilestones(ctx context.Context, projectID, projectVendorID int64) ([]domain.VendorMilestone, error) {
	if _, err := s.Get(ctx, projectID, projectVendorID); err != nil {
		return nil, err
	}
	return s.milestones.ListByProjectVendor(ctx, projectVendorID)
}

// ListMilestonesByProjectVendors backs listVendorEngagements: one query
// across every given engagement id instead of the frontend's own
// per-engagement Promise.all loop (PLAN.md "Performance remediation",
// Phase E). Returns a map so the caller can attach each engagement's own
// milestones to its response row.
func (s *VendorEngagementService) ListMilestonesByProjectVendors(ctx context.Context, projectVendorIDs []int64) (map[int64][]domain.VendorMilestone, error) {
	all, err := s.milestones.ListByProjectVendors(ctx, projectVendorIDs)
	if err != nil {
		return nil, err
	}
	byEngagement := make(map[int64][]domain.VendorMilestone, len(projectVendorIDs))
	for _, m := range all {
		byEngagement[m.ProjectVendorID] = append(byEngagement[m.ProjectVendorID], m)
	}
	return byEngagement, nil
}

func (s *VendorEngagementService) CreateMilestone(ctx context.Context, projectID, projectVendorID int64, actorStaffID int64, input VendorMilestoneInput) (*domain.VendorMilestone, error) {
	if _, err := s.Get(ctx, projectID, projectVendorID); err != nil {
		return nil, err
	}
	order, err := s.milestones.NextSortOrder(ctx, projectVendorID)
	if err != nil {
		return nil, err
	}
	m := &domain.VendorMilestone{
		ProjectVendorID: projectVendorID, SortOrder: order, Name: input.Name, Description: input.Description,
		Status: domain.MilestoneNotStarted, TargetDate: input.TargetDate, PICStaffID: input.PICStaffID,
	}
	if err := s.milestones.Create(ctx, m); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "vendor_milestone", formatID(m.ID), m.Name,
		"Timeline vendor ditambahkan: "+m.Name)
	return m, nil
}

// VendorMilestonePatch is what PATCH .../vendors/{pvid}/milestones/{mid} sent
// (ADR-0036). CompletedDate may be null (clear); Description and Notes null clear to "".
type VendorMilestonePatch struct {
	Status                    Opt[domain.MilestoneStatus]
	TargetDate, CompletedDate Opt[time.Time]
	PICStaffID                Opt[int64]
	Description, Notes        Opt[string]
}

// PatchMilestone changes the fields of a vendor milestone that were sent, and only
// those. A patch that changes nothing writes nothing; with `expected` the write is
// a compare-and-set on the milestone's updated_at (a stale one is a 409).
func (s *VendorEngagementService) PatchMilestone(ctx context.Context, projectID, projectVendorID, milestoneID int64, actorStaffID int64, patch VendorMilestonePatch, expected *time.Time) (*domain.VendorMilestone, error) {
	if _, err := s.Get(ctx, projectID, projectVendorID); err != nil {
		return nil, err
	}
	missing := apperror.NotFound("Timeline vendor tidak ditemukan")
	m, err := s.milestones.FindByID(ctx, projectVendorID, milestoneID)
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
	setVal(&ch, "picStaffId", "PIC", &m.PICStaffID, patch.PICStaffID)
	setVal(&ch, "description", "deskripsi", &m.Description, patch.Description)
	setVal(&ch, "notes", "catatan", &m.Notes, patch.Notes)
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
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.VendorMilestone, error) {
		return s.milestones.FindByID(ctx, projectVendorID, milestoneID)
	}, func(m *domain.VendorMilestone) time.Time { return m.UpdatedAt }, missing)
	if err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "vendor_milestone", formatID(fresh.ID), fresh.Name,
		"Timeline vendor diperbarui: "+ch.summary())
	return fresh, nil
}
