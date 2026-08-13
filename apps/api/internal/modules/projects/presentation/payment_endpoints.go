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
	result := make([]paymentResponse, 0, len(list))
	for _, p := range list {
		result = append(result, toPaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof)))
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
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
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
	p, err := h.payments.Create(r.Context(), projectID, claims.staffID, application.PaymentInput{
		ProjectVendorID: body.ProjectVendorID, Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	// A freshly created payment has no evidence yet at the moment this
	// response is built -- the frontend's own follow-up evidence upload(s)
	// and refetch bring the true state current, identical to how
	// createClientPayment already works.
	response.Created(w, "Pembayaran berhasil dicatat", toPaymentResponse(*p, false))
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
	response.OK(w, "Pembayaran berhasil diperbarui", toPaymentResponse(*p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof)))
}

// deletePayment hard-deletes the payment (and its own evidence, cascaded
// inside PaymentService.Delete). Owner-or-Admin -- see isOwnerOrAdmin's doc
// comment, checked here rather than in the service so the service stays
// agnostic of who's calling.
func (h *Handler) deletePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	if !isOwnerOrAdmin(claims.role) {
		response.Error(w, http.StatusForbidden, "Hanya akun Owner atau Admin yang dapat menghapus pembayaran secara permanen", nil)
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
