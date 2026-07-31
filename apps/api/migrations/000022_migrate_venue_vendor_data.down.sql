-- Best-effort only: reversing the is_active flips isn't reliable without
-- knowing each row's state before this migration ran (see ADR-0016) -- this
-- just removes the copied venues rows.
DELETE FROM venues;
