package application

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// Paket WO Tahap 4 ("Vendor dibayar klien", PLAN §3.2, §4.3, §4.4): the bills
// of a vendor or venue the client pays directly, the proofs the couple sends
// for them from the portal, and the WO checking those proofs. None of this is
// the WO's money: the payments carry payer = Client and the WO's cash, revenue,
// cash flow and payables never read them.

// SupplierBillTermRepository stores supplier_bill_terms. Every method filters
// on the workspace as well as the project (ADR-0004); lists and lookups skip
// deleted bills and the bills of a deleted supplier.
type SupplierBillTermRepository interface {
	ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.SupplierBillTerm, error)
	FindByID(ctx context.Context, tenantID, projectID, id int64) (*domain.SupplierBillTerm, error)
	FindDeletedByID(ctx context.Context, tenantID, projectID, id int64) (*domain.SupplierBillTerm, error)
	Create(ctx context.Context, tenantID int64, t *domain.SupplierBillTerm) error
	Patch(ctx context.Context, tenantID int64, t *domain.SupplierBillTerm, changed []string, expected *time.Time) (bool, error)
	SoftDelete(ctx context.Context, tenantID, projectID, id int64, at time.Time) (bool, error)
	Restore(ctx context.Context, tenantID, projectID, id int64, notBefore time.Time) (bool, error)
}

// SupplierSubmissionReview is the WO's check of a client's proof to a supplier:
// conditional writes that only touch a live, still unchecked client submission
// (idempotent under two staff members at once).
type SupplierSubmissionReview interface {
	SetVerification(ctx context.Context, projectID, id int64, at time.Time, byStaffID int64) (bool, error)
	SetRejection(ctx context.Context, projectID, id int64, at time.Time, byStaffID int64, reason string) (bool, error)
	ClearVerification(ctx context.Context, projectID, id int64) error
}

// SupplierBillService runs the bills of client-paid suppliers and the proofs
// sent for them.
type SupplierBillService struct {
	bills     SupplierBillTermRepository
	payments  SupplierPaymentRepository
	review    SupplierSubmissionReview
	suppliers SupplierRepository
	evidence  *EvidenceService
	activity  *ActivityService
	proof     *ProofSupport
}

func NewSupplierBillService(bills SupplierBillTermRepository, payments SupplierPaymentRepository, review SupplierSubmissionReview,
	suppliers SupplierRepository, evidence *EvidenceService, activity *ActivityService) *SupplierBillService {
	s := &SupplierBillService{bills: bills, payments: payments, review: review, suppliers: suppliers, evidence: evidence, activity: activity}
	evidence.OnRemoved(s)
	return s
}

// SetProofSupport wires the transaction and upload checks the writes need.
func (s *SupplierBillService) SetProofSupport(p *ProofSupport) { s.proof = p }

var errBillTermNotFound = apperror.NotFound("Tagihan vendor tidak ditemukan")

// errPaidByWO refuses a bill on a supplier the WO pays: its cost is the WO's
// business, never a bill to the client.
func errPaidByWO() error {
	msg := "Tagihan hanya untuk vendor atau venue yang dibayar klien langsung."
	return apperror.Validation(msg, map[string][]string{"projectSupplierId": {msg}}).WithCode("supplier_paid_by_wo")
}

// clientPaidSupplier is the live supplier of the project, refused unless the
// client pays it.
func (s *SupplierBillService) clientPaidSupplier(ctx context.Context, projectID, projectSupplierID int64) (*domain.ProjectSupplier, error) {
	sup, err := s.suppliers.FindByID(ctx, projectID, projectSupplierID)
	if err != nil {
		return nil, err
	}
	if sup == nil {
		msg := "Pilih vendor atau venue proyek ini."
		return nil, apperror.Validation(msg, map[string][]string{"projectSupplierId": {msg}})
	}
	if sup.PaidBy != domain.PayerClient {
		return nil, errPaidByWO()
	}
	return sup, nil
}

