-- Venue Payments (PLAN.md "Venue Payments + Pembayaran tab restructuring"):
-- money going OUT to the project's venue, tracked separately from
-- vendor_payments even though the shape mirrors it (Invoice + Bukti Transfer
-- evidence, same PaymentType enum) -- a project has at most one venue
-- (projects.venue_id), so this ties directly to project_id like
-- client_payments does, with no project_vendor_id-equivalent junction.
CREATE TABLE venue_payments (
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
  KEY idx_venue_payments_project (project_id),
  CONSTRAINT fk_venue_payments_project FOREIGN KEY (project_id) REFERENCES projects (id)
);

-- New 'venuePayment' related_kind, distinct from 'payment' (vendor) so
-- vendor_payments.id and venue_payments.id -- independent AUTO_INCREMENT
-- sequences that can collide in value -- are never ambiguous via related_id.
ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment', 'venuePayment') NOT NULL;
