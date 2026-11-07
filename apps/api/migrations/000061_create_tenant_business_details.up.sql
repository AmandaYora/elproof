-- What the Owner tells clients about the business beyond name and WhatsApp
-- (PLAN.md Fase 5): the address, and the bank account clients pay into. Kept
-- in its own table so the hot `tenants` row (read on every request for the
-- subscription lifecycle) stays narrow. One row per tenant; absent = nothing set.
CREATE TABLE tenant_business_details (
  tenant_id BIGINT UNSIGNED NOT NULL,
  address VARCHAR(255) NOT NULL DEFAULT '',
  bank_name VARCHAR(60) NOT NULL DEFAULT '',
  bank_account_number VARCHAR(40) NOT NULL DEFAULT '',
  bank_account_holder VARCHAR(100) NOT NULL DEFAULT '',
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (tenant_id),
  CONSTRAINT fk_tenant_business_details_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
