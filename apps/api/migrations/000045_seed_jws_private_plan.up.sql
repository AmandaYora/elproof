-- Gives JWS Wedding (app_02ef90c52704) its own plan, separate from
-- ElProof's own public plan 1 -- see PLAN.md D6/A14/A15. Plan 1 stays
-- unmapped (public) so the WO Testing demo tenant keeps seeing it after
-- D3's visibility rule ("mapped = private") takes effect.
--
-- Values copied from plan 1 as it stands today (A15) -- same commercial
-- terms JWS already pays, now on its own row so future price changes to
-- either no longer affect the other. sort_order is 0-based (A30) to match
-- insertFeatures' convention (mysql_plan_repository.go).
INSERT INTO subscription_plans (name, duration_months, price, is_active)
VALUES ('Paket 1 Tahun', 12, 2000000, 1);

SET @jws_plan := LAST_INSERT_ID();

INSERT INTO plan_features (plan_id, label, sort_order) VALUES
  (@jws_plan, 'VPS Regular', 0),
  (@jws_plan, 'Domain Website', 1),
  (@jws_plan, 'SSL Security', 2);

INSERT INTO subscription_plan_apps (plan_id, app_id) VALUES (@jws_plan, 'app_02ef90c52704');
