package presentation

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/apperror"
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
	// TenantName is the workspace the transaction belongs to ("" when it cannot
	// be resolved), so a list needs no second call to /tenants/names.
	TenantName string `json:"tenantName"`
}

// wib: every time leaves the API in Western Indonesia Time, whatever the
// server's own zone (the web shows these strings as they are).
var wib = time.FixedZone("WIB", 7*60*60)

const timeLayout = "2006-01-02T15:04:05Z07:00"

func toTransactionResponse(t domain.Transaction) transactionResponse {
	var paidAt *string
	if t.PaidAt != nil {
		s := t.PaidAt.In(wib).Format(timeLayout)
		paidAt = &s
	}
	return transactionResponse{
		ID: t.ID, TenantID: t.TenantID, Type: string(t.Type), Amount: t.Amount,
		PaymentMethod: t.PaymentMethod, PaymentReference: t.PaymentReference, Status: string(t.Status),
		CreatedAt: t.CreatedAt.In(wib).Format(timeLayout), PaidAt: paidAt,
	}
}

// List: platform_admin sees all tenants (optionally filtered by ?tenantId=),
// staff is always forced to their own tenant regardless of any query param —
// and, for a staff principal, Owner-only (billing/transaction history is
// part of the "Langganan" menu, which the confirmed role rule locks to
// Owner in full, not just the write actions already gated elsewhere in
// `platform/tenant_handler.go`).
func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return
	}
	if claims.PrincipalType == "staff" && !middleware.GrantsFrom(r.Context()).Can("subscription.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat melihat riwayat transaksi langganan", map[string]string{"code": "forbidden_permission"})
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

	params := pagination.FromRequest(r)
	filter, err := transactionFilter(r)
	if err != nil {
		writeAppError(w, err)
		return
	}
	columns, defaultSort := h.transactions.TransactionSortSpec()
	sort, err := pagination.SortFromRequest(r, columns, defaultSort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	rows, total, err := h.transactions.ListPage(r.Context(), tenantID, params, filter, sort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]transactionResponse, 0, len(rows))
	for _, row := range rows {
		resp := toTransactionResponse(row.Transaction)
		resp.TenantName = row.TenantName
		result = append(result, resp)
	}
	response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total).WithSort(sort))
}

// transactionFilter reads ?status=, ?search= (or q: payment reference or tenant
// name) and ?from=/?to= (days, YYYY-MM-DD, both inclusive, as Indonesian days).
func transactionFilter(r *http.Request) (domain.TransactionFilter, error) {
	q := r.URL.Query()
	f := domain.TransactionFilter{Status: q.Get("status"), Search: pagination.SearchFromRequest(r)}
	bad := map[string][]string{}
	if raw := q.Get("from"); raw != "" {
		if t, err := time.ParseInLocation("2006-01-02", raw, wib); err != nil {
			bad["from"] = []string{"Tanggal harus berformat YYYY-MM-DD"}
		} else {
			f.From = t
		}
	}
	if raw := q.Get("to"); raw != "" {
		if t, err := time.ParseInLocation("2006-01-02", raw, wib); err != nil {
			bad["to"] = []string{"Tanggal harus berformat YYYY-MM-DD"}
		} else {
			f.To = t.AddDate(0, 0, 1)
		}
	}
	if !f.From.IsZero() && !f.To.IsZero() && !f.To.After(f.From) {
		bad["to"] = []string{"Tanggal akhir tidak boleh sebelum tanggal awal"}
	}
	if len(bad) > 0 {
		return f, apperror.Validation("Filter transaksi tidak valid", bad)
	}
	return f, nil
}

// Item is GET /subscription-transactions/{id}: one transaction (the receipt
// page). A staff member only reaches their own tenant's; anyone else's id
// answers 404, never "forbidden".
func (h *TransactionHandler) Item(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return
	}
	if claims.PrincipalType == "staff" && !middleware.GrantsFrom(r.Context()).Can("subscription.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat melihat riwayat transaksi langganan", map[string]string{"code": "forbidden_permission"})
		return
	}
	id, err := strconv.ParseInt(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/subscription-transactions/"), "/"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusNotFound, "Transaksi tidak ditemukan", nil)
		return
	}
	var tenantID *int64
	if claims.PrincipalType != "platform_admin" {
		own, err := strconv.ParseInt(claims.TenantID, 10, 64)
		if err != nil {
			response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
			return
		}
		tenantID = &own
	}
	t, err := h.transactions.Get(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp := toTransactionResponse(*t)
	resp.TenantName = h.transactions.NameOf(r.Context(), *t)
	response.OK(w, "ok", resp)
}
