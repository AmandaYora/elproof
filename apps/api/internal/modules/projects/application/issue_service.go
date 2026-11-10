package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type IssueRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.VendorIssue, error)
	// ListByProjects backs ComputeProgressBatch: every matching row across
	// the given projects in one query (WHERE project_id IN (...)).
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.VendorIssue, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.VendorIssue, error)
	Create(ctx context.Context, issue *domain.VendorIssue) error
	// Patch writes the changed fields of a kendala (compare-and-set on updated_at
	// when expected is given; see infrastructure.patchRow).
	Patch(ctx context.Context, issue *domain.VendorIssue, changed []string, expected *time.Time) (bool, error)
}

type IssueService struct {
	repo             IssueRepository
	vendorMilestones VendorMilestoneRepository
	engagements      VendorEngagementRepository
	activity         *ActivityService
}

func NewIssueService(repo IssueRepository, vendorMilestones VendorMilestoneRepository, engagements VendorEngagementRepository, activity *ActivityService) *IssueService {
	return &IssueService{repo: repo, vendorMilestones: vendorMilestones, engagements: engagements, activity: activity}
}

// validateEngagement rejects a vendor engagement that is not this project's:
// the project in the path is already the caller's own, but the engagement id in
// the body is not, and an issue must never hang off another workspace's vendor.
func (s *IssueService) validateEngagement(ctx context.Context, projectID, projectVendorID int64) error {
	if projectVendorID == 0 {
		return nil // a kendala about the event in general has no vendor
	}
	pv, err := s.engagements.FindByID(ctx, projectID, projectVendorID)
	if err != nil {
		return err
	}
	if pv == nil {
		return apperror.Validation("Vendor tidak valid", map[string][]string{"projectVendorId": {"Vendor tidak ditemukan di project ini"}})
	}
	return nil
}

func (s *IssueService) List(ctx context.Context, projectID int64) ([]domain.VendorIssue, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *IssueService) Get(ctx context.Context, projectID, id int64) (*domain.VendorIssue, error) {
	issue, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, apperror.NotFound("Kendala tidak ditemukan")
	}
	return issue, nil
}

// validateVendorMilestone rejects a milestone that doesn't belong to the
// issue's own vendor engagement -- PLAN.md "Retire the standalone Kendala
// tab": the one rule that must never be skippable, otherwise a kendala could
// be silently misattributed to a different vendor's milestone. A nil id
// (general/not-tied-to-one-milestone kendala) always passes.
func (s *IssueService) validateVendorMilestone(ctx context.Context, projectVendorID int64, vendorMilestoneID *int64) error {
	if vendorMilestoneID == nil {
		return nil
	}
	if projectVendorID == 0 {
		// A deliverable belongs to a vendor: without one there is nothing to attach to.
		return apperror.Validation("Timeline vendor tidak valid", map[string][]string{"vendorMilestoneId": {"Pilih vendornya dulu untuk memilih timeline vendor"}})
	}
	milestone, err := s.vendorMilestones.FindByID(ctx, projectVendorID, *vendorMilestoneID)
	if err != nil {
		return err
	}
	if milestone == nil {
		return apperror.Validation("Timeline vendor tidak valid", map[string][]string{"vendorMilestoneId": {"Timeline tidak ditemukan untuk vendor ini"}})
	}
	return nil
}

type IssueInput struct {
	// ProjectVendorID is 0 for a kendala about the event in general.
	ProjectVendorID int64
	// VendorMilestoneID is nil when the kendala is general to the vendor
	// engagement rather than about one specific deliverable.
	VendorMilestoneID    *int64
	Title                string
	Description          string
	Impact               domain.IssueImpact
	FoundDate            time.Time
	ResolutionPlan       string
	PICStaffID           int64
	TargetResolutionDate *time.Time
	// IsClientVisible: nil means the default, shown to the client.
	IsClientVisible *bool
}

func (s *IssueService) Create(ctx context.Context, projectID int64, actorStaffID int64, input IssueInput) (*domain.VendorIssue, error) {
	if err := s.validateEngagement(ctx, projectID, input.ProjectVendorID); err != nil {
		return nil, err
	}
	if err := s.validateVendorMilestone(ctx, input.ProjectVendorID, input.VendorMilestoneID); err != nil {
		return nil, err
	}
	issue := &domain.VendorIssue{
		ProjectID: projectID, ProjectVendorID: input.ProjectVendorID, VendorMilestoneID: input.VendorMilestoneID,
		Title: input.Title, Description: input.Description,
		Impact: input.Impact, FoundDate: input.FoundDate, Status: domain.IssueOpen, ResolutionPlan: input.ResolutionPlan,
		PICStaffID: input.PICStaffID, TargetResolutionDate: input.TargetResolutionDate,
		IsClientVisible: input.IsClientVisible == nil || *input.IsClientVisible,
	}
	if err := s.repo.Create(ctx, issue); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityIssueCreated, actorStaffID, "vendor_issue", formatID(issue.ID), issue.Title,
		"Kendala baru dicatat: "+issue.Title)
	return issue, nil
}

