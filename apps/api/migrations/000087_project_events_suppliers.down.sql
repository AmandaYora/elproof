-- Reverses the shape of 000087, not its data (pre-launch, dummy data): events,
-- suppliers, their steps, payments and evidence are dropped; the old tables come
-- back empty.
DELETE FROM evidence WHERE related_kind IN ('vendorMilestone', 'supplierPayment', 'projectSupplier');
ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment', 'venuePayment', 'general', 'projectMilestone') NOT NULL;

ALTER TABLE vendor_issues
  DROP FOREIGN KEY fk_vendor_issues_supplier,
  DROP FOREIGN KEY fk_vendor_issues_vendor_milestone;
UPDATE vendor_issues SET project_supplier_id = NULL, vendor_milestone_id = NULL;
ALTER TABLE vendor_issues
  DROP KEY idx_vendor_issues_supplier,
  CHANGE COLUMN project_supplier_id project_vendor_id BIGINT UNSIGNED NULL,
  ADD KEY idx_vendor_issues_project_vendor (project_vendor_id);

DROP TABLE supplier_payments;
DROP TABLE vendor_milestones;
DROP TABLE project_event_sessions;
DROP TABLE project_events;
DROP TABLE project_suppliers;

ALTER TABLE projects
  ADD COLUMN venue VARCHAR(255) NOT NULL DEFAULT '' AFTER event_date,
  ADD COLUMN venue_id BIGINT UNSIGNED NULL,
  ADD COLUMN venue_rental_price BIGINT UNSIGNED NULL,
  ADD COLUMN venue_charge BIGINT UNSIGNED NULL;

CREATE TABLE project_vendors (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  vendor_id BIGINT UNSIGNED NOT NULL,
  category_id BIGINT UNSIGNED NOT NULL,
  scope VARCHAR(255) NOT NULL,
  contract_value BIGINT UNSIGNED NOT NULL DEFAULT 0,
  pricing_tier VARCHAR(20) NOT NULL DEFAULT 'Akad',
  engagement_status ENUM('Planned', 'Negotiation', 'Booked', 'DP Paid', 'In Progress', 'Fully Paid', 'Ready', 'Completed', 'Cancelled') NOT NULL DEFAULT 'Planned',
  booking_date DATE NULL,
  event_date DATE NOT NULL,
  event_start_time TIME NULL,
  event_end_time TIME NULL,
  dp_amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
  due_date DATE NULL,
  pic_staff_id BIGINT UNSIGNED NOT NULL,
  notes TEXT NULL,
  deleted_at DATETIME NULL,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_project_vendors_project (project_id),
  KEY idx_project_vendors_vendor (vendor_id),
  KEY idx_project_vendors_deleted (deleted_at),
  CONSTRAINT fk_project_vendors_project FOREIGN KEY (project_id) REFERENCES projects (id)
);

CREATE TABLE vendor_milestones (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_vendor_id BIGINT UNSIGNED NOT NULL,
  sort_order SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  name VARCHAR(150) NOT NULL,
  description TEXT NULL,
  status ENUM('Not Started', 'In Progress', 'Completed', 'Blocked', 'Cancelled') NOT NULL DEFAULT 'Not Started',
  target_date DATE NOT NULL,
  completed_date DATE NULL,
  pic_staff_id BIGINT UNSIGNED NOT NULL,
  notes TEXT NULL,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_vendor_milestones_project_vendor (project_vendor_id),
  CONSTRAINT fk_vendor_milestones_project_vendor FOREIGN KEY (project_vendor_id) REFERENCES project_vendors (id)
);

CREATE TABLE vendor_payments (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  project_vendor_id BIGINT UNSIGNED NOT NULL,
  type ENUM('DP', 'Termin', 'Pelunasan', 'Tambahan', 'Refund') NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  payment_date DATE NOT NULL,
  method VARCHAR(100) NOT NULL,
  reference_number VARCHAR(100) NOT NULL,
  notes TEXT NULL,
  deleted_at DATETIME NULL,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_vendor_payments_project (project_id),
  KEY idx_vendor_payments_project_vendor (project_vendor_id),
  KEY idx_vendor_payments_deleted (deleted_at),
  CONSTRAINT fk_vendor_payments_project FOREIGN KEY (project_id) REFERENCES projects (id),
  CONSTRAINT fk_vendor_payments_project_vendor FOREIGN KEY (project_vendor_id) REFERENCES project_vendors (id)
);

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
  deleted_at DATETIME NULL,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_venue_payments_project (project_id),
  KEY idx_venue_payments_deleted (deleted_at),
  CONSTRAINT fk_venue_payments_project FOREIGN KEY (project_id) REFERENCES projects (id)
);

ALTER TABLE vendor_issues
  ADD CONSTRAINT fk_vendor_issues_project_vendor FOREIGN KEY (project_vendor_id) REFERENCES project_vendors (id),
  ADD CONSTRAINT fk_vendor_issues_vendor_milestone FOREIGN KEY (vendor_milestone_id) REFERENCES vendor_milestones (id);
