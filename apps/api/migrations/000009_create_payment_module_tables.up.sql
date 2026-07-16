-- Module `payment` — "one merchant wallet, many consumers" pattern (see
-- MODULE_PAYMENT.md). None of these tables are a business ledger: they hold
-- gateway config, the App registry, a thin order_ref -> app_id dispatch
-- index, and webhook idempotency only.

CREATE TABLE payment_gateway_config (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  active_provider VARCHAR(50) NULL,
  is_sandbox BOOLEAN NOT NULL DEFAULT TRUE,
  tripay_merchant_code VARCHAR(100) NULL,
  tripay_api_key_encrypted TEXT NULL,
  tripay_private_key_encrypted TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);

-- Single row by application convention (upsert against a known id), not a
-- database constraint — see MODULE_PAYMENT.md §4.
INSERT INTO payment_gateway_config (id, active_provider, is_sandbox) VALUES (1, NULL, TRUE);

CREATE TABLE payment_apps (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  app_id VARCHAR(100) NOT NULL,
  name VARCHAR(150) NOT NULL,
  kind ENUM('internal', 'external') NOT NULL,
  -- external-only columns: secret is stored two ways on purpose (§7.5) —
  -- a one-way hash to verify inbound token exchange, and a reversible copy
  -- used only to sign outbound webhook relays.
  secret_hash VARCHAR(255) NULL,
  secret_encrypted TEXT NULL,
  callback_url VARCHAR(500) NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payment_apps_app_id (app_id)
);

CREATE TABLE payment_charge_dispatch (
  order_ref VARCHAR(100) NOT NULL,
  app_id VARCHAR(100) NOT NULL,
  provider_ref VARCHAR(150) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (order_ref)
);

CREATE TABLE payment_webhook_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  provider VARCHAR(50) NOT NULL,
  event_id VARCHAR(150) NOT NULL,
  received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payment_webhook_events (provider, event_id)
);
