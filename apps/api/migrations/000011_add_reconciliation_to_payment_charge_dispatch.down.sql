ALTER TABLE payment_charge_dispatch
  DROP INDEX idx_payment_charge_dispatch_unresolved,
  DROP COLUMN resolved_at,
  DROP COLUMN expires_at;
