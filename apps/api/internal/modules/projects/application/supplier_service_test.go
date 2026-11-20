package application

import (
	"context"
	"testing"

	"elproof/internal/modules/projects/domain"
)

// paidByFixture is a project's suppliers with the payment service that records
// money against them (through the real contract reader).
func paidByFixture() (*SupplierService, *SupplierPaymentService, *memSuppliers, *memSupplierPayments) {
	pays := &memSupplierPayments{}
	sups := &memSuppliers{pays: pays, extra: map[int64]domain.SupplierAttachments{}, rows: []domain.ProjectSupplier{
		{ID: 1, ProjectID: 10, Kind: domain.SupplierVendor, SupplierID: 5, Name: "Dekor Indah", PaidBy: domain.PayerWO,
			ShowName: true, AgreedCost: i64p(10_000_000), EngagementStatus: domain.EngagementBooked},
		{ID: 2, ProjectID: 10, Kind: domain.SupplierVendor, SupplierID: 6, Name: "Rias Ayu", PaidBy: domain.PayerClient, EngagementStatus: domain.EngagementBooked},
	}}
	svc := NewSupplierService(sups, nil, &memEvents{}, pays, nil)
	payments := NewSupplierPaymentService(pays, nil, nil, NewContractReader(nil, sups))
	return svc, payments, sups, pays
}

func toClient() SupplierPatch { return SupplierPatch{PaidBy: Some(domain.PayerClient)} }

// Tahap 2: "dibayar oleh" cannot change once the supplier has a payment.
func TestPaidByCannotChangeOnceTheSupplierHasAPayment(t *testing.T) {
	ctx := context.Background()
	svc, payments, sups, pays := paidByFixture()

	rec, err := payments.CreateWithProof(ctx, 1, 10, 9, SupplierPaymentInput{ProjectSupplierID: 1, Type: domain.PaymentDP, Amount: 2_000_000, PaymentDate: payDay}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Payment.Payer != domain.PayerWO {
		t.Fatalf("the payment takes the supplier's payer: %+v", rec.Payment)
	}

	_, err = svc.Patch(ctx, 10, 1, 9, toClient(), nil)
	f := moneyFields(t, err)
	if codeOf(err) != "paid_by_locked" || len(f["paidBy"]) == 0 {
		t.Fatalf("want 422 paid_by_locked on paidBy, got %v", err)
	}
	if want := "WO. Terkunci karena sudah ada 1 pembayaran. Hapus pembayaran itu untuk mengubah."; f["paidBy"][0] != want {
		t.Errorf("message = %q, want %q", f["paidBy"][0], want)
	}
	got, _ := svc.Get(ctx, 10, 1)
	if got.PaidBy != domain.PayerWO || got.AgreedCost == nil || sups.writes != 0 {
		t.Errorf("nothing changed: %+v, %d writes", got, sups.writes)
	}
	// Sending the payer it already has is not a change, and other fields still save.
	if got, err := svc.Patch(ctx, 10, 1, 9, SupplierPatch{PaidBy: Some(domain.PayerWO), Notes: Some("Bawa bunga segar")}, nil); err != nil || got.Notes != "Bawa bunga segar" {
		t.Errorf("an unchanged payer does not lock the rest: %+v %v", got, err)
	}

	// Once the payment is deleted, the payer may change again; switching to the
	// client drops the agreed cost and "tampilkan nama".
	if _, _, err := payments.Delete(ctx, 10, rec.Payment.ID, 9, delNow); err != nil {
		t.Fatal(err)
	}
	got, err = svc.Patch(ctx, 10, 1, 9, toClient(), nil)
	if err != nil {
		t.Fatalf("no live payment, no lock: %v", err)
	}
	if got.PaidBy != domain.PayerClient || got.AgreedCost != nil || got.ShowName {
		t.Errorf("client pays: no cost, always named: %+v", got)
	}
	if len(pays.rows) != 1 || pays.rows[0].Payer != domain.PayerWO {
		t.Errorf("the deleted payment keeps its own payer: %+v", pays.rows)
	}

	// A client-paid supplier with a payment is locked the other way too.
	if _, err := payments.CreateWithProof(ctx, 1, 10, 9, SupplierPaymentInput{ProjectSupplierID: 2, Type: domain.PaymentDP, Amount: 1, PaymentDate: payDay}, nil); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Patch(ctx, 10, 2, 9, SupplierPatch{PaidBy: Some(domain.PayerWO)}, nil)
	if f := moneyFields(t, err); codeOf(err) != "paid_by_locked" || f["paidBy"][0] != "Klien langsung. Terkunci karena sudah ada 1 pembayaran. Hapus pembayaran itu untuk mengubah." {
		t.Errorf("client-paid lock: %v", err)
	}
}

// A proof file hanging off the supplier locks the payer the same way.
func TestPaidByIsLockedByAProofFile(t *testing.T) {
	svc, _, sups, _ := paidByFixture()
	sups.extra[1] = domain.SupplierAttachments{Files: 2}
	_, err := svc.Patch(context.Background(), 10, 1, 9, toClient(), nil)
	if f := moneyFields(t, err); codeOf(err) != "paid_by_locked" || f["paidBy"][0] != "WO. Terkunci karena sudah ada 2 bukti. Hapus bukti itu untuk mengubah." {
		t.Errorf("file lock: %v", err)
	}
}
