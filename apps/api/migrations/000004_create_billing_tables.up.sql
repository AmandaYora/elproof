CREATE TABLE subscription_plans (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  duration_months SMALLINT UNSIGNED NOT NULL,
  price BIGINT UNSIGNED NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);

CREATE TABLE plan_features (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  plan_id BIGINT UNSIGNED NOT NULL,
  label VARCHAR(255) NOT NULL,
  sort_order SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_plan_features_plan (plan_id),
  CONSTRAINT fk_plan_features_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans (id)
);

CREATE TABLE subscription_transactions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  type ENUM('new', 'renewal') NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  payment_method VARCHAR(100) NOT NULL,
  payment_reference VARCHAR(100) NOT NULL,
  status ENUM('unpaid', 'paid', 'expired', 'granted') NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paid_at TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_subscription_transactions_tenant (tenant_id)
);
