package application

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// ClientSubmissionInput is what the couple sends from the Client Portal with a
// transfer proof (POST /projects/{id}/client-payments/submissions, ADR-0047).
// There is no type: the portal does not ask; see domain.SubmissionType.
type ClientSubmissionInput struct {
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}

// Field limits of the columns the submission is written to.
const (
	maxMethodLen    = 100
	maxReferenceLen = 100
	maxNotesLen     = 1000
)

// validateSubmission collects what is wrong with the couple's fields: the
// amount (the staff rule), a method, the lengths the columns hold, a date that
// is not in the future (Jakarta calendar day) and at least one proof.
func validateSubmission(fe *fieldErrors, in ClientSubmissionInput, proofs int, today time.Time) {
	if err := checkAmount(in.Amount); err != nil {
		fe.mergeErr(err)
	}
	method := strings.TrimSpace(in.Method)
	switch {
	case method == "":
		fe.add("Pilih cara pembayaran.", "method")
	case utf8.RuneCountInString(method) > maxMethodLen:
		fe.add("Cara pembayaran maksimal 100 karakter.", "method")
	}
	if utf8.RuneCountInString(strings.TrimSpace(in.ReferenceNumber)) > maxReferenceLen {
		fe.add("Nomor referensi maksimal 100 karakter.", "referenceNumber")
	}
	if utf8.RuneCountInString(strings.TrimSpace(in.Notes)) > maxNotesLen {
		fe.add("Catatan maksimal 1.000 karakter.", "notes")
	}
	if in.PaymentDate.Format(dayLayout) > domain.TodayWIB(today).Format(dayLayout) {
		fe.add("Tanggal transfer tidak boleh setelah hari ini.", "paymentDate")
	}
	if proofs == 0 {
		fe.add("Lampirkan minimal satu foto atau PDF bukti transfer.", "evidence")
	}
}

// clientOverpaymentError is the couple's version of the overpayment refusal:
// they cannot pick "Tambahan", so the staff hint is left out.
func clientOverpaymentError(err error) error {
	ae, ok := apperror.As(err)
	if !ok || ae.Code != "payment_exceeds_contract" {
		return err
	}
	msg := "Nominal ini melebihi sisa tagihan Anda. Periksa nominalnya, atau hubungi WO bila ada biaya tambahan."
	return apperror.Validation(msg, map[string][]string{"amount": {msg}}).WithCode("payment_exceeds_contract")
}

// SubmitByClient records the couple's transfer proof as ONE atomic action, the
// same path as CreateWithProof (ADR-0033): the uploads are verified first, then
// in one transaction the project is locked, the overpayment rule is checked,
// the payment (unverified, SubmittedByClientID set, no RecordedByStaffID) and
// every proof row are inserted and the activity is logged. Staff verify it with
// the usual endpoint. Every field problem is reported together.
func (s *ClientPaymentService) SubmitByClient(ctx context.Context, tenantID, projectID, clientID int64, input ClientSubmissionInput, items []ProofItem, today time.Time) (*ClientPaymentCreated, error) {
	if clientID <= 0 {
		return nil, apperror.Forbidden("Hanya klien yang dapat mengirim bukti pembayaran")
	}
	proof := s.proof
	if proof == nil {
		proof = NewProofSupport(nil, nil, s.evidence)
	}
	fe := newFieldErrors()
	validateSubmission(fe, input, len(items), today)
	resolved, err := proof.resolve(ctx, tenantID, projectID, items, clientProofTypes)
	if err != nil && !fe.mergeErr(err) {
		return nil, err
	}
	if !fe.empty() {
		return nil, fe.err()
	}

	var out ClientPaymentCreated
	err = proof.tx.InTx(ctx, func(ctx context.Context) error {
		if err := proof.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		contract, err := s.contracts.ProjectContractValue(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		all, err := s.repo.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		lines := make([]domain.PaymentLine, 0, len(all))
		for _, p := range all {
			lines = append(lines, lineOf(p.Type, p.Amount))
		}
		paymentType := domain.SubmissionType(domain.AppliedToContract(lines), contract, input.Amount)
		if err := s.checkChange(ctx, tenantID, projectID, 0, &domain.PaymentLine{Type: paymentType, Amount: input.Amount}); err != nil {
			return clientOverpaymentError(err)
		}
		p := &domain.ClientPayment{
			ProjectID: projectID, Type: paymentType, Amount: input.Amount, PaymentDate: input.PaymentDate,
			Method: strings.TrimSpace(input.Method), ReferenceNumber: strings.TrimSpace(input.ReferenceNumber), Notes: strings.TrimSpace(input.Notes),
			SubmittedByClientID: clientID,
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		evidence, err := proof.attach(ctx, proofTarget{
			ProjectID: projectID, Kind: domain.RelatedClientPayment, RelatedID: p.ID,
			ActorClientID: clientID,
			Date:          input.PaymentDate, Reference: p.ReferenceNumber,
		}, resolved)
		if err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityPaymentRecorded, 0, "client_payment", formatID(p.ID), string(p.Type),
			"Pasangan mengirim bukti pembayaran "+formatRupiah(p.Amount))
		ledger, terms, err := s.ledgerAndTerms(ctx, tenantID, projectID, today)
		if err != nil {
			return err
		}
		out = ClientPaymentCreated{Payment: p, Evidence: evidence, Ledger: ledger, Terms: terms}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
