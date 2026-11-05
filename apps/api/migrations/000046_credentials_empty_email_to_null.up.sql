-- "No email" is stored as NULL, never ''. credentials.email is UNIQUE
-- (000016): MySQL allows many NULLs under a UNIQUE key but only one '', so
-- the second account created without an email failed with a duplicate-key
-- error. The repository now writes NULL; this cleans up existing rows.
UPDATE credentials SET email = NULL WHERE email = '';
