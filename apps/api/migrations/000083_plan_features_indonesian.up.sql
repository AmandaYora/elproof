-- EP-14: plan feature lines typed in English on production ("Full Features",
-- "Full Seat", "Unlimited Project") read as Indonesian on the public pricing page
-- and in the subscription screens. Only a line that still equals one of those
-- English texts exactly (spaces around it ignored, case kept: BINARY) is
-- rewritten; anything an admin wrote differently is left alone. The wording
-- follows the public pricing copy ("semua fitur") and the glossary (anggota tim,
-- proyek). plan_features rows only; subscription_plans (and its version) is not
-- touched.
UPDATE plan_features SET label = 'Semua fitur'
WHERE BINARY TRIM(label) IN ('Full Features', 'Full Feature');

UPDATE plan_features SET label = 'Anggota tim tanpa batas'
WHERE BINARY TRIM(label) IN ('Full Seat', 'Full Seats');

UPDATE plan_features SET label = 'Proyek tanpa batas'
WHERE BINARY TRIM(label) IN ('Unlimited Project', 'Unlimited Projects');
