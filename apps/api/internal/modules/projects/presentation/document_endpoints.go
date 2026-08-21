package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/response"
)

// downloadClientPaymentReceipt streams a Kwitansi (or, for a Refund row, a
// Bukti Pengembalian Dana) PDF for one existing client_payments row -- no
// role gate, reachable by staff and client alike (PLAN.md), since it's a
// pure read/download of data both principals can already see.
func (h *Handler) downloadClientPaymentReceipt(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID pembayaran tidak valid", nil)
		return
	}
	pdf, fileName, err := h.documents.GenerateReceipt(r.Context(), claims.tenantID, projectID, paymentID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

type generateInvoiceBody struct {
	Amount  int64  `json:"amount"`
	DueDate string `json:"dueDate"`
	Note    string `json:"note"`
}

// generateClientInvoice streams a Tagihan PDF for the project's outstanding
// balance (or an ad-hoc smaller amount). Staff-only in practice -- a client
// principal is already rejected here by resolveProjectAccess (POST is never
// allowed for a `client` principal), so no additional gate is needed.
func (h *Handler) generateClientInvoice(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body generateInvoiceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	var dueDate *time.Time
	if body.DueDate != "" {
		parsed, err := parseDate(body.DueDate)
		if err != nil {
			response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"dueDate": {"Gunakan format YYYY-MM-DD"}})
			return
		}
		dueDate = &parsed
	}
	pdf, fileName, err := h.documents.GenerateInvoice(r.Context(), claims.tenantID, projectID, claims.staffID, application.GenerateInvoiceInput{
		Amount: body.Amount, DueDate: dueDate, Note: body.Note,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+fileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
