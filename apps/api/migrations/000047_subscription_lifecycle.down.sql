UPDATE tenants SET subscription_status = 'pending_payment'
WHERE subscription_status IN ('trialing');
UPDATE tenants SET subscription_status = 'expired'
WHERE subscription_status IN ('past_due', 'read_only', 'suspended');

ALTER TABLE tenants
  DROP KEY idx_tenants_subscription_status,
  DROP COLUMN trial_ends_at,
  MODIFY COLUMN subscription_status
    ENUM('active', 'expiring_soon', 'expired', 'pending_payment')
    NOT NULL DEFAULT 'pending_payment';
