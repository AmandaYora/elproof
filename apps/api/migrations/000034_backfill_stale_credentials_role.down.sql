-- No-op by design: the up migration only overwrites an already-wrong
-- `credentials.role` value with the correct one already sitting in
-- `staff_members.role` / `platform_admins.role` -- reversing it would mean
-- deliberately re-introducing the exact data corruption this migration
-- exists to fix. There is nothing else to undo; the application-code half
-- of this fix (identity.UpdateRole wiring) has its own separate revert path
-- (redeploying the previous binary), unrelated to this migration.
SELECT 1;
