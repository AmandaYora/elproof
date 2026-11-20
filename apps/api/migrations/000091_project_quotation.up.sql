-- Paket WO Tahap 5 (docs/plan/paket-wo/PLAN.md §5.4, ADR-0014): a project can
-- be created from an Accepted quotation ("Jadikan proyek"). quotation_id is a
-- primitive id into `offers` (no foreign key across modules); the unique key
-- makes the conversion idempotent: two concurrent conversions of the same
-- quotation give one project (the loser of the race reads the winner's row).
--
-- quotation_number and quotation_revision_n cache, at conversion, what the
-- banner "Dibuat dari MO-PNW-2026-0012 Revisi 2" shows, so the project detail
-- does not ask `offers` on every read. A duplicated project copies none of the three.

ALTER TABLE projects
  ADD COLUMN quotation_id BIGINT UNSIGNED NULL AFTER description,
  ADD COLUMN quotation_number VARCHAR(24) NULL AFTER quotation_id,
  ADD COLUMN quotation_revision_n INT UNSIGNED NULL AFTER quotation_number,
  ADD UNIQUE KEY uq_projects_tenant_quotation (tenant_id, quotation_id);
