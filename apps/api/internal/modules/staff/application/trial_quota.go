package application

import (
	"context"
	"fmt"

	"elproof/internal/shared/apperror"
)

// TrialQuota tells whether a tenant's free trial caps a resource (PLAN.md
// decision K6: paid plans are unlimited). Declared here, implemented by
// `platform`, wired in cmd/server — this module never imports platform.
// A nil TrialQuota (not wired) means no cap.
type TrialQuota interface {
	TrialLimit(ctx context.Context, tenantID int64, resource string) (limit int, limited bool, err error)
}

// SetTrialQuota completes two-phase wiring (platform is built after staff).
func (s *StaffService) SetTrialQuota(q TrialQuota) {
	s.quota = q
}

// checkStaffQuota refuses one more active staff member (besides the Owner)
// once a trialing tenant reached its cap.
func (s *StaffService) checkStaffQuota(ctx context.Context, tenantID int64) error {
	if s.quota == nil {
		return nil
	}
	limit, limited, err := s.quota.TrialLimit(ctx, tenantID, "staff")
	if err != nil || !limited {
		return err
	}
	n, err := s.repo.CountActiveNonOwner(ctx, tenantID)
	if err != nil {
		return err
	}
	if n >= limit {
		return apperror.Forbidden(fmt.Sprintf(
			"Masa coba dibatasi %d staf selain Owner. Pilih paket langganan untuk menambah staf tanpa batas.", limit,
		)).WithCode("trial_limit_reached")
	}
	return nil
}

// CountActiveNonOwner is the staff usage shown on the onboarding checklist.
func (s *StaffService) CountActiveNonOwner(ctx context.Context, tenantID int64) (int, error) {
	return s.repo.CountActiveNonOwner(ctx, tenantID)
}

// FindOwner returns the tenant's Owner (nil when none exists yet).
func (s *StaffService) FindOwner(ctx context.Context, tenantID int64) (*int64, error) {
	return s.repo.FindOwnerID(ctx, tenantID)
}