// Bills is where every supplier the client pays directly stands as of `today`:
// its bills (with what is paid, left and waiting for the WO's check) and its
// payments, in the suppliers' order. A supplier the WO pays is never listed.
func (s *SupplierBillService) Bills(ctx context.Context, tenantID, projectID int64, today time.Time) ([]domain.SupplierBill, error) {
	sups, err := s.suppliers.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	terms, err := s.bills.ListByProject(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	payments, err := s.payments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var out []domain.SupplierBill
	for _, sup := range sups {
		if sup.PaidBy != domain.PayerClient {
			continue
		}
		out = append(out, domain.BuildSupplierBill(sup, terms, payments, today))
	}
	return out, nil
}

// SupplierBillInput is one bill as the WO types it.
type SupplierBillInput struct {
	ProjectSupplierID int64
	Label             string
	Amount            int64
	DueDate           time.Time
}

// SupplierBillPatch is a partial change of a bill (the supplier cannot move:
// delete and add again).
type SupplierBillPatch struct {
	Label   Opt[string]
	Amount  Opt[int64]
	DueDate Opt[time.Time]
}

const maxBillLabelLen = 100

func checkBillLabel(fe *fieldErrors, label string) {
	if label == "" || utf8.RuneCountInString(label) > maxBillLabelLen {
		fe.add("Isi nama tagihan (maksimal 100 karakter).", "label")
	}
}

func checkBillAmount(fe *fieldErrors, amount int64) {
	if amount <= 0 || amount > domain.MaxMoney {
		fe.add("Nominal harus lebih dari 0 dan wajar.", "amount")
	}
}

// CreateBill adds a bill to a supplier the client pays (422 with code
// supplier_paid_by_wo for a supplier the WO pays).
func (s *SupplierBillService) CreateBill(ctx context.Context, tenantID, projectID, actorStaffID int64, in SupplierBillInput) (*domain.SupplierBillTerm, error) {
	label := strings.TrimSpace(in.Label)
	fe := newFieldErrors()
	checkBillLabel(fe, label)
	checkBillAmount(fe, in.Amount)
	if in.DueDate.IsZero() {
		fe.add("Isi tanggal jatuh tempo.", "dueDate")
	}
	if in.ProjectSupplierID <= 0 {
		fe.add("Pilih vendor atau venue proyek ini.", "projectSupplierId")
	}
	if !fe.empty() {
		return nil, fe.err()
	}
	var out *domain.SupplierBillTerm
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		sup, err := s.clientPaidSupplier(ctx, projectID, in.ProjectSupplierID)
		if err != nil {
			return err
		}
		existing, err := s.bills.ListByProject(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		order := 0
		for _, t := range existing {
			if t.ProjectSupplierID == sup.ID && t.SortOrder >= order {
				order = t.SortOrder + 1
			}
		}
		t := &domain.SupplierBillTerm{ProjectID: projectID, ProjectSupplierID: sup.ID, Label: label, Amount: in.Amount, DueDate: in.DueDate, SortOrder: order}
		if err := s.bills.Create(ctx, tenantID, t); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errBillTermNotFound
			}
			return err
		}
		out = t
		s.activity.Record(ctx, &projectID, domain.ActivityVendorUpdated, actorStaffID, "supplier_bill_term", formatID(t.ID), t.Label,
			"Tagihan "+t.Label+" "+formatRupiah(t.Amount)+" untuk "+sup.Name+" ditambahkan")
		return nil
	})
	return out, err
}

// GetBill is one live bill of the project.
func (s *SupplierBillService) GetBill(ctx context.Context, tenantID, projectID, id int64) (*domain.SupplierBillTerm, error) {
	t, err := s.bills.FindByID(ctx, tenantID, projectID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errBillTermNotFound
	}
	return t, nil
}

