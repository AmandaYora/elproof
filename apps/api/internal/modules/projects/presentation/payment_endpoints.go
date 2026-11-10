package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

func (h *Handler) listPayments(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.payments.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	evidences, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedPayment)
	ix := indexEvidence(evidences, false, nil)
	result := make([]paymentResponse, 0, len(list))
	for _, p := range list {
		resp := toPaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
		resp.Evidence = ix.refs(domain.RelatedPayment, p.ID)
		result = append(result, resp)
	}
	response.OK(w, "ok", result)
}

type paymentInputBody struct {
	ProjectVendorID int64  `json:"projectVendorId"`
	Type            string `json:"type"`
	Amount          int64  `json:"amount"`
	PaymentDate     string `json:"paymentDate"`
	Method          string `json:"method"`
	ReferenceNumber string `json:"referenceNumber"`
	Notes           string `json:"notes"`
	// Evidence names files already uploaded with POST /projects/{id}/uploads
	// ("Invoice" and/or "Transfer Proof", at most 4); they are recorded in the
	// same transaction as the payment.
	Evidence []proofBody `json:"evidence"`
}

// paymentCreatedResponse is the payment (as listed) plus where that
// engagement's price stands.
type paymentCreatedResponse struct {
	paymentResponse
	LedgerSummary ledgerSummaryResponse `json:"ledgerSummary"`
}

// createPayment records a vendor payment, optionally with its proof, as one
// atomic action, under the optional Idempotency-Key header.
func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
		h.createPaymentNow(w, r, claims, projectID)
	})
}

func (h *Handler) createPaymentNow(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body paymentInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	paymentDate, err := parseDate(body.PaymentDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"paymentDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	created, err := h.payments.CreateWithProof(r.Context(), claims.tenantID, projectID, claims.staffID, application.PaymentInput{
		ProjectVendorID: body.ProjectVendorID, Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	}, toProofItems(body.Evidence))
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(created.Evidence, domain.RelatedPayment)
	p := created.Payment
	resp := toPaymentResponse(*p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
	resp.Evidence = evidenceRefs(created.Evidence)
	response.Created(w, "Pembayaran berhasil dicatat", h.mutated(r.Context(), claims.tenantID, projectID, paymentCreatedResponse{paymentResponse: resp, LedgerSummary: toLedgerSummary(created.Ledger)}))
}

// readPaymentPatch reads the body of a PATCH on a client, vendor or venue payment:
// type, amount, paymentDate, method, referenceNumber and notes (null clears the
// note); a vendor payment's engagement is not patchable.
func readPaymentPatch(pr *patchReader) application.PaymentPatch {
	t := pr.text("type", false)
	patch := application.PaymentPatch{
		Amount: pr.integer("amount", false), PaymentDate: pr.date("paymentDate", false),
		Method: pr.text("method", false), ReferenceNumber: pr.text("referenceNumber", false), Notes: pr.text("notes", true),
	}
	if t.Present() {
		patch.Type = application.Some(domain.PaymentType(t.Value()))
	}
	return patch
}

// paymentItem is a vendor payment as GET /projects/{id}/payments lists it (with
// its proof files and whether the evidence is complete).
func (h *Handler) paymentItem(ctx context.Context, projectID int64, p domain.VendorPayment) (paymentResponse, error) {
	evidences, err := h.evidence.List(ctx, projectID)
	if err != nil {
		return paymentResponse{}, err
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedPayment)
	resp := toPaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
	resp.Evidence = indexEvidence(evidences, false, nil).refs(domain.RelatedPayment, p.ID)
	return resp, nil
}

// updatePayment is PATCH /projects/{id}/payments/{pid}: a partial update (type,
// amount, paymentDate, method, referenceNumber, notes), no role-gate beyond
// finance.record. The overpayment rule is judged on the merged result. See
// updateProject for the body, If-Match and 409 rules.
func (h *Handler) updatePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	expected, ok := ifMatch(w, r)
	if !ok {
		return
	}
	pr, ok := newPatchReader(w, r)
	if !ok {
		return
	}
	patch := readPaymentPatch(pr)
	if err := pr.finish(); err != nil {
		writeAppError(w, err)
		return
	}
	p, err := h.payments.Patch(r.Context(), projectID, paymentID, claims.staffID, patch, expected)
	if err != nil {
		writePatchError(w, err, func(cur any) (any, error) {
			return h.paymentItem(r.Context(), projectID, *cur.(*domain.VendorPayment))
		})
		return
	}
	resp, err := h.paymentItem(r.Context(), projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.payments.LedgerSummary(r.Context(), projectID, p.ProjectVendorID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran berhasil diperbarui", h.mutated(r.Context(), claims.tenantID, projectID, resp, extra{"ledgerSummary", toLedgerSummary(ledger)}))
}

// deletePayment soft-deletes the payment (finance.payment.delete); see
// deleteClientPayment for the answer. The ledger is the payment's vendor
// engagement's.
func (h *Handler) deletePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	if !claims.can("finance.payment.delete") {
		forbiddenPermission(w, "Peran Anda tidak dapat menghapus pembayaran")
		return
	}
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	p, until, err := h.payments.Delete(r.Context(), projectID, paymentID, claims.staffID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.payments.LedgerSummary(r.Context(), projectID, p.ProjectVendorID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran berhasil dihapus", h.mutated(r.Context(), claims.tenantID, projectID, toDeletedResponse(paymentID, until), extra{"ledgerSummary", toLedgerSummary(ledger)}))
}

// restorePayment is POST /projects/{id}/payments/{pid}/restore
// (finance.payment.delete); see restoreClientPayment. 422 `projectVendorId`
// when the payment's vendor engagement is itself deleted (restore it first).
func (h *Handler) restorePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	p, err := h.payments.Restore(r.Context(), projectID, paymentID, claims.staffID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	evidences, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedPayment)
	resp := toPaymentResponse(*p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
	resp.Evidence = indexEvidence(evidences, false, nil).refs(domain.RelatedPayment, p.ID)
	ledger, err := h.payments.LedgerSummary(r.Context(), projectID, p.ProjectVendorID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran dipulihkan", h.mutated(r.Context(), claims.tenantID, projectID, resp, extra{"ledgerSummary", toLedgerSummary(ledger)}))
}
