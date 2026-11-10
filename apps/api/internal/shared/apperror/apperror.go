// Package apperror defines typed domain errors and maps them to HTTP status codes,
// so presentation handlers never hardcode status numbers for business failures.
package apperror

import (
	"errors"
	"net/http"
)

type Kind string

const (
	KindValidation   Kind = "validation"
	KindNotFound     Kind = "not_found"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindConflict     Kind = "conflict"
	KindRateLimited  Kind = "rate_limited"
	KindInternal     Kind = "internal"
	// KindUnsupported: the feature exists but is switched off in this deployment.
	KindUnsupported Kind = "unsupported"
)

// AppError is a domain-level error carrying enough information for the
// presentation layer to map it to an HTTP response without knowing business rules.
type AppError struct {
	Kind    Kind
	Message string
	Fields  map[string][]string // field -> validation messages, only set for KindValidation
	// Code is an optional machine-readable reason (e.g.
	// "trial_limit_reached") the frontend can react to without parsing
	// the human message. Sent as errors.code.
	Code string
	// Data is an optional payload sent next to the error (as `data`). Only a
	// stale-version conflict uses it: the current item (see Stale).
	Data any
}

// WithCode returns a copy of e carrying a machine-readable code.
func (e *AppError) WithCode(code string) *AppError {
	c := *e
	c.Code = code
	return &c
}

// Details is the `errors` payload for a non-validation error: {code} when
// a code is set, otherwise nothing.
func (e *AppError) Details() any {
	if e.Code == "" {
		return nil
	}
	return map[string]string{"code": e.Code}
}

// Payload is the `errors` body for any error: for a validation error the
// per-field messages (plus a `code` string when the failure has a stable
// machine-readable reason, so a form can show the fields and still react to
// the code); for other kinds `{code}` or nothing.
func (e *AppError) Payload() any {
	if e.Kind == KindConflict && len(e.Fields) > 0 {
		return e.Fields
	}
	if e.Kind != KindValidation {
		return e.Details()
	}
	if e.Code == "" {
		return e.Fields
	}
	out := make(map[string]any, len(e.Fields)+1)
	for field, messages := range e.Fields {
		out[field] = messages
	}
	out["code"] = e.Code
	return out
}

func (e *AppError) Error() string {
	return e.Message
}

func NotFound(message string) *AppError {
	return &AppError{Kind: KindNotFound, Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Kind: KindUnauthorized, Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Kind: KindForbidden, Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Kind: KindConflict, Message: message}
}

// StaleMessage is the message of the 409 answered when a write carried an
// If-Match that is no longer the row's version (ADR-0036).
const StaleMessage = "Data ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru."

// Stale is the 409 for a write whose If-Match is out of date. errors is
// {"_conflict": ["stale"]} and `data.current` is the row as it is now: current is
// whatever the service holds (the handler swaps it for the response shape with
// WithData before it is sent).
func Stale(current any) *AppError {
	return &AppError{Kind: KindConflict, Message: StaleMessage, Fields: map[string][]string{"_conflict": {"stale"}}, Data: current}
}

// IsStale reports whether err is a Stale conflict and returns its payload.
func IsStale(err error) (current any, ok bool) {
	if ae, isApp := As(err); isApp && ae.Kind == KindConflict && ae.Data != nil && len(ae.Fields["_conflict"]) > 0 {
		return ae.Data, true
	}
	return nil, false
}

// WithData returns a copy of e carrying another payload.
func (e *AppError) WithData(data any) *AppError {
	c := *e
	c.Data = data
	return &c
}

func RateLimited(message string) *AppError {
	return &AppError{Kind: KindRateLimited, Message: message}
}

func Validation(message string, fields map[string][]string) *AppError {
	return &AppError{Kind: KindValidation, Message: message, Fields: fields}
}

// Unsupported marks something this deployment does not offer (501).
func Unsupported(message string) *AppError {
	return &AppError{Kind: KindUnsupported, Message: message}
}

func Internal(message string) *AppError {
	return &AppError{Kind: KindInternal, Message: message}
}

// HTTPStatus maps an error (typed or not) to the HTTP status the presentation
// layer should respond with.
func HTTPStatus(err error) int {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return http.StatusInternalServerError
	}
	switch appErr.Kind {
	case KindValidation:
		return http.StatusUnprocessableEntity
	case KindNotFound:
		return http.StatusNotFound
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindConflict:
		return http.StatusConflict
	case KindRateLimited:
		return http.StatusTooManyRequests
	case KindUnsupported:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// As extracts the *AppError from err, if any.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
