-- Reverses the shape of 000088, not its data (pre-launch, dummy data): the
-- lines are dropped and the typed package name and contract value come back
-- empty.
DROP TABLE project_lines;

ALTER TABLE projects
  DROP COLUMN lines_version,
  DROP COLUMN contract_total,
  DROP COLUMN contract_tax,
  DROP COLUMN contract_subtotal,
  DROP COLUMN is_pkp,
  ADD COLUMN package_name VARCHAR(150) NOT NULL DEFAULT '' AFTER prep_start_date,
  ADD COLUMN contract_value BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER package_name;
