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
	KindInternal     Kind = "internal"
)

// AppError is a domain-level error carrying enough information for the
// presentation layer to map it to an HTTP response without knowing business rules.
type AppError struct {
	Kind    Kind
	Message string
	Fields  map[string][]string // field -> validation messages, only set for KindValidation
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

func Validation(message string, fields map[string][]string) *AppError {
	return &AppError{Kind: KindValidation, Message: message, Fields: fields}
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
