-- Tahap 3 / E4 (docs/plan/revamp-ui-shadcn/PLAN.md): who recorded a client
-- payment and who verified it (and when), so the Client Portal can show a
-- payment as "diterima dan diverifikasi" with its proof.
--
-- The staff ids are primitive references into the `staff` module: no foreign
-- key (a module never owns another module's table, and a staff row may be
-- hard-deleted later; the name then simply resolves to nothing). Existing rows
-- stay NULL, i.e. "not verified" and "recorder unknown".
ALTER TABLE client_payments
  ADD COLUMN recorded_by_staff_id BIGINT UNSIGNED NULL,
  ADD COLUMN verified_at DATETIME NULL,
  ADD COLUMN verified_by_staff_id BIGINT UNSIGNED NULL;
