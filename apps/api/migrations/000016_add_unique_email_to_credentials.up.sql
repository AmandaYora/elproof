-- Now that login can resolve an email, two accounts sharing one email would
-- make login-by-email ambiguous by construction (whichever's password check
-- happens to match first) -- enforce the same uniqueness already guaranteed
-- for username. MySQL treats multiple NULLs as non-conflicting under a
-- UNIQUE key, so accounts that never got a backfilled email (see migration
-- 000015) aren't affected.
ALTER TABLE credentials DROP KEY idx_credentials_email;
ALTER TABLE credentials ADD UNIQUE KEY uq_credentials_email (email);
