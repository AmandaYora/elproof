ALTER TABLE subscription_transactions
  MODIFY COLUMN status ENUM('unpaid', 'pending', 'paid', 'expired', 'granted') NOT NULL;
