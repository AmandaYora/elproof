package domain

import (
	"testing"
	"time"
)

// Paket WO Tahap 4: the bills of a vendor the client pays directly.

var billToday = time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

func billDay(n int) time.Time { return billToday.AddDate(0, 0, n) }

func photographer() ProjectSupplier {
	return ProjectSupplier{ID: 7, ProjectID: 12, Kind: SupplierKind("vendor"), Name: "Foto Klien", PaidBy: PayerClient}
}

func photographerBills() []SupplierBillTerm {
	return []SupplierBillTerm{
		{ID: 1, ProjectID: 12, ProjectSupplierID: 7, Label: "DP", Amount: 3_000_000, DueDate: billDay(3)},
		{ID: 2, ProjectID: 12, ProjectSupplierID: 7, Label: "Pelunasan", Amount: 7_000_000, DueDate: billDay(40)},
		{ID: 3, ProjectID: 12, ProjectSupplierID: 8, Label: "Lain", Amount: 1_000_000, DueDate: billDay(1)}, // another supplier
	}
}

func TestSupplierPaymentStatus(t *testing.T) {
	at := billToday
	cases := []struct {
		p      SupplierPayment
		status SupplierPaymentStatus
		counts bool
	}{
		{SupplierPayment{}, SupplierPaymentRecorded, true},
		{SupplierPayment{SubmittedByClientID: 5}, SupplierPaymentPending, false},
		{SupplierPayment{SubmittedByClientID: 5, VerifiedAt: &at}, SupplierPaymentVerified, true},
		{SupplierPayment{SubmittedByClientID: 5, RejectedAt: &at}, SupplierPaymentRejected, false},
	}
	for _, c := range cases {
		if c.p.Status() != c.status || c.p.CountsAsPaid() != c.counts {
			t.Errorf("%+v: got %s/%v, want %s/%v", c.p, c.p.Status(), c.p.CountsAsPaid(), c.status, c.counts)
		}
	}
}

func TestVerifyAndRejectSubmission(t *testing.T) {
	staffMade := SupplierPayment{}
	if _, err := staffMade.VerifySubmission(billToday, 9, 1); err != ErrNotClientSubmission {
		t.Errorf("only a client's proof is checked: %v", err)
	}
	p := SupplierPayment{SubmittedByClientID: 5}
	if _, err := p.VerifySubmission(billToday, 9, 0); err != ErrVerifyNeedsEvidence {
		t.Errorf("no file, no verification: %v", err)
	}
	if changed, err := p.VerifySubmission(billToday, 9, 1); err != nil || !changed || p.VerifiedByStaffID != 9 {
		t.Fatalf("verified: %v %v %+v", changed, err, p)
	}
	if changed, err := p.VerifySubmission(billToday, 10, 1); err != nil || changed || p.VerifiedByStaffID != 9 {
		t.Errorf("verifying twice changes nothing: %v %v", changed, err)
	}
	if _, err := p.RejectSubmission(billToday, 9, "salah"); err != ErrSubmissionVerified {
		t.Errorf("a verified proof cannot be rejected: %v", err)
	}
	r := SupplierPayment{SubmittedByClientID: 5}
	if changed, err := r.RejectSubmission(billToday, 9, "Nominal tidak sesuai"); err != nil || !changed || r.RejectionReason != "Nominal tidak sesuai" {
		t.Fatalf("rejected: %v %v %+v", changed, err, r)
	}
	if _, err := r.VerifySubmission(billToday, 9, 1); err != ErrSubmissionRejected {
		t.Errorf("a rejected proof cannot be verified: %v", err)
	}
}

// Only the client's counted payments to this supplier lower its bills: a
// pending proof is shown apart, a rejected one and the WO's payment do nothing.
func TestBuildSupplierBill(t *testing.T) {
	at := billToday
	payments := []SupplierPayment{
		{ID: 1, ProjectSupplierID: 7, Payer: PayerClient, Type: PaymentDP, Amount: 2_000_000},                                              // staff recorded
		{ID: 2, ProjectSupplierID: 7, Payer: PayerClient, Type: PaymentTermin, Amount: 1_500_000, SubmittedByClientID: 5, VerifiedAt: &at}, // verified
		{ID: 3, ProjectSupplierID: 7, Payer: PayerClient, Type: PaymentTermin, Amount: 500_000, SubmittedByClientID: 5, BillTermID: 2},     // pending
		{ID: 4, ProjectSupplierID: 7, Payer: PayerClient, Type: PaymentTermin, Amount: 900_000, SubmittedByClientID: 5, RejectedAt: &at},   // rejected
		{ID: 5, ProjectSupplierID: 7, Payer: PayerWO, Type: PaymentTermin, Amount: 4_000_000},                                              // not the client's
		{ID: 6, ProjectSupplierID: 8, Payer: PayerClient, Type: PaymentDP, Amount: 1_000_000},                                              // another supplier
		{ID: 7, ProjectSupplierID: 7, Payer: PayerClient, Type: PaymentTermin, Amount: 1_000_000, DeletedAt: &at},                          // deleted
	}
	b := BuildSupplierBill(photographer(), photographerBills(), payments, billToday)
	if b.Total != 10_000_000 || b.Paid != 3_500_000 || b.Remaining != 6_500_000 || b.Pending != 500_000 {
		t.Errorf("totals: %+v", b)
	}
	if len(b.Payments) != 4 {
		t.Errorf("the client's live payments to this supplier are listed (4), got %d", len(b.Payments))
	}
	if len(b.Terms) != 2 || b.Terms[0].Status != TermPaid || b.Terms[1].Paid != 500_000 || b.Terms[1].Remaining != 6_500_000 || b.Terms[1].Pending != 500_000 {
		t.Errorf("the counted money covers the earliest bill first: %+v", b.Terms)
	}
	next := NextSupplierBill([]SupplierBill{b})
	if next == nil || next.Bill.Term.ID != 2 || next.SupplierName != "Foto Klien" || !next.NeedsClientAction() {
		t.Errorf("next bill: %+v", next)
	}
}

