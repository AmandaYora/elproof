-- Subscription lifecycle (docs/plan/registrasi-wo-self-service/FLOW-IDEAL.md
-- §1.4). The platform module now derives a tenant's status from its dates
-- (domain.ComputeLifecycle); subscription_status is a cache the status
-- sweeper keeps current for list filters and dashboards.
ALTER TABLE tenants
  MODIFY COLUMN subscription_status
    ENUM('trialing', 'active', 'expiring_soon', 'past_due', 'read_only', 'suspended', 'pending_payment', 'expired')
    NOT NULL DEFAULT 'trialing',
  ADD COLUMN trial_ends_at DATETIME NULL AFTER subscription_expires_at,
  ADD KEY idx_tenants_subscription_status (subscription_status);

-- Every tenant that has never paid gets a fresh 14-day trial instead of the
-- old open-ended "pending_payment" (which never restricted anything). All
-- current tenants are test data (product decision, 2026-11-06).
UPDATE tenants
SET trial_ends_at = DATE_ADD(NOW(), INTERVAL 14 DAY),
    subscription_status = 'trialing'
WHERE subscription_expires_at IS NULL;
