-- A kendala without a vendor cannot exist in the old shape: those rows (and the
-- evidence rows attached to them) are deleted before the column is made NOT NULL
-- again. Stored files stay in the bucket.
DELETE FROM evidence WHERE related_kind = 'issue'
  AND related_id IN (SELECT id FROM vendor_issues WHERE project_vendor_id IS NULL);
DELETE FROM vendor_issues WHERE project_vendor_id IS NULL;

ALTER TABLE vendor_issues
  DROP COLUMN is_client_visible,
  MODIFY COLUMN project_vendor_id BIGINT UNSIGNED NOT NULL;
