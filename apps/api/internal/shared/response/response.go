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

func write(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// OK writes a 200 success envelope.
func OK(w http.ResponseWriter, message string, data interface{}) {
	write(w, http.StatusOK, successBody{Success: true, Message: message, Data: data})
}

// Created writes a 201 success envelope.
func Created(w http.ResponseWriter, message string, data interface{}) {
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
