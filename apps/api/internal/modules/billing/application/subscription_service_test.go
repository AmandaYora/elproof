package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
)

// stubPlanCatalog implements PlanCatalog.
type stubPlanCatalog struct {
	mappedIDs []int64
	getIDsErr error
	plan      *domain.Plan
	getErr    error
}

func (s *stubPlanCatalog) GetAppPlanIDs(ctx context.Context, appID string) ([]int64, error) {
	return s.mappedIDs, s.getIDsErr
}

func (s *stubPlanCatalog) Get(ctx context.Context, id int64) (*domain.Plan, error) {
	return s.plan, s.getErr
}

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

func TestSubCharge_PaketTidakDipetakanJadi422(t *testing.T) {
	plans := &stubPlanCatalog{mappedIDs: []int64{1, 2}, plan: &domain.Plan{ID: 99, Price: 2000000, DurationMonths: 12}}
	charges := &stubAppChargeWriter{}
	payment := &stubPaymentClient{result: &paymentcontracts.ChargeResult{OrderRef: "ORD-1"}}
	s := NewSubscriptionService(plans, charges, payment)

	_, err := s.StartSubscriptionCharge(context.Background(), "app_x", "ORD-1", 99, paymentcontracts.ChargeOptions{})

	if err == nil {
		t.Fatal("planId 99 tidak ada di mappedIDs [1,2], harus galat 422")
	}
	if charges.createCalled {
		t.Fatal("intent TIDAK boleh ditulis untuk paket yang bukan milik App ini (8.4) — ini penjaga sebenarnya terhadap lintas-App")
	}
	if payment.createChannelCalled {
		t.Fatal("payment.Client TIDAK boleh dipanggil sama sekali")
	}
}

func TestSubCharge_NominalDariKatalogBukanBody(t *testing.T) {
	plans := &stubPlanCatalog{mappedIDs: []int64{1}, plan: &domain.Plan{ID: 1, Price: 2000000, DurationMonths: 12, Name: "Paket JWS"}}
	charges := &stubAppChargeWriter{}
	payment := &stubPaymentClient{result: &paymentcontracts.ChargeResult{OrderRef: "ORD-1"}}
	s := NewSubscriptionService(plans, charges, payment)

	// Body/caller tidak pernah punya kesempatan mengirim amount sendiri —
	// signature StartSubscriptionCharge tidak menerima parameter amount
	// sama sekali (A18); nilainya HARUS berasal dari plans.Get().
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

	plans := &stubPlanCatalog{mappedIDs: []int64{1}, plan: &domain.Plan{ID: 1, Price: 2000000, DurationMonths: 12}}
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
	plans := &stubPlanCatalog{mappedIDs: []int64{1}, plan: &domain.Plan{ID: 1, Price: 2000000, DurationMonths: 12}}
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

func TestDedupPlanIDs(t *testing.T) {
	got := dedupInt64([]int64{1, 1, 2})
	want := []int64{1, 2}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
