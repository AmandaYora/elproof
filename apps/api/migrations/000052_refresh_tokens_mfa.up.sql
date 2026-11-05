-- A refresh token remembers whether its session was established with a
-- second factor; Refresh copies it forward instead of re-deriving it from
-- the account's current state (security review, Fase 1).
ALTER TABLE refresh_tokens ADD COLUMN mfa BOOLEAN NOT NULL DEFAULT FALSE AFTER token_hash;
