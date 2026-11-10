-- T-20261110-16 (ADR-0040): every subscription plan belongs to exactly ONE
-- App of the payment registry -- the App that sells it -- replacing the
-- many-to-many visibility mapping of 000043 (subscription_plan_apps).
-- ElProof's own catalog for WO tenants is the plans of its internal App
-- 'platform-billing' (paymentcontracts.InternalAppBilling).
-- app_id is a primitive cross-module ID: no FK to payment_apps, same as
-- payment_charge_dispatch.app_id and subscription_app_charges.app_id.
--
-- Data move (development data only, no compatibility path is kept):
--   * a plan with no mapping (the old "public catalog") -> 'platform-billing'
--   * a plan mapped to one or more Apps -> its first App (MIN(app_id))
ALTER TABLE subscription_plans
  ADD COLUMN app_id VARCHAR(100) NOT NULL DEFAULT 'platform-billing' AFTER id;

UPDATE subscription_plans sp
  JOIN (SELECT plan_id, MIN(app_id) AS app_id FROM subscription_plan_apps GROUP BY plan_id) m ON m.plan_id = sp.id
   SET sp.app_id = m.app_id;

-- No default from here on: every new plan must name its App.
ALTER TABLE subscription_plans
  ALTER COLUMN app_id DROP DEFAULT,
  ADD KEY idx_subscription_plans_app (app_id);

DROP TABLE subscription_plan_apps;
