package presentation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/middleware"
)

func platformAdminClaims() *middleware.Claims {
	return &middleware.Claims{PrincipalType: "platform_admin", PrincipalID: "1"}
}

func staffClaims() *middleware.Claims {
	return &middleware.Claims{PrincipalType: "staff", PrincipalID: "1", TenantID: "1"}
}

// TestPlansList_BranchingPrincipal locks D3/A12/T9: platform_admin sees
// every plan via List/ListPaginated, anyone else sees only ListPublic* —
// tested across BOTH the all=true and paginated branches (6.2), because a
// leak through either door is still a leak.
func TestPlansList_BranchingPrincipal(t *testing.T) {
	t.Run("all=true, platform_admin", func(t *testing.T) {
		stub := &stubPlanLister{all: []domain.Plan{{ID: 1, Name: "Semua"}}}
		h := NewPlanHandler(stub)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plans?all=true", nil)
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
		w := httptest.NewRecorder()

		h.Collection(w, req)

		if !stub.calledList {
			t.Fatal("platform_admin harus memanggil List (semua paket), bukan ListPublic")
		}
		if stub.calledListPublic {
			t.Fatal("platform_admin TIDAK boleh terpanggil lewat ListPublic")
		}
	})

	t.Run("all=true, staff", func(t *testing.T) {
		stub := &stubPlanLister{public: []domain.Plan{{ID: 1, Name: "Publik"}}}
		h := NewPlanHandler(stub)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plans?all=true", nil)
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), staffClaims()))
		w := httptest.NewRecorder()

		h.Collection(w, req)

		if !stub.calledListPublic {
			t.Fatal("staff harus memanggil ListPublic, bukan List — inilah T9")
		}
		if stub.calledList {
			t.Fatal("staff TIDAK boleh melihat paket privat lewat List")
		}
	})

	t.Run("paginated, platform_admin", func(t *testing.T) {
		stub := &stubPlanLister{}
		h := NewPlanHandler(stub)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plans", nil)
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
		w := httptest.NewRecorder()

		h.Collection(w, req)

		if !stub.calledListPaginated {
			t.Fatal("platform_admin harus memanggil ListPaginated di cabang paginasi")
		}
	})

	t.Run("paginated, staff", func(t *testing.T) {
		stub := &stubPlanLister{}
		h := NewPlanHandler(stub)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plans", nil)
		req = req.WithContext(middleware.NewContextWithClaims(req.Context(), staffClaims()))
		w := httptest.NewRecorder()

		h.Collection(w, req)

		if !stub.calledListPublicPaged {
			t.Fatal("staff harus memanggil ListPublicPaginated di cabang paginasi — kebocoran bisa pindah pintu di sini (T9)")
		}
	})
}

func TestPlansList_TanpaKlaimPakaiJalurPublik(t *testing.T) {
	stub := &stubPlanLister{}
	h := NewPlanHandler(stub)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plans?all=true", nil)
	w := httptest.NewRecorder()

	h.Collection(w, req)

	if !stub.calledListPublic {
		t.Fatal("tanpa klaim harus jatuh ke jalur publik (default aman, 6.3)")
	}
	if stub.calledList {
		t.Fatal("tanpa klaim TIDAK boleh memanggil List")
	}
}

func TestPlansList_AppIdsTerkirim(t *testing.T) {
	stub := &stubPlanLister{all: []domain.Plan{
		{ID: 1, Name: "Privat JWS", AppIDs: []string{"app_02ef90c52704"}},
		{ID: 2, Name: "Publik", AppIDs: nil},
	}}
	h := NewPlanHandler(stub)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plans?all=true", nil)
	req = req.WithContext(middleware.NewContextWithClaims(req.Context(), platformAdminClaims()))
	w := httptest.NewRecorder()

	h.Collection(w, req)

	var body struct {
		Data []planResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("gagal decode: %v (%s)", err, w.Body.String())
	}
	if len(body.Data) != 2 {
		t.Fatalf("data length = %d, want 2", len(body.Data))
	}
	if len(body.Data[0].AppIDs) != 1 || body.Data[0].AppIDs[0] != "app_02ef90c52704" {
		t.Errorf("plan privat harus membawa appIds: %v", body.Data[0].AppIDs)
	}
	if body.Data[1].AppIDs == nil || len(body.Data[1].AppIDs) != 0 {
		t.Errorf("plan publik harus appIds:[] bukan null, got: %v", body.Data[1].AppIDs)
	}
}
