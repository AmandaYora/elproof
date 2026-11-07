package domain

import (
	"strconv"
	"sync"
	"time"
)

// SignupConsent is the consent evidence stored at self-service signup (UU PDP).
type SignupConsent struct {
	TermsVersion    string
	TermsAcceptedAt time.Time
	MarketingOptIn  bool
	IPAddress       string
}

// ---- Subscription policy (editable, FLOW-IDEAL 1.3 "Kebijakan langganan") ----

var (
	policyMu      sync.RWMutex
	currentPolicy = DefaultSubscriptionPolicy
)

// CurrentSubscriptionPolicy is the lifecycle policy in force: the defaults
// until the platform team edits it (loaded at startup and after each edit).
func CurrentSubscriptionPolicy() SubscriptionPolicy {
	policyMu.RLock()
	defer policyMu.RUnlock()
	return currentPolicy
}

// SetSubscriptionPolicy replaces the policy in force.
func SetSubscriptionPolicy(p SubscriptionPolicy) {
	policyMu.Lock()
	defer policyMu.Unlock()
	currentPolicy = p
}

// ValidatePolicy returns a message per invalid field (empty when fine).
// Nothing here ever deletes data: product decision K10.
func ValidatePolicy(p SubscriptionPolicy) map[string][]string {
	fields := map[string][]string{}
	check := func(key string, v, lo, hi int, label string) {
		if v < lo || v > hi {
			fields[key] = []string{label + " harus antara " + strconv.Itoa(lo) + " dan " + strconv.Itoa(hi) + " hari"}
		}
	}
	check("trialDays", p.TrialDays, 1, 90, "Masa coba")
	check("expiringSoonDays", p.ExpiringSoonDays, 0, 30, "Pengingat sebelum berakhir")
	check("graceDays", p.GraceDays, 0, 60, "Masa tenggang")
	check("readOnlyDays", p.ReadOnlyDays, 0, 180, "Masa baca-saja")
	return fields
}

// ---- Internal notes, data requests ----

// TenantNote is a note only ElProof's team can see.
type TenantNote struct {
	ID        int64
	TenantID  int64
	AdminID   string
	AdminName string
	Body      string
	CreatedAt time.Time
}

type DataRequestKind string

const (
	DataRequestExport DataRequestKind = "export"
	DataRequestDelete DataRequestKind = "delete"
)

type DataRequestStatus string

const (
	DataRequestOpen     DataRequestStatus = "open"
	DataRequestDone     DataRequestStatus = "done"
	DataRequestRejected DataRequestStatus = "rejected"
)

// DataRequestDeadlineDays is how long the platform has to answer (UU PDP).
const DataRequestDeadlineDays = 30

// DataRequest is a workspace's request to export or delete its data.
type DataRequest struct {
	ID             int64
	TenantID       int64
	TenantName     string
	Kind           DataRequestKind
	Status         DataRequestStatus
	RequestedBy    string
	Note           string
	DueAt          time.Time
	CreatedAt      time.Time
	CompletedAt    *time.Time
	CompletedBy    string
	CompletionNote string
}

// DaysUntilDue is calendar days to the deadline (negative once passed).
func (d DataRequest) DaysUntilDue(now time.Time) int { return daysUntil(now, d.DueAt) }

// ---- Announcements ----

type AnnouncementKind string

const (
	AnnouncementInfo        AnnouncementKind = "info"
	AnnouncementWarning     AnnouncementKind = "warning"
	AnnouncementMaintenance AnnouncementKind = "maintenance"
)

type AnnouncementAudience string

const (
	AudienceAll    AnnouncementAudience = "all"
	AudiencePlan   AnnouncementAudience = "plan"   // AudienceValue = plan id
	AudienceStatus AnnouncementAudience = "status" // AudienceValue = lifecycle status
)

// Announcement is an in-app message shown to workspace staff.
type Announcement struct {
	ID            int64
	Title         string
	Body          string
	Kind          AnnouncementKind
	Audience      AnnouncementAudience
	AudienceValue string
	StartsAt      time.Time
	EndsAt        *time.Time
	CreatedBy     string
	CreatedAt     time.Time
}

// ActiveAt reports whether the announcement is on air at `now`.
func (a Announcement) ActiveAt(now time.Time) bool {
	if now.Before(a.StartsAt) {
		return false
	}
	return a.EndsAt == nil || now.Before(*a.EndsAt)
}

// Targets reports whether a workspace should see the announcement.
func (a Announcement) Targets(t Tenant, lc Lifecycle) bool {
	switch a.Audience {
	case AudiencePlan:
		return t.PlanID != nil && strconv.FormatInt(*t.PlanID, 10) == a.AudienceValue
	case AudienceStatus:
		return string(lc.Status) == a.AudienceValue
	default:
		return true
	}
}

// ---- Support access ----

// SupportAccessGrant is the Owner's consent for ElProof's support team to
// view the workspace read-only.
type SupportAccessGrant struct {
	ID               int64
	TenantID         int64
	GrantedByStaffID int64
	GrantedAt        time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
}

// SupportAccessHours is how long a grant lasts; SupportSessionMinutes how
// long one viewing session may run (FLOW-IDEAL 1.5).
const (
	SupportAccessHours    = 24
	SupportSessionMinutes = 60
)

// ActiveAt: granted, not revoked and not expired.
func (g SupportAccessGrant) ActiveAt(now time.Time) bool {
	return g.RevokedAt == nil && now.Before(g.ExpiresAt)
}

// SupportSession is one "Lihat sebagai tenant" session.
type SupportSession struct {
	ID        int64
	TenantID  int64
	AdminID   string
	AdminName string
	Reason    string
	StartedAt time.Time
	ExpiresAt time.Time
	EndedAt   *time.Time
}
