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

// stubVoucherRepo keeps vouchers and redemptions in memory. Reserve runs the
// same decide callback the MySQL repository runs under its row lock.
type stubVoucherRepo struct {
	vouchers    map[string]*domain.Voucher
	redemptions []domain.VoucherRedemption
	deleted     []int64
	settled     map[string]domain.RedemptionStatus
	reserveErr  error
}

func newStubVoucherRepo(vs ...domain.Voucher) *stubVoucherRepo {
	r := &stubVoucherRepo{vouchers: map[string]*domain.Voucher{}, settled: map[string]domain.RedemptionStatus{}}
	for i := range vs {
		v := vs[i]
		r.vouchers[v.Code] = &v
	}
	return r
}

func (r *stubVoucherRepo) ListPaginated(context.Context, pagination.Params, domain.VoucherFilter) ([]domain.Voucher, int64, error) {
	return nil, 0, nil
}
func (r *stubVoucherRepo) FindByID(_ context.Context, id int64) (*domain.Voucher, error) {
	for _, v := range r.vouchers {
		if v.ID == id {
			c := *v
			return &c, nil
		}
	}
	return nil, nil
}
func (r *stubVoucherRepo) FindByCode(_ context.Context, code string) (*domain.Voucher, error) {
	if v, ok := r.vouchers[code]; ok {
		c := *v
		return &c, nil
	}
	return nil, nil
}
func (r *stubVoucherRepo) CodeTaken(_ context.Context, code string, exceptID int64) (bool, error) {
	v, ok := r.vouchers[code]
	return ok && v.ID != exceptID, nil
}
func (r *stubVoucherRepo) Create(_ context.Context, v *domain.Voucher) error {
	v.ID = int64(len(r.vouchers) + 100)
	c := *v
	r.vouchers[v.Code] = &c
	return nil
}
func (r *stubVoucherRepo) Update(_ context.Context, v *domain.Voucher, _ *time.Time) (bool, error) {
	c := *v
	r.vouchers[v.Code] = &c
	return true, nil
}
func (r *stubVoucherRepo) SetActive(context.Context, int64, bool) error { return nil }
func (r *stubVoucherRepo) CountRedemptions(_ context.Context, id int64) (int64, error) {
	var n int64
	for _, x := range r.redemptions {
		if x.VoucherID == id {
			n++
		}
	}
	return n, nil
}
func (r *stubVoucherRepo) ListRedemptions(context.Context, int64, pagination.Params) ([]domain.VoucherRedemption, int64, error) {
	return nil, 0, nil
}
func (r *stubVoucherRepo) CustomerUses(_ context.Context, id int64, c domain.Customer) (int64, error) {
	var n int64
	for _, x := range r.redemptions {
		if x.VoucherID == id && x.Customer == c && x.Status != domain.RedemptionReleased {
			n++
		}
	}
	return n, nil
}
func (r *stubVoucherRepo) Reserve(ctx context.Context, code string, c domain.Customer,
	decide func(*domain.Voucher, int64) (*domain.VoucherRedemption, error)) (*domain.VoucherRedemption, error) {
	if r.reserveErr != nil {
		return nil, r.reserveErr
	}
	v, _ := r.FindByCode(ctx, code)
	var uses int64
	if v != nil {
		uses, _ = r.CustomerUses(ctx, v.ID, c)
	}
	red, err := decide(v, uses)
	if err != nil {
		return nil, err
	}
	red.ID = int64(len(r.redemptions) + 1)
	r.redemptions = append(r.redemptions, *red)
	r.vouchers[code].ReservedCount++
	return red, nil
}
func (r *stubVoucherRepo) DeleteReservation(_ context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	return nil
}
func (r *stubVoucherRepo) Settle(_ context.Context, orderRef string, s domain.RedemptionStatus) error {
	r.settled[orderRef] = s
	return nil
}
func (r *stubVoucherRepo) ReleaseStale(context.Context, time.Time) (int64, error) { return 0, nil }

type stubVoucherPlans struct{ plans map[int64]*domain.Plan }

func (p stubVoucherPlans) FindByID(_ context.Context, id int64) (*domain.Plan, error) {
	return p.plans[id], nil
}

// fixed "today" (WIB): 10 Nov 2026, 10:00.
var voucherToday = time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

