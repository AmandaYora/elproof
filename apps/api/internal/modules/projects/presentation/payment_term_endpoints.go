package presentation

import (
	"elproof/internal/shared/apperror"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

type paymentTermResponse struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	Amount    int64  `json:"amount"`
	DueDate   string `json:"dueDate"`
	Paid      int64  `json:"paid"`
	Remaining int64  `json:"remaining"`
	// Status: paid | overdue | partial | due_soon | upcoming.
	Status string `json:"status"`
}

func toPaymentTermResponses(list []domain.TermProgress) []paymentTermResponse {
	out := make([]paymentTermResponse, 0, len(list))
	for _, t := range list {
		out = append(out, paymentTermResponse{
			ID: t.Term.ID, Label: t.Term.Label, Amount: t.Term.Amount, DueDate: t.Term.DueDate.Format(dateLayout),
			Paid: t.Paid, Remaining: t.Remaining, Status: string(t.Status),
		})
	}
	return out
}

func (h *Handler) listPaymentTerms(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	list, err := h.terms.List(r.Context(), claims.tenantID, projectID, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toPaymentTermResponses(list))
}

type paymentTermsBody struct {
	Terms []struct {
		Label   string `json:"label"`
		Amount  int64  `json:"amount"`
		DueDate string `json:"dueDate"`
	} `json:"terms"`
}

// replacePaymentTerms (PUT) swaps the whole schedule for the one sent.
func (h *Handler) replacePaymentTerms(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body paymentTermsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	// A date that is not YYYY-MM-DD is reported on its own row; the service
	// reports every other problem, and the two are merged so nothing is hidden.
	inputs := make([]application.PaymentTermInput, 0, len(body.Terms))
	badDates := map[string][]string{}
	for i, t := range body.Terms {
		due, err := parseDate(t.DueDate)
		if err != nil && t.DueDate != "" {
			badDates["terms["+strconv.Itoa(i)+"].dueDate"] = []string{"Gunakan format YYYY-MM-DD"}
			due = time.Time{}
		}
		inputs = append(inputs, application.PaymentTermInput{Label: t.Label, Amount: t.Amount, DueDate: due})
	}
	list, err := h.terms.Replace(r.Context(), claims.tenantID, projectID, claims.staffID, inputs, nowWIB())
	if err != nil {
		if appErr, ok := apperror.As(err); ok && appErr.Kind == apperror.KindValidation && len(badDates) > 0 {
			merged := *appErr
			merged.Fields = map[string][]string{}
			for k, v := range appErr.Fields {
				merged.Fields[k] = v
			}
			for k, v := range badDates {
				merged.Fields[k] = v // the format message is more precise than "isi tanggal"
			}
			err = &merged
		}
		writeAppError(w, err)
		return
	}
	response.OK(w, "Jadwal pembayaran disimpan", toPaymentTermResponses(list))
}
