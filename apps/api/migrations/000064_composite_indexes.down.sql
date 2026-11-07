ALTER TABLE tenants DROP INDEX idx_tenants_joined;

ALTER TABLE subscription_transactions
  ADD INDEX idx_subscription_transactions_tenant (tenant_id),
  DROP INDEX idx_subscription_transactions_created,
  DROP INDEX idx_subscription_transactions_status_created,
  DROP INDEX idx_subscription_transactions_tenant_created,
  DROP INDEX idx_subscription_transactions_status_paid;

ALTER TABLE projects
  ADD INDEX idx_projects_tenant (tenant_id),
  DROP INDEX idx_projects_tenant_archived_event;
