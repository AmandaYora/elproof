package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func req(body string) *http.Request {
	return httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body))
}

func TestReadObjectTellsAbsentFromNullFromZero(t *testing.T) {
	obj, _, err := ReadObject(strings.NewReader(`{"a": null, "b": 0, "c": ""}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := obj[k]; !ok {
			t.Errorf("%s was sent and must be present", k)
		}
	}
	if _, ok := obj["d"]; ok {
		t.Error("d was not sent")
	}
	if obj, _, err := ReadObject(strings.NewReader(`  {}  `)); err != nil || obj == nil || len(obj) != 0 {
		t.Errorf("an empty object is valid and empty: %v %v", obj, err)
	}
	for _, bad := range []string{``, `[]`, `null`, `"x"`, `{"a":`, `1`} {
		if _, _, err := ReadObject(strings.NewReader(bad)); err == nil {
			t.Errorf("%q is not a JSON object", bad)
		}
	}
}

func TestReadStatePatch(t *testing.T) {
	sp, err := ReadStatePatch(req(`{"isActive": false}`), "isActive")
	if err != nil || !sp.IsState || sp.Value {
		t.Errorf("state false: %+v %v", sp, err)
	}
	sp, err = ReadStatePatch(req(`{"isActive": true}`), "isActive")
	if err != nil || !sp.IsState || !sp.Value {
		t.Errorf("state true: %+v %v", sp, err)
	}
	// Without the key it is an ordinary edit and the body is handed back whole.
	sp, err = ReadStatePatch(req(`{"name": "x"}`), "isActive")
	if err != nil || sp.IsState || string(sp.Body) != `{"name": "x"}` {
		t.Errorf("edit body: %+v %v", sp, err)
	}
	var se *StateError
	for _, bad := range []string{`{"isActive": null}`, `{"isActive": "true"}`, `{"isActive": 1}`, `{"isActive": false, "name": "x"}`} {
		if _, err := ReadStatePatch(req(bad), "isActive"); !errors.As(err, &se) || se.Key != "isActive" {
			t.Errorf("%s: want a StateError on isActive, got %v", bad, err)
		}
	}
	if _, err := ReadStatePatch(req(`[true]`), "isActive"); err == nil || errors.As(err, &se) {
		t.Errorf("a body that is not an object is not a StateError: %v", err)
	}
}

func TestWriteStatePatchError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteStatePatchError(rec, &StateError{Key: "isActive", Message: "isActive harus true atau false"})
	if rec.Code != 422 || !strings.Contains(rec.Body.String(), `"isActive":["isActive harus true atau false"]`) {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	WriteStatePatchError(rec, ErrNotObject)
	if rec.Code != 400 {
		t.Errorf("%d", rec.Code)
	}
}
