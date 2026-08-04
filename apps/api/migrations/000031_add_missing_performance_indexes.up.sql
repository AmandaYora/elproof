-- PLAN.md "Performance remediation": project_vendors.vendor_id (vendor "Lihat
-- Project" history) and projects.pic_staff_id (Wedding Planner PIC scoping)
-- are filtered on but were missing an index, forcing a full table scan that
-- worsens as each table grows. Composite on (tenant_id, pic_staff_id) since
-- every real query already filters tenant_id first, then optionally
-- pic_staff_id -- one index serves both the tenant-only and PIC-scoped shapes.
ALTER TABLE project_vendors ADD KEY idx_project_vendors_vendor (vendor_id);
ALTER TABLE projects ADD KEY idx_projects_tenant_pic (tenant_id, pic_staff_id);
