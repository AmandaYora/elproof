package presentation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elproof/internal/shared/apperror"
)

func reader(t *testing.T, body string) *patchReader {
	t.Helper()
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body))
	pr, ok := newPatchReader(rec, r)
	if !ok {
		t.Fatalf("%q: unexpected %d", body, rec.Code)
	}
	return pr
}

func fieldsOf(err error) map[string][]string {
	if ae, ok := apperror.As(err); ok {
		return ae.Fields
	}
	return nil
}

func TestPatchReaderAbsentNullAndZeroAreDifferent(t *testing.T) {
	pr := reader(t, `{"name": "", "amount": 0, "notes": null, "pic": null, "ok": false}`)
	name, amount := pr.text("name", false), pr.integer("amount", false)
	notes, pic := pr.text("notes", true), pr.integer("pic", true)
	ok, other := pr.boolean("ok"), pr.text("other", false)
	if err := pr.finish(); err != nil {
		t.Fatal(err)
	}
	if !name.Present() || name.Value() != "" || !amount.Present() || amount.Value() != 0 {
		t.Error("an empty string and zero are values that were sent")
	}
	if !notes.Present() || notes.IsNull() || notes.Value() != "" {
		t.Error("null on a nullable text clears it to the empty string")
	}
	if !pic.Present() || !pic.IsNull() {
		t.Error("null on a nullable number stays null")
	}
	if !ok.Present() || ok.Value() {
		t.Error("false is a value")
	}
	if other.Present() {
		t.Error("a key that was not sent is absent")
	}
}

func TestPatchReaderRefusesWhatIsWrongOnItsOwnKey(t *testing.T) {
	pr := reader(t, `{"name": null, "amount": "banyak", "when": "besok", "flag": null, "cleared": ""}`)
	pr.text("name", false)
	pr.integer("amount", false)
	pr.date("when", false)
	pr.boolean("flag")
	if d := pr.date("cleared", true); !d.IsNull() {
		t.Error(`"" clears an optional date`)
	}
	f := fieldsOf(pr.finish())
	for _, k := range []string{"name", "amount", "when", "flag"} {
		if len(f[k]) == 0 {
			t.Errorf("missing a message for %s: %v", k, f)
		}
	}
	if _, bad := f["cleared"]; bad {
		t.Error("an optional date may be cleared with an empty string")
	}
	if f["when"][0] != "Gunakan format YYYY-MM-DD" {
		t.Errorf("the date message is the one create uses: %q", f["when"][0])
	}
}

func TestPatchReaderEmptyAndUnknownBodiesAreOneError(t *testing.T) {
	for _, body := range []string{`{}`, `{"tidakDikenal": 1}`} {
		pr := reader(t, body)
		pr.text("name", false)
		f := fieldsOf(pr.finish())
		if len(f["body"]) != 1 || len(f) != 1 {
			t.Errorf("%s: want one error on `body`, got %v", body, f)
		}
	}
	// A recognised key that is only wrong is its own error, not "empty".
	pr := reader(t, `{"name": null}`)
	pr.text("name", false)
	if f := fieldsOf(pr.finish()); len(f["body"]) != 0 || len(f["name"]) != 1 {
		t.Errorf("got %v", f)
	}
}

func TestPatchReaderBodyThatIsNotAnObject(t *testing.T) {
	for _, body := range []string{``, `[]`, `"x"`, `{"a":`, `null`} {
		rec := httptest.NewRecorder()
		if _, ok := newPatchReader(rec, httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body))); ok || rec.Code != 400 {
			t.Errorf("%q: want a 400, got ok=%v code=%d", body, ok, rec.Code)
		}
	}
}

func TestIfMatchHeader(t *testing.T) {
	get := func(h string) (bool, int) {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, "/x", nil)
		if h != "" {
			r.Header.Set("If-Match", h)
		}
		exp, ok := ifMatch(rec, r)
		if ok && h != "" && exp == nil {
			t.Errorf("%q: a valid header must give a version", h)
		}
		return ok, rec.Code
	}
	if ok, _ := get(""); !ok {
		t.Error("an absent header is fine")
	}
	if ok, _ := get(`"2026-11-10T08:15:30.123456Z"`); !ok {
		t.Error("a valid header is fine")
	}
	if ok, code := get("banana"); ok || code != 400 {
		t.Errorf("malformed: ok=%v code=%d", ok, code)
	}
}

func TestWritePatchErrorRendersTheCurrentItem(t *testing.T) {
	rec := httptest.NewRecorder()
	writePatchError(rec, apperror.Stale("domain row"), func(cur any) (any, error) {
		return map[string]any{"rendered": cur}, nil
	})
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), `"data":{"current":{"rendered":"domain row"}}`) {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	writePatchError(rec, apperror.NotFound("tidak ada"), nil)
	if rec.Code != 404 {
		t.Errorf("other errors keep their shape: %d", rec.Code)
	}
}
