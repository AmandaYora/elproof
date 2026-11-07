-- Payment schedule for what the client owes (PLAN.md Fase 4b, FLOW-IDEAL 2.4):
-- the agreed installments (label, amount, due date). Status ("Terlambat",
-- "Dibayar sebagian"...) is never stored: it is worked out on read from the
-- project's real client_payments, so it can never disagree with them.
CREATE TABLE client_payment_terms (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  label VARCHAR(100) NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  due_date DATE NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_client_payment_terms_project (project_id, due_date),
  CONSTRAINT fk_client_payment_terms_project FOREIGN KEY (project_id) REFERENCES projects (id)
);
