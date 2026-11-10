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
// (e.g. TestPlanCreate_AppTidakAdaJadi422).
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

// stubSubscriptionCharger implements subscriptionCharger.
type stubSubscriptionCharger struct {
	result *paymentcontracts.ChargeResult
	err    error

	called       bool
	lastAppID    string
	lastOrderRef string
	lastPlanID   int64
	lastVoucher  string
	discount     int64
	quote        *application.VoucherQuote
}

func (s *stubSubscriptionCharger) StartSubscriptionCharge(ctx context.Context, appID, orderRef string, planID int64, voucherCode string, opts paymentcontracts.ChargeOptions) (*application.SubscriptionCharge, error) {
	s.called = true
	s.lastAppID = appID
	s.lastOrderRef = orderRef
	s.lastPlanID = planID
	s.lastVoucher = voucherCode
	if s.err != nil {
		return nil, s.err
	}
	return &application.SubscriptionCharge{Charge: s.result, OriginalAmount: s.result.Amount + s.discount, DiscountAmount: s.discount, VoucherCode: voucherCode}, nil
}

func (s *stubSubscriptionCharger) QuoteVoucher(ctx context.Context, appID string, planID int64, voucherCode string) (*application.VoucherQuote, error) {
	s.lastAppID, s.lastPlanID, s.lastVoucher = appID, planID, voucherCode
	if s.err != nil {
		return nil, s.err
	}
	return s.quote, nil
}

// stubPlanLister implements planLister — used by TestPlansList_* to lock
// which App's plans each principal is served (ADR-0040).
type stubPlanLister struct {
	plans []domain.Plan
	total int64
	err   error

	calledList          bool
	calledListPaginated bool
	lastAppID           string // the appID the last list call was scoped to

	created *application.PlanInput
	updated *application.PlanInput
}

func (s *stubPlanLister) List(ctx context.Context, appID string) ([]domain.Plan, error) {
	s.calledList = true
	s.lastAppID = appID
	return s.plans, s.err
}

func (s *stubPlanLister) ListPaginated(ctx context.Context, params pagination.Params, search, appID string) ([]domain.Plan, int64, error) {
	s.calledListPaginated = true
	s.lastAppID = appID
	return s.plans, s.total, s.err
}

func (s *stubPlanLister) Get(ctx context.Context, id int64) (*domain.Plan, error) {
	return nil, nil
}

func (s *stubPlanLister) Create(ctx context.Context, input application.PlanInput) (*domain.Plan, error) {
	s.created = &input
	return &domain.Plan{ID: 1, AppID: input.AppID, Name: input.Name}, nil
}

func (s *stubPlanLister) Update(ctx context.Context, id int64, input application.PlanInput) (*domain.Plan, error) {
	s.updated = &input
	return &domain.Plan{ID: id, AppID: input.AppID, Name: input.Name}, nil
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
