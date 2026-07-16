-- Fase 9: `platform`'s own index remembering, for a charge still awaiting
-- gateway confirmation, which tenant+plan it was for — `payment` itself is
-- never told about tenants/plans (see MODULE_PAYMENT.md's non-goals), so this
-- mapping has to live on the business-module side, not the payment module.
CREATE TABLE pending_subscription_charges (
  order_ref VARCHAR(100) NOT NULL,
  tenant_id BIGINT UNSIGNED NOT NULL,
  plan_id BIGINT UNSIGNED NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (order_ref),
  KEY idx_pending_subscription_charges_tenant (tenant_id)
);

-- `billing` gets a `pending` status: a real charge has been created at the
-- gateway and is awaiting the webhook confirmation, distinct from `unpaid`
-- (an invoice that has never had a charge attempt at all, e.g. right after
-- tenant registration).
ALTER TABLE subscription_transactions
  MODIFY COLUMN status ENUM('unpaid', 'pending', 'paid', 'expired', 'granted') NOT NULL;
