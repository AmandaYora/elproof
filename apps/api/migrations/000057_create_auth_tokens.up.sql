-- One-time access links (PLAN.md Fase 3, decision K5): staff invitations
-- (accepted by signing in with the invited Google account) and client
-- invitations / password resets (the client sets their own password; the WO
-- never types or sees it). Only a SHA-256 hash of the link token is stored;
-- a link is single-use (used_at) and expires. Issuing a new link for the
-- same login and purpose revokes the previous one. Owned by `identity`.
CREATE TABLE auth_tokens (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  token_hash CHAR(64) NOT NULL,
  purpose ENUM('staff_invite', 'client_invite', 'client_reset') NOT NULL,
  credential_id BIGINT UNSIGNED NOT NULL,
  expires_at DATETIME NOT NULL,
  used_at DATETIME NULL,
  revoked_at DATETIME NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_auth_tokens_hash (token_hash),
  KEY idx_auth_tokens_credential (credential_id, purpose),
  CONSTRAINT fk_auth_tokens_credential FOREIGN KEY (credential_id) REFERENCES credentials (id)
);
