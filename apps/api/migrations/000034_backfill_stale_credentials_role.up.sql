-- One-time reconciliation for the bug this migration ships alongside:
-- `credentials.role` is a denormalized copy, written once at account
-- creation and (until now) never updated again when `staff_members.role` /
-- `platform_admins.role` changed via their own Update endpoints -- Login and
-- Refresh only ever read `credentials.role`, so a past role edit never took
-- effect for authorization purposes even though the owning module's own
-- table was correctly updated. This backfills every row already drifted by
-- that gap; the application code fix (StaffService.Update /
-- PlatformAdminService.Update now call identity.UpdateRole) prevents new
-- drift going forward.

-- CAST(... AS CHAR)'s result takes the connection's default collation,
-- which doesn't necessarily match `credentials.principal_id`'s stored one
-- (utf8mb4_0900_ai_ci) -- explicit COLLATE avoids an "Illegal mix of
-- collations" error regardless of what a given server's connection default
-- happens to be.
UPDATE credentials c
JOIN staff_members s
  ON c.principal_type = 'staff'
  AND c.principal_id = CAST(s.id AS CHAR) COLLATE utf8mb4_0900_ai_ci
  AND c.tenant_id = s.tenant_id
SET c.role = s.role
WHERE c.role <> s.role;

UPDATE credentials c
JOIN platform_admins p
  ON c.principal_type = 'platform_admin'
  AND c.principal_id = CAST(p.id AS CHAR) COLLATE utf8mb4_0900_ai_ci
SET c.role = p.role
WHERE c.role <> p.role;