func TestBillTermsAreForClientPaidSuppliersAndFinanceRoles(t *testing.T) {
	pengantin := NewPortalPolicy(func(p string) bool { return p == PermPortalFinance || p == PermPortalVendor })
	keluarga := NewPortalPolicy(func(p string) bool { return p == PermPortalVendor || p == PermPortalTimeline })
	sup := photographer()
	if !pengantin.BillTerm(sup) {
		t.Error("Pengantin sees the bills of a vendor it pays")
	}
	if keluarga.BillTerm(sup) {
		t.Error("Keluarga (no portal.finance.view) never sees a bill")
	}
	wo := sup
	wo.PaidBy = PayerWO
	if pengantin.BillTerm(wo) {
		t.Error("a vendor the WO pays has no client bill")
	}
	if a := (SupplierAttachments{BillTerms: 1}); a.PaidByLock(PayerClient) == "" {
		t.Error("a supplier with bills cannot change who pays it")
	}
}

func TestSupplierBillsJoinTheHomeActions(t *testing.T) {
	b := BuildSupplierBill(photographer(), photographerBills()[:1], nil, billToday)
	open := OpenSupplierBills([]SupplierBill{b})
	acts := SelectHomeActions(HomeActionInput{Today: billToday, ShowTerms: true, SupplierBills: open})
	if len(acts) != 1 || acts[0].Kind != HomeActionPaySupplier || acts[0].Amount == nil || *acts[0].Amount != 3_000_000 {
		t.Fatalf("the photographer's DP due in 3 days is an action: %+v", acts)
	}
	if got := SelectHomeActions(HomeActionInput{Today: billToday, ShowTerms: false, SupplierBills: open}); len(got) != 0 {
		t.Errorf("without finance (Keluarga) no bill action: %+v", got)
	}
	// A proof that covers what is left waits for the WO: nothing to do.
	open[0].Bill.Pending = 3_000_000
	if got := SelectHomeActions(HomeActionInput{Today: billToday, ShowTerms: true, SupplierBills: open}); len(got) != 0 {
		t.Errorf("a bill covered by a pending proof is no action: %+v", got)
	}
}

func TestSelectClientBillsDue(t *testing.T) {
	rows := []ClientBillTermRow{
		{Term: SupplierBillTerm{ID: 1, ProjectID: 12, ProjectSupplierID: 7, Label: "DP", Amount: 3_000_000, DueDate: billDay(-2)}, ProjectName: "P", SupplierName: "Foto Klien"},
		{Term: SupplierBillTerm{ID: 2, ProjectID: 12, ProjectSupplierID: 7, Label: "Termin", Amount: 2_000_000, DueDate: billDay(5)}, ProjectName: "P", SupplierName: "Foto Klien"},
		{Term: SupplierBillTerm{ID: 3, ProjectID: 12, ProjectSupplierID: 7, Label: "Pelunasan", Amount: 5_000_000, DueDate: billDay(30)}, ProjectName: "P", SupplierName: "Foto Klien"},
		{Term: SupplierBillTerm{ID: 4, ProjectID: 13, ProjectSupplierID: 9, Label: "DP", Amount: 1_000_000, DueDate: billDay(1)}, ProjectName: "Q", SupplierName: "Dekor"},
	}
	payments := []SupplierPayment{
		{ProjectSupplierID: 7, Payer: PayerClient, Type: PaymentDP, Amount: 3_000_000},                                          // pays bill 1
		{ProjectSupplierID: 7, Payer: PayerClient, Type: PaymentTermin, Amount: 500_000, SubmittedByClientID: 5, BillTermID: 2}, // pending on bill 2
	}
	got := SelectClientBillsDue(billToday, rows, payments)
	if len(got) != 2 || got[0].BillTermID != 4 || got[1].BillTermID != 2 || got[1].Pending != 500_000 || got[1].Remaining != 2_000_000 {
		t.Errorf("open bills due within 7 days, soonest first, a paid one and a far one left out: %+v", got)
	}
}
