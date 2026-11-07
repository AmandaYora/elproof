// Package application records and lists audit entries.
package application

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/audit/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
)

type Repository interface {
	Insert(ctx context.Context, e *domain.Entry) error
	ListPaginated(ctx context.Context, params pagination.Params, f domain.Filter) ([]domain.Entry, int64, error)
	// ListAll returns up to `limit` matches, newest first (CSV export).
	ListAll(ctx context.Context, f domain.Filter, limit int) ([]domain.Entry, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Record implements middleware.AuditRecorder.
func (s *Service) Record(ctx context.Context, rec middleware.AuditRecord) error {
	e := &domain.Entry{
		OccurredAt: s.now(),
		ActorType:  rec.ActorType,
		ActorID:    rec.ActorID,
		ActorRole:  rec.ActorRole,
		TenantID:   tenantOf(rec),
		Action:     ActionLabel(rec.Method, rec.Path),
		Method:     rec.Method,
		Path:       truncate(rec.Path, 255),
		StatusCode: rec.StatusCode,
		IP:         truncate(rec.IP, 64),
		UserAgent:  truncate(rec.UserAgent, 255),
	}
	return s.repo.Insert(ctx, e)
}

// tenantOf is the workspace an entry belongs to: the actor's own for WO staff
// and clients, and for a platform admin the tenant named in the path
// (/tenants/{id}/... or /console/tenants/{id}/...), so "everything done to
// this tenant" includes what ElProof's team did to it.
func tenantOf(rec middleware.AuditRecord) *int64 {
	if id := parseTenant(rec.TenantID); id != nil {
		return id
	}
	if rec.ActorType != "platform_admin" {
		return nil
	}
	segs := strings.Split(strings.Trim(strings.TrimPrefix(rec.Path, "/api/v1"), "/"), "/")
	if len(segs) >= 3 && segs[0] == "console" {
		segs = segs[1:]
	}
	if len(segs) >= 2 && segs[0] == "tenants" {
		if id, err := strconv.ParseInt(segs[1], 10, 64); err == nil && id > 0 {
			return &id
		}
	}
	return nil
}

func (s *Service) List(ctx context.Context, params pagination.Params, f domain.Filter) ([]domain.Entry, int64, error) {
	return s.repo.ListPaginated(ctx, params, f)
}

// ExportLimit caps one CSV export; narrow the filter (dates) for more.
const ExportLimit = 10000

// Export returns the matching entries for a CSV download.
func (s *Service) Export(ctx context.Context, f domain.Filter) ([]domain.Entry, error) {
	return s.repo.ListAll(ctx, f, ExportLimit)
}

var idSegment = regexp.MustCompile(`^(\d+|app_[0-9a-f]+)$`)

// ActionLabel turns "/api/v1/tenants/12/toggle-suspension" into
// "POST /tenants/{id}/toggle-suspension" so entries can be grouped by action.
func ActionLabel(method, path string) string {
	segs := strings.Split(strings.Trim(strings.TrimPrefix(path, "/api/v1"), "/"), "/")
	for i, s := range segs {
		if idSegment.MatchString(s) {
			segs[i] = "{id}"
		}
	}
	return truncate(method+" /"+strings.Join(segs, "/"), 120)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func parseTenant(v string) *int64 {
	if v == "" {
		return nil
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}
