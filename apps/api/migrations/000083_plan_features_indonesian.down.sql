-- Puts back the English seed text where a line still holds exactly what 000083
-- wrote. (Whether a row was singular or plural before is not kept; the forms
-- reported from production are used.)
UPDATE plan_features SET label = 'Full Features' WHERE BINARY label = 'Semua fitur';
UPDATE plan_features SET label = 'Full Seat' WHERE BINARY label = 'Anggota tim tanpa batas';
UPDATE plan_features SET label = 'Unlimited Project' WHERE BINARY label = 'Proyek tanpa batas';
