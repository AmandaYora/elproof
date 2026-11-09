package presentation

import (
	"encoding/json"
	"net/http"

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
	response.Created(w, "Pembayaran berhasil dicatat", paymentCreatedResponse{paymentResponse: resp, LedgerSummary: toLedgerSummary(created.Ledger)})
}

// updatePayment is a full-field overwrite, no role-gate -- same convention
// as updateProject/updateVendorEngagement/updateIssue (PLAN.md §2.3).
func (h *Handler) updatePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
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
	p, err := h.payments.Update(r.Context(), projectID, paymentID, claims.staffID, application.PaymentInput{
		Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	})
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
	response.OK(w, "Pembayaran berhasil diperbarui", resp)
}

// deletePayment hard-deletes the payment (and its own evidence, cascaded
// inside PaymentService.Delete). Owner-or-Admin -- see isOwnerOrAdmin's doc
// comment, checked here rather than in the service so the service stays
// agnostic of who's calling.
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
	if err := h.payments.Delete(r.Context(), projectID, paymentID, claims.staffID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran berhasil dihapus", nil)
}
