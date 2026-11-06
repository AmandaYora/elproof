-- Self-service WO registration (PLAN.md §3, Fase 2). One row per Google
-- account: it records the consent evidence required by UU PDP (which terms
-- version was accepted, when, from which IP) and drives the provisioning
-- saga — tenant_id is stored as soon as the tenant row exists so a retry
-- resumes the same workspace instead of creating a second one. Owned by
-- `platform`; tenant_id is a same-module reference.
CREATE TABLE signup_requests (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  google_sub VARCHAR(255) NOT NULL,
  email VARCHAR(150) NOT NULL,
  owner_name VARCHAR(150) NOT NULL,
  business_name VARCHAR(150) NOT NULL,
  terms_version VARCHAR(20) NOT NULL,
  terms_accepted_at DATETIME NOT NULL,
  marketing_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
  ip_address VARCHAR(45) NOT NULL,
  user_agent VARCHAR(255) NOT NULL,
  status ENUM('provisioning', 'completed') NOT NULL DEFAULT 'provisioning',
  tenant_id BIGINT UNSIGNED NULL,
  completed_at DATETIME NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_signup_requests_google_sub (google_sub),
  KEY idx_signup_requests_tenant (tenant_id),
  CONSTRAINT fk_signup_requests_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- How the tenant came to exist (self-service signup vs. created by ElProof's
-- team) and whether its Owner dismissed the "Mulai di sini" checklist.
ALTER TABLE tenants
  ADD COLUMN signup_source ENUM('platform', 'self_service') NOT NULL DEFAULT 'platform' AFTER custom_domain,
  ADD COLUMN onboarding_dismissed_at DATETIME NULL AFTER signup_source;
