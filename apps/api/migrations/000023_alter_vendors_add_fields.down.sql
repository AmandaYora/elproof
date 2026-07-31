-- Best-effort only (ADR-0011): reverting email/address to NOT NULL fails if
-- any row picked up a NULL value after the up-migration ran -- same caveat
-- already accepted by 000022's down migration for this exact reason.
ALTER TABLE vendors
  DROP COLUMN attachment_mime_type,
  DROP COLUMN attachment_path,
  DROP COLUMN price_akad_resepsi,
  DROP COLUMN price_akad,
  DROP COLUMN city,
  DROP COLUMN social_media,
  MODIFY COLUMN address VARCHAR(255) NOT NULL,
  MODIFY COLUMN email VARCHAR(150) NOT NULL;
