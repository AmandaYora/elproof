-- Google-only accounts get an unusable password marker ("!" never matches a
-- bcrypt hash), so rolling back cannot open a passwordless login.
UPDATE credentials SET password_hash = '!' WHERE password_hash IS NULL;
ALTER TABLE credentials
  DROP KEY uq_credentials_google_sub,
  DROP COLUMN google_sub,
  MODIFY password_hash VARCHAR(255) NOT NULL;
