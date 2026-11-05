-- Rolling back must never grant more power: the old schema only knows
-- Super Admin and Support, so Billing / Read-only admins are deactivated
-- (and stored as Support only to satisfy the old ENUM) instead of being
-- silently promoted to Support's write access.
UPDATE platform_admins SET role = 'Support', is_active = FALSE WHERE role IN ('Billing', 'Read-only');
UPDATE credentials SET role = 'Support', is_active = FALSE
WHERE principal_type = 'platform_admin' AND role IN ('Billing', 'Read-only');
ALTER TABLE platform_admins
  MODIFY COLUMN role ENUM('Super Admin', 'Support') NOT NULL;
