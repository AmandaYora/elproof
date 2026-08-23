-- 'projectMilestone' -- lets evidence attach to a Timeline item
-- (project_milestones), same ALTER pattern already used to add
-- 'clientPayment' (000027), 'venuePayment' (000030), and 'general' (000035).
ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment', 'venuePayment', 'general', 'projectMilestone') NOT NULL;
