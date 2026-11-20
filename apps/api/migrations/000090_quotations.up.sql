-- Paket WO Tahap 5 (docs/plan/paket-wo/PLAN.md §5, ADR-0050): quotations
-- ("Penawaran") in the offers module. A quotation has the same shape as a
-- project (§3.1): events with sessions, Vendor & venue, and lines; plus terms,
-- terms & conditions, revisions frozen as JSON snapshots, and an activity log.
--
-- Every table carries tenant_id (ADR-0004), child tables included, and every
-- unique key starts with it. Event kind, vendor, venue, category, package, staff
-- and project ids are primitive ids into other modules (or the catalog), with no
-- foreign key; foreign keys stay inside the quotation tables.

CREATE TABLE quotations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  -- KODE-PNW-YYMMDD-NNN, frozen when the quotation is created.
  number VARCHAR(24) NOT NULL,
  -- The WIB day in the number (and of quotation_counters).
  doc_date DATE NOT NULL,
  couple_names VARCHAR(150) NOT NULL,
  contact_whatsapp VARCHAR(30) NOT NULL DEFAULT '',
  pic_staff_id BIGINT UNSIGNED NOT NULL,
  -- Valid through the end of this WIB day; a Draft past it reads as Kedaluwarsa.
  valid_until DATE NOT NULL,
  notes TEXT NULL,
  status ENUM('Draft', 'Accepted', 'Rejected', 'Superseded') NOT NULL DEFAULT 'Draft',
  -- The first quotation of a group of alternatives (NULL for that first one).
  alternative_of_id BIGINT UNSIGNED NULL,
  -- The quotation that replaced this one (copy, or an accepted alternative).
  superseded_by_id BIGINT UNSIGNED NULL,
  accepted_revision_id BIGINT UNSIGNED NULL,
  accepted_on DATE NULL,
  accepted_channel ENUM('WhatsApp', 'InPerson', 'Email', 'Other') NULL,
  -- Optional evidence of acceptance (photo of the signed PDF) in the WO's storage path.
  accepted_attachment_key VARCHAR(500) NULL,
  accepted_attachment_name VARCHAR(255) NULL,
  accepted_attachment_type VARCHAR(100) NULL,
  rejected_reason VARCHAR(500) NULL,
  rejected_at DATETIME NULL,
  -- Cache written by projects (offers contract MarkConverted); the project is the truth.
  converted_project_id BIGINT UNSIGNED NULL,
  converted_at DATETIME NULL,
  -- Set when the converted project was deleted for good: "Proyek sudah dihapus".
  project_deleted_at DATETIME NULL,
  -- Copied from the workspace when the quotation is created.
  is_pkp BOOLEAN NOT NULL DEFAULT FALSE,
  terms_text TEXT NOT NULL,
  -- Highest revision number so far (0 = none yet).
  revision_count INT UNSIGNED NOT NULL DEFAULT 0,
  -- Working-copy caches for the list: contract value and the earliest event date.
  contract_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
  main_event_date DATE NULL,
  created_by_staff_id BIGINT UNSIGNED NULL,
  deleted_at DATETIME NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  -- Versions for If-Match (ADR-0036), each written explicitly (no ON UPDATE), so
  -- saving lines never makes an open header form stale, and the other way round.
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  lines_updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  terms_updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  -- Last change of anything (the list's default order "terakhir diubah").
  changed_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_quotations_tenant_number (tenant_id, number),
  KEY idx_quotations_tenant_status (tenant_id, status, valid_until),
  KEY idx_quotations_tenant_alternative (tenant_id, alternative_of_id),
  KEY idx_quotations_tenant_pic (tenant_id, pic_staff_id),
  KEY idx_quotations_tenant_changed (tenant_id, changed_at),
  KEY idx_quotations_tenant_project (tenant_id, converted_project_id)
);

CREATE TABLE quotation_counters (
  tenant_id BIGINT UNSIGNED NOT NULL,
  doc_date DATE NOT NULL,
  last_seq INT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, doc_date)
);

CREATE TABLE quotation_suppliers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  quotation_id BIGINT UNSIGNED NOT NULL,
  supplier_kind ENUM('vendor', 'venue') NOT NULL,
  supplier_id BIGINT UNSIGNED NOT NULL,
  supplier_name VARCHAR(150) NOT NULL,
  category_id BIGINT UNSIGNED NULL,
  paid_by ENUM('WO', 'Client') NOT NULL DEFAULT 'WO',
  show_name BOOLEAN NOT NULL DEFAULT FALSE,
  capacity_standing INT UNSIGNED NULL,
  capacity_seated INT UNSIGNED NULL,
  agreed_cost BIGINT UNSIGNED NULL,
  notes TEXT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_quotation_suppliers_quotation (tenant_id, quotation_id),
  KEY idx_quotation_suppliers_supplier (tenant_id, supplier_kind, supplier_id),
  KEY idx_quotation_suppliers_category (tenant_id, category_id),
  CONSTRAINT fk_quotation_suppliers_quotation FOREIGN KEY (quotation_id) REFERENCES quotations (id) ON DELETE CASCADE
);

CREATE TABLE quotation_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  quotation_id BIGINT UNSIGNED NOT NULL,
  event_kind_id BIGINT UNSIGNED NOT NULL,
  kind_name VARCHAR(60) NOT NULL,
  -- May stay empty in a quotation (an estimate); a project needs it.
  event_date DATE NULL,
  start_time CHAR(5) NULL,
  end_time CHAR(5) NULL,
  venue_supplier_id BIGINT UNSIGNED NULL,
  layout ENUM('Standing', 'Seated') NOT NULL DEFAULT 'Standing',
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_quotation_events_quotation (tenant_id, quotation_id),
  KEY idx_quotation_events_kind (tenant_id, event_kind_id),
  CONSTRAINT fk_quotation_events_quotation FOREIGN KEY (quotation_id) REFERENCES quotations (id) ON DELETE CASCADE,
  CONSTRAINT fk_quotation_events_venue FOREIGN KEY (venue_supplier_id) REFERENCES quotation_suppliers (id) ON DELETE SET NULL
);

