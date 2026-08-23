ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment', 'venuePayment', 'general') NOT NULL;
