package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/response"
)

// subscriptionCharger is the narrow slice of *application.SubscriptionService
// this handler needs — declared here (not the concrete struct) so a stub
// can be injected for tests without a database or a real payment.Client
// (A34).
type subscriptionCharger interface {
	StartSubscriptionCharge(ctx context.Context, appID, orderRef string, planID int64, opts paymentcontracts.ChargeOptions) (*paymentcontracts.ChargeResult, error)
}

// ExternalSubscriptionHandler serves POST /api/v1/external/subscriptions/charges
// (D9) — the correct way for an external App to purchase/renew an ElProof
// subscription, as opposed to /external/payments/charges (general-purpose
// charges, amount supplied by the caller). This handler is deliberately
// thin: guard, parse, map errors — the actual business rules (plan
// ownership, price-from-catalog, write-before-call) live in
// SubscriptionService so they're testable with stubs (A34/§7.1).
type ExternalSubscriptionHandler struct {
	subs subscriptionCharger
	apps paymentcontracts.AppRegistry
}

func NewExternalSubscriptionHandler(subs subscriptionCharger, apps paymentcontracts.AppRegistry) *ExternalSubscriptionHandler {
	return &ExternalSubscriptionHandler{subs: subs, apps: apps}
}

type externalSubscriptionChargeBody struct {
	OrderRef      string `json:"orderRef"`
	PlanID        int64  `json:"planId"`
	CustomerName  string `json:"customerName"`
	CustomerEmail string `json:"customerEmail"`
	CustomerPhone string `json:"customerPhone"`
}

// externalSubscriptionChargeResponse mirrors payment's own
// toExternalChargeResponse field-for-field (external_handler.go:54-65) so a
// consumer's existing charge parser works unchanged for a subscription
// charge (task 8.5) — deliberately NOT imported from `payment/presentation`
// (forbidden by the module-boundary rule), so it's a local duplicate.
type externalSubscriptionChargeResponse struct {
	OrderRef    string `json:"orderRef"`
	ProviderRef string `json:"providerRef"`
	Channel     string `json:"channel"`
	QRImageURL  string `json:"qrImageUrl,omitempty"`
	PayCode     string `json:"payCode,omitempty"`
	CheckoutURL string `json:"checkoutUrl,omitempty"`
	Amount      int64  `json:"amount"`
	FeeAmount   int64  `json:"feeAmount"`
	ExpiresAt   string `json:"expiresAt"`
	Status      string `json:"status"`
}

func toExternalSubscriptionChargeResponse(c *paymentcontracts.ChargeResult) externalSubscriptionChargeResponse {
	return externalSubscriptionChargeResponse{
		OrderRef: c.OrderRef, ProviderRef: c.ProviderRef, Channel: c.Channel,
		QRImageURL: c.QRImageURL, PayCode: c.PayCode, CheckoutURL: c.CheckoutURL,
		Amount: c.Amount, FeeAmount: c.FeeAmount, ExpiresAt: c.ExpiresAt.Format(time.RFC3339), Status: c.Status,
	}
}

// CreateCharge is POST /api/v1/external/subscriptions/charges. Not wrapped
// in httpx.Method (T17/A24), same reasoning as ExternalPlanHandler.List —
// the method check happens here so a 405 still carries {code:"bad_request"}.
func (h *ExternalSubscriptionHandler) CreateCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeExternalError(w, http.StatusMethodNotAllowed, "bad_request", "Metode HTTP tidak diizinkan untuk endpoint ini")
		return
	}

	appID, ok := requireActiveApp(w, r, h.apps)
	if !ok {
		return
	}

	var body externalSubscriptionChargeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeExternalError(w, http.StatusBadRequest, "bad_request", "Body permintaan tidak valid")
		return
	}
	if body.OrderRef == "" || body.PlanID == 0 {
		writeExternalError(w, http.StatusBadRequest, "bad_request", "orderRef dan planId wajib diisi")
		return
	}
	// No `amount` field is read from body at all (A18) — the price comes
	// from the catalog inside SubscriptionService, so a caller can never
	// determine what they pay for a plan.

	result, err := h.subs.StartSubscriptionCharge(r.Context(), appID, body.OrderRef, body.PlanID, paymentcontracts.ChargeOptions{
		CustomerName: body.CustomerName, CustomerEmail: body.CustomerEmail, CustomerPhone: body.CustomerPhone,
	})
	if err != nil {
		// Plan-not-mapped-to-this-App and orderRef-already-used both arrive
		// here as apperror (KindValidation -> 422 "bad_request", A18/8.4;
		// KindConflict -> 409 "conflict") — writeExternalAppError already
		// carries the same {code} vocabulary as every other external route.
		writeExternalAppError(w, err)
		return
	}
	response.Created(w, "ok", toExternalSubscriptionChargeResponse(result))
}
