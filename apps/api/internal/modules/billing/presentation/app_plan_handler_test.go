package presentation

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"elproof/internal/shared/middleware"
)

func TestAppPlans_BukanPlatformAdmin(t *testing.T) {
	apps := &stubAppRegistry{exists: true}
	mapper := &stubAppPlanMapper{}
	h := NewAppPlanHandler(mapper, apps)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-plans?appId=app_x", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), staffClaims()))
	w := httptest.NewRecorder()

	h.Collection(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	// Guard harus memutus SEBELUM query apa pun (7.2) — penelepon tak
	// berwenang tidak boleh memicu AppExists.
	if apps.existsCalled {
		t.Fatal("AppExists TIDAK boleh dipanggil untuk principal non-platform_admin")
	}
}

func TestAppPlans_AppIdTidakAdaJadi422(t *testing.T) {
	apps := &stubAppRegistry{exists: false}
	mapper := &stubAppPlanMapper{}
	h := NewAppPlanHandler(mapper, apps)
	body := bytes.NewBufferString(`{"appId":"app_tidak_ada","planIds":[1]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/app-plans", body)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
	w := httptest.NewRecorder()

	h.Collection(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", w.Code)
	}
	if mapper.setCalled {
		t.Fatal("SetAppPlans TIDAK boleh dipanggil kalau appId tidak ada (A11)")
	}
}

func TestAppPlans_GalatRegistryBukan422(t *testing.T) {
	apps := &stubAppRegistry{existsErr: errors.New("db down")}
	mapper := &stubAppPlanMapper{}
	h := NewAppPlanHandler(mapper, apps)
	body := bytes.NewBufferString(`{"appId":"app_x","planIds":[1]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/app-plans", body)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
	w := httptest.NewRecorder()

	h.Collection(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (bukan 422)", w.Code)
	}
}

func TestAppPlans_AppNonaktifTetapBolehDipetakan(t *testing.T) {
	apps := &stubAppRegistry{exists: true, isActive: false}
	mapper := &stubAppPlanMapper{}
	h := NewAppPlanHandler(mapper, apps)
	body := bytes.NewBufferString(`{"appId":"app_x","planIds":[1]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/app-plans", body)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
	w := httptest.NewRecorder()

	h.Collection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !apps.existsCalled {
		t.Fatal("harus memanggil AppExists")
	}
	if apps.isActiveCalled {
		t.Fatal("TIDAK boleh memanggil IsAppActive — App nonaktif tetap boleh dipetakan (A11)")
	}
}

func TestAppPlans_PlanIdsKosongSah(t *testing.T) {
	apps := &stubAppRegistry{exists: true}
	mapper := &stubAppPlanMapper{}
	h := NewAppPlanHandler(mapper, apps)
	body := bytes.NewBufferString(`{"appId":"app_x","planIds":[]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/app-plans", body)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
	w := httptest.NewRecorder()

	h.Collection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — planIds:[] adalah cabut semua pemetaan, sah", w.Code)
	}
	if !mapper.setCalled || len(mapper.lastSetIDs) != 0 {
		t.Fatalf("SetAppPlans harus dipanggil dengan slice kosong, got %v (called=%v)", mapper.lastSetIDs, mapper.setCalled)
	}
}
