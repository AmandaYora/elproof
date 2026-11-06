-- Usage caps while a tenant is on its free trial (decision K6, 6 Nov 2026):
-- paid plans are unlimited, plans differ only by duration. A single row,
-- editable from the Platform Console. Owned by `platform`.
CREATE TABLE trial_limits (
  id TINYINT UNSIGNED NOT NULL DEFAULT 1,
  max_staff INT UNSIGNED NOT NULL,
  max_active_projects INT UNSIGNED NOT NULL,
  max_clients INT UNSIGNED NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT chk_trial_limits_single_row CHECK (id = 1)
);

INSERT INTO trial_limits (id, max_staff, max_active_projects, max_clients) VALUES (1, 3, 5, 10);
