DELETE FROM tenant_role_permissions WHERE permission = 'portal.tasks.complete';
ALTER TABLE project_milestones DROP COLUMN completed_by_client_id, DROP COLUMN for_client;
