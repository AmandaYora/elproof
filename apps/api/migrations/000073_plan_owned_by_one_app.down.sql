-- Back to the 000043 model: plans of ElProof's own App become the unmapped
-- "public catalog" again; every other plan gets one mapping row to its App.
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

INSERT INTO subscription_plan_apps (plan_id, app_id)
SELECT id, app_id FROM subscription_plans WHERE app_id <> 'platform-billing';

ALTER TABLE subscription_plans
  DROP KEY idx_subscription_plans_app,
  DROP COLUMN app_id;