func voucherSvc(repo *stubVoucherRepo, plans ...*domain.Plan) *VoucherService {
	m := map[int64]*domain.Plan{}
	for _, p := range plans {
		m[p.ID] = p
	}
	s := NewVoucherService(repo, stubVoucherPlans{plans: m})
	s.now = func() time.Time { return voucherToday }
	return s
}

func i64(n int64) *int64 { return &n }

var (
	planElproof  = &domain.Plan{ID: 1, AppID: TenantPlansApp, Name: "Paket 1 Tahun", Price: 2_000_000, DurationMonths: 12}
	planElproof2 = &domain.Plan{ID: 2, AppID: TenantPlansApp, Name: "Paket Bulanan", Price: 250_000, DurationMonths: 1}
	planJWS      = &domain.Plan{ID: 9, AppID: "app_jws", Name: "JWS Gold", Price: 1_500_000, DurationMonths: 12}
	tenantA      = domain.Customer{Type: domain.CustomerTenant, Ref: "7"}
)

func reasonOf(err error) string {
	if ae, ok := apperror.As(err); ok {
		return ae.Code
	}
	return ""
}

func TestVoucherDiscountFor(t *testing.T) {
	for _, tc := range []struct {
		name  string
		v     domain.Voucher
		price int64
		want  int64
	}{
		{"persen dibulatkan ke bawah", domain.Voucher{Type: domain.DiscountPercent, Value: 15}, 99_999, 14_999},
		{"persen dibatasi maks", domain.Voucher{Type: domain.DiscountPercent, Value: 50, MaxDiscount: i64(100_000)}, 2_000_000, 100_000},
		{"persen di bawah batas", domain.Voucher{Type: domain.DiscountPercent, Value: 10, MaxDiscount: i64(500_000)}, 2_000_000, 200_000},
		{"nominal", domain.Voucher{Type: domain.DiscountAmount, Value: 300_000}, 2_000_000, 300_000},
	} {
		if got := tc.v.DiscountFor(tc.price); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestVoucherQuote_Alasan(t *testing.T) {
	base := domain.Voucher{ID: 1, Code: "HEMAT10", AppID: TenantPlansApp, Type: domain.DiscountPercent, Value: 10, IsActive: true}
	with := func(f func(v *domain.Voucher)) domain.Voucher { v := base; f(&v); return v }
	cases := []struct {
		name   string
		v      domain.Voucher
		plan   *domain.Plan
		code   string
		used   bool // the tenant already holds a use
		reason string
	}{
		{"kode salah", base, planElproof, "SALAH", false, VoucherNotFound},
		{"kode kosong", base, planElproof, "  ", false, VoucherNotFound},
		{"milik aplikasi lain sama dengan tidak ada", base, planJWS, "HEMAT10", false, VoucherNotFound},
		{"nonaktif", with(func(v *domain.Voucher) { v.IsActive = false }), planElproof, "HEMAT10", false, VoucherInactive},
		{"belum mulai", with(func(v *domain.Voucher) { v.ValidFrom = "2026-11-11" }), planElproof, "HEMAT10", false, VoucherNotStarted},
		{"kedaluwarsa", with(func(v *domain.Voucher) { v.ValidUntil = "2026-11-09" }), planElproof, "HEMAT10", false, VoucherExpired},
		{"beda paket", with(func(v *domain.Voucher) { v.PlanID = i64(2); v.PlanName = "Paket Bulanan" }), planElproof, "HEMAT10", false, VoucherWrongPlan},
		{"kuota habis", with(func(v *domain.Voucher) { v.Quota = i64(3); v.UsedCount = 2; v.ReservedCount = 1 }), planElproof, "HEMAT10", false, VoucherExhausted},
		{"sudah dipakai pelanggan ini", with(func(v *domain.Voucher) { v.OncePerCustomer = true }), planElproof, "HEMAT10", true, VoucherAlreadyUsed},
		{"potongan menghabiskan harga", with(func(v *domain.Voucher) { v.Type = domain.DiscountAmount; v.Value = 250_000 }), planElproof2, "HEMAT10", false, VoucherTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newStubVoucherRepo(tc.v)
			if tc.used {
				repo.redemptions = append(repo.redemptions, domain.VoucherRedemption{VoucherID: tc.v.ID, Customer: tenantA, Status: domain.RedemptionUsed})
			}
			_, err := voucherSvc(repo).Quote(context.Background(), tc.code, tc.plan, tenantA)
			if apperror.HTTPStatus(err) != 422 || reasonOf(err) != tc.reason {
				t.Fatalf("err = %v (reason %q), want 422 %s", err, reasonOf(err), tc.reason)
			}
			if ae, _ := apperror.As(err); len(ae.Fields["voucherCode"]) == 0 || ae.Message == "" {
				t.Fatalf("a refusal must carry a readable message on voucherCode: %+v", ae)
			}
		})
	}
}

func TestVoucherQuote_Berlaku(t *testing.T) {
	v := domain.Voucher{ID: 1, Code: "HEMAT10", AppID: TenantPlansApp, Type: domain.DiscountPercent, Value: 10, MaxDiscount: i64(150_000),
		IsActive: true, ValidFrom: "2026-11-10", ValidUntil: "2026-11-10", Quota: i64(5), UsedCount: 4, OncePerCustomer: true}
	repo := newStubVoucherRepo(v)
	// A released use of the same tenant no longer counts.
	repo.redemptions = append(repo.redemptions, domain.VoucherRedemption{VoucherID: 1, Customer: tenantA, Status: domain.RedemptionReleased})
	q, err := voucherSvc(repo).Quote(context.Background(), " hemat10 ", planElproof, tenantA)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if q.OriginalAmount != 2_000_000 || q.DiscountAmount != 150_000 || q.FinalAmount != 1_850_000 || q.Code != "HEMAT10" {
		t.Fatalf("quote = %+v", q)
	}
	// 99% leaves something to pay.
	repo = newStubVoucherRepo(domain.Voucher{ID: 2, Code: "HAMPIR", AppID: TenantPlansApp, Type: domain.DiscountPercent, Value: 99, IsActive: true})
	q, err = voucherSvc(repo).Quote(context.Background(), "HAMPIR", planElproof2, tenantA)
	if err != nil || q.FinalAmount != 2_500 {
		t.Fatalf("99%% of 250.000: %+v, %v", q, err)
	}
}

func TestVoucherValidate(t *testing.T) {
	ok := VoucherInput{Code: "hemat-10", AppID: TenantPlansApp, Type: domain.DiscountPercent, Value: 10}
	with := func(f func(in *VoucherInput)) VoucherInput { in := ok; f(&in); return in }
	for _, tc := range []struct {
		name  string
		in    VoucherInput
		field string
	}{
		{"kode pendek", with(func(in *VoucherInput) { in.Code = "A" }), "code"},
		{"kode berspasi", with(func(in *VoucherInput) { in.Code = "HE MAT" }), "code"},
		{"tanpa aplikasi", with(func(in *VoucherInput) { in.AppID = " " }), "appId"},
		{"persen 100", with(func(in *VoucherInput) { in.Value = 100 }), "discountValue"},
		{"persen 0", with(func(in *VoucherInput) { in.Value = 0 }), "discountValue"},
		{"jenis kosong", with(func(in *VoucherInput) { in.Type = "" }), "discountType"},
		{"tanggal terbalik", with(func(in *VoucherInput) { in.ValidFrom = "2026-12-01"; in.ValidUntil = "2026-11-01" }), "validUntil"},
		{"kuota 0", with(func(in *VoucherInput) { in.Quota = i64(0) }), "quota"},
		{"paket aplikasi lain", with(func(in *VoucherInput) { in.PlanID = i64(9) }), "planId"},
		{"nominal setara harga paket", with(func(in *VoucherInput) { in.Type = domain.DiscountAmount; in.Value = 250_000; in.PlanID = i64(2) }), "discountValue"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := voucherSvc(newStubVoucherRepo(), planElproof2, planJWS).Create(context.Background(), tc.in)
			ae, _ := apperror.As(err)
			if ae == nil || ae.Kind != apperror.KindValidation || len(ae.Fields[tc.field]) == 0 {
				t.Fatalf("want 422 on %s, got %v", tc.field, err)
			}
		})
	}
	repo := newStubVoucherRepo(domain.Voucher{ID: 5, Code: "HEMAT-10"})
	if _, err := voucherSvc(repo).Create(context.Background(), ok); reasonOrField(err) != "code" {
		t.Fatalf("duplicate code (case-insensitive) must be refused on code, got %v", err)
	}
	v, err := voucherSvc(newStubVoucherRepo()).Create(context.Background(), with(func(in *VoucherInput) {
		in.Type = domain.DiscountAmount
		in.Value = 50_000
		in.MaxDiscount = i64(10)
	}))
	if err != nil || v.Code != "HEMAT-10" || v.MaxDiscount != nil {
		t.Fatalf("an amount voucher is stored upper case without a cap: %+v, %v", v, err)
	}
}

