package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type VendorEngagementRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectVendor, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectVendor, error)
	Create(ctx context.Context, pv *domain.ProjectVendor) error
	Update(ctx context.Context, pv *domain.ProjectVendor) error
	SetStatus(ctx context.Context, projectID, id int64, status domain.EngagementStatus) error
	ListByVendor(ctx context.Context, tenantID, vendorID int64) ([]domain.VendorEngagementHistoryRow, error)
}

type VendorMilestoneRepository interface {
	ListByProjectVendor(ctx context.Context, projectVendorID int64) ([]domain.VendorMilestone, error)
	FindByID(ctx context.Context, projectVendorID, id int64) (*domain.VendorMilestone, error)
	Create(ctx context.Context, m *domain.VendorMilestone) error
	Update(ctx context.Context, m *domain.VendorMilestone) error
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
	EngagementStatus domain.EngagementStatus
	BookingDate      *time.Time
	EventDate        time.Time
	DPAmount         int64
	PaidAmount       int64
	DueDate          *time.Time
	PICStaffID       int64
	Notes            string
}

func (s *VendorEngagementService) Create(ctx context.Context, projectID int64, actorStaffID int64, input VendorEngagementInput) (*domain.ProjectVendor, error) {
	pv := &domain.ProjectVendor{
		ProjectID: projectID, VendorID: input.VendorID, CategoryID: input.CategoryID, Scope: input.Scope,
		ContractValue: input.ContractValue, EngagementStatus: input.EngagementStatus, BookingDate: input.BookingDate,
		EventDate: input.EventDate, DPAmount: input.DPAmount, PaidAmount: input.PaidAmount, DueDate: input.DueDate,
		PICStaffID: input.PICStaffID, Notes: input.Notes,
	}
	if err := s.repo.Create(ctx, pv); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorAdded, actorStaffID, "project_vendor", formatID(pv.ID), input.Scope,
		"Vendor ditambahkan ke project")
	return pv, nil
}

func (s *VendorEngagementService) Update(ctx context.Context, projectID, id int64, actorStaffID int64, input VendorEngagementInput) (*domain.ProjectVendor, error) {
	pv, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	pv.VendorID = input.VendorID
	pv.CategoryID = input.CategoryID
	pv.Scope = input.Scope
	pv.ContractValue = input.ContractValue
	pv.EngagementStatus = input.EngagementStatus
	pv.BookingDate = input.BookingDate
	pv.EventDate = input.EventDate
	pv.DPAmount = input.DPAmount
	pv.PaidAmount = input.PaidAmount
	pv.DueDate = input.DueDate
	pv.PICStaffID = input.PICStaffID
	pv.Notes = input.Notes
	if err := s.repo.Update(ctx, pv); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorStatusChanged, actorStaffID, "project_vendor", formatID(pv.ID), pv.Scope,
		"Informasi kerja sama vendor diperbarui")
	return pv, nil
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
		"Milestone vendor ditambahkan: "+m.Name)
	return m, nil
}

type VendorMilestoneUpdateInput struct {
	Status        domain.MilestoneStatus
	TargetDate    time.Time
	CompletedDate *time.Time
	PICStaffID    int64
	Description   string
	Notes         string
}

func (s *VendorEngagementService) UpdateMilestone(ctx context.Context, projectID, projectVendorID, milestoneID int64, actorStaffID int64, input VendorMilestoneUpdateInput) (*domain.VendorMilestone, error) {
	if _, err := s.Get(ctx, projectID, projectVendorID); err != nil {
		return nil, err
	}
	m, err := s.milestones.FindByID(ctx, projectVendorID, milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, apperror.NotFound("Milestone vendor tidak ditemukan")
	}
	m.Status = input.Status
	m.TargetDate = input.TargetDate
	m.CompletedDate = input.CompletedDate
	m.PICStaffID = input.PICStaffID
	m.Description = input.Description
	m.Notes = input.Notes
	if err := s.milestones.Update(ctx, m); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "vendor_milestone", formatID(m.ID), m.Name,
		"Milestone vendor diperbarui")
	return m, nil
}
