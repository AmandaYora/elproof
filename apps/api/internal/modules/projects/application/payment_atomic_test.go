package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
)

// ---- a tiny in-memory "database" with real rollback ----

type memDB struct {
	clientPays []domain.ClientPayment
	vendorPays []domain.VendorPayment
	venuePays  []domain.VenuePayment
	files      *proofFiles
	activity   *memActivity
	nextPay    int64
}

// snapshot returns the function that puts everything back as it is now.
func (d *memDB) snapshot() func() {
	cp := append([]domain.ClientPayment(nil), d.clientPays...)
	vp := append([]domain.VendorPayment(nil), d.vendorPays...)
	np := append([]domain.VenuePayment(nil), d.venuePays...)
	ev := append([]domain.Evidence(nil), d.files.rows...)
	act := append([]domain.ActivityLogEntry(nil), d.activity.entries...)
	return func() {
		d.clientPays, d.vendorPays, d.venuePays = cp, vp, np
		d.files.rows, d.activity.entries = ev, act
	}
}

// memTx is a Transactor over memDB: a failed InTx restores the snapshot.
type memTx struct {
	db                *memDB
	began, rolledBack int
	locked            []int64
}

func (t *memTx) InTx(ctx context.Context, fn func(context.Context) error) error {
	t.began++
	undo := t.db.snapshot()
	if err := fn(ctx); err != nil {
		undo()
		t.rolledBack++
		return err
	}
	return nil
}
func (t *memTx) LockProject(_ context.Context, id int64) error {
	t.locked = append(t.locked, id)
	return nil
}

type dbClientPayments struct {
	ClientPaymentRepository
	db *memDB
}

func (r dbClientPayments) ListByProject(context.Context, int64) ([]domain.ClientPayment, error) {
	return r.db.clientPays, nil
}
func (r dbClientPayments) Create(_ context.Context, p *domain.ClientPayment) error {
	r.db.nextPay++
	p.ID = r.db.nextPay
	r.db.clientPays = append(r.db.clientPays, *p)
	return nil
}

type dbVendorPayments struct {
	PaymentRepository
	db *memDB
}

func (r dbVendorPayments) ListByProject(context.Context, int64) ([]domain.VendorPayment, error) {
	return r.db.vendorPays, nil
}
func (r dbVendorPayments) Create(_ context.Context, p *domain.VendorPayment) error {
	r.db.nextPay++
	p.ID = r.db.nextPay
	r.db.vendorPays = append(r.db.vendorPays, *p)
	return nil
}

type dbVenuePayments struct {
	VenuePaymentRepository
	db *memDB
}

func (r dbVenuePayments) ListByProject(context.Context, int64) ([]domain.VenuePayment, error) {
	return r.db.venuePays, nil
}
func (r dbVenuePayments) Create(_ context.Context, p *domain.VenuePayment) error {
	r.db.nextPay++
	p.ID = r.db.nextPay
	r.db.venuePays = append(r.db.venuePays, *p)
	return nil
}

type memTerms struct{ terms []domain.PaymentTerm }

func (m memTerms) ListByProject(context.Context, int64) ([]domain.PaymentTerm, error) {
	return m.terms, nil
}
func (m memTerms) ReplaceAll(context.Context, int64, []domain.PaymentTerm) error { return nil }

type atomicFixture struct {
	db      *memDB
	tx      *memTx
	store   *objStorage
	uploads *UploadService
	client  *ClientPaymentService
	vendor  *PaymentService
	venue   *VenuePaymentService
}

var payDay = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

func newAtomicFixture(contracts fakeContracts, terms []domain.PaymentTerm) *atomicFixture {
	db := &memDB{files: &proofFiles{}, activity: &memActivity{}}
	store := newObjStorage(true)
	activity := NewActivityService(activityRepoAdapter{db.activity})
	ev := NewEvidenceService(db.files, store, buildKeyForTests, activity)
	uploads := newUploads(store, db.files)
	tx := &memTx{db: db}
	proof := NewProofSupport(tx, uploads, ev)

	client := NewClientPaymentService(dbClientPayments{db: db}, ev, activity, contracts)
	client.SetAtomicSupport(proof, memTerms{terms})
	vendor := NewPaymentService(dbVendorPayments{db: db}, ev, activity, contracts)
	vendor.SetProofSupport(proof)
	venue := NewVenuePaymentService(dbVenuePayments{db: db}, ev, activity, contracts)
	venue.SetProofSupport(proof)
	return &atomicFixture{db: db, tx: tx, store: store, uploads: uploads, client: client, vendor: vendor, venue: venue}
}

