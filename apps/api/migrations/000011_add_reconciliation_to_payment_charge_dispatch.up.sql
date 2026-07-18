-- Adds the minimum state `payment` needs to reconcile a charge whose webhook
-- was never received (network blip, App down, etc.) — see
-- knowledge/MODULE_PAYMENT.md's reconciliation section. Still not a business
-- ledger (§4): `resolved_at` is a completion marker, not an amount/history.

ALTER TABLE payment_charge_dispatch
  ADD COLUMN expires_at TIMESTAMP NULL AFTER provider_ref,
  ADD COLUMN resolved_at TIMESTAMP NULL AFTER expires_at,
  ADD INDEX idx_payment_charge_dispatch_unresolved (resolved_at, created_at);
