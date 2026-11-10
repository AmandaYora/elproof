package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/pagination"
)

// stubPublicPlanRepo implements application.PlanRepository; only ListPublic
// is exercised. The handler is wired to a real *application.PlanService so
// these tests also lock ListPublicActive's filter and ordering.
type stubPublicPlanRepo struct {
	public []domain.Plan
	err    error
}

func (s *stubPublicPlanRepo) ListPublic(ctx context.Context) ([]domain.Plan, error) {
	return s.public, s.err
}

func (s *stubPublicPlanRepo) List(ctx context.Context) ([]domain.Plan, error) { return nil, nil }
func (s *stubPublicPlanRepo) ListPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	return nil, 0, nil
}
func (s *stubPublicPlanRepo) FindByID(ctx context.Context, id int64) (*domain.Plan, error) {
	return nil, nil
}
func (s *stubPublicPlanRepo) Create(ctx context.Context, plan *domain.Plan) error { return nil }
func (s *stubPublicPlanRepo) Update(ctx context.Context, plan *domain.Plan) error { return nil }
func (s *stubPublicPlanRepo) SetActive(ctx context.Context, id int64, isActive bool) error {
	return nil
}
func (s *stubPublicPlanRepo) ListForApp(ctx context.Context, appID string) ([]domain.Plan, error) {
	return nil, nil
}
func (s *stubPublicPlanRepo) ListPublicPaginated(ctx context.Context, params pagination.Params, search string) ([]domain.Plan, int64, error) {
	return nil, 0, nil
}
func (s *stubPublicPlanRepo) ListPlanIDsForApp(ctx context.Context, appID string) ([]int64, error) {
	return nil, nil
}
func (s *stubPublicPlanRepo) ReplaceAppPlans(ctx context.Context, appID string, planIDs []int64) error {
	return nil
}

func servePublicPlans(t *testing.T, repo *stubPublicPlanRepo) *httptest.ResponseRecorder {
	t.Helper()
	h := NewPublicPlanHandler(application.NewPlanService(repo))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/plans", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	return w
}

type publicPlansEnvelope struct {
	Success bool                         `json:"success"`
	Data    []map[string]json.RawMessage `json:"data"`
}

func decodePublicPlans(t *testing.T, w *httptest.ResponseRecorder) publicPlansEnvelope {
	t.Helper()
	var env publicPlansEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("body bukan JSON: %v (%s)", err, w.Body.String())
	}
	return env
}

func TestPublicPlans_KosongJadiArray(t *testing.T) {
	w := servePublicPlans(t, &stubPublicPlanRepo{public: nil})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("body = %s, want data:[] (bukan null)", w.Body.String())
	}
}

func TestPublicPlans_NonaktifTidakMuncul(t *testing.T) {
	w := servePublicPlans(t, &stubPublicPlanRepo{public: []domain.Plan{
		{ID: 1, Name: "Aktif", DurationMonths: 1, Price: 100, IsActive: true},
		{ID: 2, Name: "Nonaktif", DurationMonths: 1, Price: 50, IsActive: false},
	}})

	env := decodePublicPlans(t, w)
	if len(env.Data) != 1 || string(env.Data[0]["id"]) != "1" {
		t.Fatalf("data = %s, want hanya paket id 1", w.Body.String())
	}
}

func TestPublicPlans_FieldDanFeaturesNil(t *testing.T) {
	now := time.Now()
	w := servePublicPlans(t, &stubPublicPlanRepo{public: []domain.Plan{
		{ID: 7, Name: "Dasar", DurationMonths: 3, Price: 300000, Features: nil, IsActive: true,
			CreatedAt: now, UpdatedAt: now, AppIDs: []string{"tidak-boleh-bocor"}},
	}})

	env := decodePublicPlans(t, w)
	if !env.Success || len(env.Data) != 1 {
		t.Fatalf("body = %s, want success dengan 1 paket", w.Body.String())
	}
	got := env.Data[0]
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	want := map[string]string{
		"id": "7", "name": `"Dasar"`, "durationMonths": "3", "price": "300000", "features": "[]",
	}
	if len(got) != len(want) {
		t.Fatalf("keys = %v, want persis %v", keys, want)
	}
	for k, v := range want {
		if string(got[k]) != v {
			t.Fatalf("%s = %s, want %s (body %s)", k, got[k], v, w.Body.String())
		}
	}
}

func TestPublicPlans_UrutDurasiLaluHarga(t *testing.T) {
	w := servePublicPlans(t, &stubPublicPlanRepo{public: []domain.Plan{
		{ID: 1, Name: "12 bln", DurationMonths: 12, Price: 1000, IsActive: true},
		{ID: 2, Name: "1 bln mahal", DurationMonths: 1, Price: 200, IsActive: true, Features: []string{"a"}},
		{ID: 3, Name: "3 bln", DurationMonths: 3, Price: 500, IsActive: true},
		{ID: 4, Name: "1 bln murah", DurationMonths: 1, Price: 100, IsActive: true},
	}})

	env := decodePublicPlans(t, w)
	ids := make([]string, 0, len(env.Data))
	for _, p := range env.Data {
		ids = append(ids, string(p["id"]))
	}
	if want := []string{"4", "2", "3", "1"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("urutan id = %v, want %v", ids, want)
	}
	if string(env.Data[1]["features"]) != `["a"]` {
		t.Fatalf("features = %s, want [\"a\"]", env.Data[1]["features"])
	}
}

func TestPublicPlans_HeaderCache(t *testing.T) {
	w := servePublicPlans(t, &stubPublicPlanRepo{})

	if got := w.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("Cache-Control = %q, want %q", got, "public, max-age=300")
	}
}

func TestPublicPlans_GalatRepoJadi500TanpaCache(t *testing.T) {
	w := servePublicPlans(t, &stubPublicPlanRepo{err: errors.New("db down")})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), "db down") {
		t.Fatalf("body = %s, galat internal bocor", w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("Cache-Control = %q, galat tidak boleh di-cache publik", got)
	}
}
