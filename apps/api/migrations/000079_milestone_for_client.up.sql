-- Client Portal finding #11 (ADR-0047): a Timeline step can be a task for the
-- couple ("Kirim daftar tamu"); the couple marks it done from the portal.
-- completed_by_client_id names who did (a primitive reference into `clients`, no
-- foreign key); NULL when staff completed it or it is not completed.
ALTER TABLE project_milestones
  ADD COLUMN for_client BOOLEAN NOT NULL DEFAULT FALSE AFTER days_before_event,
  ADD COLUMN completed_by_client_id BIGINT UNSIGNED NULL AFTER for_client;

-- New portal permission, on by default for the Pengantin role of every workspace
-- (off for Keluarga). Keep in step with access/domain/catalog.go Templates.
INSERT IGNORE INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'portal.tasks.complete' AS permission) p WHERE r.template_key = 'bride';
