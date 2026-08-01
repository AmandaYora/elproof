ALTER TABLE project_vendors ADD COLUMN paid_amount BIGINT UNSIGNED NOT NULL DEFAULT 0;

ALTER TABLE projects
  DROP COLUMN venue_rental_price,
  DROP COLUMN venue_charge;
