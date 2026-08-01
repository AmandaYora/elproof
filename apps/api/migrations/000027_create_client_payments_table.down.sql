ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue') NOT NULL;

DROP TABLE IF EXISTS client_payments;
