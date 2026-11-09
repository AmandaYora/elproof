-- Tahap 5 / E13 (docs/plan/revamp-ui-shadcn/PLAN.md): "Urungkan". A payment, a
-- vendor engagement or a client is no longer removed at once: DELETE only stamps
-- deleted_at, every read ignores a stamped row, POST .../restore clears the stamp
-- (within 24 hours), and a scheduled purge (`api purge`, also an in-process
-- ticker) hard-deletes what has been stamped for longer than that, together with
-- its evidence rows and stored files. See ADR-0034.
--
-- DATETIME (not TIMESTAMP): the value is written by the application, so it is the
-- same instant that is sent back as `restorableUntil`. NULL = a live row; every
-- existing row stays NULL. The index serves the purge ("deleted_at < cutoff") and
-- the "which rows are hidden" sub-selects of the evidence queries.
ALTER TABLE client_payments
  ADD COLUMN deleted_at DATETIME NULL,
  ADD KEY idx_client_payments_deleted (deleted_at);

ALTER TABLE vendor_payments
  ADD COLUMN deleted_at DATETIME NULL,
  ADD KEY idx_vendor_payments_deleted (deleted_at);

ALTER TABLE venue_payments
  ADD COLUMN deleted_at DATETIME NULL,
  ADD KEY idx_venue_payments_deleted (deleted_at);

ALTER TABLE project_vendors
  ADD COLUMN deleted_at DATETIME NULL,
  ADD KEY idx_project_vendors_deleted (deleted_at);

ALTER TABLE clients
  ADD COLUMN deleted_at DATETIME NULL,
  ADD KEY idx_clients_deleted (deleted_at);
