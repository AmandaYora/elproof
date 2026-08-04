ALTER TABLE vendor_issues
  DROP FOREIGN KEY fk_vendor_issues_vendor_milestone,
  DROP KEY idx_vendor_issues_vendor_milestone,
  DROP COLUMN vendor_milestone_id;
