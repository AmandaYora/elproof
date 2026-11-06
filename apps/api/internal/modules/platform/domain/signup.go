package domain

import (
	"errors"
	"time"
)

// TermsVersion identifies the Syarat & Ketentuan + Perjanjian Pemrosesan Data
// + Kebijakan Privasi text a new WO accepts at signup. Change it whenever
// those pages change materially; it is stored with every consent (UU PDP:
// proof of which text was agreed to, when, and from where).
const TermsVersion = "2026-11-06"

// ErrSignupExists: this Google account already has a signup row.
var ErrSignupExists = errors.New("signup request already exists")

type SignupStatus string

const (
	SignupProvisioning SignupStatus = "provisioning"
	SignupCompleted    SignupStatus = "completed"
)

// SignupRequest is one Google account's self-service registration (PLAN.md
// §3). It carries the consent evidence and drives the provisioning saga:
// TenantID is stored as soon as the tenant row exists, so retrying resumes
// the same workspace.
type SignupRequest struct {
	ID              int64
	GoogleSub       string
	Email           string
	OwnerName       string
	BusinessName    string
	TermsVersion    string
	TermsAcceptedAt time.Time
	MarketingOptIn  bool
	IPAddress       string
	UserAgent       string
	Status          SignupStatus
	TenantID        *int64
	CompletedAt     *time.Time
	CreatedAt       time.Time
}

// TrialLimits caps usage while a tenant is on its free trial (decision K6);
// paid plans are unlimited.
type TrialLimits struct {
	MaxStaff          int
	MaxActiveProjects int
	MaxClients        int
}

// Trial-limited resources (the modules that own each count enforce it).
const (
	ResourceStaff          = "staff"
	ResourceActiveProjects = "active_projects"
	ResourceClients        = "clients"
)

// Limit returns the cap for one resource (0, false for an unknown one).
func (l TrialLimits) Limit(resource string) (int, bool) {
	switch resource {
	case ResourceStaff:
		return l.MaxStaff, true
	case ResourceActiveProjects:
		return l.MaxActiveProjects, true
	case ResourceClients:
		return l.MaxClients, true
	}
	return 0, false
}