CREATE TABLE quotation_event_sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  event_id BIGINT UNSIGNED NOT NULL,
  label VARCHAR(60) NOT NULL,
  start_time CHAR(5) NULL,
  end_time CHAR(5) NULL,
  guests INT UNSIGNED NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_quotation_event_sessions_event (tenant_id, event_id),
  CONSTRAINT fk_quotation_event_sessions_event FOREIGN KEY (event_id) REFERENCES quotation_events (id) ON DELETE CASCADE
);

CREATE TABLE quotation_lines (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  quotation_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('Package', 'Component', 'ExtraGuests', 'Item', 'Discount') NOT NULL,
  -- Component and ExtraGuests lines point at their Package line.
  parent_line_id BIGINT UNSIGNED NULL,
  -- Package lines: the catalog package (a package referred to here can only be deactivated)
  -- and the copy of its rule taken when it was added.
  package_id BIGINT UNSIGNED NULL,
  package_rule JSON NULL,
  copied_at DATETIME NULL,
  description VARCHAR(255) NOT NULL,
  details TEXT NULL,
  category_id BIGINT UNSIGNED NULL,
  event_id BIGINT UNSIGNED NULL,
  quantity BIGINT UNSIGNED NULL,
  unit VARCHAR(20) NULL,
  unit_price BIGINT UNSIGNED NULL,
  -- Always positive; the kind says whether it adds or takes away.
  amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
  is_bonus BOOLEAN NOT NULL DEFAULT FALSE,
  -- The supplier (paid by the WO) that provides this line.
  supplier_ref BIGINT UNSIGNED NULL,
  sort_order INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_quotation_lines_quotation (tenant_id, quotation_id),
  KEY idx_quotation_lines_package (tenant_id, package_id),
  KEY idx_quotation_lines_category (tenant_id, category_id),
  CONSTRAINT fk_quotation_lines_quotation FOREIGN KEY (quotation_id) REFERENCES quotations (id) ON DELETE CASCADE,
  CONSTRAINT fk_quotation_lines_parent FOREIGN KEY (parent_line_id) REFERENCES quotation_lines (id) ON DELETE CASCADE,
  CONSTRAINT fk_quotation_lines_event FOREIGN KEY (event_id) REFERENCES quotation_events (id) ON DELETE SET NULL,
  CONSTRAINT fk_quotation_lines_supplier FOREIGN KEY (supplier_ref) REFERENCES quotation_suppliers (id) ON DELETE SET NULL
);

CREATE TABLE quotation_terms (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  quotation_id BIGINT UNSIGNED NOT NULL,
  label VARCHAR(60) NOT NULL,
  -- Exactly one of percent (of the contract value) and amount is set.
  percent INT UNSIGNED NULL,
  amount BIGINT UNSIGNED NULL,
  -- Due relative to the day the quotation is accepted, or to an event's date
  -- (due_event_id; NULL = the earliest event).
  due_basis ENUM('Accepted', 'Event') NOT NULL DEFAULT 'Accepted',
  due_offset_days INT NOT NULL DEFAULT 0,
  due_event_id BIGINT UNSIGNED NULL,
  sort_order INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_quotation_terms_quotation (tenant_id, quotation_id),
  CONSTRAINT fk_quotation_terms_quotation FOREIGN KEY (quotation_id) REFERENCES quotations (id) ON DELETE CASCADE,
  CONSTRAINT fk_quotation_terms_event FOREIGN KEY (due_event_id) REFERENCES quotation_events (id) ON DELETE SET NULL
);

CREATE TABLE quotation_revisions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  quotation_id BIGINT UNSIGNED NOT NULL,
  n INT UNSIGNED NOT NULL,
  -- Everything the PDF needs, frozen: content, WO identity (incl. the logo key), validity.
  snapshot JSON NOT NULL,
  -- sha256 of the content part of the snapshot: identical content never adds a revision.
  content_hash CHAR(64) NOT NULL,
  contract_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_by_staff_id BIGINT UNSIGNED NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_quotation_revisions_hash (tenant_id, quotation_id, content_hash),
  UNIQUE KEY uq_quotation_revisions_n (tenant_id, quotation_id, n),
  CONSTRAINT fk_quotation_revisions_quotation FOREIGN KEY (quotation_id) REFERENCES quotations (id) ON DELETE CASCADE
);

CREATE TABLE quotation_activity (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  quotation_id BIGINT UNSIGNED NOT NULL,
  type VARCHAR(40) NOT NULL,
  actor_staff_id BIGINT UNSIGNED NULL,
  summary VARCHAR(500) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_quotation_activity_quotation (tenant_id, quotation_id, created_at),
  CONSTRAINT fk_quotation_activity_quotation FOREIGN KEY (quotation_id) REFERENCES quotations (id) ON DELETE CASCADE
);

-- New permissions quotation.view and quotation.manage, on by default for the
-- Owner, Admin, and Sales roles of every workspace (PLAN §6). Keep in step with
-- access/domain/catalog.go Templates.
INSERT IGNORE INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'quotation.view' AS permission UNION ALL SELECT 'quotation.manage') p WHERE r.template_key = 'owner';
INSERT IGNORE INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'quotation.view' AS permission UNION ALL SELECT 'quotation.manage') p WHERE r.template_key = 'admin';
INSERT IGNORE INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'quotation.view' AS permission UNION ALL SELECT 'quotation.manage') p WHERE r.template_key = 'sales';
