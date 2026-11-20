ALTER TABLE projects
  DROP INDEX uq_projects_tenant_quotation,
  DROP COLUMN quotation_revision_n,
  DROP COLUMN quotation_number,
  DROP COLUMN quotation_id;