func (f *atomicFixture) token(t *testing.T, project int64, name string) string {
	t.Helper()
	return grantAndPut(t, f.uploads, f.store, 1, project, []byte("bytes-of-"+name), "image/jpeg")
}

func clientInput(amount int64) ClientPaymentInput {
	return ClientPaymentInput{Type: domain.PaymentDP, Amount: amount, PaymentDate: payDay, Method: "Transfer", ReferenceNumber: "REF-1"}
}

func TestClientPaymentWithProofIsOneAction(t *testing.T) {
	f := newAtomicFixture(fakeContracts{project: 10_000_000}, []domain.PaymentTerm{
		{ID: 1, Label: "DP", Amount: 3_000_000, DueDate: payDay.AddDate(0, 0, 3)},
		{ID: 2, Label: "Pelunasan", Amount: 7_000_000, DueDate: payDay.AddDate(0, 1, 0)},
	})
	tok := f.token(t, 10, "dp")
	res, err := f.client.CreateWithProof(context.Background(), 1, 10, 77, clientInput(4_000_000),
		[]ProofItem{{UploadToken: tok, Type: domain.EvidenceTransferProof}}, payDay)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.db.clientPays) != 1 || len(f.db.files.rows) != 1 {
		t.Fatalf("one payment and one proof: %d / %d", len(f.db.clientPays), len(f.db.files.rows))
	}
	e := f.db.files.rows[0]
	if e.RelatedKind != domain.RelatedClientPayment || e.RelatedID != res.Payment.ID || e.ProjectID != 10 || e.UploadedByStaffID != 77 ||
		e.Name != "Bukti Transfer - REF-1" || e.FileName != "bukti.jpg" || e.Type != domain.EvidenceTransferProof ||
		e.DocumentDate == nil || !e.DocumentDate.Equal(payDay) || e.IsClientVisible {
		t.Errorf("evidence defaults are those of the two-call flow: %+v", e)
	}
	if res.Payment.RecordedByStaffID != 77 || len(res.Evidence) != 1 {
		t.Errorf("result: %+v", res)
	}
	if res.Ledger != (domain.Ledger{ContractValue: 10_000_000, Paid: 4_000_000, Remaining: 6_000_000}) {
		t.Errorf("ledger: %+v", res.Ledger)
	}
	if len(res.Terms) != 2 || res.Terms[0].Paid != 3_000_000 || res.Terms[0].Status != domain.TermPaid || res.Terms[1].Paid != 1_000_000 || res.Terms[1].Remaining != 6_000_000 {
		t.Errorf("the server allocates the payment over the schedule: %+v", res.Terms)
	}
	kinds := map[domain.ActivityType]int{}
	for _, a := range f.db.activity.entries {
		kinds[a.Type]++
	}
	if kinds[domain.ActivityPaymentRecorded] != 1 || kinds[domain.ActivityEvidenceUploaded] != 1 {
		t.Errorf("activity entries are kept: %v", kinds)
	}
	if len(f.tx.locked) != 1 || f.tx.locked[0] != 10 {
		t.Errorf("the project is locked while money is recorded: %v", f.tx.locked)
	}
}

func TestBadProofCreatesNothing(t *testing.T) {
	ctx := context.Background()
	f := newAtomicFixture(fakeContracts{project: 10_000_000}, nil)
	good := f.token(t, 10, "good")

	// A tampered second token: refused before anything is written.
	_, err := f.client.CreateWithProof(ctx, 1, 10, 77, clientInput(1000),
		[]ProofItem{{UploadToken: good, Type: domain.EvidenceTransferProof}, {UploadToken: good + "x", Type: domain.EvidenceTransferProof}}, payDay)
	if f := fieldsOf(t, err); len(f["evidence[1].uploadToken"]) == 0 {
		t.Errorf("keys: %v", f)
	}
	if len(f.db.clientPays) != 0 || len(f.db.files.rows) != 0 || f.tx.began != 0 {
		t.Errorf("nothing may be written for a bad token: %d payments, %d proofs, %d tx", len(f.db.clientPays), len(f.db.files.rows), f.tx.began)
	}
	// ...and a bad amount and a bad token are reported together.
	_, err = f.client.CreateWithProof(ctx, 1, 10, 77, ClientPaymentInput{Type: "Cicilan", Amount: 0, PaymentDate: payDay},
		[]ProofItem{{UploadToken: "nope", Type: domain.EvidenceTransferProof}}, payDay)
	if got := sortedKeys(fieldsOf(t, err)); len(got) != 4 { // type, amount, evidence[0], evidence[0].uploadToken
		t.Errorf("all field problems at once: %v", got)
	}
	// Client payments accept only a transfer proof.
	_, err = f.client.CreateWithProof(ctx, 1, 10, 77, clientInput(1000), []ProofItem{{UploadToken: good, Type: domain.EvidenceInvoice}}, payDay)
	if f := fieldsOf(t, err); len(f["evidence[0].type"]) == 0 {
		t.Errorf("an invoice is not a client proof: %v", f)
	}
}

