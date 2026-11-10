ALTER TABLE evidence DROP COLUMN updated_at;
ALTER TABLE venue_payments DROP COLUMN updated_at;
ALTER TABLE vendor_payments DROP COLUMN updated_at;
ALTER TABLE client_payments DROP COLUMN updated_at;
ALTER TABLE vendor_issues DROP COLUMN updated_at;
ALTER TABLE vendor_milestones DROP COLUMN updated_at;
ALTER TABLE project_vendors DROP COLUMN updated_at;
ALTER TABLE project_milestones DROP COLUMN updated_at;

-- projects keeps its column, back to whole seconds as migration 000008 made it.
ALTER TABLE projects
  MODIFY COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
