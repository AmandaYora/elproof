// Package domain holds the audit module's single record type.
package domain

import "time"

// Entry is one audited, state-changing request.
type Entry struct {
	ID         int64
	OccurredAt time.Time
	ActorType  string
	ActorID    string
	ActorRole  string
	// Action is a readable label derived from method + path, e.g.
	// "POST /tenants/{id}/toggle-suspension".
	Action     string
	Method     string
	Path       string
	StatusCode int
	IP         string
	UserAgent  string
}

// Succeeded reports whether the audited request was applied (2xx).
func (e Entry) Succeeded() bool { return e.StatusCode >= 200 && e.StatusCode < 300 }
