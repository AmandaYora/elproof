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
	result := make([]paymentResponse, 0, len(list))
	for _, p := range list {
		result = append(result, toPaymentResponse(p))
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
	response.Created(w, "Pembayaran berhasil dicatat", toPaymentResponse(*p))
}
