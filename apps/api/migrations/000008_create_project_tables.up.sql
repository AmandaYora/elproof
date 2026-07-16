CREATE TABLE projects (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(150) NOT NULL,
  bride_name VARCHAR(150) NOT NULL,
  groom_name VARCHAR(150) NOT NULL,
  event_date DATE NOT NULL,
  venue VARCHAR(255) NOT NULL,
  prep_start_date DATE NOT NULL,
  package_name VARCHAR(150) NOT NULL,
  contract_value BIGINT UNSIGNED NOT NULL DEFAULT 0,
  status ENUM('Draft', 'Preparation', 'Ready', 'Completed', 'Cancelled') NOT NULL DEFAULT 'Draft',
  pic_staff_id BIGINT UNSIGNED NOT NULL,
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_projects_tenant (tenant_id)
);

CREATE TABLE project_milestones (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  sort_order SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  name VARCHAR(150) NOT NULL,
  status ENUM('Not Started', 'In Progress', 'Completed', 'Blocked', 'Cancelled') NOT NULL DEFAULT 'Not Started',
  target_date DATE NOT NULL,
  completed_date DATE NULL,
  PRIMARY KEY (id),
  KEY idx_project_milestones_project (project_id),
  CONSTRAINT fk_project_milestones_project FOREIGN KEY (project_id) REFERENCES projects (id)
);

CREATE TABLE project_vendors (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  vendor_id BIGINT UNSIGNED NOT NULL,
  category_id BIGINT UNSIGNED NOT NULL,
  scope VARCHAR(255) NOT NULL,
  contract_value BIGINT UNSIGNED NOT NULL DEFAULT 0,
  engagement_status ENUM('Planned', 'Negotiation', 'Booked', 'DP Paid', 'In Progress', 'Fully Paid', 'Ready', 'Completed', 'Cancelled') NOT NULL DEFAULT 'Planned',
  booking_date DATE NULL,
  event_date DATE NOT NULL,
  dp_amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
  paid_amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
  due_date DATE NULL,
  pic_staff_id BIGINT UNSIGNED NOT NULL,
  notes TEXT NULL,
  PRIMARY KEY (id),
  KEY idx_project_vendors_project (project_id),
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
  invoice_evidence_id BIGINT UNSIGNED NULL,
  proof_evidence_id BIGINT UNSIGNED NULL,
  notes TEXT NULL,
  PRIMARY KEY (id),
  KEY idx_vendor_payments_project (project_id),
  KEY idx_vendor_payments_project_vendor (project_vendor_id),
  CONSTRAINT fk_vendor_payments_project FOREIGN KEY (project_id) REFERENCES projects (id),
  CONSTRAINT fk_vendor_payments_project_vendor FOREIGN KEY (project_vendor_id) REFERENCES project_vendors (id)
);

CREATE TABLE vendor_issues (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  project_vendor_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  impact ENUM('Low', 'Medium', 'High', 'Critical') NOT NULL,
  found_date DATE NOT NULL,
  status ENUM('Open', 'In Review', 'In Resolution', 'Resolved', 'Closed') NOT NULL DEFAULT 'Open',
  resolution_plan TEXT NULL,
  pic_staff_id BIGINT UNSIGNED NOT NULL,
  target_resolution_date DATE NULL,
  resolved_date DATE NULL,
  resolution_notes TEXT NULL,
  PRIMARY KEY (id),
  KEY idx_vendor_issues_project (project_id),
  KEY idx_vendor_issues_project_vendor (project_vendor_id),
  CONSTRAINT fk_vendor_issues_project FOREIGN KEY (project_id) REFERENCES projects (id),
  CONSTRAINT fk_vendor_issues_project_vendor FOREIGN KEY (project_vendor_id) REFERENCES project_vendors (id)
);

CREATE TABLE evidence (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  type ENUM('Quotation', 'Invoice', 'Contract', 'Transfer Proof', 'Receipt', 'Purchase Order', 'Photo', 'Document', 'Screenshot', 'Minutes of Meeting', 'Other') NOT NULL,
  storage_path VARCHAR(500) NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  document_date DATE NULL,
  uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  description VARCHAR(255) NULL,
  uploaded_by_staff_id BIGINT UNSIGNED NOT NULL,
  related_kind ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue') NOT NULL,
  related_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (id),
  KEY idx_evidence_project (project_id),
  KEY idx_evidence_related (related_kind, related_id),
  CONSTRAINT fk_evidence_project FOREIGN KEY (project_id) REFERENCES projects (id)
);

CREATE TABLE activity_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  project_id BIGINT UNSIGNED NULL,
  type VARCHAR(50) NOT NULL,
  actor_staff_id BIGINT UNSIGNED NOT NULL,
  entity_type VARCHAR(50) NOT NULL,
  entity_id VARCHAR(50) NOT NULL,
  entity_label VARCHAR(255) NOT NULL,
  description VARCHAR(500) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_activity_log_project (project_id)
);
