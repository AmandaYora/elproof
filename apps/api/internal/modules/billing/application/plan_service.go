package application

import (
	"context"
	"sort"

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
	ListForApp(ctx context.Context, appID string) ([]domain.Plan, error)
	ListPublic(ctx context.Context) ([]domain.Plan, error)
	ListPublicPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error)
	ListPlanIDsForApp(ctx context.Context, appID string) ([]int64, error)
	ReplaceAppPlans(ctx context.Context, appID string, planIDs []int64) error
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

// ListForApp returns the catalog visible to appID (D3) — every plan mapped
// to it, including non-active ones (A3).
func (s *PlanService) ListForApp(ctx context.Context, appID string) ([]domain.Plan, error) {
	if appID == "" {
		return nil, apperror.Validation("appId wajib diisi", map[string][]string{"appId": {"appId wajib diisi"}})
	}
	return s.repo.ListForApp(ctx, appID)
}

// ListPublic returns ElProof's own catalog — plans with no App mapping (D3).
func (s *PlanService) ListPublic(ctx context.Context) ([]domain.Plan, error) {
	return s.repo.ListPublic(ctx)
}

// ListPublicActive is the catalog shown to anonymous visitors on the public
// homepage (GET /api/v1/public/plans): ListPublic narrowed to active plans,
// ordered by duration then price (id breaks ties so the order is stable).
// ListPublic itself keeps returning inactive plans for its existing callers.
func (s *PlanService) ListPublicActive(ctx context.Context) ([]domain.Plan, error) {
	plans, err := s.repo.ListPublic(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]domain.Plan, 0, len(plans))
	for _, p := range plans {
		if p.IsActive {
			active = append(active, p)
		}
	}
	sort.SliceStable(active, func(i, j int) bool {
		a, b := active[i], active[j]
		if a.DurationMonths != b.DurationMonths {
			return a.DurationMonths < b.DurationMonths
		}
		if a.Price != b.Price {
			return a.Price < b.Price
		}
		return a.ID < b.ID
	})
	return active, nil
}

// ListPublicPaginated is ListPublic with pagination — backs the branching
// list() paginated cabang (A12).
func (s *PlanService) ListPublicPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	return s.repo.ListPublicPaginated(ctx, params, search)
}

// GetAppPlanIDs returns the raw plan_id set mapped to appID — backs
// GET /api/v1/app-plans (task 7.3).
func (s *PlanService) GetAppPlanIDs(ctx context.Context, appID string) ([]int64, error) {
	if appID == "" {
		return nil, apperror.Validation("appId wajib diisi", map[string][]string{"appId": {"appId wajib diisi"}})
	}
	return s.repo.ListPlanIDsForApp(ctx, appID)
}

// SetAppPlans replaces appID's entire mapping set (A8, replace-set
// semantics). Duplicate planIDs are dropped before writing so a sensible
// payload never trips the UNIQUE (app_id, plan_id) constraint.
func (s *PlanService) SetAppPlans(ctx context.Context, appID string, planIDs []int64) error {
	if appID == "" {
		return apperror.Validation("appId wajib diisi", map[string][]string{"appId": {"appId wajib diisi"}})
	}
	return s.repo.ReplaceAppPlans(ctx, appID, dedupInt64(planIDs))
}

// dedupInt64 drops duplicates while keeping first-seen order stable.
func dedupInt64(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
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