// IssuePatch is what PATCH /projects/{id}/issues/{iid} sent (ADR-0036).
// ProjectVendorID null or 0 makes it a kendala about the event in general;
// VendorMilestoneID, TargetResolutionDate may be null (clear); ResolutionPlan null
// clears to "".
type IssuePatch struct {
	ProjectVendorID, VendorMilestoneID, PICStaffID Opt[int64]
	Title, Description, ResolutionPlan             Opt[string]
	Impact                                         Opt[domain.IssueImpact]
	Status                                         Opt[domain.IssueStatus]
	FoundDate, TargetResolutionDate                Opt[time.Time]
	IsClientVisible                                Opt[bool]
}

func validImpact(i domain.IssueImpact) bool {
	return i == domain.ImpactLow || i == domain.ImpactMedium || i == domain.ImpactHigh || i == domain.ImpactCritical
}

func validIssueStatus(s domain.IssueStatus) bool {
	switch s {
	case domain.IssueOpen, domain.IssueInReview, domain.IssueInResolution, domain.IssueResolved, domain.IssueClosed:
		return true
	}
	return false
}

// Patch changes the fields of a kendala that were sent, and only those. The rules
// are judged on the merged result: the vendor must be an engagement of this
// project, and a vendor milestone must belong to the (merged) vendor, so moving a
// kendala to another vendor without moving its milestone is refused. A status
// change to Resolved/Closed stamps resolvedDate exactly once, never re-stamped.
// ResolutionNotes survives untouched (no UI writes it yet). A patch that changes
// nothing writes nothing; with `expected` the write is a compare-and-set on
// updated_at (a stale one is a 409 carrying the current kendala). The caller has
// already checked that the PIC named in the patch is this workspace's staff.
func (s *IssueService) Patch(ctx context.Context, projectID, id int64, actorStaffID int64, patch IssuePatch, asOf time.Time, expected *time.Time) (*domain.VendorIssue, error) {
	issue, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	fe := newFieldErrors()
	requiredText(fe, "title", "Judul kendala wajib diisi", patch.Title)
	if patch.Impact.Present() && !validImpact(patch.Impact.Value()) {
		fe.add("Dampak harus Low, Medium, High, atau Critical", "impact")
	}
	if patch.Status.Present() && !validIssueStatus(patch.Status.Value()) {
		fe.add("Status kendala tidak valid", "status")
	}
	if err := fe.err(); err != nil {
		return nil, err
	}

	loaded := *issue
	var ch changeSet
	vendor := patch.ProjectVendorID
	if vendor.IsNull() {
		vendor = Some(int64(0))
	}
	setVal(&ch, "projectVendorId", "vendor", &issue.ProjectVendorID, vendor)
	setNullable(&ch, "vendorMilestoneId", "timeline vendor", &issue.VendorMilestoneID, patch.VendorMilestoneID)
	setVal(&ch, "title", "judul", &issue.Title, patch.Title)
	setVal(&ch, "description", "deskripsi", &issue.Description, patch.Description)
	setVal(&ch, "impact", "dampak", &issue.Impact, patch.Impact)
	setDate(&ch, "foundDate", "tanggal ditemukan", &issue.FoundDate, patch.FoundDate)
	setVal(&ch, "resolutionPlan", "rencana penyelesaian", &issue.ResolutionPlan, patch.ResolutionPlan)
	setVal(&ch, "picStaffId", "PIC", &issue.PICStaffID, patch.PICStaffID)
	setNullableDate(&ch, "targetResolutionDate", "target penyelesaian", &issue.TargetResolutionDate, patch.TargetResolutionDate)
	setVal(&ch, "status", "status", &issue.Status, patch.Status)
	setVal(&ch, "isClientVisible", "tampil ke client", &issue.IsClientVisible, patch.IsClientVisible)
	if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
		return nil, err
	}

	if ch.has("projectVendorId") {
		if err := s.validateEngagement(ctx, projectID, issue.ProjectVendorID); err != nil {
			return nil, err
		}
	}
	if ch.has("projectVendorId") || ch.has("vendorMilestoneId") {
		if err := s.validateVendorMilestone(ctx, issue.ProjectVendorID, issue.VendorMilestoneID); err != nil {
			return nil, err
		}
	}
	if ch.has("status") && (issue.Status == domain.IssueResolved || issue.Status == domain.IssueClosed) && issue.ResolvedDate == nil {
		stamp := asOf
		issue.ResolvedDate = &stamp
		ch.note("resolvedDate", "")
	}
	if !ch.any() {
		return issue, nil
	}
	wrote, err := s.repo.Patch(ctx, issue, ch.keys, expected)
	if err != nil {
		return nil, err
	}
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.VendorIssue, error) {
		return s.repo.FindByID(ctx, projectID, id)
	}, func(i *domain.VendorIssue) time.Time { return i.UpdatedAt }, apperror.NotFound("Kendala tidak ditemukan"))
	if err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityIssueUpdated, actorStaffID, "vendor_issue", formatID(fresh.ID), fresh.Title,
		"Kendala diperbarui: "+fresh.Title+" ("+ch.summary()+")")
	return fresh, nil
}
