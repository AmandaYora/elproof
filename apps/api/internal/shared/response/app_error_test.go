package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"elproof/internal/shared/apperror"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestFromErrorKeepsFieldsAndCodeTogether(t *testing.T) {
	rec := httptest.NewRecorder()
	err := apperror.Validation("Nominal melebihi sisa tagihan", map[string][]string{"terms[2].amount": {"Nominal melebihi sisa tagihan"}}).WithCode("payment_exceeds_contract")
	FromError(rec, err)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", rec.Code)
	}
	errs := decode(t, rec)["errors"].(map[string]any)
	if errs["code"] != "payment_exceeds_contract" {
		t.Errorf("a validation error with a code must still carry it: %v", errs)
	}
	if msgs, ok := errs["terms[2].amount"].([]any); !ok || len(msgs) != 1 {
		t.Errorf("per-field messages (with the row index) must stay arrays: %v", errs)
	}
}

func TestFromErrorShapes(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		errors any
	}{
		{"validation without a code", apperror.Validation("x", map[string][]string{"f": {"m"}}), 422, map[string]any{"f": []any{"m"}}},
		{"not found", apperror.NotFound("tidak ada"), 404, nil},
		{"forbidden with a code", apperror.Forbidden("no").WithCode("forbidden_permission"), 403, map[string]any{"code": "forbidden_permission"}},
		{"unknown error is a generic 500", errors.New("boom: secret detail"), 500, nil},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		FromError(rec, c.err)
		body := decode(t, rec)
		if rec.Code != c.status {
			t.Errorf("%s: status %d, want %d", c.name, rec.Code, c.status)
		}
		got, _ := json.Marshal(body["errors"])
		want, _ := json.Marshal(c.errors)
		if string(got) != string(want) {
			t.Errorf("%s: errors %s, want %s", c.name, got, want)
		}
		if body["success"] != false {
			t.Errorf("%s: success must be false", c.name)
		}
		if c.status == 500 && body["message"] == "boom: secret detail" {
			t.Errorf("%s: internal error text leaked", c.name)
		}
	}
}

func TestStaleConflictCarriesTheCurrentItem(t *testing.T) {
	rec := httptest.NewRecorder()
	FromError(rec, apperror.Stale(nil).WithData(map[string]any{"current": map[string]any{"id": 7, "updatedAt": "2026-11-10T08:15:30.123456Z"}}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d", rec.Code)
	}
	body := decode(t, rec)
	if body["success"] != false || body["message"] != "Data ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru." {
		t.Errorf("envelope: %v", body)
	}
	errs, _ := body["errors"].(map[string]any)
	if c, _ := errs["_conflict"].([]any); len(c) != 1 || c[0] != "stale" {
		t.Errorf("errors = %v", body["errors"])
	}
	cur := body["data"].(map[string]any)["current"].(map[string]any)
	if cur["id"] != float64(7) || cur["updatedAt"] != "2026-11-10T08:15:30.123456Z" {
		t.Errorf("data.current = %v", cur)
	}
	// A plain conflict keeps its old shape: no data key at all.
	rec = httptest.NewRecorder()
	FromError(rec, apperror.Conflict("bentrok").WithCode("x"))
	if _, has := decode(t, rec)["data"]; has {
		t.Error("an ordinary conflict has no data")
	}
}
