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

// SetTrialQuota completes two-phase wiring (main.go).
func (s *ClientService) SetTrialQuota(q TrialQuota) {
	s.quota = q
}

// checkClientQuota refuses one more active client account once a trialing
// tenant reached its cap.
func (s *ClientService) checkClientQuota(ctx context.Context, tenantID int64) error {
	if s.quota == nil {
		return nil
	}
	limit, limited, err := s.quota.TrialLimit(ctx, tenantID, "clients")
	if err != nil || !limited {
		return err
	}
	n, err := s.repo.CountActive(ctx, tenantID)
	if err != nil {
		return err
	}
	if n >= limit {
		return apperror.Forbidden(fmt.Sprintf(
			"Masa coba dibatasi %d akun client. Pilih paket langganan untuk menambah client tanpa batas.", limit,
		)).WithCode("trial_limit_reached")
	}
	return nil
}

// CountActive is the client usage shown on the onboarding checklist.
func (s *ClientService) CountActive(ctx context.Context, tenantID int64) (int, error) {
	return s.repo.CountActive(ctx, tenantID)
}
