package presentation

import (
	"net/http"
	"strconv"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type TransactionHandler struct {
	transactions *application.TransactionService
}

func NewTransactionHandler(transactions *application.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactions: transactions}
}

type transactionResponse struct {
	ID               int64   `json:"id"`
	TenantID         int64   `json:"tenantId"`
	Type             string  `json:"type"`
	Amount           int64   `json:"amount"`
	PaymentMethod    string  `json:"paymentMethod"`
	PaymentReference string  `json:"paymentReference"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"createdAt"`
	PaidAt           *string `json:"paidAt"`
}

func toTransactionResponse(t domain.Transaction) transactionResponse {
	var paidAt *string
	if t.PaidAt != nil {
		s := t.PaidAt.Format("2006-01-02T15:04:05Z07:00")
		paidAt = &s
	}
	return transactionResponse{
		ID: t.ID, TenantID: t.TenantID, Type: string(t.Type), Amount: t.Amount,
		PaymentMethod: t.PaymentMethod, PaymentReference: t.PaymentReference, Status: string(t.Status),
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), PaidAt: paidAt,
	}
}

// List: platform_admin sees all tenants (optionally filtered by ?tenantId=),
// staff is always forced to their own tenant regardless of any query param.
func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return
	}

	var tenantID *int64
	if claims.PrincipalType == "platform_admin" {
		if raw := r.URL.Query().Get("tenantId"); raw != "" {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "tenantId tidak valid", nil)
				return
			}
			tenantID = &id
		}
	} else {
		id, err := strconv.ParseInt(claims.TenantID, 10, 64)
		if err != nil {
			response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
			return
		}
		tenantID = &id
	}

	if r.URL.Query().Get("all") == "true" {
		transactions, err := h.transactions.List(r.Context(), tenantID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]transactionResponse, 0, len(transactions))
		for _, t := range transactions {
			result = append(result, toTransactionResponse(t))
		}
		response.OK(w, "ok", result)
		return
	}

	params := pagination.FromRequest(r)
	status := r.URL.Query().Get("status")
	transactions, total, err := h.transactions.ListPaginated(r.Context(), tenantID, params, status)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]transactionResponse, 0, len(transactions))
	for _, t := range transactions {
		result = append(result, toTransactionResponse(t))
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
}
