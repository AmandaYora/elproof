package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

// stubPlanRepo implements PlanRepository with one stored plan, so these tests
// run the real PlanService.GetForApp ownership check (ADR-0040).
type stubPlanRepo struct {
	plan *domain.Plan
}

func (s *stubPlanRepo) List(ctx context.Context, appID string) ([]domain.Plan, error) {
	return nil, nil
}
func (s *stubPlanRepo) ListPaginated(ctx context.Context, params pagination.Params, search, appID string) ([]domain.Plan, int64, error) {
	return nil, 0, nil
}
func (s *stubPlanRepo) FindByID(ctx context.Context, id int64) (*domain.Plan, error) {
	if s.plan == nil || s.plan.ID != id {
		return nil, nil
	}
	p := *s.plan
	return &p, nil
}
func (s *stubPlanRepo) Create(ctx context.Context, plan *domain.Plan) error          { return nil }
func (s *stubPlanRepo) Update(ctx context.Context, plan *domain.Plan) error          { return nil }
func (s *stubPlanRepo) SetActive(ctx context.Context, id int64, isActive bool) error { return nil }

func catalogWith(plan *domain.Plan) *PlanService { return NewPlanService(&stubPlanRepo{plan: plan}) }

// stubAppChargeWriter implements AppChargeWriter. onCreate/onDelete let a
// test observe call order relative to stubPaymentClient's own hook (A20).
type stubAppChargeWriter struct {
	createErr       error
	createCalled    bool
	onCreate        func()
	deleteCalled    bool
	deletedOrderRef string
}

func (s *stubAppChargeWriter) Create(ctx context.Context, c *domain.AppCharge) error {
	s.createCalled = true
	if s.onCreate != nil {
		s.onCreate()
	}
	return s.createErr
}

func (s *stubAppChargeWriter) Delete(ctx context.Context, orderRef string) error {
	s.deleteCalled = true
	s.deletedOrderRef = orderRef
	return nil
}

// stubPaymentClient implements paymentcontracts.Client — only
// CreateChannelCharge is exercised by SubscriptionService, the rest are
// unused no-ops.
type stubPaymentClient struct {
	result                *paymentcontracts.ChargeResult
	err                   error
	createChannelCalled   bool
	lastAmount            int64
	onCreateChannelCharge func()
}

func (s *stubPaymentClient) Enabled(ctx context.Context) (bool, error) { return true, nil }
func (s *stubPaymentClient) QuoteFee(ctx context.Context, channel string, amount int64) (int64, error) {
	return 0, nil
}
func (s *stubPaymentClient) CreateCharge(ctx context.Context, appID, contextID, orderRef string, amount int64) (*paymentcontracts.ChargeResult, error) {
	return nil, nil
}
func (s *stubPaymentClient) CreateChannelCharge(ctx context.Context, appID, contextID, orderRef string, amount int64, channel string, opts paymentcontracts.ChargeOptions) (*paymentcontracts.ChargeResult, error) {
	s.createChannelCalled = true
	s.lastAmount = amount
	if s.onCreateChannelCharge != nil {
		s.onCreateChannelCharge()
	}
	return s.result, s.err
}
func (s *stubPaymentClient) ListChannels(ctx context.Context) ([]paymentcontracts.Channel, error) {
	return nil, nil
}
func (s *stubPaymentClient) CheckStatus(ctx context.Context, providerRef string) (*paymentcontracts.ChargeResult, error) {
	return nil, nil
}
func (s *stubPaymentClient) CheckStatusForApp(ctx context.Context, appID, orderRef string) (*paymentcontracts.ChargeResult, error) {
	return nil, nil
}

func TestSubCharge_PaketMilikAppLainJadi422(t *testing.T) {
	for _, tc := range []struct {
		name   string
		planID int64
	}{{"paket milik App lain", 99}, {"paket tidak ada", 100}} {
		t.Run(tc.name, func(t *testing.T) {
			plans := catalogWith(&domain.Plan{ID: 99, AppID: "app_lain", Price: 2000000, DurationMonths: 12})
			charges := &stubAppChargeWriter{}
			payment := &stubPaymentClient{result: &paymentcontracts.ChargeResult{OrderRef: "ORD-1"}}
			s := NewSubscriptionService(plans, charges, payment)

			_, err := s.StartSubscriptionCharge(context.Background(), "app_x", "ORD-1", tc.planID, paymentcontracts.ChargeOptions{})

			if apperror.HTTPStatus(err) != 422 {
				t.Fatalf("galat = %v (status %d), want 422 — paket milik App lain dijawab sama dengan paket yang tidak ada", err, apperror.HTTPStatus(err))
			}
			if charges.createCalled {
				t.Fatal("intent TIDAK boleh ditulis untuk paket yang bukan milik App ini — ini penjaga sebenarnya terhadap lintas-App")
			}
			if payment.createChannelCalled {
				t.Fatal("payment.Client TIDAK boleh dipanggil sama sekali")
			}
		})
	}
}

func TestSubCharge_PaketTanpaAppIdTidakLolos(t *testing.T) {
	plans := catalogWith(&domain.Plan{ID: 1, AppID: "app_x", Price: 1})
	s := NewSubscriptionService(plans, &stubAppChargeWriter{}, &stubPaymentClient{})
	if _, err := s.StartSubscriptionCharge(context.Background(), "", "ORD-1", 1, paymentcontracts.ChargeOptions{}); err == nil {
		t.Fatal("appId kosong harus ditolak")
	}
}

