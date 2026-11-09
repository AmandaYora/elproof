-- Tahap 4 / E3: a signed upload names its stored object before any payment
-- exists, so the same upload token could be offered to two payments. One stored
-- object may belong to at most one evidence row; the second insert fails on
-- this index and the payment that tried to use it is rolled back.
--
-- Every key written so far contains a random UUID, so existing rows are
-- distinct. If this migration ever fails on a duplicate, two rows really do
-- share one file: resolve those rows by hand, then run it again.
ALTER TABLE evidence ADD UNIQUE KEY uq_evidence_storage_path (storage_path);
