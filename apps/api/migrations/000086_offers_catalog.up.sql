-- Catalog of the offers module (docs/plan/paket-wo/PLAN.md §2.1, ADR-0049):
-- the kinds of event a WO organises, and its packages with the kinds they
-- cover and what they contain. Every table carries tenant_id (ADR-0004), child
-- tables included. Venue, vendor, and vendor category are primitive ids into
-- the vendors module, with no foreign key; foreign keys stay inside offers.

CREATE TABLE event_kinds (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(60) NOT NULL,
  -- Which built-in kind this row was seeded as (akad, resepsi, ...); NULL for a
  -- kind the WO added. Survives a rename, so seeding stays idempotent.
  default_key VARCHAR(20) NULL,
  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_event_kinds_tenant_name (tenant_id, name),
  UNIQUE KEY uq_event_kinds_tenant_default (tenant_id, default_key)
);

CREATE TABLE packages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(150) NOT NULL,
  price BIGINT UNSIGNED NOT NULL,
  included_guests INT UNSIGNED NULL,
  extra_guest_step INT UNSIGNED NULL,
  extra_guest_step_price BIGINT UNSIGNED NULL,
  default_venue_id BIGINT UNSIGNED NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uq_packages_tenant_name (tenant_id, name),
  KEY idx_packages_tenant_venue (tenant_id, default_venue_id)
);

CREATE TABLE package_event_kinds (
  tenant_id BIGINT UNSIGNED NOT NULL,
  package_id BIGINT UNSIGNED NOT NULL,
  event_kind_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (package_id, event_kind_id),
  KEY idx_package_event_kinds_kind (tenant_id, event_kind_id),
  CONSTRAINT fk_package_event_kinds_package FOREIGN KEY (package_id) REFERENCES packages (id) ON DELETE CASCADE,
  CONSTRAINT fk_package_event_kinds_kind FOREIGN KEY (event_kind_id) REFERENCES event_kinds (id)
);

CREATE TABLE package_components (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  package_id BIGINT UNSIGNED NOT NULL,
  description VARCHAR(255) NOT NULL,
  category_id BIGINT UNSIGNED NULL,
  vendor_id BIGINT UNSIGNED NULL,
  estimated_cost BIGINT UNSIGNED NULL,
  sort_order INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_package_components_package (tenant_id, package_id),
  KEY idx_package_components_vendor (tenant_id, vendor_id),
  KEY idx_package_components_category (tenant_id, category_id),
  CONSTRAINT fk_package_components_package FOREIGN KEY (package_id) REFERENCES packages (id) ON DELETE CASCADE
);

-- The six built-in kinds for every workspace that already exists; new ones get
-- them at sign-up (offers SeedDefaultEventKinds). Keep in step with
-- offers/domain/event_kind.go DefaultEventKinds.
INSERT IGNORE INTO event_kinds (tenant_id, name, default_key, sort_order)
  SELECT t.id, k.name, k.default_key, k.sort_order FROM tenants t, (
    SELECT 'Lamaran' AS name, 'lamaran' AS default_key, 1 AS sort_order
    UNION ALL SELECT 'Pengajian', 'pengajian', 2
    UNION ALL SELECT 'Siraman', 'siraman', 3
    UNION ALL SELECT 'Akad', 'akad', 4
    UNION ALL SELECT 'Resepsi', 'resepsi', 5
    UNION ALL SELECT 'Ngunduh mantu', 'ngunduh_mantu', 6
  ) k;

-- New permission package.manage (Katalog: packages and event kinds), on by
-- default for the Owner and Admin roles of every workspace. Keep in step with
-- access/domain/catalog.go Templates.
INSERT IGNORE INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'package.manage' AS permission) p WHERE r.template_key = 'owner';
INSERT IGNORE INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'package.manage' AS permission) p WHERE r.template_key = 'admin';
