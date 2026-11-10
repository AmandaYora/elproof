package presentation

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/billing/application"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/middleware"
)

// NOTE: TestSubCharge_PaketTidakDipetakanJadi422, TestSubCharge_
// NominalDariKatalogBukanBody, TestSubCharge_UrutanIntentSebelumCharge, and
// TestSubCharge_ChargeGagalMenghapusIntent live in
// application/subscription_service_test.go instead of here — what they
// assert (ListPlanIDsForApp/Create/CreateChannelCharge/Delete call order
// and arguments) are SubscriptionService's own collaborators, invisible
// from this handler's subscriptionCharger interface (A34 deliberately
// hides them so the handler stays thin). Only the two tests below are
// genuinely about this handler's own HTTP-mapping behavior.

func TestSubCharge_OrderRefDuplikatJadi409(t *testing.T) {
	subs := &stubSubscriptionCharger{err: apperror.Conflict(`order_ref "ORD-1" sudah pernah dipakai`)}
	h := NewExternalSubscriptionHandler(subs, &stubAppRegistry{isActive: true})
	body := bytes.NewBufferString(`{"orderRef":"ORD-1","planId":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/external/subscriptions/charges", body)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_x")))
	w := httptest.NewRecorder()

	h.CreateCharge(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"code":"conflict"`) {
		t.Fatalf("body = %s, want code:conflict", w.Body.String())
	}
}

func TestSubCharge_BentukResponseSamaDenganChargeLama(t *testing.T) {
	expires := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	subs := &stubSubscriptionCharger{result: &paymentcontracts.ChargeResult{
		OrderRef: "ORD-1", ProviderRef: "PROV-1", Channel: "QRIS",
		QRImageURL: "https://example/qr.png", Amount: 2000000, FeeAmount: 1000,
		ExpiresAt: expires, Status: "unpaid",
	}}
	h := NewExternalSubscriptionHandler(subs, &stubAppRegistry{isActive: true})
	body := bytes.NewBufferString(`{"orderRef":"ORD-1","planId":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/external/subscriptions/charges", body)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_x")))
	w := httptest.NewRecorder()

	h.CreateCharge(w, req)

	for _, field := range []string{"orderRef", "providerRef", "channel", "qrImageUrl", "amount", "feeAmount", "expiresAt", "status"} {
		if !strings.Contains(w.Body.String(), `"`+field+`"`) {
			t.Errorf("field %q hilang dari response: %s", field, w.Body.String())
		}
	}
}

func TestExternalSubCharge_VoucherDiteruskanDanNominalTercatat(t *testing.T) {
	subs := &stubSubscriptionCharger{discount: 300000, result: &paymentcontracts.ChargeResult{
		OrderRef: "ORD-1", Amount: 1200000, ExpiresAt: time.Now(), Status: "unpaid",
	}}
	h := NewExternalSubscriptionHandler(subs, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/external/subscriptions/charges",
		bytes.NewBufferString(`{"orderRef":"ORD-1","planId":9,"voucherCode":"JWS20","amount":1}`))
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_jws")))
	w := httptest.NewRecorder()
	h.CreateCharge(w, req)
	if w.Code != http.StatusCreated || subs.lastVoucher != "JWS20" || subs.lastAppID != "app_jws" {
		t.Fatalf("status %d voucher %q app %q: %s", w.Code, subs.lastVoucher, subs.lastAppID, w.Body.String())
	}
	for _, want := range []string{`"amount":1200000`, `"originalAmount":1500000`, `"discountAmount":300000`, `"voucherCode":"JWS20"`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("response lacks %s: %s", want, w.Body.String())
		}
	}
}

func TestExternalVoucherCheck(t *testing.T) {
	refused := apperror.Validation("Voucher ini sudah kedaluwarsa.", map[string][]string{"voucherCode": {"x"}}).WithCode("voucher_expired")
	for _, tc := range []struct {
		name   string
		subs   *stubSubscriptionCharger
		body   string
		status int
		want   []string
	}{
		{"berlaku", &stubSubscriptionCharger{quote: &application.VoucherQuote{Code: "JWS20", PlanID: 9, OriginalAmount: 1500000, DiscountAmount: 300000, FinalAmount: 1200000}},
			`{"planId":9,"voucherCode":"jws20"}`, http.StatusOK, []string{`"finalAmount":1200000`, `"voucherCode":"JWS20"`}},
		{"ditolak dengan alasan", &stubSubscriptionCharger{err: refused}, `{"planId":9,"voucherCode":"LAMA"}`, http.StatusUnprocessableEntity,
			[]string{`"code":"bad_request"`, `"reason":"voucher_expired"`, `kedaluwarsa`}},
		{"tanpa kode", &stubSubscriptionCharger{}, `{"planId":9}`, http.StatusBadRequest, []string{`"code":"bad_request"`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewExternalSubscriptionHandler(tc.subs, &stubAppRegistry{isActive: true})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/external/subscriptions/voucher-check", bytes.NewBufferString(tc.body))
			req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_jws")))
			w := httptest.NewRecorder()
			h.VoucherCheck(w, req)
			if w.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
			for _, s := range tc.want {
				if !strings.Contains(w.Body.String(), s) {
					t.Errorf("body lacks %s: %s", s, w.Body.String())
				}
			}
		})
	}
	// Not an App token -> 403, and a WO staff can never reach another App's codes.
	h := NewExternalSubscriptionHandler(&stubSubscriptionCharger{}, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/external/subscriptions/voucher-check", bytes.NewBufferString(`{"planId":9,"voucherCode":"X"}`))
	w := httptest.NewRecorder()
	h.VoucherCheck(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("no app principal: %d", w.Code)
	}
}
