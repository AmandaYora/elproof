UPDATE platform_admins SET role = 'Support' WHERE role IN ('Billing', 'Read-only');
UPDATE credentials SET role = 'Support'
WHERE principal_type = 'platform_admin' AND role IN ('Billing', 'Read-only');
ALTER TABLE platform_admins
  MODIFY COLUMN role ENUM('Super Admin', 'Support') NOT NULL;
