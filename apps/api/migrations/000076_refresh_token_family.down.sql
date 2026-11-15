ALTER TABLE refresh_tokens
  DROP KEY idx_refresh_tokens_family,
  DROP COLUMN family_id;
