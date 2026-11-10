package application

import (
	"context"
	"sort"
	"strings"

	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

// TenantPlansApp is the App whose plans make up ElProof's own catalog for WO
// tenants: the internal App `platform` already charges tenant subscriptions
// through. `payment` knows nothing about plans — only this module decides
// that "ElProof's plans" means "the plans of that App" (ADR-0040).
const TenantPlansApp = paymentcontracts.InternalAppBilling

type PlanRepository interface {
	// List returns the plans of appID, or of every App when appID is "".
	List(ctx context.Context, appID string) ([]domain.Plan, error)
	ListPaginated(ctx context.Context, params pagination.Params, search, appID string) ([]domain.Plan, int64, error)
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

// List returns the plans of appID ("" = every App) — the Platform Console's
// view, and with TenantPlansApp the catalog every other principal sees.
func (s *PlanService) List(ctx context.Context, appID string) ([]domain.Plan, error) {
	return s.repo.List(ctx, appID)
}

func (s *PlanService) ListPaginated(ctx context.Context, params pagination.Params, search, appID string) ([]domain.Plan, int64, error) {
	return s.repo.ListPaginated(ctx, params, search, appID)
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

// GetForApp returns plan id only when it belongs to appID. A plan of another
// App answers exactly like a plan that does not exist — a 422 on planId — so
// a guessed id never confirms another App's catalog.
func (s *PlanService) GetForApp(ctx context.Context, appID string, id int64) (*domain.Plan, error) {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.AppID != appID {
		return nil, apperror.Validation("Paket ini tidak tersedia", map[string][]string{"planId": {"Pilih paket yang tersedia"}})
	}
	return plan, nil
}

// GetTenantPlan is GetForApp for ElProof's own catalog: the only plans a WO
// tenant may be subscribed to (self-service payment and manual activation).
func (s *PlanService) GetTenantPlan(ctx context.Context, id int64) (*domain.Plan, error) {
	return s.GetForApp(ctx, TenantPlansApp, id)
}

type PlanInput struct {
	AppID          string
	Name           string
	DurationMonths int
	Price          int64
	Features       []string
}

func validatePlanInput(input *PlanInput) error {
	input.AppID = strings.TrimSpace(input.AppID)
	if input.AppID == "" {
		return apperror.Validation("Pilih aplikasi pemilik paket", map[string][]string{"appId": {"Pilih aplikasi pemilik paket"}})
	}
	return nil
}

func (s *PlanService) Create(ctx context.Context, input PlanInput) (*domain.Plan, error) {
	if err := validatePlanInput(&input); err != nil {
		return nil, err
	}
	plan := &domain.Plan{
		AppID:          input.AppID,
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
	if err := validatePlanInput(&input); err != nil {
		return nil, err
	}
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperror.NotFound("Paket tidak ditemukan")
	}

	existing.AppID = input.AppID
	existing.Name = input.Name
	existing.DurationMonths = input.DurationMonths
	existing.Price = input.Price
	existing.Features = input.Features

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// ListForApp returns the catalog of appID — including non-active plans, with
// their flag, since a consumer marks its tenant's current plan with it (A3) —
// ordered by price, then id.
func (s *PlanService) ListForApp(ctx context.Context, appID string) ([]domain.Plan, error) {
	if appID == "" {
		return nil, apperror.Validation("appId wajib diisi", map[string][]string{"appId": {"appId wajib diisi"}})
	}
	plans, err := s.repo.List(ctx, appID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(plans, func(i, j int) bool { return plans[i].Price < plans[j].Price })
	return plans, nil
}

// ListPublicActive is the catalog shown to anonymous visitors on the public
// homepage (GET /api/v1/public/plans): ElProof's own active plans, ordered by
// duration then price (id breaks ties so the order is stable).
func (s *PlanService) ListPublicActive(ctx context.Context) ([]domain.Plan, error) {
	plans, err := s.repo.List(ctx, TenantPlansApp)
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
