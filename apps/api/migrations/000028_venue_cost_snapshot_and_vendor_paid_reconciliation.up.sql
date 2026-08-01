-- Part 1: per-project venue cost snapshot (PLAN.md "Financial Calculation
-- Correctness" §1.1) -- mirrors project_vendors' own contract_value column;
-- a project has at most one venue, so these live directly on `projects`,
-- not a join table. Historical Margin must never drift just because
-- venue master data changed later -- see the plan for the full reasoning.
ALTER TABLE projects
  ADD COLUMN venue_rental_price BIGINT UNSIGNED NULL AFTER venue_id,
  ADD COLUMN venue_charge BIGINT UNSIGNED NULL AFTER venue_rental_price;

-- Backfill: every project that already has a venue attached gets that
-- venue's CURRENT price copied in as its starting snapshot -- otherwise
-- every already-attached venue would show venueCost = 0 in Margin the
-- moment this ships, until someone happens to re-open and re-save it.
-- This is a deliberate, one-time cross-module JOIN (projects <-> venues,
-- normally forbidden per .claude/rules/database.md) -- acceptable here
-- because it's a single backfill migration, not a live application-code
-- query path; going forward the snapshot is populated by
-- ProjectService.Update alone, never a join at read time. Same precedent
-- as migration 000022/000026's own one-time cross-module backfills.
UPDATE projects p
JOIN venues v ON v.id = p.venue_id
SET p.venue_rental_price = v.rental_price,
    p.venue_charge = v.charge
WHERE p.venue_id IS NOT NULL;

-- Part 2: reconcile vendor "paid amount" before dropping the manual field
-- (PLAN.md §1.2) -- backfill only the shortfall where the old manual field
-- claims MORE was paid than the real vendor_payments ledger accounts for;
-- never invents a negative adjustment when the ledger already has more
-- recorded than the old field (that case needs no backfill -- the ledger
-- is already the more complete number).
INSERT INTO vendor_payments (project_id, project_vendor_id, type, amount, payment_date, method, reference_number, notes)
SELECT
  pv.project_id,
  pv.id,
  'Tambahan',
  (pv.paid_amount - COALESCE(vp.net_paid, 0)),
  CURDATE(),
  'Migrasi data',
  'MIGRASI-000028',
  'Backfill otomatis dari field paid_amount lama (migrasi 000028) -- selisih yang belum tercatat sebagai transaksi individual di vendor_payments.'
FROM project_vendors pv
LEFT JOIN (
  SELECT project_vendor_id,
         SUM(CASE WHEN type = 'Refund' THEN -amount ELSE amount END) AS net_paid
  FROM vendor_payments
  GROUP BY project_vendor_id
) vp ON vp.project_vendor_id = pv.id
WHERE pv.paid_amount > COALESCE(vp.net_paid, 0);

ALTER TABLE project_vendors DROP COLUMN paid_amount;
