package application

import (
	"context"
	"net/http"
	"testing"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

func moneyFields(t *testing.T, err error) map[string][]string {
	t.Helper()
	if err == nil {
		t.Fatal("want a 422, got nil")
	}
	if got := apperror.HTTPStatus(err); got != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (%v)", got, err)
	}
	ae, _ := apperror.As(err)
	return ae.Fields
}

func TestCheckAgreedCost(t *testing.T) {
	cases := []struct {
		name   string
		paidBy domain.Payer
		cost   *int64
		want   bool
	}{
		{"not set", domain.PayerWO, nil, false},
		{"WO pays, a sane cost", domain.PayerWO, i64p(12_500_000), false},
		{"WO pays, 0 (not agreed yet)", domain.PayerWO, i64p(0), false},
		{"negative", domain.PayerWO, i64p(-1), true},
		{"absurd", domain.PayerWO, i64p(domain.MaxMoney + 1), true},
		{"the client pays: no cost for the WO", domain.PayerClient, i64p(1_000_000), true},
		{"the client pays, not set", domain.PayerClient, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fe := newFieldErrors()
			checkAgreedCost(fe, tc.paidBy, tc.cost)
			if got := len(fe.fields["agreedCost"]) > 0; got != tc.want || len(fe.fields) > 1 {
				t.Fatalf("fields = %v, want refused=%v", fe.fields, tc.want)
			}
		})
	}
	fe := newFieldErrors()
	checkAgreedCost(fe, domain.PayerClient, i64p(5))
	if want := "Biaya rekanan hanya diisi untuk vendor atau venue yang dibayar WO"; fe.fields["agreedCost"][0] != want {
		t.Fatalf("message = %q", fe.fields["agreedCost"][0])
	}
}

func TestCostBelowPaid(t *testing.T) {
	cases := []struct {
		name            string
		old, next, paid int64
		refused         bool
	}{
		{"lower but still above paid", 10_000_000, 8_000_000, 5_000_000, false},
		{"lower to exactly paid", 10_000_000, 5_000_000, 5_000_000, false},
		{"lower under paid", 10_000_000, 4_000_000, 5_000_000, true},
		{"set a cost under paid where none was set", 0, 4_000_000, 5_000_000, true},
		{"clear the cost (0 = not set)", 10_000_000, 0, 5_000_000, false},
		{"raise a cost that is still under paid", 3_000_000, 4_000_000, 5_000_000, false},
		{"nothing paid", 10_000_000, 1, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := costBelowPaid(tc.old, tc.next, tc.paid) != ""; got != tc.refused {
				t.Fatalf("refused = %v, want %v", got, tc.refused)
			}
		})
	}
	want := "Biaya rekanan tidak bisa di bawah yang sudah dibayar WO (Rp5.000.000). Naikkan nilainya, atau koreksi pembayarannya lebih dulu."
	if got := costBelowPaid(10_000_000, 4_000_000, 5_000_000); got != want {
		t.Fatalf("message = %q", got)
	}
}

