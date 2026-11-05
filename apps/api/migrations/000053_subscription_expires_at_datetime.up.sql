-- A DATE reads back as 00:00, so a tenant shown "Berlaku hingga 20 Des"
-- already lost its paid status at the start of 20 Dec (security review,
-- Fase 1). Store the exact end instead; existing dates become the end of
-- that same day, keeping their inclusive meaning.
ALTER TABLE tenants MODIFY COLUMN subscription_expires_at DATETIME NULL;
UPDATE tenants
SET subscription_expires_at = DATE_ADD(subscription_expires_at, INTERVAL 86399 SECOND)
WHERE subscription_expires_at IS NOT NULL;
