package presentation

import (
	"encoding/json"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

func (h *Handler) listVenuePayments(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.venuePayments.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	evidences, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedVenuePayment)
	ix := indexEvidence(evidences, false, nil)
	result := make([]venuePaymentResponse, 0, len(list))
	for _, p := range list {
		resp := toVenuePaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
		resp.Evidence = ix.refs(domain.RelatedVenuePayment, p.ID)
		result = append(result, resp)
	}
	response.OK(w, "ok", result)
}

type venuePaymentInputBody struct {
	Type            string `json:"type"`
	Amount          int64  `json:"amount"`
	PaymentDate     string `json:"paymentDate"`
	Method          string `json:"method"`
	ReferenceNumber string `json:"referenceNumber"`
	Notes           string `json:"notes"`
	// Evidence: see paymentInputBody.
	Evidence []proofBody `json:"evidence"`
}

// venuePaymentCreatedResponse is the payment (as listed) plus where the venue's
// cost (rental price + charge) stands.
type venuePaymentCreatedResponse struct {
	venuePaymentResponse
	LedgerSummary ledgerSummaryResponse `json:"ledgerSummary"`
}

// createVenuePayment records a venue payment, optionally with its proof, as one
// atomic action, under the optional Idempotency-Key header.
func (h *Handler) createVenuePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
		h.createVenuePaymentNow(w, r, claims, projectID)
	})
}

func (h *Handler) createVenuePaymentNow(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body venuePaymentInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	paymentDate, err := parseDate(body.PaymentDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"paymentDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	created, err := h.venuePayments.CreateWithProof(r.Context(), claims.tenantID, projectID, claims.staffID, application.VenuePaymentInput{
		Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	}, toProofItems(body.Evidence))
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(created.Evidence, domain.RelatedVenuePayment)
	p := created.Payment
	resp := toVenuePaymentResponse(*p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
	resp.Evidence = evidenceRefs(created.Evidence)
	response.Created(w, "Pembayaran venue berhasil dicatat", venuePaymentCreatedResponse{venuePaymentResponse: resp, LedgerSummary: toLedgerSummary(created.Ledger)})
}

// updateVenuePayment is a full-field overwrite, no role-gate — see
// updatePayment's doc comment.
func (h *Handler) updateVenuePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body venuePaymentInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	paymentDate, err := parseDate(body.PaymentDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"paymentDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	p, err := h.venuePayments.Update(r.Context(), claims.tenantID, projectID, paymentID, claims.staffID, application.VenuePaymentInput{
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
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedVenuePayment)
	resp := toVenuePaymentResponse(*p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
	resp.Evidence = indexEvidence(evidences, false, nil).refs(domain.RelatedVenuePayment, p.ID)
	response.OK(w, "Pembayaran venue berhasil diperbarui", resp)
}

// deleteVenuePayment hard-deletes the payment (and its own evidence).
// Owner-or-Admin — see deletePayment's doc comment.
func (h *Handler) deleteVenuePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	if !claims.can("finance.payment.delete") {
		forbiddenPermission(w, "Peran Anda tidak dapat menghapus pembayaran")
		return
	}
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	if err := h.venuePayments.Delete(r.Context(), claims.tenantID, projectID, paymentID, claims.staffID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran venue berhasil dihapus", nil)
}
