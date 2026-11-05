package application

import (
	"context"
	"sync"
	"time"

	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
)

// accessCacheTTL bounds how stale the per-request access gate can be. Every
// change made through this service (pay, manual activation, freeze) also
// invalidates the entry immediately, so the TTL only matters for time
// simply passing (a trial ending), where a minute's delay is harmless.
const accessCacheTTL = time.Minute

type accessEntry struct {
	lifecycle domain.Lifecycle
	found     bool
	at        time.Time
}

type accessCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[int64]accessEntry
	// gen counts invalidations per tenant. A lookup remembers the
	// generation it started with and only stores its result if nothing was
	// invalidated meanwhile — otherwise a read that began before a freeze
	// committed could re-cache the un-frozen state for a whole TTL.
	gen map[int64]uint64
}

func newAccessCache(ttl time.Duration) *accessCache {
	return &accessCache{ttl: ttl, entries: make(map[int64]accessEntry), gen: make(map[int64]uint64)}
}

func (c *accessCache) generation(id int64) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gen[id]
}

func (c *accessCache) get(id int64, now time.Time) (accessEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[id]
	if !ok || now.Sub(e.at) > c.ttl {
		return accessEntry{}, false
	}
	return e, true
}

// put stores e unless the tenant was invalidated after generation g.
func (c *accessCache) put(id int64, e accessEntry, g uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gen[id] != g {
		return
	}
	c.entries[id] = e
}

func (c *accessCache) invalidate(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, id)
	c.gen[id]++
}

// Lifecycle returns a tenant's subscription state right now.
func (s *TenantService) Lifecycle(t domain.Tenant) domain.Lifecycle {
	return domain.ComputeLifecycle(t, time.Now(), domain.DefaultSubscriptionPolicy)
}

func (s *TenantService) cachedLifecycle(ctx context.Context, tenantID int64) (domain.Lifecycle, bool, error) {
	now := time.Now()
	if e, ok := s.access.get(tenantID, now); ok {
		return e.lifecycle, e.found, nil
	}
	g := s.access.generation(tenantID)
	tenant, err := s.repo.FindByID(ctx, tenantID)
	if err != nil {
		return domain.Lifecycle{}, false, err
	}
	e := accessEntry{found: tenant != nil, at: now}
	if tenant != nil {
		e.lifecycle = domain.ComputeLifecycle(*tenant, now, domain.DefaultSubscriptionPolicy)
	}
	s.access.put(tenantID, e, g)
	return e.lifecycle, e.found, nil
}

// TenantAccessLevel implements middleware.TenantAccessResolver (wired in
// cmd/server): the access level the gate enforces for a tenant's requests.
// A tenant id that no longer exists is treated as blocked.
func (s *TenantService) TenantAccessLevel(ctx context.Context, tenantID int64) (string, error) {
	lc, found, err := s.cachedLifecycle(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if !found {
		return string(domain.AccessBlocked), nil
	}
	return string(lc.Access), nil
}

// CheckTenantLogin implements identity's TenantGuard: a tenant frozen by the
// platform team cannot log in or refresh a session. A billing downgrade does
// NOT block login — staff must still be able to sign in to pay, and the
// couple keeps reading the portal; the access gate handles the rest.
func (s *TenantService) CheckTenantLogin(ctx context.Context, tenantID int64) error {
	lc, found, err := s.cachedLifecycle(ctx, tenantID)
	if err != nil {
		return err
	}
	if !found {
		return apperror.Unauthorized("Akun ini tidak lagi terdaftar")
	}
	if lc.Frozen {
		return apperror.Forbidden("Akun WO ini sedang dibekukan oleh ElProof. Hubungi tim ElProof untuk bantuan.")
	}
	return nil
}

// SweepSubscriptionStatuses refreshes the cached subscription_status column
// of every tenant whose computed lifecycle status changed (list filters and
// the platform dashboard read that column). Access itself never depends on
// this sweep — see ComputeLifecycle.
func (s *TenantService) SweepSubscriptionStatuses(ctx context.Context) (changed int, err error) {
	tenants, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	for _, t := range tenants {
		lc := domain.ComputeLifecycle(t, now, domain.DefaultSubscriptionPolicy)
		if lc.Status == t.SubscriptionStatus {
			continue
		}
		if err := s.repo.SetSubscriptionStatus(ctx, t.ID, lc.Status); err != nil {
			return changed, err
		}
		s.access.invalidate(t.ID)
		changed++
	}
	return changed, nil
}

// StartSubscriptionStatusSweeper runs SweepSubscriptionStatuses once now and
// then every interval for the life of the process (same in-process pattern
// as the payment/billing reconcilers, ADR-0011).
func (s *TenantService) StartSubscriptionStatusSweeper(ctx context.Context, interval time.Duration) {
	sweep := func() {
		n, err := s.SweepSubscriptionStatuses(ctx)
		if err != nil {
			logger.Error("platform: subscription status sweep failed: %v", err)
			return
		}
		if n > 0 {
			logger.Info("platform: subscription status sweep updated %d tenant(s)", n)
		}
	}
	go func() {
		sweep()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweep()
			}
		}
	}()
}
