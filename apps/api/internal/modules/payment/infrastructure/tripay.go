package infrastructure

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/payment/domain"
)

// tripayGateway implements `gateway` against the real Tripay API.
//
// NOTE for whoever wires this up with real credentials: the base URL
// (https://tripay.co.id/api-sandbox), the `/merchant/payment-channel` path,
// and the `Authorization: Bearer <api_key>` scheme were all confirmed live
// against Tripay's real sandbox during development — an unauthenticated/
// invalid-key request correctly returned Tripay's own
// `{"success":false,"message":"Invalid API Key"}` shape, matching this
// file's response structs exactly. What was NOT verified (no real merchant
// credentials were available in that environment): `/transaction/create`'s
// full request/response shape, `/transaction/detail`, and the webhook
// callback signature/body shape — those match Tripay's publicly documented
// Closed Payment API as of this module's authoring, but double-check them
// against https://tripay.co.id/developer once real sandbox credentials are
// in hand (see PLAN.md Fase 9 notes).
type tripayGateway struct {
	httpClient    *http.Client
	baseURL       string
	merchantCode  string
	apiKey        string
	privateKey    string
	defaultExpiry time.Duration
}

const (
	tripaySandboxBaseURL = "https://tripay.co.id/api-sandbox"
	tripayProdBaseURL    = "https://tripay.co.id/api"
)

func newTripayGateway(merchantCode, apiKey, privateKey string, sandbox bool) *tripayGateway {
	base := tripayProdBaseURL
	if sandbox {
		base = tripaySandboxBaseURL
	}
	return &tripayGateway{
		httpClient:    &http.Client{Timeout: 20 * time.Second},
		baseURL:       base,
		merchantCode:  merchantCode,
		apiKey:        apiKey,
		privateKey:    privateKey,
		defaultExpiry: 24 * time.Hour,
	}
}

func (g *tripayGateway) Name() string { return "tripay" }

func (g *tripayGateway) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tripay: request gagal: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tripay: status %d — %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

type tripayChannelResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Group       string `json:"group"`
		FeeMerchant struct {
			Flat    int64   `json:"flat"`
			Percent float64 `json:"percent"`
		} `json:"fee_merchant"`
		FeeCustomer struct {
			Flat    int64   `json:"flat"`
			Percent float64 `json:"percent"`
		} `json:"fee_customer"`
		IconURL string `json:"icon_url"`
		Active  bool   `json:"active"`
	} `json:"data"`
}

func (g *tripayGateway) ListChannels(ctx context.Context) ([]domain.Channel, error) {
	raw, err := g.doRequest(ctx, http.MethodGet, "/merchant/payment-channel", nil)
	if err != nil {
		return nil, err
	}
	var parsed tripayChannelResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("tripay: parse channel list gagal: %w", err)
	}
	channels := make([]domain.Channel, 0, len(parsed.Data))
	for _, c := range parsed.Data {
		channels = append(channels, domain.Channel{
			Code:               c.Code,
			Name:               c.Name,
			Type:               c.Group,
			FeeMerchantFlat:    c.FeeMerchant.Flat,
			FeeMerchantPercent: c.FeeMerchant.Percent,
			FeeCustomerFlat:    c.FeeCustomer.Flat,
			FeeCustomerPercent: c.FeeCustomer.Percent,
			IconURL:            c.IconURL,
			Active:             c.Active,
		})
	}
	return channels, nil
}

type tripayCreateTransactionRequest struct {
	Method        string                       `json:"method"`
	MerchantRef   string                       `json:"merchant_ref"`
	Amount        int64                        `json:"amount"`
	CustomerName  string                       `json:"customer_name"`
	CustomerEmail string                       `json:"customer_email"`
	CustomerPhone string                       `json:"customer_phone"`
	OrderItems    []tripayOrderItem            `json:"order_items"`
	CallbackURL   string                       `json:"callback_url"`
	ExpiredTime   int64                        `json:"expired_time"`
	Signature     string                       `json:"signature"`
}

type tripayOrderItem struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
}

type tripayTransactionData struct {
	Reference   string `json:"reference"`
	MerchantRef string `json:"merchant_ref"`
	PaymentName string `json:"payment_name"`
	Amount      int64  `json:"amount"`
	FeeMerchant int64  `json:"fee_merchant"`
	PayCode     string `json:"pay_code"`
	CheckoutURL string `json:"checkout_url"`
	QRURL       string `json:"qr_url"`
	Status      string `json:"status"`
	ExpiredTime int64  `json:"expired_time"`
}

type tripayCreateTransactionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    tripayTransactionData  `json:"data"`
}

