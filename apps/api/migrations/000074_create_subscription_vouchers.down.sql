ALTER TABLE subscription_app_charges
  DROP COLUMN voucher_code,
  DROP COLUMN amount,
  DROP COLUMN discount_amount;

ALTER TABLE subscription_transactions
  DROP COLUMN voucher_code,
  DROP COLUMN discount_amount,
  DROP COLUMN original_amount;

DROP TABLE subscription_voucher_redemptions;
DROP TABLE subscription_vouchers;
