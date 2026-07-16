package application

import (
	"context"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type PlanRepository interface {
	List(ctx context.Context) ([]domain.Plan, error)
	ListPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error)
	FindByID(ctx context.Context, id int64) (*domain.Plan, error)
	Create(ctx context.Context, plan *domain.Plan) error
	Update(ctx context.Context, plan *domain.Plan) error
	SetActive(ctx context.Context, id int64, isActive bool) error
}

type PlanService struct {
	repo PlanRepository
}

func NewPlanService(repo PlanRepository) *PlanService {
	return &PlanService{repo: repo}
}

func (s *PlanService) List(ctx context.Context) ([]domain.Plan, error) {
	return s.repo.List(ctx)
}

func (s *PlanService) ListPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	return s.repo.ListPaginated(ctx, params, search)
}

func (s *PlanService) Get(ctx context.Context, id int64) (*domain.Plan, error) {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, apperror.NotFound("Paket tidak ditemukan")
	}
	return plan, nil
}

type PlanInput struct {
	Name           string
	DurationMonths int
	Price          int64
	Features       []string
}

func (s *PlanService) Create(ctx context.Context, input PlanInput) (*domain.Plan, error) {
	plan := &domain.Plan{
		Name:           input.Name,
		DurationMonths: input.DurationMonths,
		Price:          input.Price,
		Features:       input.Features,
		IsActive:       true,
	}
	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *PlanService) Update(ctx context.Context, id int64, input PlanInput) (*domain.Plan, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperror.NotFound("Paket tidak ditemukan")
	}

	existing.Name = input.Name
	existing.DurationMonths = input.DurationMonths
	existing.Price = input.Price
	existing.Features = input.Features

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *PlanService) ToggleActive(ctx context.Context, id int64) (*domain.Plan, error) {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, apperror.NotFound("Paket tidak ditemukan")
	}
	if err := s.repo.SetActive(ctx, id, !plan.IsActive); err != nil {
		return nil, err
	}
	plan.IsActive = !plan.IsActive
	return plan, nil
}