func reasonOrField(err error) string {
	ae, ok := apperror.As(err)
	if !ok {
		return ""
	}
	for f := range ae.Fields {
		return f
	}
	return ae.Code
}

func TestVoucherUpdate_AplikasiTerkunciSetelahDipakai(t *testing.T) {
	repo := newStubVoucherRepo(domain.Voucher{ID: 3, Code: "JWS10", AppID: "app_jws", Type: domain.DiscountPercent, Value: 10, IsActive: true})
	repo.redemptions = append(repo.redemptions, domain.VoucherRedemption{VoucherID: 3, Status: domain.RedemptionReleased})
	_, err := voucherSvc(repo).Update(context.Background(), 3, VoucherInput{Code: "JWS10", AppID: TenantPlansApp, Type: domain.DiscountPercent, Value: 10}, nil)
	if reasonOrField(err) != "appId" {
		t.Fatalf("changing the App of a used voucher must be refused on appId, got %v", err)
	}
	stale := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = voucherSvc(repo).Update(context.Background(), 3, VoucherInput{Code: "JWS10", AppID: "app_jws", Type: domain.DiscountPercent, Value: 20}, &stale)
	if _, ok := apperror.IsStale(err); !ok {
		t.Fatalf("an old If-Match must be a stale 409, got %v", err)
	}
}

