package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// ---- Paket WO Tahap 4: the vendors the client pays directly ----

// memBillTerms is supplier_bill_terms of one workspace.
type memBillTerms struct {
	rows []domain.SupplierBillTerm
	next int64
}

func (m *memBillTerms) ListByProject(_ context.Context, _, projectID int64) ([]domain.SupplierBillTerm, error) {
	var out []domain.SupplierBillTerm
	for _, r := range m.rows {
		if r.ProjectID == projectID && r.DeletedAt == nil {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *memBillTerms) FindByID(_ context.Context, _, projectID, id int64) (*domain.SupplierBillTerm, error) {
	for _, r := range m.rows {
		if r.ProjectID == projectID && r.ID == id && r.DeletedAt == nil {
			c := r
			return &c, nil
		}
	}
	return nil, nil
}
func (m *memBillTerms) FindDeletedByID(context.Context, int64, int64, int64) (*domain.SupplierBillTerm, error) {
	return nil, nil
}
func (m *memBillTerms) Create(_ context.Context, _ int64, t *domain.SupplierBillTerm) error {
	m.next++
	t.ID = m.next
	m.rows = append(m.rows, *t)
	return nil
}
func (m *memBillTerms) Patch(context.Context, int64, *domain.SupplierBillTerm, []string, *time.Time) (bool, error) {
	return true, nil
}
func (m *memBillTerms) SoftDelete(context.Context, int64, int64, int64, time.Time) (bool, error) {
	return true, nil
}
func (m *memBillTerms) Restore(context.Context, int64, int64, int64, time.Time) (bool, error) {
	return true, nil
}

// billPayments adds the lookups and the conditional verify/reject writes of
// supplier_payments to the in-memory database.
type billPayments struct{ dbSupplierPayments }

func (r billPayments) FindByID(_ context.Context, projectID, id int64) (*domain.SupplierPayment, error) {
	for _, p := range r.db.supplierPays {
		if p.ProjectID == projectID && p.ID == id && p.DeletedAt == nil {
			c := p
			return &c, nil
		}
	}
	return nil, nil
}
func (r billPayments) open(projectID, id int64) *domain.SupplierPayment {
	for i := range r.db.supplierPays {
		p := &r.db.supplierPays[i]
		if p.ProjectID == projectID && p.ID == id && p.AwaitsReview() {
			return p
		}
	}
	return nil
}
func (r billPayments) SetVerification(_ context.Context, projectID, id int64, at time.Time, by int64) (bool, error) {
	p := r.open(projectID, id)
	if p == nil {
		return false, nil
	}
	p.VerifiedAt, p.VerifiedByStaffID = &at, by
	return true, nil
}
func (r billPayments) SetRejection(_ context.Context, projectID, id int64, at time.Time, by int64, reason string) (bool, error) {
	p := r.open(projectID, id)
	if p == nil {
		return false, nil
	}
	p.RejectedAt, p.RejectedByStaffID, p.RejectionReason = &at, by, reason
	return true, nil
}
func (r billPayments) ClearVerification(_ context.Context, projectID, id int64) error {
	for i := range r.db.supplierPays {
		if p := &r.db.supplierPays[i]; p.ProjectID == projectID && p.ID == id {
			p.VerifiedAt, p.VerifiedByStaffID = nil, 0
		}
	}
	return nil
}

// billFiles lets the evidence service list a payment's files.
type billFiles struct{ *proofFiles }

func (f billFiles) ListByRelated(_ context.Context, kind domain.EvidenceRelatedKind, id int64) ([]domain.Evidence, error) {
	var out []domain.Evidence
	for _, e := range f.rows {
		if e.RelatedKind == kind && e.RelatedID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

type billFixture struct {
	*atomicFixture
	bills     *memBillTerms
	suppliers *memSuppliers
	svc       *SupplierBillService
}

const (
	photographerID = int64(7) // the client pays it
	decorID        = int64(8) // the WO pays it
)

func newBillFixture() *billFixture {
	f := newAtomicFixture(fakeContracts{project: 50_000_000}, nil)
	activity := NewActivityService(activityRepoAdapter{f.db.activity})
	ev := NewEvidenceService(billFiles{f.db.files}, f.store, buildKeyForTests, activity)
	sups := &memSuppliers{rows: []domain.ProjectSupplier{
		{ID: photographerID, ProjectID: 10, Kind: domain.SupplierKind("vendor"), Name: "Foto Klien", PaidBy: domain.PayerClient, EngagementStatus: domain.EngagementStatus("Booked")},
		{ID: decorID, ProjectID: 10, Kind: domain.SupplierKind("vendor"), Name: "Dekor WO", PaidBy: domain.PayerWO, EngagementStatus: domain.EngagementStatus("Booked")},
	}}
	bills := &memBillTerms{rows: []domain.SupplierBillTerm{
		{ID: 1, ProjectID: 10, ProjectSupplierID: photographerID, Label: "DP fotografer", Amount: 3_000_000, DueDate: payDay.AddDate(0, 0, 5)},
		{ID: 2, ProjectID: 10, ProjectSupplierID: photographerID, Label: "Pelunasan fotografer", Amount: 7_000_000, DueDate: payDay.AddDate(0, 1, 0)},
	}, next: 2}
	pays := billPayments{dbSupplierPayments{db: f.db}}
	svc := NewSupplierBillService(bills, pays, pays, sups, ev, activity)
	svc.SetProofSupport(NewProofSupport(f.tx, f.uploads, ev))
	return &billFixture{atomicFixture: f, bills: bills, suppliers: sups, svc: svc}
}

func (f *billFixture) submit(t *testing.T, clientID, billID, amount int64) (*SupplierSubmitted, error) {
	t.Helper()
	return f.svc.SubmitByClient(context.Background(), 1, 10, clientID, SupplierSubmissionInput{
		BillTermID: billID, Amount: amount, PaymentDate: payDay, Method: "Transfer Bank", ReferenceNumber: "FOTO-1",
	}, []ProofItem{{UploadToken: f.token(t, 10, "bukti"), Type: domain.EvidenceTransferProof}}, nowOnPayDay)
}

// PLAN §8 Tahap 4 "selesai bila": Pengantin sends the proof of paying the
// photographer, the WO verifies it, and the WO's cash and revenue do not move.
func TestClientPaysThePhotographerAndTheWOCashStays(t *testing.T) {
	f := newBillFixture()
	res, err := f.submit(t, 5, 1, 3_000_000)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Payment
	if p.Payer != domain.PayerClient || p.SubmittedByClientID != 5 || p.RecordedByStaffID != 0 || p.BillTermID != 1 ||
		p.ProjectSupplierID != photographerID || p.Status() != domain.SupplierPaymentPending || p.Type != domain.PaymentDP {
		t.Errorf("a pending proof by the client to the photographer: %+v", p)
	}
	if len(f.db.files.rows) != 1 || f.db.files.rows[0].RelatedKind != domain.RelatedSupplierPayment || f.db.files.rows[0].RelatedID != p.ID ||
		f.db.files.rows[0].UploadedByStaffID != 0 {
		t.Errorf("the proof file hangs off the payment, uploaded by the client: %+v", f.db.files.rows)
	}
	if res.Bill.Paid != 0 || res.Bill.Pending != 3_000_000 || res.Bill.Terms[0].Pending != 3_000_000 {
		t.Errorf("an unchecked proof does not count as paid yet: %+v", res.Bill)
	}

	verified, err := f.svc.Verify(context.Background(), 10, p.ID, 77, nowOnPayDay)
	if err != nil {
		t.Fatal(err)
	}
	if verified.VerifiedAt == nil || verified.VerifiedByStaffID != 77 || verified.Status() != domain.SupplierPaymentVerified {
		t.Errorf("verified by staff 77: %+v", verified)
	}
	bills, err := f.svc.Bills(context.Background(), 1, 10, payDay)
	if err != nil {
		t.Fatal(err)
	}
	if len(bills) != 1 || bills[0].Supplier.ID != photographerID {
		t.Fatalf("only the client-paid supplier has bills: %+v", bills)
	}
	if b := bills[0]; b.Paid != 3_000_000 || b.Pending != 0 || b.Remaining != 7_000_000 || b.Terms[0].Status != domain.TermPaid {
		t.Errorf("once verified the DP bill is paid: %+v", b)
	}

	// The WO's money: no client payment (revenue, contract) was made, and the WO
	// paid no supplier (cash out, payables): every WO figure reads payer = WO.
	if len(f.db.clientPays) != 0 {
		t.Errorf("the client's payment to its vendor is never a payment to the WO: %+v", f.db.clientPays)
	}
	for _, sp := range f.db.supplierPays {
		if sp.Payer == domain.PayerWO {
			t.Errorf("no WO payment appeared: %+v", sp)
		}
	}
	if lines := supplierLines(f.db.supplierPays, photographerID, domain.PayerWO, 0); len(lines) != 0 {
		t.Errorf("the WO's ledger to the photographer stays empty: %+v", lines)
	}
}

func TestBillTermsRefusedOnAWOPaidSupplier(t *testing.T) {
	f := newBillFixture()
	_, err := f.svc.CreateBill(context.Background(), 1, 10, 77, SupplierBillInput{
		ProjectSupplierID: decorID, Label: "DP dekor", Amount: 1_000_000, DueDate: payDay.AddDate(0, 0, 10),
	})
	ae, ok := apperror.As(err)
	if !ok || ae.Code != "supplier_paid_by_wo" || len(ae.Fields["projectSupplierId"]) == 0 {
		t.Fatalf("a bill on a vendor the WO pays is refused (supplier_paid_by_wo): %v", err)
	}
	if len(f.bills.rows) != 2 {
		t.Errorf("nothing was written: %+v", f.bills.rows)
	}
	created, err := f.svc.CreateBill(context.Background(), 1, 10, 77, SupplierBillInput{
		ProjectSupplierID: photographerID, Label: "Album", Amount: 1_500_000, DueDate: payDay.AddDate(0, 2, 0),
	})
	if err != nil || created.ProjectSupplierID != photographerID || created.SortOrder <= 0 {
		t.Errorf("a bill on the client-paid photographer is fine: %v %+v", err, created)
	}
}

// Only the couple sends a proof (staff record payments in Keuangan), and only
// for a bill of this project's client-paid vendors.
func TestSupplierSubmissionRefusals(t *testing.T) {
	f := newBillFixture()
	if _, err := f.submit(t, 0, 1, 1_000_000); err == nil {
		t.Error("no client, no submission")
	}
	if _, err := f.submit(t, 5, 99, 1_000_000); err == nil {
		t.Error("a bill that is not this project's is refused")
	}
	f.bills.rows = append(f.bills.rows, domain.SupplierBillTerm{ID: 3, ProjectID: 11, ProjectSupplierID: 70, Label: "Lain", Amount: 1_000_000, DueDate: payDay})
	if _, err := f.submit(t, 5, 3, 1_000_000); err == nil {
		t.Error("another project's bill is refused")
	}
	if len(f.db.supplierPays) != 0 {
		t.Errorf("nothing was written: %+v", f.db.supplierPays)
	}
}

func TestRejectedProofNeverCounts(t *testing.T) {
	f := newBillFixture()
	res, err := f.submit(t, 5, 1, 3_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Reject(context.Background(), 10, res.Payment.ID, 77, "", nowOnPayDay); err == nil {
		t.Error("a rejection needs a reason")
	}
	rejected, err := f.svc.Reject(context.Background(), 10, res.Payment.ID, 77, "Nominal tidak sesuai", nowOnPayDay)
	if err != nil || rejected.Status() != domain.SupplierPaymentRejected || rejected.RejectionReason != "Nominal tidak sesuai" {
		t.Fatalf("rejected: %v %+v", err, rejected)
	}
	if _, err := f.svc.Verify(context.Background(), 10, res.Payment.ID, 77, nowOnPayDay); err == nil {
		t.Error("a rejected proof cannot be verified")
	}
	bills, _ := f.svc.Bills(context.Background(), 1, 10, payDay)
	if len(bills) != 1 || bills[0].Paid != 0 || bills[0].Pending != 0 {
		t.Errorf("a rejected proof is neither paid nor pending: %+v", bills)
	}
}
