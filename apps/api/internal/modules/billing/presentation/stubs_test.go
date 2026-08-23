package presentation

import (
	"context"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/pagination"
)

// stubAppRegistry implements paymentcontracts.AppRegistry — shared across
// this package's handler tests (A34). Records whether each method was
// called so a test can assert a guard short-circuited before reaching it
// (e.g. TestAppPlans_BukanPlatformAdmin).
type stubAppRegistry struct {
	isActive       bool
	isActiveErr    error
	isActiveCalled bool

	exists       bool
	existsErr    error
	existsCalled bool
}

func (s *stubAppRegistry) IsAppActive(ctx context.Context, appID string) (bool, error) {
	s.isActiveCalled = true
	return s.isActive, s.isActiveErr
}

func (s *stubAppRegistry) AppExists(ctx context.Context, appID string) (bool, error) {
	s.existsCalled = true
	return s.exists, s.existsErr
}

// stubExternalPlanLister implements externalPlanLister.
type stubExternalPlanLister struct {
	plans []domain.Plan
	err   error
}

func (s *stubExternalPlanLister) ListForApp(ctx context.Context, appID string) ([]domain.Plan, error) {
	return s.plans, s.err
}

// stubAppPlanMapper implements appPlanMapper.
type stubAppPlanMapper struct {
	planIDs      []int64
	getErr       error
	setErr       error
	setCalled    bool
	lastSetAppID string
	lastSetIDs   []int64
}

func (s *stubAppPlanMapper) GetAppPlanIDs(ctx context.Context, appID string) ([]int64, error) {
	return s.planIDs, s.getErr
}

func (s *stubAppPlanMapper) SetAppPlans(ctx context.Context, appID string, planIDs []int64) error {
	s.setCalled = true
	s.lastSetAppID = appID
	s.lastSetIDs = planIDs
	return s.setErr
}

// stubSubscriptionCharger implements subscriptionCharger.
type stubSubscriptionCharger struct {
	result *paymentcontracts.ChargeResult
	err    error

	called       bool
	lastAppID    string
	lastOrderRef string
	lastPlanID   int64
}

func (s *stubSubscriptionCharger) StartSubscriptionCharge(ctx context.Context, appID, orderRef string, planID int64, opts paymentcontracts.ChargeOptions) (*paymentcontracts.ChargeResult, error) {
	s.called = true
	s.lastAppID = appID
	s.lastOrderRef = orderRef
	s.lastPlanID = planID
	return s.result, s.err
}

// stubPlanLister implements planLister — used by TestPlansList_* to lock
// the principal-aware branching (D3/A12) that closes T9/T16.
type stubPlanLister struct {
	all    []domain.Plan
	public []domain.Plan
	total  int64
	err    error

	calledList            bool
	calledListPaginated   bool
	calledListPublic      bool
	calledListPublicPaged bool
}

func (s *stubPlanLister) List(ctx context.Context) ([]domain.Plan, error) {
	s.calledList = true
	return s.all, s.err
}

func (s *stubPlanLister) ListPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	s.calledListPaginated = true
	return s.all, s.total, s.err
}

func (s *stubPlanLister) ListPublic(ctx context.Context) ([]domain.Plan, error) {
	s.calledListPublic = true
	return s.public, s.err
}

func (s *stubPlanLister) ListPublicPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	s.calledListPublicPaged = true
	return s.public, s.total, s.err
}

func (s *stubPlanLister) Get(ctx context.Context, id int64) (*domain.Plan, error) {
	return nil, nil
}

func (s *stubPlanLister) Create(ctx context.Context, input application.PlanInput) (*domain.Plan, error) {
	return nil, nil
}

func (s *stubPlanLister) Update(ctx context.Context, id int64, input application.PlanInput) (*domain.Plan, error) {
	return nil, nil
}

func (s *stubPlanLister) ToggleActive(ctx context.Context, id int64) (*domain.Plan, error) {
	return nil, nil
}

// stubAppSubscriptionLister implements appSubscriptionLister.
type stubAppSubscriptionLister struct {
	summaries []domain.AppSubscriptionSummary
	err       error
}

func (s *stubAppSubscriptionLister) LatestPaidPerApp(ctx context.Context) ([]domain.AppSubscriptionSummary, error) {
	return s.summaries, s.err
}
