-- Workspace-side audit (ADR-0026): changes a WO makes to its own roles and
-- permissions are recorded too, so entries carry the workspace they belong to.
ALTER TABLE audit_log
  ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER actor_role,
  ADD KEY idx_audit_log_tenant (tenant_id, occurred_at);
