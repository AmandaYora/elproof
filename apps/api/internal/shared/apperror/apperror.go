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
