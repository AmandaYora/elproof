-- Least-privilege Platform Console roles (FLOW-IDEAL.md §1.3). Enforced in
-- cmd/server/platform_scope.go; credentials.role is a VARCHAR copy.
ALTER TABLE platform_admins
  MODIFY COLUMN role ENUM('Super Admin', 'Billing', 'Support', 'Read-only') NOT NULL;
