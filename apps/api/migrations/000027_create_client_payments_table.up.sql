-- Client Payments (PLAN.md "Uang Masuk dari Client"): money coming IN from
-- the client against the project's own Nilai Kontrak, tracked separately
-- from vendor_payments (money going OUT to vendors) -- opposite accounting
-- directions, so a distinct table instead of a "direction" flag on the
-- existing one. No project_vendor_id: this belongs to the project itself,
-- not any vendor.
CREATE TABLE client_payments (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  type ENUM('DP', 'Termin', 'Pelunasan', 'Tambahan', 'Refund') NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  payment_date DATE NOT NULL,
  method VARCHAR(100) NOT NULL,
  reference_number VARCHAR(100) NOT NULL,
  notes TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_client_payments_project (project_id),
  CONSTRAINT fk_client_payments_project FOREIGN KEY (project_id) REFERENCES projects (id)
);

-- New 'clientPayment' related_kind for a client payment's single evidence
-- slot (transfer proof only -- no "invoice" concept for money coming in).
ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment') NOT NULL;
