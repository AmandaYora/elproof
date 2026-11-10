package presentation

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/middleware"
)

func platformAdminClaims() *middleware.Claims {
	return &middleware.Claims{PrincipalType: "platform_admin", PrincipalID: "1"}
}

func staffClaims() *middleware.Claims {
	return &middleware.Claims{PrincipalType: "staff", PrincipalID: "1", TenantID: "1"}
}

func serveList(stub *stubPlanLister, claims *middleware.Claims, url string) *httptest.ResponseRecorder {
	h := NewPlanHandler(stub, &stubAppRegistry{})
	req := httptest.NewRequest(http.MethodGet, url, nil)
	if claims != nil {
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), claims))
	}
	w := httptest.NewRecorder()
	h.Collection(w, req)
	return w
}

// TestPlansList_CakupanAplikasi locks ADR-0040: platform_admin lists every
// App's plans (or one App's with ?appId=); every other principal — and no
// claim at all — is pinned to ElProof's own catalog, even when it sends
// ?appId= itself. Checked on BOTH the all=true and paginated branches,
// because a leak through either door is still a leak.
func TestPlansList_CakupanAplikasi(t *testing.T) {
	cases := []struct {
		name   string
		claims *middleware.Claims
		url    string
		want   string
	}{
		{"all, admin tanpa saring", platformAdminClaims(), "/api/v1/plans?all=true", ""},
		{"all, admin saring App", platformAdminClaims(), "/api/v1/plans?all=true&appId=app_jws", "app_jws"},
		{"all, staff", staffClaims(), "/api/v1/plans?all=true", application.TenantPlansApp},
		{"all, staff mencoba appId lain", staffClaims(), "/api/v1/plans?all=true&appId=app_jws", application.TenantPlansApp},
		{"all, tanpa klaim", nil, "/api/v1/plans?all=true&appId=app_jws", application.TenantPlansApp},
		{"paginasi, admin saring App", platformAdminClaims(), "/api/v1/plans?appId=app_jws", "app_jws"},
		{"paginasi, staff mencoba appId lain", staffClaims(), "/api/v1/plans?appId=app_jws", application.TenantPlansApp},
		{"paginasi, tanpa klaim", nil, "/api/v1/plans", application.TenantPlansApp},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubPlanLister{}
			w := serveList(stub, tc.claims, tc.url)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
			}
			if !stub.calledList && !stub.calledListPaginated {
				t.Fatal("tidak ada kueri daftar")
			}
			if strings.Contains(tc.url, "all=true") != stub.calledList {
				t.Fatalf("cabang salah: all=%v calledList=%v", strings.Contains(tc.url, "all=true"), stub.calledList)
			}
			if stub.lastAppID != tc.want {
				t.Fatalf("appID = %q, want %q", stub.lastAppID, tc.want)
			}
		})
	}
}

func TestPlansList_AppIdTerkirim(t *testing.T) {
	stub := &stubPlanLister{plans: []domain.Plan{
		{ID: 1, AppID: "app_02ef90c52704", Name: "JWS"},
		{ID: 2, AppID: application.TenantPlansApp, Name: "ElProof"},
	}}
	w := serveList(stub, platformAdminClaims(), "/api/v1/plans?all=true")

	var body struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("gagal decode: %v (%s)", err, w.Body.String())
	}
	if len(body.Data) != 2 {
		t.Fatalf("data length = %d, want 2", len(body.Data))
	}
	if string(body.Data[0]["appId"]) != `"app_02ef90c52704"` || string(body.Data[1]["appId"]) != `"platform-billing"` {
		t.Errorf("appId = %s / %s", body.Data[0]["appId"], body.Data[1]["appId"])
	}
	if _, ok := body.Data[0]["appIds"]; ok {
		t.Error("field lama appIds tidak boleh dikirim lagi")
	}
	if string(body.Data[0]["features"]) != "[]" {
		t.Errorf("features = %s, want [] bukan null", body.Data[0]["features"])
	}
}

func postPlan(h *PlanHandler, method, url, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
	w := httptest.NewRecorder()
	if method == http.MethodPost {
		h.Collection(w, req)
	} else {
		h.Item(w, req)
	}
	return w
}

const planBody = `{"appId":%q,"name":"Paket","durationMonths":12,"price":1000,"features":["a"]}`

func TestPlanCreate_AppTidakAdaJadi422(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		stub := &stubPlanLister{}
		h := NewPlanHandler(stub, &stubAppRegistry{exists: false})
		w := postPlan(h, method, map[string]string{http.MethodPost: "/api/v1/plans", http.MethodPatch: "/api/v1/plans/3"}[method],
			strings.Replace(planBody, "%q", `"app_salah_ketik"`, 1))
		if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), `"appId"`) {
			t.Fatalf("%s status = %d (%s), want 422 pada appId", method, w.Code, w.Body.String())
		}
		if stub.created != nil || stub.updated != nil {
			t.Fatalf("%s: paket TIDAK boleh tersimpan untuk App yang tidak ada", method)
		}
	}
}

func TestPlanCreate_GalatRegistryBukan422(t *testing.T) {
	h := NewPlanHandler(&stubPlanLister{}, &stubAppRegistry{existsErr: errors.New("db down")})
	w := postPlan(h, http.MethodPost, "/api/v1/plans", strings.Replace(planBody, "%q", `"app_x"`, 1))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (registry gagal = tidak tahu, bukan salah klien)", w.Code)
	}
}

func TestPlanCreate_ElProofTanpaRegistry(t *testing.T) {
	reg := &stubAppRegistry{exists: false}
	stub := &stubPlanLister{}
	h := NewPlanHandler(stub, reg)
	w := postPlan(h, http.MethodPost, "/api/v1/plans", strings.Replace(planBody, "%q", `"platform-billing"`, 1))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s), want 201", w.Code, w.Body.String())
	}
	if reg.existsCalled {
		t.Fatal("App ElProof selalu ada; registry tidak perlu ditanya (api seed mengosongkannya sampai server start)")
	}
	if stub.created == nil || stub.created.AppID != "platform-billing" {
		t.Fatalf("appId tidak diteruskan ke service: %+v", stub.created)
	}
}

func TestPlanUpdate_AppIdDiteruskan(t *testing.T) {
	stub := &stubPlanLister{}
	h := NewPlanHandler(stub, &stubAppRegistry{exists: true})
	w := postPlan(h, http.MethodPatch, "/api/v1/plans/3", strings.Replace(planBody, "%q", `"app_jws"`, 1))
	if w.Code != http.StatusOK || stub.updated == nil || stub.updated.AppID != "app_jws" {
		t.Fatalf("status = %d, updated = %+v", w.Code, stub.updated)
	}
}

func TestPlanCreate_BukanPlatformAdmin(t *testing.T) {
	reg := &stubAppRegistry{exists: true}
	h := NewPlanHandler(&stubPlanLister{}, reg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plans", strings.NewReader(strings.Replace(planBody, "%q", `"app_x"`, 1)))
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), staffClaims()))
	w := httptest.NewRecorder()
	h.Collection(w, req)
	if w.Code != http.StatusForbidden || reg.existsCalled {
		t.Fatalf("status = %d existsCalled=%v, want 403 tanpa kueri", w.Code, reg.existsCalled)
	}
}