func TestFailureInsideTheTransactionRollsBackThePayment(t *testing.T) {
	ctx := context.Background()
	f := newAtomicFixture(fakeContracts{project: 10_000_000}, nil)
	a, b := f.token(t, 10, "a"), f.token(t, 10, "b")
	items := []ProofItem{{UploadToken: a, Type: domain.EvidenceTransferProof}, {UploadToken: b, Type: domain.EvidenceTransferProof}}

	// The database fails on the second proof row.
	f.db.files.failOn = 2
	if _, err := f.client.CreateWithProof(ctx, 1, 10, 77, clientInput(1000), items, payDay); err == nil {
		t.Fatal("must fail")
	}
	if len(f.db.clientPays) != 0 || len(f.db.files.rows) != 0 || len(f.db.activity.entries) != 0 {
		t.Errorf("a failed proof must take the payment, the first proof and the activity with it: %d/%d/%d",
			len(f.db.clientPays), len(f.db.files.rows), len(f.db.activity.entries))
	}
	if f.tx.rolledBack != 1 {
		t.Errorf("rolled back %d times", f.tx.rolledBack)
	}
	f.db.files.failOn = 0

	// A token a concurrent request claimed after it was checked: the unique
	// path refuses it, on its own item, and the payment is rolled back.
	c, _ := testSigner("secret-one").Verify(b, 1, 10)
	f.db.files.rows = append(f.db.files.rows, domain.Evidence{StoragePath: c.Key}) // taken, but ExistsByStoragePath is bypassed below
	pre := len(f.db.files.rows)
	spy := &raceFiles{proofFiles: f.db.files, hideFrom: c.Key}
	f.uploads.files = spy
	_, err := f.client.CreateWithProof(ctx, 1, 10, 77, clientInput(1000), []ProofItem{{UploadToken: b, Type: domain.EvidenceTransferProof}}, payDay)
	if fl := fieldsOf(t, err); len(fl["evidence[0]"]) == 0 || len(fl["evidence[0].uploadToken"]) == 0 {
		t.Errorf("the spent token is reported on its item: %v", fl)
	}
	if len(f.db.clientPays) != 0 || len(f.db.files.rows) != pre {
		t.Errorf("no payment row without its proof: %d payments, %d proofs", len(f.db.clientPays), len(f.db.files.rows))
	}
}

// raceFiles hides one path from the early "already used?" check, as if another
// request claimed it just after the check.
type raceFiles struct {
	*proofFiles
	hideFrom string
}

func (r *raceFiles) ExistsByStoragePath(ctx context.Context, p string) (bool, error) {
	if p == r.hideFrom {
		return false, nil
	}
	return r.proofFiles.ExistsByStoragePath(ctx, p)
}

func TestOverpaymentRollsBackAndKeepsTheUpload(t *testing.T) {
	ctx := context.Background()
	f := newAtomicFixture(fakeContracts{project: 1_000}, nil)
	tok := f.token(t, 10, "x")
	_, err := f.client.CreateWithProof(ctx, 1, 10, 77, clientInput(5_000), []ProofItem{{UploadToken: tok, Type: domain.EvidenceTransferProof}}, payDay)
	if codeOf(err) != "payment_exceeds_contract" {
		t.Fatalf("overpayment keeps its stable code: %v", err)
	}
	if len(f.db.clientPays) != 0 || len(f.db.files.rows) != 0 {
		t.Error("nothing written")
	}
	if len(f.store.deleted) != 0 {
		t.Error("the upload stays so the corrected request can reuse the same token")
	}
	// The corrected retry with the same token works.
	if _, err := f.client.CreateWithProof(ctx, 1, 10, 77, clientInput(500), []ProofItem{{UploadToken: tok, Type: domain.EvidenceTransferProof}}, payDay); err != nil {
		t.Errorf("retry with the same token: %v", err)
	}
	// A token is single-use across payments.
	if _, err := f.client.CreateWithProof(ctx, 1, 10, 77, clientInput(100), []ProofItem{{UploadToken: tok, Type: domain.EvidenceTransferProof}}, payDay); err == nil {
		t.Error("the same token cannot prove a second payment")
	}
	if len(f.db.clientPays) != 1 {
		t.Errorf("the refused second use creates no payment: %d", len(f.db.clientPays))
	}
}

