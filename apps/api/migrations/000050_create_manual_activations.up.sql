-- Every manual subscription activation by ElProof's team, with its reason and
-- proof (FLOW-IDEAL.md §1.3 "Aktivasi manual"). Owned by `platform`; the
-- matching ledger row lives in billing's subscription_transactions
-- (transaction_reference), linked by value, not by foreign key.
CREATE TABLE manual_activations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  plan_id BIGINT UNSIGNED NOT NULL,
  actor_admin_id VARCHAR(64) NOT NULL,
  reason VARCHAR(40) NOT NULL,
  proof_reference VARCHAR(255) NOT NULL,
  note VARCHAR(500) NOT NULL DEFAULT '',
  transaction_reference VARCHAR(100) NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_manual_activations_tenant (tenant_id, created_at)
);
