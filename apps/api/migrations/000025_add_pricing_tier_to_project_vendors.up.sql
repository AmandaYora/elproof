-- Vendor Pricing Tier (PLAN.md): records which of the vendor's own preset
-- prices (priceAkad / priceAkadResepsi) a project vendor engagement's Nilai
-- Kerja Sama started from -- purely an informational label going forward
-- (it never locks contract_value, which stays freely negotiable), so an
-- arbitrary default on existing rows is harmless.
ALTER TABLE project_vendors ADD COLUMN pricing_tier VARCHAR(20) NOT NULL DEFAULT 'Akad' AFTER contract_value;
