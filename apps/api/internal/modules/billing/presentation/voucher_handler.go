package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

// voucherManager is the slice of *application.VoucherService the console
// endpoints need (an interface so tests can stub it).
type voucherManager interface {
	List(ctx context.Context, params pagination.Params, f domain.VoucherFilter) ([]domain.Voucher, int64, error)
	Get(ctx context.Context, id int64) (*domain.Voucher, error)
	Create(ctx context.Context, in application.VoucherInput) (*domain.Voucher, error)
	Update(ctx context.Context, id int64, in application.VoucherInput, expected *time.Time) (*domain.Voucher, error)
	SetActive(ctx context.Context, id int64, active bool) (*domain.Voucher, error)
	ListRedemptions(ctx context.Context, voucherID int64, params pagination.Params) ([]application.RedemptionRow, int64, error)
	Quote(ctx context.Context, code string, plan *domain.Plan, c domain.Customer) (*application.VoucherQuote, error)
	Today() string
}

// tenantPlanReader reads a plan of ElProof's own catalog (422 otherwise).
type tenantPlanReader interface {
	GetTenantPlan(ctx context.Context, id int64) (*domain.Plan, error)
}

// VoucherHandler serves the Platform Console's voucher management
// (/api/v1/vouchers, platform_admin only; writes need the Billing role,
// cmd/server/platform_scope.go) and a WO Owner's preview of a code before
// paying (POST /api/v1/subscriptions/voucher-check).
type VoucherHandler struct {
	vouchers voucherManager
	plans    tenantPlanReader
	apps     paymentcontracts.AppRegistry
}

func NewVoucherHandler(vouchers voucherManager, plans tenantPlanReader, apps paymentcontracts.AppRegistry) *VoucherHandler {
	return &VoucherHandler{vouchers: vouchers, plans: plans, apps: apps}
}

type voucherResponse struct {
	ID              int64   `json:"id"`
	Code            string  `json:"code"`
	AppID           string  `json:"appId"`
	PlanID          *int64  `json:"planId"`
	PlanName        *string `json:"planName"`
	DiscountType    string  `json:"discountType"`
	DiscountValue   int64   `json:"discountValue"`
	MaxDiscount     *int64  `json:"maxDiscount"`
	ValidFrom       *string `json:"validFrom"`
	ValidUntil      *string `json:"validUntil"`
	Quota           *int64  `json:"quota"`
	OncePerCustomer bool    `json:"oncePerCustomer"`
	IsActive        bool    `json:"isActive"`
	// State is the voucher's standing today (WIB): active, inactive,
	// scheduled, expired or exhausted.
	State         string `json:"state"`
	UsedCount     int64  `json:"usedCount"`
	ReservedCount int64  `json:"reservedCount"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toVoucherResponse(v domain.Voucher, today string) voucherResponse {
	out := voucherResponse{
		ID: v.ID, Code: v.Code, AppID: v.AppID, PlanID: v.PlanID,
		DiscountType: string(v.Type), DiscountValue: v.Value, MaxDiscount: v.MaxDiscount,
		ValidFrom: optString(v.ValidFrom), ValidUntil: optString(v.ValidUntil), Quota: v.Quota,
		OncePerCustomer: v.OncePerCustomer, IsActive: v.IsActive, State: string(v.StateOn(today)),
		UsedCount: v.UsedCount, ReservedCount: v.ReservedCount,
		CreatedAt: v.CreatedAt.In(wib).Format(timeLayout), UpdatedAt: etag.Format(v.UpdatedAt),
	}
	if v.PlanID != nil {
		out.PlanName = optString(v.PlanName)
	}
	return out
}

type voucherBody struct {
	Code            string `json:"code"`
	AppID           string `json:"appId"`
	PlanID          *int64 `json:"planId"`
	DiscountType    string `json:"discountType"`
	DiscountValue   int64  `json:"discountValue"`
	MaxDiscount     *int64 `json:"maxDiscount"`
	ValidFrom       string `json:"validFrom"`
	ValidUntil      string `json:"validUntil"`
	Quota           *int64 `json:"quota"`
	OncePerCustomer bool   `json:"oncePerCustomer"`
}

func (b voucherBody) input() application.VoucherInput {
	return application.VoucherInput{
		Code: b.Code, AppID: b.AppID, PlanID: b.PlanID, Type: domain.DiscountType(b.DiscountType), Value: b.DiscountValue,
		MaxDiscount: b.MaxDiscount, ValidFrom: b.ValidFrom, ValidUntil: b.ValidUntil, Quota: b.Quota, OncePerCustomer: b.OncePerCustomer,
	}
}

// Collection is /api/v1/vouchers: GET lists, POST creates.
func (h *VoucherHandler) Collection(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		params := pagination.FromRequest(r)
		q := r.URL.Query()
		list, total, err := h.vouchers.List(r.Context(), params, domain.VoucherFilter{Search: q.Get("search"), AppID: q.Get("appId")})
		if err != nil {
			writeAppError(w, err)
			return
		}
		today := h.vouchers.Today()
		out := make([]voucherResponse, 0, len(list))
		for _, v := range list {
			out = append(out, toVoucherResponse(v, today))
		}
		response.OKPaginated(w, "ok", out, pagination.BuildMeta(params, total))
	case http.MethodPost:
		var body voucherBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		if !requireKnownApp(w, r, h.apps, body.AppID) {
			return
		}
		v, err := h.vouchers.Create(r.Context(), body.input())
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.Created(w, "Voucher berhasil dibuat", toVoucherResponse(*v, h.vouchers.Today()))
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

// Item is /api/v1/vouchers/{id} (GET, PATCH) and /api/v1/vouchers/{id}/redemptions (GET).
func (h *VoucherHandler) Item(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/vouchers/")
	if len(segments) == 0 || len(segments) > 2 {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID voucher tidak valid", nil)
		return
	}
	switch {
	case len(segments) == 2 && segments[1] == "redemptions" && r.Method == http.MethodGet:
		h.redemptions(w, r, id)
	case len(segments) == 1 && r.Method == http.MethodGet:
		v, err := h.vouchers.Get(r.Context(), id)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "ok", toVoucherResponse(*v, h.vouchers.Today()))
	case len(segments) == 1 && r.Method == http.MethodPatch:
		h.patch(w, r, id)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// patch: {"isActive": true|false} alone is the explicit state (ADR-0036);
// any other body is the full edit, guarded by If-Match: <updatedAt>.
func (h *VoucherHandler) patch(w http.ResponseWriter, r *http.Request, id int64) {
	sp, err := httpx.ReadStatePatch(r, "isActive")
	if err != nil {
		httpx.WriteStatePatchError(w, err)
		return
	}
	today := h.vouchers.Today()
	if sp.IsState {
		v, err := h.vouchers.SetActive(r.Context(), id, sp.Value)
		if err != nil {
			writeAppError(w, err)
			return
		}
		msg := "Voucher dinonaktifkan"
		if sp.Value {
			msg = "Voucher diaktifkan"
		}
		response.OK(w, msg, toVoucherResponse(*v, today))
		return
	}
	expected, err := etag.ParseIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "If-Match tidak valid", nil)
		return
	}
	var body voucherBody
	if err := json.Unmarshal(sp.Body, &body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	if !requireKnownApp(w, r, h.apps, body.AppID) {
		return
	}
	v, err := h.vouchers.Update(r.Context(), id, body.input(), expected)
	if err != nil {
		if current, ok := staleVoucher(err); ok {
			response.ErrorWithData(w, http.StatusConflict, "Data ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru.",
				map[string][]string{"_conflict": {"stale"}}, toVoucherResponse(*current, today))
			return
		}
		writeAppError(w, err)
		return
	}
	response.OK(w, "Voucher berhasil diperbarui", toVoucherResponse(*v, today))
}

type redemptionResponse struct {
	ID             int64   `json:"id"`
	OrderRef       string  `json:"orderRef"`
	CustomerType   string  `json:"customerType"`
	CustomerRef    string  `json:"customerRef"`
	CustomerName   string  `json:"customerName"`
	PlanID         int64   `json:"planId"`
	OriginalAmount int64   `json:"originalAmount"`
	DiscountAmount int64   `json:"discountAmount"`
	FinalAmount    int64   `json:"finalAmount"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	ResolvedAt     *string `json:"resolvedAt"`
}

