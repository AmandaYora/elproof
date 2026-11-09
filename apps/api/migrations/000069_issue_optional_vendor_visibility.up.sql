-- Tahap 5 / E13 (docs/plan/revamp-ui-shadcn/PLAN.md): a kendala can be about the
-- event in general (no vendor), and staff decide whether the couple sees it.
--
-- project_vendor_id becomes NULLable (the foreign key to project_vendors stays:
-- same module). is_client_visible is NOT NULL DEFAULT 1, so every existing row
-- keeps being shown to its client exactly as before.
ALTER TABLE vendor_issues
  MODIFY COLUMN project_vendor_id BIGINT UNSIGNED NULL,
  ADD COLUMN is_client_visible BOOLEAN NOT NULL DEFAULT TRUE;