// PatchBill changes the sent fields of a bill (If-Match: compare-and-set).
func (s *SupplierBillService) PatchBill(ctx context.Context, tenantID, projectID, id, actorStaffID int64, patch SupplierBillPatch, expected *time.Time) (*domain.SupplierBillTerm, error) {
	fe := newFieldErrors()
	if patch.Label.Present() {
		if patch.Label.IsNull() {
			checkBillLabel(fe, "")
		} else {
			patch.Label = Some(strings.TrimSpace(patch.Label.Value()))
			checkBillLabel(fe, patch.Label.Value())
		}
	}
	if patch.Amount.Present() {
		checkBillAmount(fe, patch.Amount.Value())
	}
	if patch.DueDate.Present() && (patch.DueDate.IsNull() || patch.DueDate.Value().IsZero()) {
		fe.add("Isi tanggal jatuh tempo.", "dueDate")
	}
	if !fe.empty() {
		return nil, fe.err()
	}
	var out *domain.SupplierBillTerm
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		t, err := s.GetBill(ctx, tenantID, projectID, id)
		if err != nil {
			return err
		}
		loaded := *t
		var ch changeSet
		setVal(&ch, "label", "nama", &t.Label, patch.Label)
		setVal(&ch, "amount", "nominal", &t.Amount, patch.Amount)
		setDate(&ch, "dueDate", "jatuh tempo", &t.DueDate, patch.DueDate)
		if !ch.any() {
			out = t
			return nil
		}
		if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
			return err
		}
		wrote, err := s.bills.Patch(ctx, tenantID, t, ch.keys, expected)
		if err != nil {
			return err
		}
		fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.SupplierBillTerm, error) {
			return s.bills.FindByID(ctx, tenantID, projectID, id)
		}, func(t *domain.SupplierBillTerm) time.Time { return t.UpdatedAt }, errBillTermNotFound)
		if err != nil {
			return err
		}
		out = fresh
		s.activity.Record(ctx, &projectID, domain.ActivityVendorUpdated, actorStaffID, "supplier_bill_term", formatID(fresh.ID), fresh.Label,
			"Tagihan vendor "+fresh.Label+" diperbarui: "+ch.summary())
		return nil
	})
	return out, err
}

// DeleteBill soft-deletes a bill (ADR-0034). A proof already sent for it stays
// (its link to the bill goes when the bill is purged).
func (s *SupplierBillService) DeleteBill(ctx context.Context, tenantID, projectID, id, actorStaffID int64, now time.Time) (*domain.SupplierBillTerm, time.Time, error) {
	t, err := s.GetBill(ctx, tenantID, projectID, id)
	if err != nil {
		return nil, time.Time{}, err
	}
	ok, err := s.bills.SoftDelete(ctx, tenantID, projectID, id, now)
	if err != nil {
		return nil, time.Time{}, err
	}
	if !ok {
		return nil, time.Time{}, errBillTermNotFound
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorUpdated, actorStaffID, "supplier_bill_term", formatID(t.ID), t.Label,
		"Tagihan vendor "+t.Label+" dihapus")
	return t, domain.RestorableUntil(now), nil
}

// RestoreBill brings a deleted bill back while its supplier is live and still
// paid by the client.
func (s *SupplierBillService) RestoreBill(ctx context.Context, tenantID, projectID, id, actorStaffID int64, now time.Time) (*domain.SupplierBillTerm, error) {
	var out *domain.SupplierBillTerm
	err := inLockedTx(ctx, s.proof, projectID, func(ctx context.Context) error {
		t, err := s.bills.FindDeletedByID(ctx, tenantID, projectID, id)
		if err != nil {
			return err
		}
		if t == nil || t.DeletedAt == nil || !domain.CanRestore(*t.DeletedAt, now) {
			return errNotRestorable()
		}
		if _, err := s.clientPaidSupplier(ctx, projectID, t.ProjectSupplierID); err != nil {
			if ae, ok := apperror.As(err); ok && ae.Code != "supplier_paid_by_wo" {
				msg := "Vendor atau venue untuk tagihan ini sudah dihapus. Pulihkan lebih dulu."
				return apperror.Validation(msg, map[string][]string{"projectSupplierId": {msg}})
			}
			return err
		}
		ok, err := s.bills.Restore(ctx, tenantID, projectID, id, domain.PurgeCutoff(now))
		if err != nil {
			return err
		}
		if !ok {
			return errNotRestorable()
		}
		t.DeletedAt = nil
		out = t
		s.activity.Record(ctx, &projectID, domain.ActivityRestored, actorStaffID, "supplier_bill_term", formatID(t.ID), t.Label,
			"Tagihan vendor "+t.Label+" dipulihkan")
		return nil
	})
	return out, err
}

// SupplierSubmissionInput is what the couple sends from the portal with the
// proof of a payment to a supplier they pay directly
// (POST /projects/{id}/supplier-bills/submissions).
type SupplierSubmissionInput struct {
	BillTermID      int64
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}

// SupplierSubmitted is the outcome of a client's proof: the payment (pending
// the WO's check), its files and where that supplier now stands.
type SupplierSubmitted struct {
	Payment  *domain.SupplierPayment
	Evidence []domain.Evidence
	Bill     domain.SupplierBill
}

