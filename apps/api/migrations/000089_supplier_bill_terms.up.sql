-- Paket WO Tahap 4 (docs/plan/paket-wo/PLAN.md §3.2, §4.4, §8 Tahap 4,
-- ADR-0047): a vendor or venue the client pays directly ("dibayar klien") gets
-- its own bills (label, amount, due date), and the couple can send proof of a
-- payment to it from the portal. A sent proof is an unverified supplier
-- payment (submitted_by_client_id set); it counts as paid only after the WO
-- marks it right (verified_at), and the WO can refuse it with a reason
-- (rejected_at). These payments are the client's money: they never enter the
-- WO's cash, revenue, cash flow or payables (those read payer = 'WO' only).
--
-- Every new table carries tenant_id (ADR-0004); the foreign keys stay inside
-- projects.

CREATE TABLE supplier_bill_terms (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  project_id BIGINT UNSIGNED NOT NULL,
  -- Only a supplier with paid_by = 'Client' (checked by the service).
  project_supplier_id BIGINT UNSIGNED NOT NULL,
  label VARCHAR(100) NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  due_date DATE NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_supplier_bill_terms_project (tenant_id, project_id),
  KEY idx_supplier_bill_terms_supplier (tenant_id, project_supplier_id),
  KEY idx_supplier_bill_terms_due (tenant_id, due_date),
  KEY idx_supplier_bill_terms_deleted (deleted_at),
  CONSTRAINT fk_supplier_bill_terms_project FOREIGN KEY (project_id) REFERENCES projects (id),
  CONSTRAINT fk_supplier_bill_terms_supplier FOREIGN KEY (project_supplier_id) REFERENCES project_suppliers (id)
);

ALTER TABLE supplier_payments
  -- The bill a client's proof pays (optional for a payment the WO records).
  ADD COLUMN bill_term_id BIGINT UNSIGNED NULL AFTER project_supplier_id,
  -- Set when the client sent the proof from the portal (ADR-0047); a primitive
  -- id into clients, no foreign key.
  ADD COLUMN submitted_by_client_id BIGINT UNSIGNED NULL AFTER recorded_by_staff_id,
  ADD COLUMN verified_at DATETIME NULL AFTER submitted_by_client_id,
  ADD COLUMN verified_by_staff_id BIGINT UNSIGNED NULL AFTER verified_at,
  ADD COLUMN rejected_at DATETIME NULL AFTER verified_by_staff_id,
  ADD COLUMN rejected_by_staff_id BIGINT UNSIGNED NULL AFTER rejected_at,
  ADD COLUMN rejection_reason VARCHAR(500) NULL AFTER rejected_by_staff_id,
  ADD KEY idx_supplier_payments_bill_term (tenant_id, bill_term_id),
  ADD KEY idx_supplier_payments_pending (tenant_id, payer, verified_at, rejected_at),
  ADD CONSTRAINT fk_supplier_payments_bill_term FOREIGN KEY (bill_term_id) REFERENCES supplier_bill_terms (id) ON DELETE SET NULL;
