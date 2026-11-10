package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/modules/billing/presentation"
)

type stubPublicActiveLister struct{ called bool }

func (s *stubPublicActiveLister) ListPublicActive(ctx context.Context) ([]domain.Plan, error) {
	s.called = true
	return nil, nil
}

// The public plans route answers without any Authorization header (it is
// registered outside `authed`) and only to GET.
func TestPublicRoutes_PlansTanpaSesiDanHanyaGet(t *testing.T) {
	lister := &stubPublicActiveLister{}
	mux := http.NewServeMux()
	registerPublicRoutes(mux, presentation.NewPublicPlanHandler(lister))

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/public/plans", nil))
	if w.Code != http.StatusOK || !lister.called {
		t.Fatalf("GET status = %d (called=%v), want 200 tanpa sesi", w.Code, lister.called)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		lister.called = false
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/public/plans", nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405", method, w.Code)
		}
		if lister.called {
			t.Fatalf("%s mencapai service; harus ditolak sebelum kueri", method)
		}
	}
}
