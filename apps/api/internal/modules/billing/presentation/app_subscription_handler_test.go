package presentation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elproof/internal/shared/middleware"
)

func TestAppSubscriptions_PlatformAdminOnly(t *testing.T) {
	h := NewAppSubscriptionHandler(&stubAppSubscriptionLister{})

	t.Run("staff", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/app-subscriptions", nil)
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), staffClaims()))
		w := httptest.NewRecorder()

		h.List(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("platform_admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/app-subscriptions", nil)
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
		w := httptest.NewRecorder()

		h.List(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}

func TestAppSubscriptions_KosongJadiArray(t *testing.T) {
	h := NewAppSubscriptionHandler(&stubAppSubscriptionLister{summaries: nil})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-subscriptions", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (bukan galat)", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("data harus [] bukan null: %s", w.Body.String())
	}
}
