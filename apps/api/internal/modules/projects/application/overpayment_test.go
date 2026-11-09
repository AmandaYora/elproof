package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type fakeContracts struct{ project, venue, engagement int64 }

func (f fakeContracts) ProjectContractValue(context.Context, int64, int64) (int64, error) {
	return f.project, nil
}
func (f fakeContracts) VenueCost(context.Context, int64, int64) (int64, error) { return f.venue, nil }
func (f fakeContracts) EngagementContractValue(context.Context, int64, int64) (int64, error) {
	return f.engagement, nil
}

type fakeClientPayments struct {
	rows    []domain.ClientPayment
	created int
}

func (f *fakeClientPayments) ListByProject(context.Context, int64) ([]domain.ClientPayment, error) {
	return f.rows, nil
}
func (f *fakeClientPayments) FindByID(_ context.Context, _, id int64) (*domain.ClientPayment, error) {
	for i := range f.rows {
		if f.rows[i].ID == id {
			p := f.rows[i] // a copy, like a database read
			return &p, nil
		}
	}
	return nil, nil
}
func (f *fakeClientPayments) Create(context.Context, *domain.ClientPayment) error {
	f.created++
	return nil
}
func (f *fakeClientPayments) Update(_ context.Context, _, id int64, p domain.ClientPayment) error {
	for i := range f.rows {
		if f.rows[i].ID == id {
			f.rows[i] = p
		}
	}
	return nil
}
func (f *fakeClientPayments) Delete(context.Context, int64, int64) error { return nil }
func (f *fakeClientPayments) SetVerification(_ context.Context, _, id int64, at time.Time, by int64) (bool, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].VerifiedAt == nil {
			f.rows[i].VerifiedAt, f.rows[i].VerifiedByStaffID = &at, by
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeClientPayments) ClearVerification(_ context.Context, _, id int64) error {
	for i := range f.rows {
		if f.rows[i].ID == id {
			f.rows[i].VerifiedAt, f.rows[i].VerifiedByStaffID = nil, 0
		}
	}
	return nil
}

func codeOf(err error) string {
	if ae, ok := apperror.As(err); ok {
		return ae.Code
	}
	return ""
}

// A DP/Termin/Pelunasan past the contract value is refused with a stable
// code, and nothing is written. (Accepted payments also record activity,
// which needs the database, so the accepting side is covered by the domain
// rule's own tests and the end-to-end run.)
func TestClientPaymentRefusesOverpayment(t *testing.T) {
	repo := &fakeClientPayments{rows: []domain.ClientPayment{
		{ID: 1, Type: domain.PaymentDP, Amount: 60_000_000},
		{ID: 2, Type: domain.PaymentTambahan, Amount: 20_000_000},
	}}
	svc := NewClientPaymentService(repo, nil, nil, fakeContracts{project: 100_000_000})

	_, err := svc.Create(context.Background(), 1, 10, 5, ClientPaymentInput{
		Type: domain.PaymentPelunasan, Amount: 40_000_001, PaymentDate: time.Now(),
	})
	if codeOf(err) != "payment_exceeds_contract" {
		t.Fatalf("Create past the contract: got %v", err)
	}
	if repo.created != 0 {
		t.Fatalf("a refused payment must not be written, created=%d", repo.created)
	}

	// Editing payment 1 from 60 to 100 million leaves no room problem
	// (itself excluded), but 100,000,001 does.
	_, err = svc.Update(context.Background(), 1, 10, 1, 5, ClientPaymentInput{
		Type: domain.PaymentDP, Amount: 100_000_001, PaymentDate: time.Now(),
	})
	if codeOf(err) != "payment_exceeds_contract" {
		t.Fatalf("Update past the contract: got %v", err)
	}
}

func TestFormatRupiah(t *testing.T) {
	for in, want := range map[int64]string{0: "Rp0", 999: "Rp999", 1000: "Rp1.000", 1_500_000: "Rp1.500.000", 100_000_000: "Rp100.000.000"} {
		if got := formatRupiah(in); got != want {
			t.Errorf("formatRupiah(%d) = %s, want %s", in, got, want)
		}
	}
}

func TestClientPaymentDeleteOfRefundIsChecked(t *testing.T) {
	repo := &fakeClientPayments{rows: []domain.ClientPayment{
		{ID: 1, Type: domain.PaymentDP, Amount: 100},
		{ID: 2, Type: domain.PaymentRefund, Amount: 50},
		{ID: 3, Type: domain.PaymentTermin, Amount: 50},
	}}
	svc := NewClientPaymentService(repo, nil, nil, fakeContracts{project: 100})
	err := svc.Delete(context.Background(), 1, 10, 2, 5)
	if codeOf(err) != "payment_exceeds_contract" {
		t.Fatalf("deleting the refund must be refused, got %v", err)
	}
}

func TestPaymentAmountMustBeSane(t *testing.T) {
	svc := NewClientPaymentService(&fakeClientPayments{}, nil, nil, fakeContracts{project: 100})
	for _, amount := range []int64{0, -5, domain.MaxMoney + 1} {
		_, err := svc.Create(context.Background(), 1, 10, 5, ClientPaymentInput{Type: domain.PaymentDP, Amount: amount, PaymentDate: time.Now()})
		if ae, ok := apperror.As(err); !ok || ae.Kind != apperror.KindValidation {
			t.Errorf("amount %d: want a validation error, got %v", amount, err)
		}
	}
}
