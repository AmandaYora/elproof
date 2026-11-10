-- Platform Console "Audit Log" (Tahap 6): the list can be filtered by an exact
-- action label and ordered by it. (action, id) serves both: the equality filter
-- reads only that action's rows, and ORDER BY action, id walks the index instead
-- of sorting the whole append-only table.
ALTER TABLE audit_log ADD KEY idx_audit_log_action (action, id);
