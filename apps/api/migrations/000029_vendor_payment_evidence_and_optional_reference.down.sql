ALTER TABLE vendor_payments
  ADD COLUMN invoice_evidence_id BIGINT UNSIGNED NULL,
  ADD COLUMN proof_evidence_id BIGINT UNSIGNED NULL;
