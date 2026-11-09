-- Tahap 4 / E3, E7 (docs/plan/revamp-ui-shadcn/PLAN.md): replay protection for
-- the business calls that create money or a project (POST client-payments,
-- payments, venue-payments and POST /projects). A client sends an
-- `Idempotency-Key` header; the first request with that key reserves a row
-- (status_code NULL = still running), and when it succeeds its response is
-- stored here, so a retry (a dropped connection, a double tap) returns the same
-- answer instead of recording the payment twice.
--
-- Owned by `projects`. tenant_id/staff_id are primitive references into other
-- modules: no foreign key. Rows are deleted after 24 hours by the application
-- (opportunistically, on each reservation); created_at is indexed for that.
-- idempotency_key and route are compared byte for byte (utf8mb4_bin): a key is
-- an opaque token, "Abc" and "abc" are different keys.
CREATE TABLE idempotency_keys (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  staff_id BIGINT UNSIGNED NOT NULL,
  route VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  idempotency_key VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  request_hash CHAR(64) NOT NULL,
  status_code SMALLINT UNSIGNED NULL,
  response_body MEDIUMTEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_idempotency_tenant_key_route (tenant_id, idempotency_key, route),
  KEY idx_idempotency_created (created_at)
);
