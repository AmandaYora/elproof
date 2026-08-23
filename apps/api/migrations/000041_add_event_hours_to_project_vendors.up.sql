-- Jam Acara (PLAN.md revisi-timeline-vendor-role-sales): 2 nullable TIME
-- columns on a vendor engagement -- preset (5 options) or custom, optional
-- (analyst decision, confirmed by user).
ALTER TABLE project_vendors
  ADD COLUMN event_start_time TIME NULL AFTER event_date,
  ADD COLUMN event_end_time TIME NULL AFTER event_start_time;
