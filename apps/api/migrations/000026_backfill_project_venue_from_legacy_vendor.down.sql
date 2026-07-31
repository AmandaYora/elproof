-- Irreversible by design: the up migration permanently deletes legacy
-- project_vendors rows (and their milestones/payments/issues/evidence) --
-- their original field values are gone, not just hidden, so there is
-- nothing to restore them from. This down migration can only undo the
-- venue_id attachment it added, not bring the deleted rows back.
UPDATE projects p
JOIN activity_log a ON a.project_id = p.id
  AND a.description = 'Venue ditautkan otomatis dari data vendor lama (migrasi pasca ADR-0016)'
SET p.venue_id = NULL;

DELETE FROM activity_log
WHERE description = 'Venue ditautkan otomatis dari data vendor lama (migrasi pasca ADR-0016)';