// ---- subscription charge with a voucher (external App) ----

func TestSubCharge_DenganVoucher(t *testing.T) {
	repo := newStubVoucherRepo(domain.Voucher{ID: 4, Code: "JWS20", AppID: "app_jws", Type: domain.DiscountPercent, Value: 20, IsActive: true, OncePerCustomer: true})
	charges := &stubAppChargeWriter{}
	payment := &stubPaymentClient{result: &paymentcontracts.ChargeResult{OrderRef: "ORD-V1", Amount: 1_200_000}}
	s := NewSubscriptionService(catalogWith(planJWS), charges, payment)
	s.SetVouchers(voucherSvc(repo))

	out, err := s.StartSubscriptionCharge(context.Background(), "app_jws", "ORD-V1", 9, "jws20", paymentcontracts.ChargeOptions{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if payment.lastAmount != 1_200_000 || out.OriginalAmount != 1_500_000 || out.DiscountAmount != 300_000 || out.VoucherCode != "JWS20" {
		t.Fatalf("charged %d, out %+v", payment.lastAmount, out)
	}
	if len(repo.redemptions) != 1 || repo.redemptions[0].Customer != (domain.Customer{Type: domain.CustomerApp, Ref: "app_jws"}) || repo.redemptions[0].OrderRef != "ORD-V1" {
		t.Fatalf("the use must be reserved for the App on this order: %+v", repo.redemptions)
	}

	// The same App again: once per customer.
	_, err = s.StartSubscriptionCharge(context.Background(), "app_jws", "ORD-V2", 9, "JWS20", paymentcontracts.ChargeOptions{})
	if reasonOf(err) != VoucherAlreadyUsed {
		t.Fatalf("second use by the same App: %v", err)
	}
}

func TestSubCharge_VoucherAplikasiLainDitolak(t *testing.T) {
	repo := newStubVoucherRepo(domain.Voucher{ID: 4, Code: "ELP10", AppID: TenantPlansApp, Type: domain.DiscountPercent, Value: 10, IsActive: true})
	charges := &stubAppChargeWriter{}
	payment := &stubPaymentClient{result: &paymentcontracts.ChargeResult{OrderRef: "ORD-V1"}}
	s := NewSubscriptionService(catalogWith(planJWS), charges, payment)
	s.SetVouchers(voucherSvc(repo))
	_, err := s.StartSubscriptionCharge(context.Background(), "app_jws", "ORD-V1", 9, "ELP10", paymentcontracts.ChargeOptions{})
	if reasonOf(err) != VoucherNotFound || charges.createCalled || payment.createChannelCalled {
		t.Fatalf("a code of another App must look unknown and stop before any write: %v", err)
	}
	if _, err := s.QuoteVoucher(context.Background(), "app_jws", 9, "ELP10"); reasonOf(err) != VoucherNotFound {
		t.Fatalf("voucher-check: %v", err)
	}
}

func TestSubCharge_GagalMelepasReservasi(t *testing.T) {
	for _, tc := range []struct {
		name       string
		createErr  error
		paymentErr error
	}{{"intent gagal ditulis", errors.New("dup"), nil}, {"gateway gagal", nil, errors.New("gateway down")}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newStubVoucherRepo(domain.Voucher{ID: 4, Code: "JWS20", AppID: "app_jws", Type: domain.DiscountPercent, Value: 20, IsActive: true})
			charges := &stubAppChargeWriter{createErr: tc.createErr}
			payment := &stubPaymentClient{result: &paymentcontracts.ChargeResult{}, err: tc.paymentErr}
			s := NewSubscriptionService(catalogWith(planJWS), charges, payment)
			s.SetVouchers(voucherSvc(repo))
			if _, err := s.StartSubscriptionCharge(context.Background(), "app_jws", "ORD-V1", 9, "JWS20", paymentcontracts.ChargeOptions{}); err == nil {
				t.Fatal("the error must reach the caller")
			}
			if len(repo.deleted) != 1 || repo.deleted[0] != repo.redemptions[0].ID {
				t.Fatalf("the reservation must be deleted, deleted=%v", repo.deleted)
			}
			if tc.createErr != nil && charges.deleteCalled {
				t.Fatal("an intent that was never written by us (e.g. a duplicate orderRef) must not be deleted")
			}
		})
	}
}

