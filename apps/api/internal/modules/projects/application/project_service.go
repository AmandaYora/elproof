package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type ProjectRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.Project, error)
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, status string) ([]domain.Project, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Project, error)
	Create(ctx context.Context, p *domain.Project) error
	Update(ctx context.Context, p *domain.Project) error
	SetStatus(ctx context.Context, tenantID, id int64, status domain.ProjectStatus) error
}

type MilestoneRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectMilestone, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectMilestone, error)
	Create(ctx context.Context, m *domain.ProjectMilestone) error
	Update(ctx context.Context, m *domain.ProjectMilestone) error
	NextSortOrder(ctx context.Context, projectID int64) (int, error)
}

type ProjectService struct {
	repo           ProjectRepository
	milestones     MilestoneRepository
	vendorEngagements VendorEngagementRepository
	vendorMilestones  VendorMilestoneRepository
	issues         IssueRepository
	payments       PaymentRepository
	activity       *ActivityService
}

func NewProjectService(
	repo ProjectRepository,
	milestones MilestoneRepository,
	vendorEngagements VendorEngagementRepository,
	vendorMilestones VendorMilestoneRepository,
	issues IssueRepository,
	payments PaymentRepository,
	activity *ActivityService,
) *ProjectService {
	return &ProjectService{
		repo: repo, milestones: milestones, vendorEngagements: vendorEngagements,
		vendorMilestones: vendorMilestones, issues: issues, payments: payments, activity: activity,
	}
}

func (s *ProjectService) List(ctx context.Context, tenantID int64) ([]domain.Project, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *ProjectService) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, status string) ([]domain.Project, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, params, search, status)
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
	s.activity.Record(ctx, &p.ID, domain.ActivityProjectCreated, actorStaffID, "project", formatID(p.ID), p.Name,
		"Project baru dibuat: "+p.Name)
	return p, nil
}

func (s *ProjectService) Update(ctx context.Context, tenantID, id int64, actorStaffID int64, input ProjectInput) (*domain.Project, error) {
	p, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
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
		"Milestone project ditambahkan: "+m.Name)
	return m, nil
}

func (s *ProjectService) UpdateMilestoneStatus(ctx context.Context, tenantID, projectID, milestoneID int64, actorStaffID int64, status domain.MilestoneStatus, asOf time.Time) (*domain.ProjectMilestone, error) {
	if _, err := s.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	m, err := s.milestones.FindByID(ctx, projectID, milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, apperror.NotFound("Milestone tidak ditemukan")
	}
	m.Status = status
	if status == domain.MilestoneCompleted && m.CompletedDate == nil {
		m.CompletedDate = &asOf
	}
	if err := s.milestones.Update(ctx, m); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "project_milestone", formatID(m.ID), m.Name,
		"Status milestone diubah menjadi "+string(status))
	return m, nil
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
