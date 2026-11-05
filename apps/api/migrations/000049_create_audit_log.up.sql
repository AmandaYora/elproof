-- Platform Console audit trail (docs/plan/registrasi-wo-self-service PLAN.md
-- Fase 1): one row per state-changing request by ElProof's own team. Owned
-- by the `audit` module; append-only from the application (no UPDATE or
-- DELETE path exists). Never stores request bodies, so no secret or password
-- can land here.
CREATE TABLE audit_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  occurred_at DATETIME(3) NOT NULL,
  actor_type VARCHAR(30) NOT NULL,
  actor_id VARCHAR(64) NOT NULL,
  actor_role VARCHAR(50) NOT NULL,
  action VARCHAR(120) NOT NULL,
  method VARCHAR(10) NOT NULL,
  path VARCHAR(255) NOT NULL,
  status_code SMALLINT UNSIGNED NOT NULL,
  ip VARCHAR(64) NOT NULL,
  user_agent VARCHAR(255) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  KEY idx_audit_log_occurred_at (occurred_at),
  KEY idx_audit_log_actor (actor_type, actor_id, occurred_at)
);