func (h *VoucherHandler) redemptions(w http.ResponseWriter, r *http.Request, id int64) {
	params := pagination.FromRequest(r)
	rows, total, err := h.vouchers.ListRedemptions(r.Context(), id, params)
	if err != nil {
		writeAppError(w, err)
		return
	}
	out := make([]redemptionResponse, 0, len(rows))
	for _, x := range rows {
		item := redemptionResponse{
			ID: x.ID, OrderRef: x.OrderRef, CustomerType: string(x.Customer.Type), CustomerRef: x.Customer.Ref,
			CustomerName: x.CustomerName, PlanID: x.PlanID, OriginalAmount: x.OriginalAmount, DiscountAmount: x.DiscountAmount,
			FinalAmount: x.FinalAmount, Status: string(x.Status), CreatedAt: x.CreatedAt.In(wib).Format(timeLayout),
		}
		if x.ResolvedAt != nil {
			s := x.ResolvedAt.In(wib).Format(timeLayout)
			item.ResolvedAt = &s
		}
		out = append(out, item)
	}
	response.OKPaginated(w, "ok", out, pagination.BuildMeta(params, total))
}

// ---- preview of a code before paying ----

type voucherCheckBody struct {
	PlanID      int64  `json:"planId"`
	VoucherCode string `json:"voucherCode"`
}

type voucherQuoteResponse struct {
	VoucherCode    string `json:"voucherCode"`
	PlanID         int64  `json:"planId"`
	OriginalAmount int64  `json:"originalAmount"`
	DiscountAmount int64  `json:"discountAmount"`
	FinalAmount    int64  `json:"finalAmount"`
}

func toQuoteResponse(q application.VoucherQuote) voucherQuoteResponse {
	return voucherQuoteResponse{VoucherCode: q.Code, PlanID: q.PlanID, OriginalAmount: q.OriginalAmount, DiscountAmount: q.DiscountAmount, FinalAmount: q.FinalAmount}
}

// TenantCheck is POST /api/v1/subscriptions/voucher-check: a WO Owner (the
// subscription.manage permission, same as paying) previews what a code does
// to one of ElProof's plans. Nothing is reserved; paying checks again.
func (h *VoucherHandler) TenantCheck(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" || !middleware.GrantsFrom(r.Context()).Can("subscription.manage") {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak dapat mengelola langganan", map[string]string{"code": "forbidden_permission"})
		return
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return
	}
	var body voucherCheckBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	plan, err := h.plans.GetTenantPlan(r.Context(), body.PlanID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	q, err := h.vouchers.Quote(r.Context(), body.VoucherCode, plan,
		domain.Customer{Type: domain.CustomerTenant, Ref: strconv.FormatInt(tenantID, 10)})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Voucher dapat dipakai", toQuoteResponse(*q))
}

// staleVoucher unwraps the 409 of a stale If-Match (its payload is the
// voucher as it is now).
func staleVoucher(err error) (*domain.Voucher, bool) {
	data, ok := apperror.IsStale(err)
	if !ok {
		return nil, false
	}
	v, ok := data.(*domain.Voucher)
	return v, ok && v != nil
}
