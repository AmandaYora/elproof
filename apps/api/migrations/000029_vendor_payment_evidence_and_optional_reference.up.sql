-- Retires the two direct-FK evidence columns on vendor_payments (PLAN.md
-- "Payment evidence (Invoice/Bukti Transfer)..."). Confirmed via a full-repo
-- search: no code path has ever written a non-null value to either column --
-- completeness is computed instead by cross-referencing the polymorphic
-- `evidence` table (related_kind = 'payment', already in the ENUM since the
-- first migration, distinguished by evidence.type Invoice vs Transfer
-- Proof), the same pattern already used for Client Payments and the vendor
-- paid-amount fix.
ALTER TABLE vendor_payments
  DROP COLUMN invoice_evidence_id,
  DROP COLUMN proof_evidence_id;
