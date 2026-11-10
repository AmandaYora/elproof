-- T-20261110-11 (ADR-0039): ElProof is one product for every WO, so the
-- per-tenant colour choice is gone. The WO Console, Client Portal and
-- Platform Console share one palette (plum + seal gold) from theme.css.
-- The tenant's logo (logo_storage_path) stays.
ALTER TABLE tenants
  DROP COLUMN brand_color_preset;
