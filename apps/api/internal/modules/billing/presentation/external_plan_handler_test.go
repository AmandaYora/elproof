package presentation

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/middleware"
)

func appClaims(appID string) *middleware.Claims {
	return &middleware.Claims{PrincipalType: "app", PrincipalID: appID}
}

func TestExternalPlans_PrincipalBukanApp(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{}, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/subscription-plans", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), &middleware.Claims{PrincipalType: "staff", PrincipalID: "1"}))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if !strings.Contains(w.Body.String(), `"code":"forbidden"`) {
		t.Fatalf("body = %s, want code:forbidden", w.Body.String())
	}
}

func TestExternalPlans_TanpaKlaim(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{}, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/subscription-plans", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestExternalPlans_AppDinonaktifkan(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{}, &stubAppRegistry{isActive: false})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/subscription-plans", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_x")))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestExternalPlans_GalatRegistryBukan403(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{}, &stubAppRegistry{isActiveErr: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/subscription-plans", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_x")))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (bukan 403)", w.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(w.Body.String(), `"code":"internal"`) {
		t.Fatalf("body = %s, want code:internal", w.Body.String())
	}
}

func TestExternalPlans_MethodSelainGet(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{}, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/external/subscription-plans", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_x")))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	// Ini yang mengunci A24: kalau rutenya dibungkus httpx.Method,
	// {code} akan hilang dari body (T17).
	if !strings.Contains(w.Body.String(), `"code":"bad_request"`) {
		t.Fatalf("body = %s, want code:bad_request", w.Body.String())
	}
}

func TestExternalPlans_BentukFieldHarfiah(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{plans: []domain.Plan{
		{ID: 1, Name: "Paket JWS", DurationMonths: 12, Price: 2000000, IsActive: true, Features: []string{"VPS Regular"}},
	}}, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/subscription-plans", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_x")))
	w := httptest.NewRecorder()

	h.List(w, req)

	var body struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("gagal decode body: %v (%s)", err, w.Body.String())
	}
	if len(body.Data) != 1 {
		t.Fatalf("data length = %d, want 1", len(body.Data))
	}
	item := body.Data[0]
	for _, field := range []string{"id", "name", "durationMonths", "price", "active", "features"} {
		if _, ok := item[field]; !ok {
			t.Errorf("field %q hilang dari response: %v", field, item)
		}
	}
	if _, ok := item["isActive"]; ok {
		t.Errorf("field isActive TIDAK boleh ada di kontrak eksternal (T3), tapi ada: %v", item)
	}
}

func TestExternalPlans_FeaturesKosongJadiArray(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{plans: []domain.Plan{
		{ID: 1, Name: "Tanpa Fitur", DurationMonths: 12, Price: 1000000, IsActive: true, Features: nil},
	}}, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/subscription-plans", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_x")))
	w := httptest.NewRecorder()

	h.List(w, req)

	if strings.Contains(w.Body.String(), `"features":null`) {
		t.Fatalf("features harus [] bukan null: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"features":[]`) {
		t.Fatalf("features harus [] eksplisit: %s", w.Body.String())
	}
}

func TestExternalPlans_TanpaPemetaanKembali200Kosong(t *testing.T) {
	h := NewExternalPlanHandler(&stubExternalPlanLister{plans: nil}, &stubAppRegistry{isActive: true})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/external/subscription-plans", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), appClaims("app_elkasir")))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (bukan 404) — R2/A4", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("data harus [] bukan null: %s", w.Body.String())
	}
}
