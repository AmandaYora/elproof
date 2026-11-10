package application

import (
	"context"
	"errors"
	"testing"

	billingcontracts "elproof/internal/modules/billing/contracts"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/domain"
)

// T-20261110-17: Pay with a voucher reserves it in billing, charges and records
// the discounted amount, and gives the use back when no pending charge results.

type payTenantRepo struct{ TenantRepository }

func (payTenantRepo) FindByID(_ context.Context, id int64) (*domain.Tenant, error) {
	return &domain.Tenant{ID: id}, nil
}

type payPending struct {
	created   bool
	createErr error
}

func (p *payPending) Create(context.Context, string, int64, int64) error {
	p.created = true
	return p.createErr
}
func (p *payPending) FindByOrderRef(context.Context, string) (*domain.PendingCharge, error) {
	return nil, nil
}
func (p *payPending) FindByTenant(context.Context, int64) ([]domain.PendingCharge, error) {
	return nil, nil
}
func (p *payPending) Delete(context.Context, string) error { return nil }

type payBilling struct {
	billingcontracts.Contracts
	reserveErr error
	reserved   string // orderRef
	released   bool
	recorded   *billingcontracts.RecordTransactionInput
	recordErr  error
	statuses   []billingcontracts.TransactionStatus
}

func (b *payBilling) GetTenantPlan(context.Context, int64) (*billingcontracts.Plan, error) {
	return &billingcontracts.Plan{ID: 1, Name: "Paket 1 Tahun", DurationMonths: 12, Price: 2_000_000}, nil
}
func (b *payBilling) ReserveTenantVoucher(_ context.Context, _, _ int64, code, orderRef string) (*billingcontracts.VoucherReservation, error) {
	if b.reserveErr != nil {
		return nil, b.reserveErr
	}
	b.reserved = orderRef
	return &billingcontracts.VoucherReservation{VoucherQuote: billingcontracts.VoucherQuote{Code: "HEMAT10", OriginalAmount: 2_000_000, DiscountAmount: 200_000, FinalAmount: 1_800_000}}, nil
}
func (b *payBilling) ReleaseVoucher(context.Context, *billingcontracts.VoucherReservation) error {
	b.released = true
	return nil
}
func (b *payBilling) RecordTransaction(_ context.Context, in billingcontracts.RecordTransactionInput) error {
	b.recorded = &in
	return b.recordErr
}
func (b *payBilling) UpdateTransactionStatus(_ context.Context, _ string, s billingcontracts.TransactionStatus) error {
	b.statuses = append(b.statuses, s)
	return nil
}

type payGateway struct {
	paymentcontracts.Client
	err      error
	amount   int64
	orderRef string
}

func (g *payGateway) CreateCharge(_ context.Context, _, _, orderRef string, amount int64) (*paymentcontracts.ChargeResult, error) {
	g.amount, g.orderRef = amount, orderRef
	if g.err != nil {
		return nil, g.err
	}
	return &paymentcontracts.ChargeResult{OrderRef: orderRef, Channel: "QRIS", Amount: amount}, nil
}

func payService(b *payBilling, g *payGateway, p *payPending) *TenantService {
	return NewTenantService(payTenantRepo{}, p, nil, nil, b, g, nil, nil)
}

func TestPay_VoucherDipotongDanDicatat(t *testing.T) {
	b, g, p := &payBilling{}, &payGateway{}, &payPending{}
	charge, err := payService(b, g, p).Pay(context.Background(), 7, 1, "hemat10")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if g.amount != 1_800_000 || charge.Amount != 1_800_000 || b.reserved != g.orderRef {
		t.Fatalf("charged %d for %s, reserved for %s", g.amount, g.orderRef, b.reserved)
	}
	r := b.recorded
	if r == nil || r.Amount != 1_800_000 || r.OriginalAmount != 2_000_000 || r.DiscountAmount != 200_000 || r.VoucherCode != "HEMAT10" || r.PaymentReference != g.orderRef {
		t.Fatalf("ledger row: %+v", r)
	}
	if b.released || !p.created {
		t.Fatal("a successful charge keeps its reservation and is tracked")
	}
}

func TestPay_TanpaVoucherTidakMenyentuhVoucher(t *testing.T) {
	b, g := &payBilling{}, &payGateway{}
	if _, err := payService(b, g, &payPending{}).Pay(context.Background(), 7, 1, "  "); err != nil {
		t.Fatal(err)
	}
	if b.reserved != "" || g.amount != 2_000_000 || b.recorded.DiscountAmount != 0 {
		t.Fatalf("no voucher: reserved=%q amount=%d", b.reserved, g.amount)
	}
}

func TestPay_VoucherDilepasBilaTagihanTidakJadi(t *testing.T) {
	// Refused code: nothing is charged.
	b, g := &payBilling{reserveErr: errors.New("voucher_expired")}, &payGateway{}
	if _, err := payService(b, g, &payPending{}).Pay(context.Background(), 7, 1, "LAMA"); err == nil || g.orderRef != "" {
		t.Fatalf("a refused voucher must stop before the gateway: %v", err)
	}
	// Gateway fails.
	b, g = &payBilling{}, &payGateway{err: errors.New("gateway down")}
	if _, err := payService(b, g, &payPending{}).Pay(context.Background(), 7, 1, "HEMAT10"); err == nil || !b.released {
		t.Fatalf("gateway failure must release the voucher: released=%v", b.released)
	}
	// Ledger write fails.
	b, g = &payBilling{recordErr: errors.New("db down")}, &payGateway{}
	if _, err := payService(b, g, &payPending{}).Pay(context.Background(), 7, 1, "HEMAT10"); err == nil || !b.released {
		t.Fatalf("ledger failure must release the voucher: released=%v", b.released)
	}
	// Tracking row fails: the ledger row is cancelled (which settles the voucher).
	b, g = &payBilling{}, &payGateway{}
	if _, err := payService(b, g, &payPending{createErr: errors.New("db down")}).Pay(context.Background(), 7, 1, "HEMAT10"); err == nil ||
		len(b.statuses) != 1 || b.statuses[0] != billingcontracts.StatusCancelled {
		t.Fatalf("untracked charge must be cancelled in the ledger: %v", b.statuses)
	}
}
