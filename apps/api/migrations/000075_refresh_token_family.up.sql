-- Refresh-token families (ADR-0042). Every token minted by rotating another
-- one carries the id of the first token of its sign-in (the family root).
-- NULL means the token is itself a root, so existing rows need no backfill.
-- Presenting a token that was rotated away longer ago than the reuse grace
-- revokes the whole family: whoever holds the newest token is signed out
-- too, which is the point (one of the two holders is not the user).
ALTER TABLE refresh_tokens
  ADD COLUMN family_id BIGINT UNSIGNED NULL DEFAULT NULL AFTER credential_id,
  ADD KEY idx_refresh_tokens_family (family_id);
