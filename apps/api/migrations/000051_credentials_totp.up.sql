-- Mandatory two-factor authentication (TOTP, RFC 6238) for Platform Console
-- admins (PLAN.md Fase 1, decision K14). The shared secret is stored
-- encrypted (AES-GCM); totp_last_step blocks reusing a code.
ALTER TABLE credentials
  ADD COLUMN totp_secret_enc VARBINARY(255) NULL AFTER password_hash,
  ADD COLUMN totp_enabled BOOLEAN NOT NULL DEFAULT FALSE AFTER totp_secret_enc,
  ADD COLUMN totp_last_step BIGINT NULL AFTER totp_enabled;
