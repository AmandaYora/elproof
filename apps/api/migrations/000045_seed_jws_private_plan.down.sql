-- Identifies the JWS plan by its mapping + seeded values (A26) -- NEVER by
-- name alone, since the JWS plan's name is intentionally identical to
-- plan 1 ("Paket 1 Tahun", 12 months, 2000000). Matching on name would
-- delete ElProof's own public demo plan too (T19).
--
-- Captured to a variable BEFORE any DELETE: plan_id's ON DELETE CASCADE FK
-- (000043) would remove the subscription_plan_apps row the moment the plan
-- itself is deleted, so the id must be resolved first or the mapping used
-- to find it would already be gone.
--
-- If @jws_plan resolves to NULL (e.g. this seed was already removed by
-- hand), both DELETEs below become no-ops -- this file is safe to run twice.
SET @jws_plan := (
  SELECT spa.plan_id FROM subscription_plan_apps spa
  JOIN subscription_plans p ON p.id = spa.plan_id
  WHERE spa.app_id = 'app_02ef90c52704'
    AND p.name = 'Paket 1 Tahun' AND p.duration_months = 12 AND p.price = 2000000
  ORDER BY p.id LIMIT 1
);

DELETE FROM plan_features WHERE plan_id = @jws_plan;
DELETE FROM subscription_plans WHERE id = @jws_plan;
