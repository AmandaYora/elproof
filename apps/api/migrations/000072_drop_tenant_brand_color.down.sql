-- Restores the column with migration 000017's definition. The colour each
-- tenant had chosen is not recoverable: every row gets the old default.
ALTER TABLE tenants
  ADD COLUMN brand_color_preset VARCHAR(20) NOT NULL DEFAULT 'navy';
