-- Vendor field adjustment (per the "DATA VENDOR" slide): email/address drop
-- their mandatory-ness, and social media / city / two package prices /
-- a single attachment slot are added -- same TEXT/nullable-int conventions
-- Venue's own Revisi 2 already established (no JSON, no array columns).
-- price_akad/price_akad_resepsi are nullable at the DB despite being
-- mandatory on the create form, so this migration doesn't fail against
-- existing vendor rows that have neither (see vendor_service.go's
-- validateVendorCity/VendorInput doc comment for the enforcement split).
ALTER TABLE vendors
  MODIFY COLUMN email VARCHAR(150) NULL,
  MODIFY COLUMN address VARCHAR(255) NULL,
  ADD COLUMN social_media TEXT NULL AFTER email,
  ADD COLUMN city VARCHAR(100) NULL AFTER address,
  ADD COLUMN price_akad BIGINT UNSIGNED NULL AFTER city,
  ADD COLUMN price_akad_resepsi BIGINT UNSIGNED NULL AFTER price_akad,
  ADD COLUMN attachment_path VARCHAR(500) NULL AFTER notes,
  ADD COLUMN attachment_mime_type VARCHAR(100) NULL AFTER attachment_path;