func TestVendorAndVenuePaymentsWithProof(t *testing.T) {
	ctx := context.Background()
	f := newAtomicFixture(fakeContracts{engagement: 5_000_000, venue: 8_000_000}, nil)
	inv, proof := f.token(t, 10, "inv"), f.token(t, 10, "proof")
	items := []ProofItem{{UploadToken: inv, Type: domain.EvidenceInvoice}, {UploadToken: proof, Type: domain.EvidenceTransferProof, Name: "Bukti DP dekor"}}

	v, err := f.vendor.CreateWithProof(ctx, 1, 10, 77, PaymentInput{ProjectVendorID: 3, Type: domain.PaymentDP, Amount: 2_000_000, PaymentDate: payDay, ReferenceNumber: "V-1"}, items)
	if err != nil {
		t.Fatal(err)
	}
	if v.Ledger != (domain.Ledger{ContractValue: 5_000_000, Paid: 2_000_000, Remaining: 3_000_000}) || len(v.Evidence) != 2 {
		t.Errorf("vendor result: %+v", v)
	}
	names := map[domain.EvidenceType]string{}
	for _, e := range v.Evidence {
		names[e.Type] = e.Name
		if e.RelatedKind != domain.RelatedPayment || e.RelatedID != v.Payment.ID {
			t.Errorf("proof attaches to the vendor payment: %+v", e)
		}
	}
	if names[domain.EvidenceInvoice] != "Invoice - V-1" || names[domain.EvidenceTransferProof] != "Bukti DP dekor" {
		t.Errorf("names: %v", names)
	}
	// A second engagement's payments do not count toward the first.
	if _, err := f.vendor.CreateWithProof(ctx, 1, 10, 77, PaymentInput{ProjectVendorID: 4, Type: domain.PaymentDP, Amount: 1_000_000, PaymentDate: payDay}, nil); err != nil {
		t.Fatal(err)
	}
	v2, _ := f.vendor.CreateWithProof(ctx, 1, 10, 77, PaymentInput{ProjectVendorID: 3, Type: domain.PaymentTermin, Amount: 1_000_000, PaymentDate: payDay}, nil)
	if v2.Ledger.Paid != 3_000_000 || v2.Ledger.Remaining != 2_000_000 {
		t.Errorf("ledger is per engagement: %+v", v2.Ledger)
	}

	n, err := f.venue.CreateWithProof(ctx, 1, 10, 77, VenuePaymentInput{Type: domain.PaymentDP, Amount: 3_000_000, PaymentDate: payDay},
		[]ProofItem{{UploadToken: f.token(t, 10, "vp"), Type: domain.EvidenceTransferProof}})
	if err != nil || n.Ledger != (domain.Ledger{ContractValue: 8_000_000, Paid: 3_000_000, Remaining: 5_000_000}) || n.Evidence[0].RelatedKind != domain.RelatedVenuePayment {
		t.Errorf("venue result: %+v %v", n, err)
	}
	// Overpaying the venue rolls back with its proof.
	before := len(f.db.files.rows)
	if _, err := f.venue.CreateWithProof(ctx, 1, 10, 77, VenuePaymentInput{Type: domain.PaymentPelunasan, Amount: 9_000_000, PaymentDate: payDay},
		[]ProofItem{{UploadToken: f.token(t, 10, "vp2"), Type: domain.EvidenceTransferProof}}); codeOf(err) != "payment_exceeds_contract" {
		t.Errorf("venue overpayment: %v", err)
	}
	if len(f.db.venuePays) != 1 || len(f.db.files.rows) != before {
		t.Errorf("the refused venue payment left rows behind: %d payments, %d proofs", len(f.db.venuePays), len(f.db.files.rows)-before)
	}
}

func TestPaymentWithoutProofStillWorksWithoutTransactionSupport(t *testing.T) {
	db := &memDB{files: &proofFiles{}, activity: &memActivity{}}
	activity := NewActivityService(activityRepoAdapter{db.activity})
	ev := NewEvidenceService(db.files, newObjStorage(false), buildKeyForTests, activity)
	svc := NewClientPaymentService(dbClientPayments{db: db}, ev, activity, fakeContracts{project: 1_000_000})
	res, err := svc.CreateWithProof(context.Background(), 1, 10, 77, clientInput(100), nil, payDay)
	if err != nil || len(db.clientPays) != 1 || res.Ledger.Remaining != 999_900 || len(res.Terms) != 0 {
		t.Errorf("legacy wiring: %+v %v", res, err)
	}
}
