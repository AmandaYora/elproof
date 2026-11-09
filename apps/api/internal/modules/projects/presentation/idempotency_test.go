package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elproof/internal/modules/projects/application"
)

type memIdemRepo struct {
	rec *application.IdempotencyRecord
}

func (m *memIdemRepo) Reserve(_ context.Context, _, staff int64, _, _, hash string) (*application.IdempotencyRecord, error) {
	if m.rec != nil {
		return m.rec, nil
	}
	m.rec = &application.IdempotencyRecord{StaffID: staff, RequestHash: hash}
	return nil, nil
}
func (m *memIdemRepo) TakeOver(context.Context, int64, int64, string, string, string) (bool, error) {
	return false, nil
}
func (m *memIdemRepo) Complete(_ context.Context, _ int64, _, _ string, code int, body []byte) error {
	m.rec.StatusCode, m.rec.Body = &code, body
	return nil
}
func (m *memIdemRepo) Release(context.Context, int64, string, string) error { m.rec = nil; return nil }

func idemHandler() (*Handler, *memIdemRepo) {
	repo := &memIdemRepo{}
	return &Handler{idem: application.NewIdempotencyService(repo)}, repo
}

func post(h *Handler, key, body string, next func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/1/client-payments", strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	h.idempotent(rec, req, staffClaims{tenantID: 1, staffID: 5}, next)
	return rec
}

func TestIdempotentReplaysTheSameCreate(t *testing.T) {
	h, _ := idemHandler()
	runs := 0
	create := func(w http.ResponseWriter, r *http.Request) {
		runs++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"id":` + string(rune('0'+runs)) + `}}`))
	}
	first := post(h, "key-aaaaaaaa", `{"amount":1}`, create)
	// The same request, written differently, with the same key.
	second := post(h, "key-aaaaaaaa", ` { "amount" : 1 } `, create)
	if runs != 1 {
		t.Fatalf("the handler must run once, ran %d times", runs)
	}
	if first.Code != 201 || second.Code != 201 || first.Body.String() != second.Body.String() {
		t.Errorf("the replay is the same status and body: %d %q vs %d %q", first.Code, first.Body.String(), second.Code, second.Body.String())
	}
	if second.Header().Get("Idempotent-Replay") != "true" || first.Header().Get("Idempotent-Replay") != "" {
		t.Errorf("only the replay is marked: %q / %q", first.Header().Get("Idempotent-Replay"), second.Header().Get("Idempotent-Replay"))
	}
}

func TestIdempotentRefusesAChangedBodyAndABadKey(t *testing.T) {
	h, _ := idemHandler()
	ok := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) }
	post(h, "key-bbbbbbbb", `{"amount":1}`, ok)
	rec := post(h, "key-bbbbbbbb", `{"amount":2}`, ok)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("different body: %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Errors map[string]any `json:"errors"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Errors["code"] != "idempotency_key_reuse" || body.Errors["Idempotency-Key"] == nil {
		t.Errorf("422 names the header and carries the code: %v", body.Errors)
	}
	rec = post(h, "short", `{}`, ok)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "Idempotency-Key") {
		t.Errorf("malformed key: %d %s", rec.Code, rec.Body.String())
	}
}

func TestIdempotentConflictWhileRunningAndReleaseOnFailure(t *testing.T) {
	h, repo := idemHandler()
	// The first request is "still running": another arrives with the same key.
	var inner *httptest.ResponseRecorder
	post(h, "key-cccccccc", `{"a":1}`, func(w http.ResponseWriter, r *http.Request) {
		inner = post(h, "key-cccccccc", `{"a":1}`, func(http.ResponseWriter, *http.Request) { t.Error("must not run twice") })
		w.WriteHeader(http.StatusCreated)
	})
	if inner.Code != http.StatusConflict {
		t.Errorf("a retry while the first is running is a 409, got %d %s", inner.Code, inner.Body.String())
	}

	// A failed request (422) releases the key: the corrected request with the same
	// key is a new attempt, not a reuse.
	repo.rec = nil
	rec := post(h, "key-dddddddd", `{"a":1}`, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnprocessableEntity) })
	if rec.Code != 422 || repo.rec != nil {
		t.Fatalf("failure answer passes through and frees the key: %d %v", rec.Code, repo.rec)
	}
	rec = post(h, "key-dddddddd", `{"a":2}`, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) })
	if rec.Code != http.StatusCreated {
		t.Errorf("corrected retry: %d", rec.Code)
	}
}

func TestNoHeaderMeansNoIdempotency(t *testing.T) {
	h, repo := idemHandler()
	runs := 0
	next := func(w http.ResponseWriter, r *http.Request) { runs++; w.WriteHeader(http.StatusCreated) }
	post(h, "", `{}`, next)
	post(h, "", `{}`, next)
	if runs != 2 || repo.rec != nil {
		t.Errorf("without the header every request runs: %d", runs)
	}
}

func TestPanicReleasesTheKey(t *testing.T) {
	h, repo := idemHandler()
	func() {
		defer func() { _ = recover() }()
		post(h, "key-eeeeeeee", `{}`, func(http.ResponseWriter, *http.Request) { panic("boom") })
	}()
	if repo.rec != nil {
		t.Error("a request that blew up must not hold its key")
	}
}
