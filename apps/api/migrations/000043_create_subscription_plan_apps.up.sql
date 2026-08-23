-- App-to-plan visibility mapping (PLAN.md docs/plan/external-subscription-plans).
-- A plan WITH a row here is private to that app_id and hidden from the WO
-- tenant subscription catalog; a plan with NO row is ElProof's own public
-- catalog (demo/internal). app_id is a primitive cross-module ID (A2) --
-- same convention as payment_charge_dispatch.app_id, no FK to payment_apps.
CREATE TABLE subscription_plan_apps (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  plan_id BIGINT UNSIGNED NOT NULL,
  app_id VARCHAR(100) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_subscription_plan_apps (app_id, plan_id),
  KEY idx_subscription_plan_apps_app (app_id),
  CONSTRAINT fk_subscription_plan_apps_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans (id) ON DELETE CASCADE
);
