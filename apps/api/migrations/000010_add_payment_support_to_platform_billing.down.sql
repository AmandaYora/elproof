ALTER TABLE subscription_transactions
  MODIFY COLUMN status ENUM('unpaid', 'paid', 'expired', 'granted') NOT NULL;

DROP TABLE IF EXISTS pending_subscription_charges;
