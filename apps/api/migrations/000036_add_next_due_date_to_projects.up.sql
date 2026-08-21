-- Kwitansi & Tagihan PDF feature (PLAN.md) -- the single new column for this
-- whole feature. Set ad-hoc when a Tagihan (bill) is generated with an
-- explicit due date; cleared back to NULL server-side by
-- ClientPaymentService whenever a payment mutation brings the project's
-- outstanding balance to zero or below (see ProjectRepository.SetNextDueDate).
ALTER TABLE projects ADD COLUMN next_due_date DATE NULL AFTER contract_value;
