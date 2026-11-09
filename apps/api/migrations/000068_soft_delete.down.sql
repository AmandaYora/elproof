-- Rolling back to the code that deletes at once: whatever is still waiting in
-- the "deleted" state is deleted for real first (it was deleted from the user's
-- point of view; dropping the column would bring it back to life, money
-- included). Evidence rows hang off these items by (related_kind, related_id)
-- without a foreign key, so they are removed explicitly; their stored files are
-- left in the bucket (an orphan costs bytes, never correctness). Order matters:
-- vendor_issues -> vendor_milestones/vendor_payments -> project_vendors.
DELETE FROM evidence WHERE related_kind = 'clientPayment'
  AND related_id IN (SELECT id FROM client_payments WHERE deleted_at IS NOT NULL);
DELETE FROM client_payments WHERE deleted_at IS NOT NULL;

DELETE FROM evidence WHERE related_kind = 'venuePayment'
  AND related_id IN (SELECT id FROM venue_payments WHERE deleted_at IS NOT NULL);
DELETE FROM venue_payments WHERE deleted_at IS NOT NULL;

DELETE FROM evidence WHERE related_kind = 'payment' AND related_id IN (
  SELECT id FROM vendor_payments WHERE deleted_at IS NOT NULL
     OR project_vendor_id IN (SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL));
DELETE FROM evidence WHERE related_kind = 'issue' AND related_id IN (
  SELECT id FROM vendor_issues WHERE project_vendor_id IN (SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL));
DELETE FROM evidence WHERE related_kind = 'vendorMilestone' AND related_id IN (
  SELECT vm.id FROM vendor_milestones vm JOIN project_vendors pv ON pv.id = vm.project_vendor_id WHERE pv.deleted_at IS NOT NULL);
DELETE FROM evidence WHERE related_kind = 'projectVendor'
  AND related_id IN (SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL);
DELETE FROM vendor_issues WHERE project_vendor_id IN (SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL);
DELETE FROM vendor_payments WHERE deleted_at IS NOT NULL
  OR project_vendor_id IN (SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL);
DELETE FROM vendor_milestones WHERE project_vendor_id IN (SELECT id FROM project_vendors WHERE deleted_at IS NOT NULL);
DELETE FROM project_vendors WHERE deleted_at IS NOT NULL;

-- A deleted client keeps a deactivated login in `identity` (outside this module's
-- tables); removing the row leaves that login pointing at nothing, which the
-- portal already treats as "no access".
DELETE FROM clients WHERE deleted_at IS NOT NULL;

ALTER TABLE clients DROP KEY idx_clients_deleted, DROP COLUMN deleted_at;
ALTER TABLE project_vendors DROP KEY idx_project_vendors_deleted, DROP COLUMN deleted_at;
ALTER TABLE venue_payments DROP KEY idx_venue_payments_deleted, DROP COLUMN deleted_at;
ALTER TABLE vendor_payments DROP KEY idx_vendor_payments_deleted, DROP COLUMN deleted_at;
ALTER TABLE client_payments DROP KEY idx_client_payments_deleted, DROP COLUMN deleted_at;
