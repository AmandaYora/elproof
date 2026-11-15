package application

import (
	"context"
	"net/http"
	"testing"
	"time"

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

func TestEngagementMoneyErrors(t *testing.T) {
	cases := []struct {
		name         string
		contract, dp int64
		want         []string
	}{
		{"DP within price", 12_500_000, 5_000_000, nil},
		{"DP equal to price", 12_500_000, 12_500_000, nil},
		{"no DP", 12_500_000, 0, nil},
		{"QA repro: DP above price", 12_500_000, 99_000_000, []string{"dpAmount"}},
		{"price not set yet limits nothing", 0, 5_000_000, nil},
		{"negative price", -1, 0, []string{"contractValue"}},
		{"negative DP", 12_500_000, -1, []string{"dpAmount"}},
		{"absurd price", domain.MaxMoney + 1, 0, []string{"contractValue"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EngagementMoneyErrors(tc.contract, tc.dp)
			if len(got) != len(tc.want) {
				t.Fatalf("fields = %v, want keys %v", got, tc.want)
			}
			for _, k := range tc.want {
				if len(got[k]) == 0 {
					t.Fatalf("no message on %q: %v", k, got)
				}
			}
		})
	}
	msg := EngagementMoneyErrors(12_500_000, 99_000_000)["dpAmount"][0]
	if want := "Jumlah DP (Rp99.000.000) melebihi nilai kerja sama (Rp12.500.000). Turunkan DP atau naikkan nilai kerja sama."; msg != want {
		t.Fatalf("message = %q, want %q", msg, want)
	}
}

func TestContractBelowPaid(t *testing.T) {
	cases := []struct {
		name            string
		old, next, paid int64
		refused         bool
	}{
		{"lower but still above paid", 10_000_000, 8_000_000, 5_000_000, false},
		{"lower to exactly paid", 10_000_000, 5_000_000, 5_000_000, false},
		{"lower under paid", 10_000_000, 4_000_000, 5_000_000, true},
		{"set a price under paid where none was set", 0, 4_000_000, 5_000_000, true},
		{"clear the price (0 = not set)", 10_000_000, 0, 5_000_000, false},
		{"raise a price that is still under paid", 3_000_000, 4_000_000, 5_000_000, false},
		{"nothing paid", 10_000_000, 1, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := contractBelowPaid(tc.old, tc.next, tc.paid) != ""; got != tc.refused {
				t.Fatalf("refused = %v, want %v", got, tc.refused)
			}
		})
	}
}

// patchableEngagements is fakeEngagements that also writes a patch.
type patchableEngagements struct {
	*fakeEngagements
	writes int
}

func (p *patchableEngagements) Patch(_ context.Context, pv *domain.ProjectVendor, _ []string, _ *time.Time) (bool, error) {
	for i := range p.rows {
		if p.rows[i].ID == pv.ID {
			p.rows[i] = *pv
			p.writes++
			return true, nil
		}
	}
	return false, nil
}

func TestEngagementCreateRefusesDPAboveContract(t *testing.T) {
	// The repository has no Create: reaching it would panic.
	svc := NewVendorEngagementService(&fakeEngagements{}, nil, nil, nil)
	_, err := svc.Create(context.Background(), 10, 1, VendorEngagementInput{VendorID: 5, Scope: "Dekor", ContractValue: 12_500_000, DPAmount: 99_000_000})
	if f := moneyFields(t, err); len(f["dpAmount"]) == 0 {
		t.Fatalf("fields = %v", f)
	}
}

func TestEngagementPatchMoneyRules(t *testing.T) {
	ctx := context.Background()
	setup := func() (*VendorEngagementService, *patchableEngagements) {
		repo := &patchableEngagements{fakeEngagements: &fakeEngagements{rows: []domain.ProjectVendor{
			{ID: 1, VendorID: 5, Scope: "Dekor", ContractValue: 12_500_000, DPAmount: 2_000_000},
			// An older row that already breaks the DP rule.
			{ID: 2, VendorID: 6, Scope: "Rias", ContractValue: 1_000_000, DPAmount: 3_000_000},
		}}}
		payments := &fakeVendorPayments{rows: []domain.VendorPayment{
			{ID: 1, ProjectVendorID: 1, Type: domain.PaymentDP, Amount: 2_000_000},
			{ID: 2, ProjectVendorID: 1, Type: domain.PaymentTermin, Amount: 4_000_000},
			{ID: 3, ProjectVendorID: 1, Type: domain.PaymentTambahan, Amount: 9_000_000}, // outside the price
			{ID: 4, ProjectVendorID: 1, Type: domain.PaymentRefund, Amount: 500_000},
			{ID: 5, ProjectVendorID: 2, Type: domain.PaymentTermin, Amount: 50_000_000}, // another engagement
		}}
		return NewVendorEngagementService(repo, nil, nil, payments), repo
	}

	t.Run("DP raised above the price", func(t *testing.T) {
		svc, repo := setup()
		_, err := svc.Patch(ctx, 10, 1, 1, VendorEngagementPatch{DPAmount: Some[int64](99_000_000)}, nil)
		if f := moneyFields(t, err); len(f["dpAmount"]) == 0 {
			t.Fatalf("fields = %v", f)
		}
		if repo.writes != 0 {
			t.Fatal("a refused patch was written")
		}
	})
	t.Run("price lowered under the DP", func(t *testing.T) {
		svc, _ := setup()
		_, err := svc.Patch(ctx, 10, 1, 1, VendorEngagementPatch{ContractValue: Some[int64](1_000_000)}, nil)
		if f := moneyFields(t, err); len(f["dpAmount"]) == 0 {
			t.Fatalf("fields = %v", f)
		}
	})
	t.Run("price lowered under what was paid", func(t *testing.T) {
		// Paid towards the price: 2.000.000 + 4.000.000 − 500.000 = 5.500.000 (Tambahan left out).
		svc, repo := setup()
		_, err := svc.Patch(ctx, 10, 1, 1, VendorEngagementPatch{ContractValue: Some[int64](5_000_000)}, nil)
		f := moneyFields(t, err)
		if len(f["contractValue"]) == 0 {
			t.Fatalf("fields = %v", f)
		}
		if want := "Nilai kerja sama tidak bisa di bawah yang sudah dibayar ke vendor (Rp5.500.000). Naikkan nilainya, atau koreksi pembayarannya lebih dulu."; f["contractValue"][0] != want {
			t.Fatalf("message = %q", f["contractValue"][0])
		}
		if repo.writes != 0 {
			t.Fatal("a refused patch was written")
		}
	})
	t.Run("price lowered to exactly what was paid", func(t *testing.T) {
		svc, repo := setup()
		pv, err := svc.Patch(ctx, 10, 1, 1, VendorEngagementPatch{ContractValue: Some[int64](5_500_000)}, nil)
		if err != nil || pv.ContractValue != 5_500_000 || repo.writes != 1 {
			t.Fatalf("pv = %+v, err = %v, writes = %d", pv, err, repo.writes)
		}
	})
	t.Run("older row breaking the DP rule keeps its other fields editable", func(t *testing.T) {
		svc, repo := setup()
		pv, err := svc.Patch(ctx, 10, 2, 1, VendorEngagementPatch{Notes: Some("Ganti jam rias"), ContractValue: Some[int64](1_000_000), DPAmount: Some[int64](3_000_000)}, nil)
		if err != nil || pv.Notes != "Ganti jam rias" || repo.writes != 1 {
			t.Fatalf("pv = %+v, err = %v", pv, err)
		}
	})
}
