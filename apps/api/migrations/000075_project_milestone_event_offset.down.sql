-- Target dates re-computed by the up migration are kept (they are valid dates);
-- only the link to the event date goes.
ALTER TABLE project_milestones DROP COLUMN days_before_event;