func (g *tripayGateway) signCreate(merchantRef string, amount int64) string {
	mac := hmac.New(sha256.New, []byte(g.privateKey))
	mac.Write([]byte(g.merchantCode + merchantRef + strconv.FormatInt(amount, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (g *tripayGateway) CreateCharge(ctx context.Context, req ChargeRequest) (*domain.Charge, error) {
	expiresIn := g.defaultExpiry
	if req.ExpiresInSecs > 0 {
		expiresIn = time.Duration(req.ExpiresInSecs) * time.Second
	}

	body := tripayCreateTransactionRequest{
		Method:        req.Channel,
		MerchantRef:   req.OrderRef,
		Amount:        req.Amount,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		CustomerPhone: req.CustomerPhone,
		OrderItems: []tripayOrderItem{
			{SKU: req.OrderRef, Name: "Pembayaran " + req.OrderRef, Price: req.Amount, Quantity: 1},
		},
		CallbackURL: req.CallbackURL,
		ExpiredTime: time.Now().Add(expiresIn).Unix(),
		Signature:   g.signCreate(req.OrderRef, req.Amount),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	raw, err := g.doRequest(ctx, http.MethodPost, "/transaction/create", payload)
	if err != nil {
		return nil, err
	}
	var parsed tripayCreateTransactionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("tripay: parse create-transaction response gagal: %w", err)
	}
	if !parsed.Success {
		return nil, fmt.Errorf("tripay: %s", parsed.Message)
	}
	return toDomainCharge(parsed.Data), nil
}

func toDomainCharge(d tripayTransactionData) *domain.Charge {
	return &domain.Charge{
		OrderRef:    d.MerchantRef,
		ProviderRef: d.Reference,
		Channel:     d.PaymentName,
		QRImageURL:  d.QRURL,
		PayCode:     d.PayCode,
		CheckoutURL: d.CheckoutURL,
		Amount:      d.Amount,
		FeeAmount:   d.FeeMerchant,
		ExpiresAt:   time.Unix(d.ExpiredTime, 0),
		Status:      tripayStatusToDomain(d.Status),
	}
}

func tripayStatusToDomain(status string) domain.ChargeStatus {
	switch status {
	case "PAID":
		return domain.ChargePaid
	case "EXPIRED":
		return domain.ChargeExpired
	case "FAILED":
		return domain.ChargeFailed
	case "REFUND":
		return domain.ChargeRefund
	default:
		return domain.ChargeUnpaid
	}
}

type tripayDetailResponse struct {
	Success bool                  `json:"success"`
	Data    tripayTransactionData `json:"data"`
}

func (g *tripayGateway) GetChargeStatus(ctx context.Context, providerRef string) (*domain.Charge, error) {
	raw, err := g.doRequest(ctx, http.MethodGet, "/transaction/detail?reference="+providerRef, nil)
	if err != nil {
		return nil, err
	}
	var parsed tripayDetailResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("tripay: parse transaction detail gagal: %w", err)
	}
	if !parsed.Success {
		return nil, errors.New("tripay: transaksi tidak ditemukan")
	}
	return toDomainCharge(parsed.Data), nil
}

type tripayCallbackBody struct {
	Reference   string `json:"reference"`
	MerchantRef string `json:"merchant_ref"`
	Status      string `json:"status"`
	TotalAmount int64  `json:"total_amount"`
	PaidAt      int64  `json:"paid_at"`
}

// VerifyAndParseWebhook checks `X-Callback-Signature` — HMAC-SHA256 (hex) of
// the raw request body, keyed with the merchant's private key — using a
// constant-time comparison (see MODULE_PAYMENT.md §8), then parses the body.
func (g *tripayGateway) VerifyAndParseWebhook(signature string, rawBody []byte) (*domain.WebhookEvent, string, error) {
	mac := hmac.New(sha256.New, []byte(g.privateKey))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, "", errors.New("tripay: signature webhook tidak valid")
	}

	var body tripayCallbackBody
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return nil, "", fmt.Errorf("tripay: parse body webhook gagal: %w", err)
	}

	var paidAt time.Time
	if body.PaidAt > 0 {
		paidAt = time.Unix(body.PaidAt, 0)
	}
	event := &domain.WebhookEvent{
		ProviderRef: body.Reference,
		OrderRef:    body.MerchantRef,
		Status:      tripayStatusToDomain(body.Status),
		Amount:      body.TotalAmount,
		PaidAt:      paidAt,
	}
	// Tripay doesn't send a dedicated event id — its own transaction
	// reference plus status is a stable enough dedupe key for our webhook
	// idempotency log (a second delivery of the same status is a no-op;
	// Tripay does not re-deliver a transitioned-away status for the same ref).
	eventID := body.Reference + ":" + body.Status
	return event, eventID, nil
}
