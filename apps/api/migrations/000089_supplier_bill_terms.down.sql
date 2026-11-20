-- Reverses the shape of 000089, not its data (pre-launch, dummy data): the
-- client's sent proofs stay as plain client-paid supplier payments.
ALTER TABLE supplier_payments
  DROP FOREIGN KEY fk_supplier_payments_bill_term,
  DROP KEY idx_supplier_payments_bill_term,
  DROP KEY idx_supplier_payments_pending,
  DROP COLUMN rejection_reason,
  DROP COLUMN rejected_by_staff_id,
  DROP COLUMN rejected_at,
  DROP COLUMN verified_by_staff_id,
  DROP COLUMN verified_at,
  DROP COLUMN submitted_by_client_id,
  DROP COLUMN bill_term_id;

DROP TABLE supplier_bill_terms;
