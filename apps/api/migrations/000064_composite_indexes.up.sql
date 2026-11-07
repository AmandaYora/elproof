-- Fase 7 (scale): composite indexes for the list queries that filesorted or
-- scanned a whole table. Each one matches a WHERE + ORDER BY in a repository;
-- the single-column index each replaces is its leftmost prefix, so nothing
-- that used it is lost.

-- Project list: WHERE tenant_id = ? AND is_archived = ? ORDER BY event_date DESC, id DESC.
-- A busy WO had every project sorted in memory for each page.
ALTER TABLE projects
  ADD INDEX idx_projects_tenant_archived_event (tenant_id, is_archived, event_date, id),
  DROP INDEX idx_projects_tenant;

-- Subscription transactions: the platform list (all tenants, optional status),
-- one tenant's list, and the paid-revenue window used by the metrics.
ALTER TABLE subscription_transactions
  ADD INDEX idx_subscription_transactions_created (created_at, id),
  ADD INDEX idx_subscription_transactions_status_created (status, created_at, id),
  ADD INDEX idx_subscription_transactions_tenant_created (tenant_id, created_at, id),
  ADD INDEX idx_subscription_transactions_status_paid (status, paid_at),
  DROP INDEX idx_subscription_transactions_tenant;

-- Tenant list: ORDER BY joined_at DESC, id DESC, and the dashboard's join counts.
ALTER TABLE tenants ADD INDEX idx_tenants_joined (joined_at, id);
