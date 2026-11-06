ALTER TABLE tenants
  DROP COLUMN onboarding_dismissed_at,
  DROP COLUMN signup_source;
DROP TABLE signup_requests;
