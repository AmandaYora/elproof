// Package response builds the project-wide API envelope:
// success: {success, message, data}; error: {success:false, message, errors}.
// See .claude/rules/api-standard.md.
package response

import (
	"encoding/json"
	"net/http"

	"elproof/internal/shared/pagination"
)

type successBody struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type successBodyPaginated struct {
	Success bool               `json:"success"`
	Message string             `json:"message"`
	Data    interface{}        `json:"data"`
	Meta    pagination.Meta    `json:"meta"`
}

type errorBody struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

// errorBodyWithData is the error envelope plus `data` (the 409 of a stale write
// carries the current item there).
type errorBodyWithData struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
	Data    interface{} `json:"data"`
}

// Versioned is implemented by a response item that carries a version (the
// `updatedAt` of ADR-0036): OK and Created then set the ETag header from it, so
// every module's items answer the same way without each handler remembering to.
type Versioned interface {
	// ETag is the quoted validator, or "" when the item has no version.
	ETag() string
}

func setValidator(w http.ResponseWriter, data interface{}) {
	if v, ok := data.(Versioned); ok {
		if tag := v.ETag(); tag != "" {
			w.Header().Set("ETag", tag)
		}
	}
}

func write(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// OK writes a 200 success envelope.
func OK(w http.ResponseWriter, message string, data interface{}) {
	setValidator(w, data)
	write(w, http.StatusOK, successBody{Success: true, Message: message, Data: data})
}

// Created writes a 201 success envelope.
func Created(w http.ResponseWriter, message string, data interface{}) {
	setValidator(w, data)
	write(w, http.StatusCreated, successBody{Success: true, Message: message, Data: data})
}

// OKPaginated writes a 200 success envelope with pagination meta.
func OKPaginated(w http.ResponseWriter, message string, data interface{}, meta pagination.Meta) {
	write(w, http.StatusOK, successBodyPaginated{Success: true, Message: message, Data: data, Meta: meta})
}

// Error writes an error envelope at the given HTTP status.
func Error(w http.ResponseWriter, status int, message string, errors interface{}) {
	write(w, status, errorBody{Success: false, Message: message, Errors: errors})
}

// ErrorWithData writes an error envelope that also carries `data`.
func ErrorWithData(w http.ResponseWriter, status int, message string, errors interface{}, data interface{}) {
	write(w, status, errorBodyWithData{Success: false, Message: message, Errors: errors, Data: data})
}
