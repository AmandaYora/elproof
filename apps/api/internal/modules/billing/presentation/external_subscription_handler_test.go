package presentation

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
