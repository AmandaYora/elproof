ALTER TABLE credentials
  DROP COLUMN totp_last_step,
  DROP COLUMN totp_enabled,
  DROP COLUMN totp_secret_enc;
