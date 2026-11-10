-- T-20261110-17: discount vouchers for subscription payments, owned by
-- `billing` (ADR-0041). A voucher belongs to exactly one App (the App whose
-- plans it discounts, same key as subscription_plans.app_id, ADR-0040) and
-- may be narrowed to one of that App's plans.
--
-- app_id is a primitive cross-module ID (no FK to payment_apps), binary
-- collation so SQL compares like Go. plan_id points at this module's own
-- table, so it keeps a real FK. code is stored upper case and compared
-- exactly (binary): the service upper-cases every input before reading.
CREATE TABLE subscription_vouchers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  app_id VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  plan_id BIGINT UNSIGNED NULL DEFAULT NULL,
  discount_type ENUM('percent', 'amount') NOT NULL,
  -- percent: 1..99; amount: rupiah.
  discount_value BIGINT UNSIGNED NOT NULL,
  -- Cap on a percent discount, in rupiah (NULL = no cap; always NULL for amount).
  max_discount BIGINT UNSIGNED NULL DEFAULT NULL,
  -- Inclusive calendar days in WIB (NULL = open-ended).
  valid_from DATE NULL DEFAULT NULL,
  valid_until DATE NULL DEFAULT NULL,
  -- Total uses (NULL = unlimited); a reserved use (charge still pending) counts.
  quota INT UNSIGNED NULL DEFAULT NULL,
  once_per_customer BOOLEAN NOT NULL DEFAULT FALSE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_subscription_vouchers_code (code),
  KEY idx_subscription_vouchers_app (app_id),
  CONSTRAINT fk_subscription_vouchers_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans (id)
);

-- One row per use of a voucher on one subscription charge. A use is
-- `reserved` while its charge is pending, `used` once paid and `released`
-- when the charge expires, fails or is cancelled (it then no longer counts
-- against the quota or the once-per-customer rule). Reservations are taken
-- under a row lock on the voucher, so two charges at once can never both
-- take the last unit of a quota.
--
-- customer_type/customer_ref name who used it: a WO tenant ('tenant', its
-- id) or an external App ('app', its app_id) -- primitive IDs, no FK.
-- order_ref is the charge's reference: subscription_transactions.
-- payment_reference for a tenant, subscription_app_charges.order_ref for an App.
CREATE TABLE subscription_voucher_redemptions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  voucher_id BIGINT UNSIGNED NOT NULL,
  voucher_code VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  order_ref VARCHAR(100) NOT NULL,
  customer_type ENUM('tenant', 'app') NOT NULL,
  customer_ref VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  plan_id BIGINT UNSIGNED NOT NULL,
  original_amount BIGINT UNSIGNED NOT NULL,
  discount_amount BIGINT UNSIGNED NOT NULL,
  final_amount BIGINT UNSIGNED NOT NULL,
  status ENUM('reserved', 'used', 'released') NOT NULL DEFAULT 'reserved',
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  resolved_at TIMESTAMP(6) NULL DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_voucher_redemptions_order (order_ref),
  KEY idx_voucher_redemptions_voucher (voucher_id, status),
  KEY idx_voucher_redemptions_customer (voucher_id, customer_type, customer_ref, status),
  KEY idx_voucher_redemptions_stale (status, created_at),
  CONSTRAINT fk_voucher_redemptions_voucher FOREIGN KEY (voucher_id) REFERENCES subscription_vouchers (id)
);

-- Both ledgers record what was charged before and after the discount.
-- amount stays the amount actually charged (revenue reports keep reading it).
ALTER TABLE subscription_transactions
  ADD COLUMN original_amount BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER amount,
  ADD COLUMN discount_amount BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER original_amount,
  ADD COLUMN voucher_code VARCHAR(32) NULL DEFAULT NULL AFTER discount_amount;
UPDATE subscription_transactions SET original_amount = amount;

-- plan_price is the snapshot of the plan's price (the original amount);
-- amount is what the gateway was asked to charge.
ALTER TABLE subscription_app_charges
  ADD COLUMN discount_amount BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER plan_price,
  ADD COLUMN amount BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER discount_amount,
  ADD COLUMN voucher_code VARCHAR(32) NULL DEFAULT NULL AFTER amount;
UPDATE subscription_app_charges SET amount = plan_price;
