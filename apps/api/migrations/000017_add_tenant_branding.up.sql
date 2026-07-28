-- Per-tenant branding (see PLAN.md): a superadmin-configured logo and one of
-- 15 fixed brand-color presets, applied across WO Console and Client Portal.
-- brand_color_preset defaults to 'navy' -- the app's existing look -- so
-- every already-registered tenant renders unchanged until reconfigured.
ALTER TABLE tenants
  ADD COLUMN brand_color_preset VARCHAR(20) NOT NULL DEFAULT 'navy',
  ADD COLUMN logo_storage_path VARCHAR(500) NULL;
