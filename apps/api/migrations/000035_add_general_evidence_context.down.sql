ALTER TABLE evidence DROP COLUMN is_client_visible;

ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment', 'venuePayment') NOT NULL;
