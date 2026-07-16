package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type IssueRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.VendorIssue, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.VendorIssue, error)
	Create(ctx context.Context, issue *domain.VendorIssue) error
	Update(ctx context.Context, issue *domain.VendorIssue) error
}

type IssueService struct {
	repo     IssueRepository
	activity *ActivityService
}

func NewIssueService(repo IssueRepository, activity *ActivityService) *IssueService {
	return &IssueService{repo: repo, activity: activity}
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

type IssueInput struct {
	ProjectVendorID      int64
	Title                string
	Description          string
	Impact               domain.IssueImpact
	FoundDate            time.Time
	ResolutionPlan       string
	PICStaffID           int64
	TargetResolutionDate *time.Time
}

func (s *IssueService) Create(ctx context.Context, projectID int64, actorStaffID int64, input IssueInput) (*domain.VendorIssue, error) {
	issue := &domain.VendorIssue{
		ProjectID: projectID, ProjectVendorID: input.ProjectVendorID, Title: input.Title, Description: input.Description,
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

// UpdateStatus mirrors the frontend's exact rule: a status change to
// Resolved/Closed auto-stamps resolvedDate exactly once, never re-stamped on
// subsequent changes.
func (s *IssueService) UpdateStatus(ctx context.Context, projectID, id int64, actorStaffID int64, status domain.IssueStatus, asOf time.Time) (*domain.VendorIssue, error) {
	issue, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	issue.Status = status
	shouldStamp := (status == domain.IssueResolved || status == domain.IssueClosed) && issue.ResolvedDate == nil
	if shouldStamp {
		issue.ResolvedDate = &asOf
	}
	if err := s.repo.Update(ctx, issue); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityIssueUpdated, actorStaffID, "vendor_issue", formatID(issue.ID), issue.Title,
		"Status kendala diubah menjadi "+string(status))
	return issue, nil
}
