DELETE FROM tenant_role_permissions WHERE permission IN ('quotation.view', 'quotation.manage');

DROP TABLE IF EXISTS quotation_activity;
DROP TABLE IF EXISTS quotation_revisions;
DROP TABLE IF EXISTS quotation_terms;
DROP TABLE IF EXISTS quotation_lines;
DROP TABLE IF EXISTS quotation_event_sessions;
DROP TABLE IF EXISTS quotation_events;
DROP TABLE IF EXISTS quotation_suppliers;
DROP TABLE IF EXISTS quotation_counters;
DROP TABLE IF EXISTS quotations;
