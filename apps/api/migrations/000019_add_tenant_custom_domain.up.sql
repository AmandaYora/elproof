-- Optional custom domain a tenant can point at this server so their own
-- staff/clients see pre-login branding on their own hostname instead of the
-- platform's -- see knowledge/decisions/ADR-0015-tenant-custom-domain.md.
-- Nullable + unique: most tenants never set one, and two tenants must never
-- resolve to the same domain (MySQL's UNIQUE index permits multiple NULLs).
ALTER TABLE tenants ADD COLUMN custom_domain VARCHAR(255) NULL UNIQUE;
