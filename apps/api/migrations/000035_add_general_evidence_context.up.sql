-- 'general' -- a document with no specific vendor/payment/vendor-engagement/
-- issue to attach to (rundown, buku acara, teks juru bicara, banquet order,
-- rekap order, rekap dekor, etc.) -- see PLAN.md. Same ALTER pattern already
-- used to add 'clientPayment' (000027) and 'venuePayment' (000030).
ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment', 'venuePayment', 'general') NOT NULL;

-- Only ever meaningful/enforced for related_kind = 'general' -- the other 6
-- kinds keep their existing unconditional client visibility unchanged, this
-- column is never read for them. Defaults to FALSE (safe-by-default): a
-- staff member must explicitly opt a document in before a client sees it.
ALTER TABLE evidence ADD COLUMN is_client_visible BOOLEAN NOT NULL DEFAULT FALSE AFTER related_id;
