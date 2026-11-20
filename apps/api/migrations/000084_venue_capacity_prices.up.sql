-- Paket-WO Tahap 1 (docs/plan/paket-wo/PLAN.md §3.3): a venue holds a capacity
-- per layout (standing, seated) instead of one number, and a price per kind of
-- event (akad, resepsi, akad + resepsi) instead of one rental price. `charge`
-- stays the optional extra charge. The old single capacity becomes the standing
-- capacity and the old rental price becomes the akad + resepsi price, then the
-- old columns go (pre-launch: no compatibility columns are kept).
ALTER TABLE venues
  ADD COLUMN capacity_standing INT UNSIGNED NULL AFTER city,
  ADD COLUMN capacity_seated INT UNSIGNED NULL AFTER capacity_standing,
  ADD COLUMN price_akad BIGINT UNSIGNED NULL AFTER capacity_seated,
  ADD COLUMN price_resepsi BIGINT UNSIGNED NULL AFTER price_akad,
  ADD COLUMN price_akad_resepsi BIGINT UNSIGNED NULL AFTER price_resepsi;

UPDATE venues SET capacity_standing = capacity, price_akad_resepsi = rental_price;

ALTER TABLE venues
  DROP COLUMN capacity,
  DROP COLUMN rental_price;
