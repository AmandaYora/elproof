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
