-- PLAN.md "City filter (Vendor, Venue)" -- cheap composite index to add while
-- touching this area; not required at current data volumes (a plain
-- tenant-scoped table scan is fine today), just avoids revisiting the schema
-- once it isn't.
ALTER TABLE vendors ADD KEY idx_vendors_tenant_city (tenant_id, city);
ALTER TABLE venues ADD KEY idx_venues_tenant_city (tenant_id, city);
