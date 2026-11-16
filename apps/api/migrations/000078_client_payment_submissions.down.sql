DELETE FROM tenant_role_permissions WHERE permission = 'portal.finance.submit';
-- Submitted payments stay as ordinary client payments.
ALTER TABLE client_payments DROP COLUMN submitted_by_client_id;
