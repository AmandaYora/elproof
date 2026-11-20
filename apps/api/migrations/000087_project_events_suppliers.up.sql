-- Paket WO Tahap 2 (docs/plan/paket-wo/PLAN.md §3, ADR-0048): a project gets
-- events (acara) with sessions, and "Vendor & venue" (project_suppliers) replaces
-- the vendor engagement plus the single project venue. Payments to a supplier
-- live in supplier_payments with their own payer.
--
-- ElProof is pre-launch with dummy data (PLAN §0): the old engagements, vendor
-- payments, venue payments, vendor milestones and their evidence are dropped,
-- not converted. Kendala stay, as kendala about the event in general.
-- Every new table carries tenant_id (ADR-0004). event_kind_id, supplier_id,
-- category_id and pic_staff_id are primitive ids into other modules, with no
-- foreign key; foreign keys stay inside projects.

-- 1. Evidence of the dropped tables, and the links kendala had to them.
DELETE FROM evidence WHERE related_kind IN ('vendorMilestone', 'payment', 'projectVendor', 'venuePayment');

ALTER TABLE vendor_issues
  DROP FOREIGN KEY fk_vendor_issues_project_vendor,
  DROP FOREIGN KEY fk_vendor_issues_vendor_milestone;
UPDATE vendor_issues SET project_vendor_id = NULL, vendor_milestone_id = NULL;

DROP TABLE vendor_payments;
DROP TABLE venue_payments;
DROP TABLE vendor_milestones;
DROP TABLE project_vendors;

-- 2. The project's own venue is now a supplier of an event.
ALTER TABLE projects
  DROP COLUMN venue,
  DROP COLUMN venue_id,
  DROP COLUMN venue_rental_price,
  DROP COLUMN venue_charge;

-- 3. Vendor & venue of a project.
CREATE TABLE project_suppliers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  project_id BIGINT UNSIGNED NOT NULL,
  supplier_kind ENUM('vendor', 'venue') NOT NULL,
  supplier_id BIGINT UNSIGNED NOT NULL,
  -- Copied when chosen: the directory row may be renamed or deleted later.
  supplier_name VARCHAR(150) NOT NULL,
  category_id BIGINT UNSIGNED NULL,
  paid_by ENUM('WO', 'Client') NOT NULL DEFAULT 'WO',
  -- Only for a vendor paid by the WO; a venue or a supplier the client pays is always named.
  show_name BOOLEAN NOT NULL DEFAULT FALSE,
  -- Venue only: copied from the directory, refreshed by "Perbarui dari direktori".
  capacity_standing INT UNSIGNED NULL,
  capacity_seated INT UNSIGNED NULL,
  -- Biaya rekanan: only for a supplier paid by the WO, optional, margin permission only.
  agreed_cost BIGINT UNSIGNED NULL,
  engagement_status ENUM('Planned', 'Negotiation', 'Booked', 'DP Paid', 'In Progress', 'Fully Paid', 'Ready', 'Completed', 'Cancelled') NOT NULL DEFAULT 'Planned',
  booking_date DATE NULL,
  due_date DATE NULL,
  pic_staff_id BIGINT UNSIGNED NULL,
  scope VARCHAR(255) NOT NULL DEFAULT '',
  notes TEXT NULL,
  deleted_at DATETIME NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_project_suppliers_project (tenant_id, project_id),
  KEY idx_project_suppliers_supplier (tenant_id, supplier_kind, supplier_id),
  KEY idx_project_suppliers_deleted (deleted_at),
  CONSTRAINT fk_project_suppliers_project FOREIGN KEY (project_id) REFERENCES projects (id)
);

