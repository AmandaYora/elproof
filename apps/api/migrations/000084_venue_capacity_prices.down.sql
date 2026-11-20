-- Reverses 000084: one capacity (the standing one, else the seated one) and one
-- rental price (akad + resepsi, else resepsi, else akad) come back.
ALTER TABLE venues
  ADD COLUMN rental_price BIGINT UNSIGNED NULL AFTER city,
  ADD COLUMN capacity INT UNSIGNED NULL AFTER charge;

UPDATE venues SET
  capacity = COALESCE(capacity_standing, capacity_seated),
  rental_price = COALESCE(price_akad_resepsi, price_resepsi, price_akad);

ALTER TABLE venues
  DROP COLUMN capacity_standing,
  DROP COLUMN capacity_seated,
  DROP COLUMN price_akad,
  DROP COLUMN price_resepsi,
  DROP COLUMN price_akad_resepsi;
