-- Paket WO Tahap 3 (docs/plan/paket-wo/PLAN.md §3.1, §3.2, ADR-0048): a
-- project's contract value is no longer typed; it is the sum of its lines
-- ("Rincian"): packages (with a copy of their rule), their components, the
-- extra-guest charge, items, bonuses and discounts. PPN applies when the
-- workspace was PKP when the project was created.
--
-- ElProof is pre-launch with dummy data (PLAN §0): the typed package name and
-- contract value are dropped, not converted. Every new table carries tenant_id
-- (ADR-0004); category_id is a primitive id into `vendors`; event_id,
-- supplier_ref and parent_line_id are foreign keys inside projects.

ALTER TABLE projects
  DROP COLUMN package_name,
  DROP COLUMN contract_value,
  -- Copied from the workspace (tenant_business_details.is_pkp) when the project is created.
  ADD COLUMN is_pkp BOOLEAN NOT NULL DEFAULT FALSE AFTER prep_start_date,
  -- Cache of the live lines, rewritten with UPDATE ... SET ... = (SELECT SUM ...)
  -- in the same transaction as every change of a line, a guest number, a soft
  -- delete, a restore or a duplicate.
  ADD COLUMN contract_subtotal BIGINT NOT NULL DEFAULT 0 AFTER is_pkp,
  ADD COLUMN contract_tax BIGINT NOT NULL DEFAULT 0 AFTER contract_subtotal,
  ADD COLUMN contract_total BIGINT NOT NULL DEFAULT 0 AFTER contract_tax,
  -- The lines' own version (If-Match of the Rincian editor, ADR-0036), so a
  -- line change never makes an open project form stale (and the other way).
  ADD COLUMN lines_version TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) AFTER contract_total;

CREATE TABLE project_lines (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  project_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('Package', 'Component', 'ExtraGuests', 'Item', 'Discount') NOT NULL,
  -- Component and ExtraGuests point at their Package.
  parent_line_id BIGINT UNSIGNED NULL,
  -- Package only: the copy of the package's rule when it was added (offers/domain.PackageRule).
  package_rule JSON NULL,
  description VARCHAR(255) NOT NULL,
  category_id BIGINT UNSIGNED NULL,
  event_id BIGINT UNSIGNED NULL,
  -- Item only: amount = quantity x unit_price.
  quantity INT UNSIGNED NULL,
  unit VARCHAR(30) NOT NULL DEFAULT '',
  unit_price BIGINT UNSIGNED NULL,
  -- Always >= 0; the kind says whether it adds (Package, ExtraGuests, Item) or
  -- takes away (Discount). Component = 0. ExtraGuests is always computed.
  amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
  is_bonus BOOLEAN NOT NULL DEFAULT FALSE,
  -- The project supplier (paid by the WO) that provides the line.
  supplier_ref BIGINT UNSIGNED NULL,
  sort_order INT NOT NULL DEFAULT 0,
  deleted_at DATETIME NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  KEY idx_project_lines_project (tenant_id, project_id, deleted_at),
  KEY idx_project_lines_parent (tenant_id, parent_line_id),
  KEY idx_project_lines_deleted (deleted_at),
  CONSTRAINT fk_project_lines_project FOREIGN KEY (project_id) REFERENCES projects (id),
  CONSTRAINT fk_project_lines_parent FOREIGN KEY (parent_line_id) REFERENCES project_lines (id) ON DELETE CASCADE,
  CONSTRAINT fk_project_lines_event FOREIGN KEY (event_id) REFERENCES project_events (id) ON DELETE SET NULL,
  CONSTRAINT fk_project_lines_supplier FOREIGN KEY (supplier_ref) REFERENCES project_suppliers (id) ON DELETE SET NULL
);