-- 4. Events and their sessions.
CREATE TABLE project_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  project_id BIGINT UNSIGNED NOT NULL,
  event_kind_id BIGINT UNSIGNED NOT NULL,
  -- Copied: renaming the kind does not change a project.
  kind_name VARCHAR(60) NOT NULL,
  event_date DATE NOT NULL,
  start_time CHAR(5) NULL,
  end_time CHAR(5) NULL,
  venue_supplier_id BIGINT UNSIGNED NULL,
  layout ENUM('Standing', 'Seated') NOT NULL DEFAULT 'Standing',
  -- Acara utama: its date is the project's event_date (Hari H of the schedule). One per project, kept by the service.
  is_main BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_project_events_project (tenant_id, project_id),
  KEY idx_project_events_date (tenant_id, event_date),
  KEY idx_project_events_kind (tenant_id, event_kind_id),
  CONSTRAINT fk_project_events_project FOREIGN KEY (project_id) REFERENCES projects (id),
  CONSTRAINT fk_project_events_venue FOREIGN KEY (venue_supplier_id) REFERENCES project_suppliers (id)
);

CREATE TABLE project_event_sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  event_id BIGINT UNSIGNED NOT NULL,
  label VARCHAR(60) NOT NULL,
  start_time CHAR(5) NULL,
  end_time CHAR(5) NULL,
  guests INT UNSIGNED NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_project_event_sessions_event (tenant_id, event_id),
  CONSTRAINT fk_project_event_sessions_event FOREIGN KEY (event_id) REFERENCES project_events (id) ON DELETE CASCADE
);

-- 5. Progress steps of a supplier (shown as "Tahapan vendor").
CREATE TABLE vendor_milestones (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  project_supplier_id BIGINT UNSIGNED NOT NULL,
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
  KEY idx_vendor_milestones_supplier (tenant_id, project_supplier_id),
  CONSTRAINT fk_vendor_milestones_supplier FOREIGN KEY (project_supplier_id) REFERENCES project_suppliers (id)
);

-- 6. Money to (or, for a supplier the client pays, from the client to) a supplier.
-- payer is the row's own: queries filter on it, never on the supplier's paid_by,
-- so an old payment never changes meaning.
CREATE TABLE supplier_payments (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  project_id BIGINT UNSIGNED NOT NULL,
  project_supplier_id BIGINT UNSIGNED NOT NULL,
  payer ENUM('WO', 'Client') NOT NULL,
  type ENUM('DP', 'Termin', 'Pelunasan', 'Tambahan', 'Refund') NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  payment_date DATE NOT NULL,
  method VARCHAR(100) NOT NULL,
  reference_number VARCHAR(100) NOT NULL DEFAULT '',
  notes TEXT NULL,
  recorded_by_staff_id BIGINT UNSIGNED NULL,
  deleted_at DATETIME NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_supplier_payments_project (tenant_id, project_id, payer),
  KEY idx_supplier_payments_supplier (tenant_id, project_supplier_id),
  KEY idx_supplier_payments_date (tenant_id, payment_date),
  KEY idx_supplier_payments_deleted (deleted_at),
  CONSTRAINT fk_supplier_payments_project FOREIGN KEY (project_id) REFERENCES projects (id),
  CONSTRAINT fk_supplier_payments_supplier FOREIGN KEY (project_supplier_id) REFERENCES project_suppliers (id)
);

-- 7. Kendala point at a supplier and one of its steps.
ALTER TABLE vendor_issues
  DROP KEY idx_vendor_issues_project_vendor,
  CHANGE COLUMN project_vendor_id project_supplier_id BIGINT UNSIGNED NULL,
  ADD KEY idx_vendor_issues_supplier (project_supplier_id),
  ADD CONSTRAINT fk_vendor_issues_supplier FOREIGN KEY (project_supplier_id) REFERENCES project_suppliers (id),
  ADD CONSTRAINT fk_vendor_issues_vendor_milestone FOREIGN KEY (vendor_milestone_id) REFERENCES vendor_milestones (id);

-- 8. Evidence kinds: projectSupplier and supplierPayment replace projectVendor, payment and venuePayment.
ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'supplierPayment', 'projectSupplier', 'issue', 'clientPayment', 'general', 'projectMilestone') NOT NULL;
