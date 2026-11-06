package application

import (
	"context"
	"fmt"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// TrialQuota tells whether a tenant's free trial caps a resource (PLAN.md
// decision K6: paid plans are unlimited). Declared here, implemented by
// `platform`, wired in cmd/server — this module never imports platform.
// A nil TrialQuota (not wired) means no cap.
type TrialQuota interface {
	TrialLimit(ctx context.Context, tenantID int64, resource string) (limit int, limited bool, err error)
}

// SetTrialQuota completes two-phase wiring (platform is built first, but
// this keeps the dependency pointing from main.go, not from projects).
func (s *ProjectService) SetTrialQuota(q TrialQuota) {
	s.quota = q
}

// isActiveProject: counts toward the trial cap — not finished, not
// cancelled, not archived.
func isActiveProject(status domain.ProjectStatus, archived bool) bool {
	return !archived && status != domain.StatusCompleted && status != domain.StatusCancelled
}

// checkActiveProjectQuota refuses one more active project once a trialing
// tenant reached its cap.
func (s *ProjectService) checkActiveProjectQuota(ctx context.Context, tenantID int64) error {
	if s.quota == nil {
		return nil
	}
	limit, limited, err := s.quota.TrialLimit(ctx, tenantID, "active_projects")
	if err != nil || !limited {
		return err
	}
	n, err := s.repo.CountActive(ctx, tenantID)
	if err != nil {
		return err
	}
	if n >= limit {
		return apperror.Forbidden(fmt.Sprintf(
			"Masa coba dibatasi %d project aktif. Selesaikan atau arsipkan project lain, atau pilih paket langganan untuk project tanpa batas.", limit,
		)).WithCode("trial_limit_reached")
	}
	return nil
}

// CountActive is the active-project usage shown on the onboarding checklist.
func (s *ProjectService) CountActive(ctx context.Context, tenantID int64) (int, error) {
	return s.repo.CountActive(ctx, tenantID)
}