// SubmitByClient records the couple's proof for one bill as ONE atomic action
// (ADR-0033, ADR-0047): the uploads are verified first, then in one
// transaction the project is locked, the bill (of THIS project and workspace,
// on a live supplier the client pays) is checked, the payment (payer Client,
// unverified, SubmittedByClientID set) and every proof row are inserted and the
// activity is logged. It counts as paid only after the WO verifies it.
func (s *SupplierBillService) SubmitByClient(ctx context.Context, tenantID, projectID, clientID int64, input SupplierSubmissionInput, items []ProofItem, today time.Time) (*SupplierSubmitted, error) {
	if clientID <= 0 {
		return nil, apperror.Forbidden("Hanya klien yang dapat mengirim bukti pembayaran")
	}
	proof := s.proof
	if proof == nil {
		proof = NewProofSupport(nil, nil, s.evidence)
	}
	fe := newFieldErrors()
	if input.BillTermID <= 0 {
		fe.add("Pilih tagihan yang dibayar.", "billTermId")
	}
	validateSubmission(fe, ClientSubmissionInput{
		Amount: input.Amount, PaymentDate: input.PaymentDate, Method: input.Method,
		ReferenceNumber: input.ReferenceNumber, Notes: input.Notes,
	}, len(items), today)
	resolved, err := proof.resolve(ctx, tenantID, projectID, items, clientProofTypes)
	if err != nil && !fe.mergeErr(err) {
		return nil, err
	}
	if !fe.empty() {
		return nil, fe.err()
	}

	var out SupplierSubmitted
	err = proof.tx.InTx(ctx, func(ctx context.Context) error {
		if err := proof.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		bill, err := s.bills.FindByID(ctx, tenantID, projectID, input.BillTermID)
		if err != nil {
			return err
		}
		if bill == nil {
			msg := "Tagihan ini tidak ditemukan."
			return apperror.Validation(msg, map[string][]string{"billTermId": {msg}})
		}
		sup, err := s.suppliers.FindByID(ctx, projectID, bill.ProjectSupplierID)
		if err != nil {
			return err
		}
		if sup == nil || sup.PaidBy != domain.PayerClient {
			msg := "Tagihan ini tidak ditemukan."
			return apperror.Validation(msg, map[string][]string{"billTermId": {msg}})
		}
		all, err := s.payments.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		var lines []domain.PaymentLine
		for _, p := range all {
			if p.ProjectSupplierID == sup.ID && p.Payer == domain.PayerClient && p.Status() != domain.SupplierPaymentRejected {
				lines = append(lines, lineOf(p.Type, p.Amount))
			}
		}
		terms, err := s.bills.ListByProject(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		var total int64
		for _, t := range terms {
			if t.ProjectSupplierID == sup.ID {
				total += t.Amount
			}
		}
		p := &domain.SupplierPayment{
			ProjectID: projectID, ProjectSupplierID: sup.ID, BillTermID: bill.ID, Payer: domain.PayerClient,
			Type:   domain.SubmissionType(domain.AppliedToContract(lines), total, input.Amount),
			Amount: input.Amount, PaymentDate: input.PaymentDate,
			Method: strings.TrimSpace(input.Method), ReferenceNumber: strings.TrimSpace(input.ReferenceNumber), Notes: strings.TrimSpace(input.Notes),
			SubmittedByClientID: clientID,
		}
		if err := s.payments.Create(ctx, p); err != nil {
			return err
		}
		evidence, err := proof.attach(ctx, proofTarget{
			ProjectID: projectID, Kind: domain.RelatedSupplierPayment, RelatedID: p.ID,
			ActorClientID: clientID, Date: input.PaymentDate, Reference: p.ReferenceNumber,
		}, resolved)
		if err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, 0, "supplier_payment", formatID(p.ID), string(p.Type),
			"Pasangan mengirim bukti pembayaran "+formatRupiah(p.Amount)+" ke "+sup.Name)
		after, err := s.payments.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		out = SupplierSubmitted{Payment: p, Evidence: evidence, Bill: domain.BuildSupplierBill(*sup, terms, after, today)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// submission is a live client proof to a supplier of the project (404 for a
// payment the WO recorded: there is nothing to check).
func (s *SupplierBillService) submission(ctx context.Context, projectID, id int64) (*domain.SupplierPayment, error) {
	p, err := s.payments.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.IsClientSubmission() || p.Payer != domain.PayerClient {
		return nil, errSupplierPaymentNotFound
	}
	return p, nil
}

func errSubmissionRejected() error {
	msg := "Bukti ini sudah ditolak. Klien perlu mengirim bukti baru."
	return apperror.Validation(msg, map[string][]string{"status": {msg}}).WithCode("submission_rejected")
}

func errSubmissionVerified() error {
	msg := "Bukti ini sudah ditandai sesuai. Ubah atau hapus pembayarannya bila ada yang salah."
	return apperror.Validation(msg, map[string][]string{"status": {msg}}).WithCode("already_verified")
}

// Verify marks the client's proof to a supplier right ("Tandai sesuai"). It is
// idempotent, refuses a proof with no file (ADR-0032) and a refused one.
func (s *SupplierBillService) Verify(ctx context.Context, projectID, id, actorStaffID int64, now time.Time) (*domain.SupplierPayment, error) {
	p, err := s.submission(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if p.VerifiedAt != nil {
		return p, nil
	}
	files, err := s.evidence.ListForRelated(ctx, domain.RelatedSupplierPayment, id)
	if err != nil {
		return nil, err
	}
	if _, err := p.VerifySubmission(now, actorStaffID, len(files)); err != nil {
		switch {
		case errors.Is(err, domain.ErrVerifyNeedsEvidence):
			return nil, errVerifyNeedsEvidence()
		case errors.Is(err, domain.ErrSubmissionRejected):
			return nil, errSubmissionRejected()
		}
		return nil, err
	}
	won, err := s.review.SetVerification(ctx, projectID, id, *p.VerifiedAt, actorStaffID)
	if err != nil {
		return nil, err
	}
	fresh, err := s.submission(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if !won {
		// Someone checked it between our read and write: report theirs.
		if fresh.RejectedAt != nil {
			return nil, errSubmissionRejected()
		}
		return fresh, nil
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "supplier_payment", formatID(p.ID), string(p.Type),
		"Bukti pembayaran klien ke vendor ditandai sesuai")
	return fresh, nil
}

// Reject refuses the client's proof to a supplier with a reason the couple
// reads ("Tolak dengan alasan"). Idempotent on a refused proof; a verified
// proof cannot be refused.
func (s *SupplierBillService) Reject(ctx context.Context, projectID, id, actorStaffID int64, reason string, now time.Time) (*domain.SupplierPayment, error) {
	reason = strings.TrimSpace(reason)
	switch {
	case reason == "":
		msg := "Tulis alasan penolakan agar klien tahu apa yang perlu diperbaiki."
		return nil, apperror.Validation(msg, map[string][]string{"reason": {msg}})
	case utf8.RuneCountInString(reason) > domain.MaxRejectionReasonLen:
		msg := "Alasan maksimal 500 karakter."
		return nil, apperror.Validation(msg, map[string][]string{"reason": {msg}})
	}
	p, err := s.submission(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	changed, err := p.RejectSubmission(now, actorStaffID, reason)
	if err != nil {
		if errors.Is(err, domain.ErrSubmissionVerified) {
			return nil, errSubmissionVerified()
		}
		return nil, err
	}
	if !changed {
		return p, nil
	}
	won, err := s.review.SetRejection(ctx, projectID, id, *p.RejectedAt, actorStaffID, reason)
	if err != nil {
		return nil, err
	}
	fresh, err := s.submission(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if !won {
		if fresh.VerifiedAt != nil {
			return nil, errSubmissionVerified()
		}
		return fresh, nil
	}
	s.activity.Record(ctx, &projectID, domain.ActivityPaymentUpdated, actorStaffID, "supplier_payment", formatID(p.ID), string(p.Type),
		"Bukti pembayaran klien ke vendor ditolak: "+reason)
	return fresh, nil
}

// AfterEvidenceRemoved keeps "a verified proof always has at least one file":
// when the last file of a verified client proof to a supplier goes, the
// verification goes with it.
func (s *SupplierBillService) AfterEvidenceRemoved(ctx context.Context, e domain.Evidence, _ int64) {
	if e.RelatedKind != domain.RelatedSupplierPayment {
		return
	}
	p, err := s.payments.FindByID(ctx, e.ProjectID, e.RelatedID)
	if err != nil || p == nil || !p.IsClientSubmission() || p.VerifiedAt == nil {
		return
	}
	left, err := s.evidence.ListForRelated(ctx, domain.RelatedSupplierPayment, e.RelatedID)
	if err != nil || len(left) > 0 {
		return
	}
	_ = s.review.ClearVerification(ctx, e.ProjectID, e.RelatedID)
}
