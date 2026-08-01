package presentation

import (
	"encoding/json"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

// listClientPayments computes each payment's evidenceComplete the same way
// the response DTO needs it (§3.4 of PLAN.md's Client Payments design):
// client_payments has no direct evidence-ID column (unlike vendor_payments),
// so completeness is a simple existence check against the polymorphic
// evidence table instead of a struct method.
func (h *Handler) listClientPayments(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.clientPayments.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	evidences, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasEvidence := make(map[int64]bool, len(evidences))
	for _, e := range evidences {
		if e.RelatedKind == domain.RelatedClientPayment {
			hasEvidence[e.RelatedID] = true
		}
	}
	result := make([]clientPaymentResponse, 0, len(list))
	for _, p := range list {
		result = append(result, toClientPaymentResponse(p, hasEvidence[p.ID]))
	}
	response.OK(w, "ok", result)
}

type clientPaymentInputBody struct {
	Type            string `json:"type"`
	Amount          int64  `json:"amount"`
	PaymentDate     string `json:"paymentDate"`
	Method          string `json:"method"`
	ReferenceNumber string `json:"referenceNumber"`
	Notes           string `json:"notes"`
}

func (h *Handler) createClientPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body clientPaymentInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	paymentDate, err := parseDate(body.PaymentDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"paymentDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	p, err := h.clientPayments.Create(r.Context(), projectID, claims.staffID, application.ClientPaymentInput{
		Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Pembayaran client berhasil dicatat", toClientPaymentResponse(*p, false))
}
