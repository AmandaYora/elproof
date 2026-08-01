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
	result := make([]venuePaymentResponse, 0, len(list))
	for _, p := range list {
		result = append(result, toVenuePaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof)))
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
}

func (h *Handler) createVenuePayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
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
	p, err := h.venuePayments.Create(r.Context(), projectID, claims.staffID, application.VenuePaymentInput{
		Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	// A freshly created payment has no evidence yet -- the frontend's own
	// follow-up evidence upload(s) and refetch bring the true state current,
	// identical to createPayment/createClientPayment.
	response.Created(w, "Pembayaran venue berhasil dicatat", toVenuePaymentResponse(*p, false))
}
