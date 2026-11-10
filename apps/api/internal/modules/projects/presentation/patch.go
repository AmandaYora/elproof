package presentation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/response"
)

// Partial PATCH reading and the If-Match / ETag plumbing (ADR-0036). A PATCH body
// is read as a JSON object whose members are looked at one by one, so a key that
// was left out, a key sent as null and a key sent with the zero value are three
// different things.

// patchReader reads the members of one PATCH body. Every getter returns an
// application.Opt (absent, null or a value) and records what is wrong with a sent
// member under its own key; finish turns the problems into one 422.
type patchReader struct {
	obj   map[string]json.RawMessage
	errs  map[string][]string
	known int
}

// newPatchReader reads the request body, which must be a JSON object (a 400
// otherwise, written here).
func newPatchReader(w http.ResponseWriter, r *http.Request) (*patchReader, bool) {
	obj, _, err := httpx.ReadObject(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return nil, false
	}
	return &patchReader{obj: obj, errs: map[string][]string{}}, true
}

func (p *patchReader) fail(key, msg string) { p.errs[key] = append(p.errs[key], msg) }

// raw returns a recognised member; ok is false when it was not sent.
func (p *patchReader) raw(key string) (json.RawMessage, bool) {
	raw, ok := p.obj[key]
	if ok {
		p.known++
	}
	return raw, ok
}

func isNull(raw json.RawMessage) bool { return string(bytes.TrimSpace(raw)) == "null" }

// has reports whether the key was sent at all (without counting it as read).
func (p *patchReader) has(key string) bool { _, ok := p.obj[key]; return ok }

const msgRequired = "Wajib diisi"

// text reads a string member. A required one cannot be null; a nullable one sent
// as null clears to "" (the stored form of "none" for text).
func (p *patchReader) text(key string, nullable bool) application.Opt[string] {
	raw, ok := p.raw(key)
	if !ok {
		return application.Opt[string]{}
	}
	if isNull(raw) {
		if !nullable {
			p.fail(key, msgRequired)
			return application.Opt[string]{}
		}
		return application.Some("")
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		p.fail(key, "Harus berupa teks")
		return application.Opt[string]{}
	}
	return application.Some(v)
}

// nullableText reads a string member where null means "clear" and is passed on as
// null (for example a time of day).
func (p *patchReader) nullableText(key string) application.Opt[string] {
	raw, ok := p.raw(key)
	if !ok {
		return application.Opt[string]{}
	}
	if isNull(raw) {
		return application.Null[string]()
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		p.fail(key, "Harus berupa teks")
		return application.Opt[string]{}
	}
	if v == "" {
		return application.Null[string]()
	}
	return application.Some(v)
}

// integer reads a whole-number member; null is only allowed where nullable.
func (p *patchReader) integer(key string, nullable bool) application.Opt[int64] {
	raw, ok := p.raw(key)
	if !ok {
		return application.Opt[int64]{}
	}
	if isNull(raw) {
		if !nullable {
			p.fail(key, msgRequired)
			return application.Opt[int64]{}
		}
		return application.Null[int64]()
	}
	var v int64
	if err := json.Unmarshal(raw, &v); err != nil {
		p.fail(key, "Harus berupa angka bulat")
		return application.Opt[int64]{}
	}
	return application.Some(v)
}

// boolean reads a true/false member (never null).
func (p *patchReader) boolean(key string) application.Opt[bool] {
	raw, ok := p.raw(key)
	if !ok {
		return application.Opt[bool]{}
	}
	var v bool
	if isNull(raw) || json.Unmarshal(raw, &v) != nil {
		p.fail(key, "Harus true atau false")
		return application.Opt[bool]{}
	}
	return application.Some(v)
}

// date reads a YYYY-MM-DD member. A nullable one may be null or "" (cleared); a
// required one may be neither.
func (p *patchReader) date(key string, nullable bool) application.Opt[time.Time] {
	raw, ok := p.raw(key)
	if !ok {
		return application.Opt[time.Time]{}
	}
	if isNull(raw) {
		if !nullable {
			p.fail(key, msgRequired)
			return application.Opt[time.Time]{}
		}
		return application.Null[time.Time]()
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		p.fail(key, "Gunakan format YYYY-MM-DD")
		return application.Opt[time.Time]{}
	}
	if v == "" && nullable {
		return application.Null[time.Time]()
	}
	t, err := parseDate(v)
	if err != nil {
		p.fail(key, "Gunakan format YYYY-MM-DD")
		return application.Opt[time.Time]{}
	}
	return application.Some(t)
}

// finish is nil when the body is usable. A body with no recognised field (an
// empty object, or only keys this resource does not know) is a 422 on `body`, so a
// client bug does not look like a successful edit; a member that is sent but wrong
// is a 422 on its own key.
func (p *patchReader) finish() error {
	if len(p.errs) > 0 {
		keys := make([]string, 0, len(p.errs))
		for k := range p.errs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		msg := p.errs[keys[0]][0]
		if len(keys) > 1 {
			msg += " Ada isian lain yang perlu diperbaiki."
		}
		return apperror.Validation(msg, p.errs)
	}
	if p.known == 0 {
		msg := "Kirim minimal satu field yang ingin diubah"
		return apperror.Validation(msg, map[string][]string{"body": {msg}})
	}
	return nil
}

// ifMatch reads the optional If-Match header. A malformed value is a 400 (written
// here); an absent header is (nil, true): the write is then last-write-wins.
func ifMatch(w http.ResponseWriter, r *http.Request) (*time.Time, bool) {
	expected, err := etag.ParseIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		msg := "Header If-Match tidak valid. Kirim nilai updatedAt dari data yang Anda lihat."
		response.Error(w, http.StatusBadRequest, msg, map[string][]string{"If-Match": {msg}})
		return nil, false
	}
	return expected, true
}

// writePatchError answers a failed PATCH. A stale version is the 409 whose
// data.current is the item as it is now, rendered with the same builder the
// success answer uses (render receives what the service held, a domain pointer);
// every other error goes through writeAppError as before.
func writePatchError(w http.ResponseWriter, err error, render func(current any) (any, error)) {
	if current, ok := apperror.IsStale(err); ok {
		item, rerr := render(current)
		if rerr != nil {
			writeAppError(w, rerr)
			return
		}
		ae, _ := apperror.As(err)
		response.FromError(w, ae.WithData(map[string]any{"current": item}))
		return
	}
	writeAppError(w, err)
}
