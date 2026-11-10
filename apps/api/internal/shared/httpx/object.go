package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"elproof/internal/shared/response"
)

// ErrNotObject: the request body is not a single JSON object.
var ErrNotObject = errors.New("httpx: request body is not a JSON object")

// maxObjectBody bounds what ReadObject will read (these bodies are tiny JSON).
const maxObjectBody = 1 << 20

// ReadObject reads a request body that must be one JSON object and returns its
// raw members, so a handler can tell a key that was sent (even as null) from one
// that was left out: the difference a partial PATCH is built on. The returned
// bytes are the whole body, for handlers that still decode it into a struct.
func ReadObject(body io.Reader) (map[string]json.RawMessage, []byte, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxObjectBody+1))
	if err != nil {
		return nil, nil, err
	}
	if len(raw) > maxObjectBody {
		return nil, nil, ErrNotObject
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, raw, ErrNotObject
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return nil, raw, err
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	return obj, raw, nil
}

// ExplicitBool reads a state flag that must be a real JSON boolean. present is
// false when the key was not sent; ok is false when it was sent but is not true
// or false (null, a string, a number).
func ExplicitBool(obj map[string]json.RawMessage, key string) (value, present, ok bool) {
	raw, present := obj[key]
	if !present {
		return false, false, true
	}
	switch string(bytes.TrimSpace(raw)) {
	case "true":
		return true, true, true
	case "false":
		return false, true, true
	}
	return false, true, false
}

// StatePatch is a PATCH body that either carries one explicit state flag on its
// own (IsState, with Value) or is some other edit (Body is the whole body, for the
// handler's usual decoding).
type StatePatch struct {
	Value   bool
	IsState bool
	Body    []byte
}

// StateError: the state flag was sent wrongly (not a boolean, or mixed with other
// fields). Handlers answer it as a 422 on the flag's key.
type StateError struct {
	Key     string
	Message string
}

func (e *StateError) Error() string { return e.Message }

// ReadStatePatch reads a PATCH body and recognises the explicit-state form
// {"<key>": true|false} (ADR-0036). Setting a state shares the route with the
// resource's ordinary edit, so the form is told apart by the key being present:
// the body must then be the flag alone (mixing it with other fields would silently
// drop the other edits, so it is a *StateError) and the flag must be a real
// boolean. Without the key the body is returned untouched for the edit path. A
// body that is not a JSON object is ErrNotObject (or the read/parse error).
func ReadStatePatch(r *http.Request, key string) (StatePatch, error) {
	obj, raw, err := ReadObject(r.Body)
	if err != nil {
		// Keep the old edit path's behaviour for a body that is not an object: let it
		// fail to decode there.
		return StatePatch{Body: raw}, err
	}
	value, present, ok := ExplicitBool(obj, key)
	if !present {
		return StatePatch{Body: raw}, nil
	}
	if !ok {
		return StatePatch{}, &StateError{Key: key, Message: key + " harus true atau false"}
	}
	if len(obj) > 1 {
		return StatePatch{}, &StateError{Key: key, Message: "Kirim " + key + " sendiri, terpisah dari perubahan data lain"}
	}
	return StatePatch{Value: value, IsState: true, Body: raw}, nil
}

// WriteStatePatchError answers an error of ReadStatePatch: a wrongly sent flag is
// a 422 on its key, anything else (not JSON, not an object) a 400.
func WriteStatePatchError(w http.ResponseWriter, err error) {
	var se *StateError
	if errors.As(err, &se) {
		response.Error(w, http.StatusUnprocessableEntity, se.Message, map[string][]string{se.Key: {se.Message}})
		return
	}
	response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
}
