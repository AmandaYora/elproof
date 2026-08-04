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
	Update(ctx context.Context, issue *domain.VendorIssue) error
}

type IssueService struct {
	repo             IssueRepository
	vendorMilestones VendorMilestoneRepository
	activity         *ActivityService
}

func NewIssueService(repo IssueRepository, vendorMilestones VendorMilestoneRepository, activity *ActivityService) *IssueService {
	return &IssueService{repo: repo, vendorMilestones: vendorMilestones, activity: activity}
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
}

func (s *IssueService) Create(ctx context.Context, projectID int64, actorStaffID int64, input IssueInput) (*domain.VendorIssue, error) {
	if err := s.validateVendorMilestone(ctx, input.ProjectVendorID, input.VendorMilestoneID); err != nil {
		return nil, err
	}
	issue := &domain.VendorIssue{
		ProjectID: projectID, ProjectVendorID: input.ProjectVendorID, VendorMilestoneID: input.VendorMilestoneID,
		Title: input.Title, Description: input.Description,
		Impact: input.Impact, FoundDate: input.FoundDate, Status: domain.IssueOpen, ResolutionPlan: input.ResolutionPlan,
		PICStaffID: input.PICStaffID, TargetResolutionDate: input.TargetResolutionDate,
	}
	if err := s.repo.Create(ctx, issue); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityIssueCreated, actorStaffID, "vendor_issue", formatID(issue.ID), issue.Title,
		"Kendala baru dicatat: "+issue.Title)
	return issue, nil
}

// IssueUpdateInput backs full edit (PLAN.md "Retire the standalone Kendala
// tab") -- previously only a status change was possible. Status rides along
// here too (rather than a separate status-only input) so the frontend's
// quick status dropdown can reuse this exact same method, spreading every
// existing field and overriding just Status -- the same pattern
// VendorMilestoneUpdateInput already uses.
type IssueUpdateInput struct {
	ProjectVendorID      int64
	VendorMilestoneID    *int64
	Title                string
	Description          string
	Impact               domain.IssueImpact
	FoundDate            time.Time
	Status               domain.IssueStatus
	ResolutionPlan       string
	PICStaffID           int64
	TargetResolutionDate *time.Time
}

// Update overwrites every editable field and mirrors the frontend's exact
// rule: a status change to Resolved/Closed auto-stamps resolvedDate exactly
// once, never re-stamped on subsequent changes. ResolutionNotes isn't part
// of IssueUpdateInput (no UI writes it yet) so it survives untouched from
// the fetched row.
func (s *IssueService) Update(ctx context.Context, projectID, id int64, actorStaffID int64, input IssueUpdateInput, asOf time.Time) (*domain.VendorIssue, error) {
	issue, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateVendorMilestone(ctx, input.ProjectVendorID, input.VendorMilestoneID); err != nil {
		return nil, err
	}
	issue.ProjectVendorID = input.ProjectVendorID
	issue.VendorMilestoneID = input.VendorMilestoneID
	issue.Title = input.Title
	issue.Description = input.Description
	issue.Impact = input.Impact
	issue.FoundDate = input.FoundDate
	issue.ResolutionPlan = input.ResolutionPlan
	issue.PICStaffID = input.PICStaffID
	issue.TargetResolutionDate = input.TargetResolutionDate
	issue.Status = input.Status
	shouldStamp := (input.Status == domain.IssueResolved || input.Status == domain.IssueClosed) && issue.ResolvedDate == nil
	if shouldStamp {
		issue.ResolvedDate = &asOf
	}
	if err := s.repo.Update(ctx, issue); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityIssueUpdated, actorStaffID, "vendor_issue", formatID(issue.ID), issue.Title,
		"Kendala diperbarui: "+issue.Title)
	return issue, nil
}
