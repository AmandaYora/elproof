package presentation

import (
	"bytes"
	"context"
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
	pr, err := readPatchBody(r)
	if err != nil {
		writeBadBody(w)
		return nil, false
	}
	return pr, true
}

// readPatchBody reads the body without answering, for the one handler that must
// decide about If-Match before it answers a bad body (the project PATCH).
func readPatchBody(r *http.Request) (*patchReader, error) {
	obj, _, err := httpx.ReadObject(r.Body)
	if err != nil {
		return nil, err
	}
	return &patchReader{obj: obj, errs: map[string][]string{}}, nil
}

func writeBadBody(w http.ResponseWriter) {
	response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
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

// only reports whether the body is exactly this one key (so no other member,
// recognised or not, rides along). Reading it does not count the key as read.
func (p *patchReader) only(key string) bool { return len(p.obj) == 1 && p.has(key) }

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

// MsgIfMatchRequired is the message of the 428 a field edit gets when it carries no
// If-Match header (ADR-0036, Tahap 9).
const MsgIfMatchRequired = "Sertakan versi data (If-Match) agar perubahan tidak menimpa orang lain. Muat ulang data lalu coba lagi."

// readIfMatch is the one place that decides what the If-Match header means for a
// PATCH (ADR-0036). Order of answers:
//
//  1. a malformed value is a 400 (the header leaks nothing, so it goes first);
//  2. a valid value is returned and the write becomes a compare-and-set;
//  3. no header and not required (an explicit-state call, which is idempotent):
//     (nil, true), the write proceeds as it always did;
//  4. no header and required: nothing is written and nothing about the body is
//     looked at. exists, when given, is asked first and its error answered as is,
//     so a row that is not there (or is another workspace's) is the same 404 with
//     and without the header and the 428 never tells the two apart; only a row
//     the caller may see gets `428 Precondition Required`.
func readIfMatch(w http.ResponseWriter, r *http.Request, required bool, exists func(context.Context) error) (*time.Time, bool) {
	expected, err := etag.ParseIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		msg := "Header If-Match tidak valid. Kirim nilai updatedAt dari data yang Anda lihat."
		response.Error(w, http.StatusBadRequest, msg, map[string][]string{"If-Match": {msg}})
		return nil, false
	}
	if expected == nil && required {
		if exists != nil {
			if err := exists(r.Context()); err != nil {
				writeAppError(w, err)
				return nil, false
			}
		}
		response.Error(w, http.StatusPreconditionRequired, MsgIfMatchRequired, map[string][]string{"If-Match": {"wajib"}})
		return nil, false
	}
	return expected, true
}

// ifMatch reads an If-Match that is optional (the evidence visibility state: an
// idempotent call that still honours the header when it is sent).
func ifMatch(w http.ResponseWriter, r *http.Request) (*time.Time, bool) {
	return readIfMatch(w, r, false, nil)
}

// requireIfMatch reads the If-Match of a field edit, which must carry one (428
// without). exists is the row's existence check, run only when the header is
// missing; see readIfMatch for the order.
func requireIfMatch(w http.ResponseWriter, r *http.Request, exists func(context.Context) error) (*time.Time, bool) {
	return readIfMatch(w, r, true, exists)
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
