package application

import (
	"context"
	"strings"
	"testing"
)

func TestValidIdempotencyKey(t *testing.T) {
	ok := []string{"abcdefgh", "A1_b2-C3d4", strings.Repeat("x", 128), "0f8fad5b-d9cb-469f-a165-70867728950e"}
	bad := []string{"", "short", strings.Repeat("x", 129), "has space here", "semi;colon;key", "unicode-kunci-é1", "line\nbreak123"}
	for _, k := range ok {
		if !ValidIdempotencyKey(k) {
			t.Errorf("%q must be accepted", k)
		}
	}
	for _, k := range bad {
		if ValidIdempotencyKey(k) {
			t.Errorf("%q must be refused", k)
		}
	}
}

func TestHashRequestIgnoresKeyOrderAndWhitespace(t *testing.T) {
	a := HashRequest([]byte(`{"amount": 5000000, "type":"DP", "evidence":[{"uploadToken":"x","type":"Transfer Proof"}]}`))
	b := HashRequest([]byte("{\n \"evidence\":[{\"type\":\"Transfer Proof\",\"uploadToken\":\"x\"}],\"type\":\"DP\",\"amount\":5000000}"))
	if a != b {
		t.Error("the same JSON in another key order / layout must hash the same")
	}
	if a == HashRequest([]byte(`{"amount":5000001,"type":"DP","evidence":[{"uploadToken":"x","type":"Transfer Proof"}]}`)) {
		t.Error("a different amount must hash differently")
	}
	// A very large integer must not be rounded through float64.
	if HashRequest([]byte(`{"n":9007199254740993}`)) == HashRequest([]byte(`{"n":9007199254740992}`)) {
		t.Error("integers beyond 2^53 must stay distinct")
	}
	if HashRequest([]byte("not json")) == HashRequest([]byte("not json!")) {
		t.Error("a non-JSON body is hashed as it is")
	}
}

func TestDecideIdempotency(t *testing.T) {
	status := 201
	done := &IdempotencyRecord{StaffID: 5, RequestHash: "h1", StatusCode: &status, Body: []byte(`{}`)}
	running := &IdempotencyRecord{StaffID: 5, RequestHash: "h1"}
	dead := &IdempotencyRecord{StaffID: 5, RequestHash: "h1", Stale: true}
	cases := []struct {
		name     string
		existing *IdempotencyRecord
		staff    int64
		hash     string
		want     IdemOutcome
	}{
		{"first time", nil, 5, "h1", IdemProceed},
		{"same request, finished: replay", done, 5, "h1", IdemReplay},
		{"same key, different body: reuse", done, 5, "h2", IdemReuse},
		{"same key, different body while running: reuse", running, 5, "h2", IdemReuse},
		{"same key, another person: reuse", done, 6, "h1", IdemReuse},
		{"same request while running: in progress", running, 5, "h1", IdemInProgress},
		{"same request, first attempt died: take over", dead, 5, "h1", IdemTakeOver},
		{"dead, but different body: reuse", dead, 5, "h2", IdemReuse},
	}
	for _, c := range cases {
		if got := DecideIdempotency(c.existing, c.staff, c.hash); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

// memIdem is an in-memory IdempotencyRepository.
type memIdem struct {
	rec     *IdempotencyRecord
	takeWin bool
	took    bool
	reserve int
}

func (m *memIdem) Reserve(_ context.Context, _, staff int64, _, _, hash string) (*IdempotencyRecord, error) {
	m.reserve++
	if m.rec != nil {
		return m.rec, nil
	}
	m.rec = &IdempotencyRecord{StaffID: staff, RequestHash: hash}
	return nil, nil
}
func (m *memIdem) TakeOver(context.Context, int64, int64, string, string, string) (bool, error) {
	m.took = true
	return m.takeWin, nil
}
func (m *memIdem) Complete(_ context.Context, _ int64, _, _ string, code int, body []byte) error {
	m.rec.StatusCode, m.rec.Body = &code, body
	return nil
}
func (m *memIdem) Release(context.Context, int64, string, string) error { m.rec = nil; return nil }

func TestIdempotencyBeginFlow(t *testing.T) {
	ctx := context.Background()
	repo := &memIdem{}
	svc := NewIdempotencyService(repo)

	out, _, err := svc.Begin(ctx, 1, 5, "POST /x", "key-12345678", "h1")
	if err != nil || out != IdemProceed {
		t.Fatalf("first: %v %v", out, err)
	}
	if out, _, _ := svc.Begin(ctx, 1, 5, "POST /x", "key-12345678", "h1"); out != IdemInProgress {
		t.Errorf("second while running: want in progress, got %v", out)
	}
	if err := svc.Complete(ctx, 1, "POST /x", "key-12345678", 201, []byte(`{"id":9}`)); err != nil {
		t.Fatal(err)
	}
	out, rec, _ := svc.Begin(ctx, 1, 5, "POST /x", "key-12345678", "h1")
	if out != IdemReplay || *rec.StatusCode != 201 || string(rec.Body) != `{"id":9}` {
		t.Errorf("replay must hand back the stored response: %v %+v", out, rec)
	}
	if out, _, _ := svc.Begin(ctx, 1, 5, "POST /x", "key-12345678", "other"); out != IdemReuse {
		t.Errorf("different body: want reuse, got %v", out)
	}
	// A failed request releases the key: the corrected retry is a fresh attempt.
	repo.rec = nil
	_, _, _ = svc.Begin(ctx, 1, 5, "POST /x", "key-87654321", "h1")
	_ = svc.Release(ctx, 1, "POST /x", "key-87654321")
	if out, _, _ := svc.Begin(ctx, 1, 5, "POST /x", "key-87654321", "h2"); out != IdemProceed {
		t.Errorf("after a release, a retry with a corrected body must proceed, got %v", out)
	}
}

func TestIdempotencyTakeOver(t *testing.T) {
	ctx := context.Background()
	for _, win := range []bool{true, false} {
		repo := &memIdem{rec: &IdempotencyRecord{StaffID: 5, RequestHash: "h1", Stale: true}, takeWin: win}
		out, _, err := NewIdempotencyService(repo).Begin(ctx, 1, 5, "POST /x", "key-12345678", "h1")
		if err != nil || !repo.took {
			t.Fatalf("takeover must be attempted: %v", err)
		}
		want := IdemProceed
		if !win {
			want = IdemInProgress // two retries cannot both run
		}
		if out != want {
			t.Errorf("win=%v: got %v, want %v", win, out, want)
		}
	}
}
