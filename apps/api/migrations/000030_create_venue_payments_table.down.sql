ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment') NOT NULL;

DROP TABLE IF EXISTS venue_payments;