func TestPlanService_GetTenantPlanHanyaKatalogElProof(t *testing.T) {
	if _, err := catalogWith(&domain.Plan{ID: 5, AppID: "app_jws"}).GetTenantPlan(context.Background(), 5); apperror.HTTPStatus(err) != 422 {
		t.Fatalf("paket milik App eksternal harus 422 untuk tenant WO, got %v", err)
	}
	p, err := catalogWith(&domain.Plan{ID: 6, AppID: TenantPlansApp}).GetTenantPlan(context.Background(), 6)
	if err != nil || p.ID != 6 {
		t.Fatalf("paket ElProof harus lolos, got %v, %v", p, err)
	}
}

func TestPlanService_AppIdWajib(t *testing.T) {
	s := catalogWith(nil)
	if _, err := s.Create(context.Background(), PlanInput{AppID: "  ", Name: "X"}); apperror.HTTPStatus(err) != 422 {
		t.Fatalf("Create tanpa appId harus 422, got %v", err)
	}
	if _, err := s.Update(context.Background(), 1, PlanInput{Name: "X"}); apperror.HTTPStatus(err) != 422 {
		t.Fatalf("Update tanpa appId harus 422, got %v", err)
	}
}

func TestSubCharge_NominalDariKatalogBukanBody(t *testing.T) {
	plans := catalogWith(&domain.Plan{ID: 1, AppID: "app_x", Price: 2000000, DurationMonths: 12, Name: "Paket JWS"})
	charges := &stubAppChargeWriter{}
	payment := &stubPaymentClient{result: &paymentcontracts.ChargeResult{OrderRef: "ORD-1"}}
	s := NewSubscriptionService(plans, charges, payment)

	// Body/caller tidak pernah punya kesempatan mengirim amount sendiri —
	// signature StartSubscriptionCharge tidak menerima parameter amount
	// sama sekali (A18); nilainya HARUS berasal dari katalog (GetForApp).
	_, err := s.StartSubscriptionCharge(context.Background(), "app_x", "ORD-1", 1, paymentcontracts.ChargeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.lastAmount != 2000000 {
		t.Fatalf("amount ke payment.Client = %d, want 2000000 (dari katalog)", payment.lastAmount)
	}
}

func TestSubCharge_UrutanIntentSebelumCharge(t *testing.T) {
	var intentWritten bool
	var chargeCalledAfterIntentWritten bool

	plans := catalogWith(&domain.Plan{ID: 1, AppID: "app_x", Price: 2000000, DurationMonths: 12})
	charges := &stubAppChargeWriter{onCreate: func() { intentWritten = true }}
	payment := &stubPaymentClient{
		result:                &paymentcontracts.ChargeResult{OrderRef: "ORD-1"},
		onCreateChannelCharge: func() { chargeCalledAfterIntentWritten = intentWritten },
	}
	s := NewSubscriptionService(plans, charges, payment)

	if _, err := s.StartSubscriptionCharge(context.Background(), "app_x", "ORD-1", 1, paymentcontracts.ChargeOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !chargeCalledAfterIntentWritten {
		t.Fatal("intent HARUS sudah tertulis sebelum payment.Client dipanggil (A20)")
	}
}

func TestSubCharge_ChargeGagalMenghapusIntent(t *testing.T) {
	plans := catalogWith(&domain.Plan{ID: 1, AppID: "app_x", Price: 2000000, DurationMonths: 12})
	charges := &stubAppChargeWriter{}
	payment := &stubPaymentClient{err: errors.New("gateway down")}
	s := NewSubscriptionService(plans, charges, payment)

	_, err := s.StartSubscriptionCharge(context.Background(), "app_x", "ORD-1", 1, paymentcontracts.ChargeOptions{})

	if err == nil {
		t.Fatal("galat dari payment.Client harus diteruskan")
	}
	if !charges.deleteCalled || charges.deletedOrderRef != "ORD-1" {
		t.Fatalf("intent harus dihapus (orderRef=%q, called=%v) supaya orderRef sama bisa dicoba lagi", charges.deletedOrderRef, charges.deleteCalled)
	}
}

// timeCloseEnough tolerates the few-microsecond drift between the test's
// own time.Now() and the one ComputeUntil calls internally — the function
// deliberately has no injectable clock (matches the class diagram's
// ComputeUntil(currentUntil, durationMonths) signature), so exact equality
// isn't meaningful here.
func timeCloseEnough(t *testing.T, got, want time.Time) {
	t.Helper()
	diff := got.Sub(want)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Fatalf("got %v, want ~%v (diff %v)", got, want, diff)
	}
}

func TestComputeUntil_DariKosong(t *testing.T) {
	s := &SubscriptionService{}
	got := s.ComputeUntil(nil, 12)
	want := time.Now().AddDate(0, 12, 0)
	timeCloseEnough(t, got, want)
}

func TestComputeUntil_PerpanjanganMenumpuk(t *testing.T) {
	s := &SubscriptionService{}
	currentUntil := time.Now().AddDate(0, 0, 10) // 10 hari lagi, masih berjalan
	got := s.ComputeUntil(&currentUntil, 1)
	want := currentUntil.AddDate(0, 1, 0)
	timeCloseEnough(t, got, want)

	// Sanity: hasilnya HARUS bukan today+1bulan — itu akan merugikan
	// pelanggan yang bayar lebih awal.
	wrong := time.Now().AddDate(0, 1, 0)
	if got.Sub(wrong) < 5*24*time.Hour {
		t.Fatalf("ComputeUntil tampaknya dihitung dari hari ini, bukan dari currentUntil: got=%v", got)
	}
}

func TestComputeUntil_LanggananSudahLewat(t *testing.T) {
	s := &SubscriptionService{}
	currentUntil := time.Now().AddDate(0, 0, -10) // 10 hari lalu, sudah lewat
	got := s.ComputeUntil(&currentUntil, 1)
	want := time.Now().AddDate(0, 1, 0) // dihitung dari HARI INI, bukan dari tanggal basi
	timeCloseEnough(t, got, want)
}
