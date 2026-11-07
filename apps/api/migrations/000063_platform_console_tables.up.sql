-- Platform Console (PLAN.md Fase 6, FLOW-IDEAL 1.2-1.5): internal notes, data
-- requests (UU PDP), announcements, the editable subscription policy, and the
-- support-access consent plus the read-only sessions it allows.

CREATE TABLE tenant_notes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  admin_id VARCHAR(64) NOT NULL,
  admin_name VARCHAR(100) NOT NULL,
  body TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_tenant_notes_tenant (tenant_id, created_at),
  CONSTRAINT fk_tenant_notes_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- A request from a workspace to export or delete its data. The platform team
-- must answer within due_at (30 days); completion records who and what proof.
CREATE TABLE data_requests (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  kind ENUM('export', 'delete') NOT NULL,
  status ENUM('open', 'done', 'rejected') NOT NULL DEFAULT 'open',
  requested_by VARCHAR(200) NOT NULL,
  note TEXT NULL,
  due_at DATETIME NOT NULL,
  created_by_admin VARCHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at DATETIME NULL,
  completed_by VARCHAR(64) NULL,
  completion_note TEXT NULL,
  PRIMARY KEY (id),
  KEY idx_data_requests_status (status, due_at),
  KEY idx_data_requests_tenant (tenant_id),
  CONSTRAINT fk_data_requests_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

CREATE TABLE announcements (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  title VARCHAR(150) NOT NULL,
  body TEXT NOT NULL,
  kind ENUM('info', 'warning', 'maintenance') NOT NULL DEFAULT 'info',
  audience ENUM('all', 'plan', 'status') NOT NULL DEFAULT 'all',
  audience_value VARCHAR(50) NULL,
  starts_at DATETIME NOT NULL,
  ends_at DATETIME NULL,
  created_by VARCHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_announcements_window (starts_at, ends_at)
);

-- One row (id 1): the lifecycle durations in days, editable without a deploy.
CREATE TABLE subscription_policy (
  id TINYINT UNSIGNED NOT NULL,
  trial_days INT NOT NULL,
  expiring_soon_days INT NOT NULL,
  grace_days INT NOT NULL,
  read_only_days INT NOT NULL,
  updated_by VARCHAR(64) NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);
INSERT INTO subscription_policy (id, trial_days, expiring_soon_days, grace_days, read_only_days) VALUES (1, 14, 7, 7, 23);

-- The Owner's consent for ElProof's support team to look at the workspace.
CREATE TABLE support_access_grants (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  granted_by_staff_id BIGINT UNSIGNED NOT NULL,
  granted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME NOT NULL,
  revoked_at DATETIME NULL,
  PRIMARY KEY (id),
  KEY idx_support_access_tenant (tenant_id, expires_at),
  CONSTRAINT fk_support_access_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);

-- Every "Lihat sebagai tenant" session: who, why, how long. Visible to the Owner.
CREATE TABLE support_sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  admin_id VARCHAR(64) NOT NULL,
  admin_name VARCHAR(100) NOT NULL,
  reason VARCHAR(255) NOT NULL,
  started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME NOT NULL,
  ended_at DATETIME NULL,
  PRIMARY KEY (id),
  KEY idx_support_sessions_tenant (tenant_id, started_at),
  CONSTRAINT fk_support_sessions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