func TestSupplierCreateMoneyRules(t *testing.T) {
	ctx := context.Background()
	repo := &memSuppliers{}
	svc := NewSupplierService(repo, nil, &memEvents{}, &memSupplierPayments{}, nil)

	_, err := svc.Create(ctx, 10, 1, SupplierInput{VendorID: 5, Name: "Dekor Indah", PaidBy: domain.PayerClient, AgreedCost: i64p(5_000_000)})
	if f := moneyFields(t, err); len(f["agreedCost"]) == 0 {
		t.Fatalf("a cost for a supplier the client pays: %v", f)
	}
	_, err = svc.Create(ctx, 10, 1, SupplierInput{VendorID: 5, Name: "Dekor Indah", AgreedCost: i64p(-1)})
	if f := moneyFields(t, err); len(f["agreedCost"]) == 0 {
		t.Fatalf("a negative cost: %v", f)
	}
	if len(repo.rows) != 0 {
		t.Fatalf("a refused supplier was written: %+v", repo.rows)
	}

	got, err := svc.Create(ctx, 10, 1, SupplierInput{VendorID: 5, Name: "Dekor Indah", AgreedCost: i64p(12_500_000), ShowName: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != domain.SupplierVendor || got.PaidBy != domain.PayerWO || got.EngagementStatus != domain.EngagementPlanned ||
		got.AgreedCost == nil || *got.AgreedCost != 12_500_000 || !got.ShowName {
		t.Errorf("a vendor the WO pays by default, Planned, with its cost: %+v", got)
	}
	// "Tampilkan nama" means nothing for a supplier the client pays: always named.
	got, err = svc.Create(ctx, 10, 1, SupplierInput{VendorID: 6, Name: "Rias Ayu", PaidBy: domain.PayerClient, ShowName: true})
	if err != nil || got.ShowName || !got.ClientSeesName() {
		t.Errorf("client-paid supplier: %+v %v", got, err)
	}
}

func TestSupplierPatchMoneyRules(t *testing.T) {
	ctx := context.Background()
	setup := func() (*SupplierService, *memSuppliers) {
		pays := &memSupplierPayments{rows: []domain.SupplierPayment{
			{ID: 1, ProjectID: 10, ProjectSupplierID: 1, Payer: domain.PayerWO, Type: domain.PaymentDP, Amount: 2_000_000},
			{ID: 2, ProjectID: 10, ProjectSupplierID: 1, Payer: domain.PayerWO, Type: domain.PaymentTermin, Amount: 4_000_000},
			{ID: 3, ProjectID: 10, ProjectSupplierID: 1, Payer: domain.PayerWO, Type: domain.PaymentTambahan, Amount: 9_000_000}, // outside the cost
			{ID: 4, ProjectID: 10, ProjectSupplierID: 1, Payer: domain.PayerWO, Type: domain.PaymentRefund, Amount: 500_000},
			{ID: 5, ProjectID: 10, ProjectSupplierID: 2, Payer: domain.PayerWO, Type: domain.PaymentTermin, Amount: 50_000_000}, // another supplier
		}}
		repo := &memSuppliers{pays: pays, rows: []domain.ProjectSupplier{
			{ID: 1, ProjectID: 10, Kind: domain.SupplierVendor, SupplierID: 5, Name: "Dekor Indah", PaidBy: domain.PayerWO, AgreedCost: i64p(12_500_000), EngagementStatus: domain.EngagementBooked},
			{ID: 2, ProjectID: 10, Kind: domain.SupplierVendor, SupplierID: 6, Name: "Rias Ayu", PaidBy: domain.PayerWO, EngagementStatus: domain.EngagementBooked},
			{ID: 3, ProjectID: 10, Kind: domain.SupplierVendor, SupplierID: 7, Name: "Foto Kita", PaidBy: domain.PayerClient, EngagementStatus: domain.EngagementBooked},
		}}
		return NewSupplierService(repo, nil, &memEvents{}, pays, nil), repo
	}

	t.Run("cost lowered under what the WO paid", func(t *testing.T) {
		// Paid towards the cost: 2.000.000 + 4.000.000 - 500.000 = 5.500.000 (Tambahan left out).
		svc, repo := setup()
		_, err := svc.Patch(ctx, 10, 1, 9, SupplierPatch{AgreedCost: Some[int64](5_000_000)}, nil)
		f := moneyFields(t, err)
		if want := "Biaya rekanan tidak bisa di bawah yang sudah dibayar WO (Rp5.500.000). Naikkan nilainya, atau koreksi pembayarannya lebih dulu."; len(f["agreedCost"]) == 0 || f["agreedCost"][0] != want {
			t.Fatalf("fields = %v", f)
		}
		if repo.writes != 0 {
			t.Fatal("a refused patch was written")
		}
	})
	t.Run("cost lowered to exactly what was paid", func(t *testing.T) {
		svc, repo := setup()
		got, err := svc.Patch(ctx, 10, 1, 9, SupplierPatch{AgreedCost: Some[int64](5_500_000)}, nil)
		if err != nil || *got.AgreedCost != 5_500_000 || repo.writes != 1 {
			t.Fatalf("got = %+v, err = %v, writes = %d", got, err, repo.writes)
		}
	})
	t.Run("cost cleared", func(t *testing.T) {
		svc, _ := setup()
		got, err := svc.Patch(ctx, 10, 1, 9, SupplierPatch{AgreedCost: Null[int64]()}, nil)
		if err != nil || got.AgreedCost != nil {
			t.Fatalf("got = %+v, err = %v", got, err)
		}
	})
	t.Run("each supplier counts only its own payments", func(t *testing.T) {
		svc, _ := setup()
		if _, err := svc.Patch(ctx, 10, 2, 9, SupplierPatch{AgreedCost: Some[int64](60_000_000)}, nil); err != nil {
			t.Fatal(err)
		}
		_, err := svc.Patch(ctx, 10, 2, 9, SupplierPatch{AgreedCost: Some[int64](1_000_000)}, nil)
		if f := moneyFields(t, err); len(f["agreedCost"]) == 0 {
			t.Fatalf("its own 50.000.000 counts: %v", f)
		}
	})
	t.Run("no cost for a supplier the client pays", func(t *testing.T) {
		svc, repo := setup()
		_, err := svc.Patch(ctx, 10, 3, 9, SupplierPatch{AgreedCost: Some[int64](1_000_000)}, nil)
		if f := moneyFields(t, err); len(f["agreedCost"]) == 0 || repo.writes != 0 {
			t.Fatalf("fields = %v, writes = %d", f, repo.writes)
		}
	})
}
