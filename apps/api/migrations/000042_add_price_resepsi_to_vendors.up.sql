-- "Resepsi Only" package preset price (PLAN.md revisi-timeline-vendor-role-sales)
-- -- sits alongside price_akad/price_akad_resepsi (000023), same nullable
-- convention.
ALTER TABLE vendors ADD COLUMN price_resepsi BIGINT UNSIGNED NULL AFTER price_akad_resepsi;
