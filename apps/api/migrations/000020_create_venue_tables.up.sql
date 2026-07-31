-- Venue extraction (see knowledge/decisions/ADR-0016-venue-extraction.md): a
-- venue is no longer just a vendor category, it gets its own directory with
-- its own fields. facilities/social_media are plain TEXT -- free text the WO
-- edits per venue directly, not a fixed enum or JSON array (no marshal code
-- needed in the repository at all). city is nullable at the DB despite being
-- mandatory on the create form -- the data migration below inserts legacy
-- rows with no city information, same "DB permits it, the create-schema
-- enforces it" split already used for rental_price. One venue has at most
-- one attachment (document or photo) -- no more venue_photos gallery table.
CREATE TABLE venues (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(150) NOT NULL,
  pic_name VARCHAR(150) NOT NULL,
  phone_pic VARCHAR(30) NOT NULL,
  phone_venue VARCHAR(30) NULL,
  email VARCHAR(150) NULL,
  address VARCHAR(255) NULL,
  city VARCHAR(100) NULL,
  rental_price BIGINT UNSIGNED NULL,
  charge BIGINT UNSIGNED NULL,
  capacity INT UNSIGNED NULL,
  facilities TEXT NULL,
  social_media TEXT NULL,
  notes TEXT NULL,
  attachment_path VARCHAR(500) NULL,
  attachment_mime_type VARCHAR(100) NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_venues_tenant (tenant_id)
);
