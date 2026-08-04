-- PLAN.md "Retire the standalone Kendala tab": a kendala can optionally tie
-- to one specific vendor milestone (nullable -- a kendala can also be
-- general to the vendor, not about any one deliverable). Same module owns
-- both tables (vendor_issues, vendor_milestones), so this FK is allowed
-- under the modular-monolith rule.
ALTER TABLE vendor_issues
  ADD COLUMN vendor_milestone_id BIGINT UNSIGNED NULL,
  ADD KEY idx_vendor_issues_vendor_milestone (vendor_milestone_id),
  ADD CONSTRAINT fk_vendor_issues_vendor_milestone
    FOREIGN KEY (vendor_milestone_id) REFERENCES vendor_milestones (id);
