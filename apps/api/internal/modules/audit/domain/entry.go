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
	// TenantID is set for actions inside a workspace (nil for ElProof's team).
	TenantID *int64
	// Action is a readable label derived from method + path, e.g.
	// "POST /tenants/{id}/toggle-suspension".
	Action     string
	Method     string
	Path       string
	StatusCode int
	IP         string
	UserAgent  string
}

// Filter narrows an audit query (Platform Console "Audit Log").
type Filter struct {
	ActorID   string
	ActorType string
	TenantID  *int64
	// Outcome: "ok" (2xx) or "failed" (anything else); "" for both.
	Outcome string
	// Action keeps only entries with exactly this action label
	// ("POST /tenants/{id}/toggle-suspension").
	Action string
	// Query matches the action or path (substring).
	Query string
	// From is inclusive, To exclusive; zero means unbounded.
	From, To time.Time
}

// Succeeded reports whether the audited request was applied (2xx).
func (e Entry) Succeeded() bool { return e.StatusCode >= 200 && e.StatusCode < 300 }
