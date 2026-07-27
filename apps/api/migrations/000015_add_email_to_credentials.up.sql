ALTER TABLE credentials ADD COLUMN email VARCHAR(150) NULL AFTER username;
ALTER TABLE credentials ADD KEY idx_credentials_email (email);

-- One-time backfill for accounts created before this column existed, so
-- existing users can log in by email immediately too, not just new ones.
-- This is schema/data migration tooling, not request-path application code
-- (see internal/adminseed's own doc comment for the same distinction) — the
-- ongoing lookup path never joins across modules like this; only this
-- one-off backfill does, reading each owning module's already-existing
-- email column exactly once.
UPDATE credentials c
JOIN staff_members s ON c.principal_type = 'staff' AND c.principal_id = CAST(s.id AS CHAR) COLLATE utf8mb4_unicode_ci
SET c.email = s.email;

UPDATE credentials c
JOIN clients cl ON c.principal_type = 'client' AND c.principal_id = CAST(cl.id AS CHAR) COLLATE utf8mb4_unicode_ci
SET c.email = cl.email;

UPDATE credentials c
JOIN platform_admins pa ON c.principal_type = 'platform_admin' AND c.principal_id = CAST(pa.id AS CHAR) COLLATE utf8mb4_unicode_ci
SET c.email = pa.email;
