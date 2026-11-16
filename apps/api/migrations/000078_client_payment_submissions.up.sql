-- Client Portal finding #10 (ADR-0047): the couple sends a transfer proof from the
-- portal. It becomes an unverified client payment that staff verify as usual.
-- submitted_by_client_id names the client (a primitive reference into `clients`,
-- no foreign key); recorded_by_staff_id stays NULL for such a row.
ALTER TABLE client_payments ADD COLUMN submitted_by_client_id BIGINT UNSIGNED NULL AFTER recorded_by_staff_id;

-- New portal permission, on by default for the Pengantin role of every workspace
-- (off for Keluarga). Keep in step with access/domain/catalog.go Templates.
INSERT IGNORE INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'portal.finance.submit' AS permission) p WHERE r.template_key = 'bride';