// ---- settlement through the ledger ----

type stubTxRepo struct {
	TransactionRepository
	updated domain.TransactionStatus
}

func (s *stubTxRepo) UpdateStatusByReference(_ context.Context, _ string, status domain.TransactionStatus, _ *time.Time) error {
	s.updated = status
	return nil
}

type recordingSettler struct {
	calls map[string]bool
	err   error
}

func (r *recordingSettler) Settle(_ context.Context, orderRef string, paid bool) error {
	r.calls[orderRef] = paid
	return r.err
}

func TestTransactionStatusMenyelesaikanVoucher(t *testing.T) {
	for _, tc := range []struct {
		status  domain.TransactionStatus
		settled bool
		paid    bool
	}{
		{domain.StatusPaid, true, true},
		{domain.StatusExpired, true, false},
		{domain.StatusCancelled, true, false},
		{domain.StatusPending, false, false},
	} {
		settler := &recordingSettler{calls: map[string]bool{}}
		s := NewTransactionService(&stubTxRepo{})
		s.SetVouchers(settler)
		if err := s.UpdateStatus(context.Background(), "INV-1", tc.status); err != nil {
			t.Fatal(err)
		}
		paid, settled := settler.calls["INV-1"]
		if settled != tc.settled || paid != tc.paid {
			t.Errorf("%s: settled=%v paid=%v, want %v/%v", tc.status, settled, paid, tc.settled, tc.paid)
		}
	}
}

// A voucher settle that fails must not fail the status change: a paid webhook
// would otherwise never activate the subscription (the sweep settles it later).
func TestTransactionStatus_GagalCatatVoucherTidakMenggagalkan(t *testing.T) {
	s := NewTransactionService(&stubTxRepo{})
	s.SetVouchers(&recordingSettler{calls: map[string]bool{}, err: errors.New("lock wait timeout")})
	if err := s.UpdateStatus(context.Background(), "INV-1", domain.StatusPaid); err != nil {
		t.Fatalf("paid must go through even if the voucher cannot be settled: %v", err)
	}
}

func TestVoucherValidate_KuotaDalamBatasKolom(t *testing.T) {
	in := VoucherInput{Code: "BESAR", AppID: TenantPlansApp, Type: domain.DiscountPercent, Value: 10, Quota: i64(5_000_000_000)}
	if _, err := voucherSvc(newStubVoucherRepo()).Create(context.Background(), in); reasonOrField(err) != "quota" {
		t.Fatalf("quota beyond INT UNSIGNED must be a 422 on quota, got %v", err)
	}
}
