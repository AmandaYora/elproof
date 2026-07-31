-- Timeline Default Template (per tenant, PLAN.md): the checklist auto-seeded
-- into every newly created project's Timeline tab, now configurable from
-- Pengaturan -> Timeline Default instead of a single hardcoded list shared by
-- every tenant. No is_active column (unlike vendor_categories) -- items are
-- hard-deleted (PLAN.md's decision), so a row's existence is the only state.
CREATE TABLE project_milestone_templates (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  tenant_id BIGINT UNSIGNED NOT NULL,
  sort_order INT NOT NULL,
  name VARCHAR(255) NOT NULL,
  days_before_event INT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_project_milestone_templates_tenant (tenant_id)
);

-- Backfill every already-registered tenant with today's hardcoded template
-- (previously apps/api/internal/modules/projects/application/default_milestones.go's
-- defaultMilestoneTemplate) so no existing tenant's next new project silently
-- loses its starter checklist the moment this migration ships. Guarded by
-- WHERE NOT EXISTS so a forced/manual re-run of this file (outside
-- golang-migrate's own version tracking) never double-inserts a second set
-- of rows for a tenant that already has one.
INSERT INTO project_milestone_templates (tenant_id, sort_order, name, days_before_event)
SELECT id, 1, 'Survei Venue & Vendor', 90 FROM tenants
WHERE NOT EXISTS (SELECT 1 FROM project_milestone_templates t WHERE t.tenant_id = tenants.id)
UNION ALL SELECT id, 2, 'DP / Tanda Jadi ke Vendor', 60 FROM tenants
WHERE NOT EXISTS (SELECT 1 FROM project_milestone_templates t WHERE t.tenant_id = tenants.id)
UNION ALL SELECT id, 3, 'Technical Meeting', 14 FROM tenants
WHERE NOT EXISTS (SELECT 1 FROM project_milestone_templates t WHERE t.tenant_id = tenants.id)
UNION ALL SELECT id, 4, 'Pelunasan Vendor', 7 FROM tenants
WHERE NOT EXISTS (SELECT 1 FROM project_milestone_templates t WHERE t.tenant_id = tenants.id)
UNION ALL SELECT id, 5, 'Gladi Resik', 1 FROM tenants
WHERE NOT EXISTS (SELECT 1 FROM project_milestone_templates t WHERE t.tenant_id = tenants.id)
UNION ALL SELECT id, 6, 'Hari-H Pernikahan', 0 FROM tenants
WHERE NOT EXISTS (SELECT 1 FROM project_milestone_templates t WHERE t.tenant_id = tenants.id);
