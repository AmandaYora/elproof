-- Subscription ledger for external Apps (A17) -- NOT a reuse of
-- subscription_transactions, whose tenant_id is NOT NULL and therefore
-- cannot carry an App-keyed row without changing the nullability of a
-- live table already used by the WO tenant subscription flow.
--
-- order_ref ties back to payment_charge_dispatch.order_ref as a primitive
-- cross-module ID (no FK, different module). plan_id/plan_name/plan_price/
-- plan_duration_months are a snapshot taken when the intent row is written
-- (before payment.Client is ever called, A20) so a later price change to
-- the catalog never rewrites what a customer already agreed to pay.
--
-- resolved_at is written in the SAME UPDATE as status (A25) by
-- MarkPaidIfUnresolved/MarkResolvedIfUnresolved -- there is no separate
-- "claim" step. `WHERE resolved_at IS NULL` is what gives the reconciler
-- its single-winner guarantee without a claim-then-write window that could
-- otherwise lose an already-paid charge (T18).
CREATE TABLE subscription_app_charges (
  order_ref VARCHAR(100) NOT NULL,
  app_id VARCHAR(100) NOT NULL,
  plan_id BIGINT UNSIGNED NOT NULL,
  plan_name VARCHAR(150) NOT NULL,
  plan_price BIGINT UNSIGNED NOT NULL,
  plan_duration_months INT NOT NULL,
  status ENUM('pending', 'paid', 'expired', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
  subscription_until DATE NULL DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paid_at TIMESTAMP NULL DEFAULT NULL,
  resolved_at TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (order_ref),
  KEY idx_subscription_app_charges_app_status (app_id, status),
  KEY idx_subscription_app_charges_resolved (resolved_at, created_at),
  CONSTRAINT fk_subscription_app_charges_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans (id)
);
